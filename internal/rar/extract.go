package rar

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/austin/go-unrarall/internal/fsutil"
	"github.com/nwaples/rardecode/v2"
)

const extractCopyBufferSize = 256 * 1024

type archiveReader interface {
	Next() (*rardecode.FileHeader, error)
	io.Reader
}

// ExtractToDir streams an archive into tmpDir. It supports full-path and
// flattened extraction modes and returns the volume paths consumed by the
// archive reader.
func ExtractToDir(archivePath, tmpDir string, fullPath bool, opts ...rardecode.Option) ([]string, error) {
	return extractToDirWithOpener(openArchiveReader, archivePath, tmpDir, fullPath, opts...)
}

// ExtractToDirWithSettings is a convenience wrapper around ExtractToDir that
// converts OpenSettings into decoder options.
func ExtractToDirWithSettings(archivePath, tmpDir string, fullPath bool, settings OpenSettings) ([]string, error) {
	return ExtractToDir(archivePath, tmpDir, fullPath, settings.DecodeOptions()...)
}

func extractToDirWithOpener(
	opener openReaderFunc,
	archivePath string,
	tmpDir string,
	fullPath bool,
	opts ...rardecode.Option,
) ([]string, error) {
	reader, err := opener(archivePath, opts...)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if err := extractFromArchiveReader(reader, tmpDir, fullPath); err != nil {
		return nil, err
	}
	return reader.Volumes(), nil
}

func extractFromArchiveReader(reader archiveReader, tmpDir string, fullPath bool) error {
	buf := make([]byte, extractCopyBufferSize)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a symlink and is not supported", header.Name)
		}

		relPath, err := entryPath(header, fullPath)
		if err != nil {
			return err
		}
		if relPath == "" {
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
