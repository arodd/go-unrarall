package rar

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/arodd/go-unrarall/internal/fsutil"
	"github.com/nwaples/rardecode/v2"
)

const extractCopyBufferSize = 256 * 1024
const maxSymlinkTargetBytes = 4096

type archiveReader interface {
	Next() (*rardecode.FileHeader, error)
	io.Reader
}

// ExtractToDir streams an archive into tmpDir. It supports full-path and
// flattened extraction modes and returns the volume paths consumed by the
// archive reader.
func ExtractToDir(
	archivePath,
	tmpDir string,
	fullPath bool,
	allowSymlinks bool,
	opts ...rardecode.Option,
) ([]string, error) {
	return extractToDirWithOpener(openArchiveReader, archivePath, tmpDir, fullPath, allowSymlinks, opts...)
}

// ExtractToDirWithSettings is a convenience wrapper around ExtractToDir that
// converts OpenSettings into decoder options.
func ExtractToDirWithSettings(archivePath, tmpDir string, fullPath bool, settings OpenSettings) ([]string, error) {
	return ExtractToDir(archivePath, tmpDir, fullPath, settings.AllowSymlinks, settings.DecodeOptions()...)
}

func extractToDirWithOpener(
	opener openReaderFunc,
	archivePath string,
	tmpDir string,
	fullPath bool,
	allowSymlinks bool,
	opts ...rardecode.Option,
) ([]string, error) {
	reader, err := opener(archivePath, opts...)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if err := extractFromArchiveReader(reader, tmpDir, fullPath, allowSymlinks); err != nil {
		return nil, err
	}
	return reader.Volumes(), nil
}

func extractFromArchiveReader(reader archiveReader, tmpDir string, fullPath bool, allowSymlinks bool) error {
	buf := make([]byte, extractCopyBufferSize)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		relPath, err := entryPath(header, fullPath)
		if err != nil {
			return err
		}
		if relPath == "" {
			continue
		}

		if header.Mode()&os.ModeSymlink != 0 {
			if !allowSymlinks {
				return fmt.Errorf(
					"archive entry %q is a symlink and symlink extraction is disabled (use --allow-symlinks to override)",
					header.Name,
				)
			}
			if err := extractSymlink(reader, tmpDir, relPath); err != nil {
				return fmt.Errorf("extract symlink %q: %w", header.Name, err)
			}
			continue
		}

		target := filepath.Join(tmpDir, relPath)

		if header.IsDir {
			if err := os.MkdirAll(target, dirModeForHeader(header)); err != nil {
				return err
			}
			applyModTime(target, header.ModificationTime)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileModeForHeader(header))
		if err != nil {
			return err
		}

		_, copyErr := io.CopyBuffer(out, reader, buf)
		syncErr := out.Sync()
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(target)
			return fmt.Errorf("extract %q: %w", header.Name, copyErr)
		}
		if syncErr != nil {
			_ = os.Remove(target)
			return syncErr
		}
		if closeErr != nil {
			_ = os.Remove(target)
			return closeErr
		}

		applyModTime(target, header.ModificationTime)
	}
	return nil
}

func extractSymlink(reader io.Reader, tmpDir, relPath string) error {
	targetValue, err := decodeSymlinkTarget(reader)
	if err != nil {
		return err
	}

	linkTarget, err := sanitizeSymlinkTarget(relPath, targetValue)
	if err != nil {
		return err
	}

	linkPath := filepath.Join(tmpDir, relPath)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(linkTarget, linkPath); err != nil {
		return err
	}
	return nil
}

func decodeSymlinkTarget(reader io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxSymlinkTargetBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > maxSymlinkTargetBytes {
		return "", fmt.Errorf("symlink target exceeds %d bytes", maxSymlinkTargetBytes)
	}

	value := strings.TrimRight(string(raw), "\x00")
	if value == "" {
		return "", fmt.Errorf("symlink target is empty")
	}
	if strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("symlink target contains NUL")
	}
	return value, nil
}

func sanitizeSymlinkTarget(linkRelPath, rawTarget string) (string, error) {
	normalized := strings.ReplaceAll(rawTarget, "\\", "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == ".." || cleaned == "/" {
		return "", fmt.Errorf("unsafe symlink target %q", rawTarget)
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("absolute symlink target %q is not allowed", rawTarget)
	}
	if hasDrivePrefix(cleaned) {
		return "", fmt.Errorf("symlink target %q has a drive prefix", rawTarget)
	}

	baseDir := path.Dir(filepath.ToSlash(linkRelPath))
	resolved := path.Clean(path.Join(baseDir, cleaned))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("symlink target %q escapes extraction root", rawTarget)
	}
	return filepath.FromSlash(cleaned), nil
}

func hasDrivePrefix(pathValue string) bool {
	if len(pathValue) < 2 || pathValue[1] != ':' {
		return false
	}
	return unicode.IsLetter(rune(pathValue[0]))
}

func entryPath(header *rardecode.FileHeader, fullPath bool) (string, error) {
	normalized := strings.ReplaceAll(header.Name, "\\", "/")

	if !fullPath {
		// Flatten mode mirrors "unrar e": files land at the extraction root.
		if header.IsDir {
			return "", nil
		}
		normalized = path.Base(normalized)
	}

	sanitized, ok := fsutil.SanitizeRelPath(normalized)
	if !ok {
		return "", fmt.Errorf("unsafe path in archive: %q", header.Name)
	}
	return sanitized, nil
}

func fileModeForHeader(header *rardecode.FileHeader) os.FileMode {
	perm := header.Mode().Perm()
	if perm == 0 {
		return 0o644
	}
	return perm
}

func dirModeForHeader(header *rardecode.FileHeader) os.FileMode {
	perm := header.Mode().Perm()
	if perm == 0 {
		return 0o755
	}
	return perm
}

func applyModTime(path string, modTime time.Time) {
	if modTime.IsZero() {
		return
	}
	_ = os.Chtimes(path, time.Now(), modTime)
}
