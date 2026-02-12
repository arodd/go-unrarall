package fsutil

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var renamePath = os.Rename

const copyBufferSize = 256 * 1024

// SafeMove moves src to dst. If dst already exists, a .N suffix is appended
// until a free path is found. Cross-device moves fall back to copy+remove.
func SafeMove(src, dst string) (string, error) {
	if _, err := os.Stat(src); err != nil {
		return "", err
	}

	for attempt := 0; ; attempt++ {
		candidate := dst
		if attempt > 0 {
			candidate = fmt.Sprintf("%s.%d", dst, attempt)
		}

		_, err := os.Stat(candidate)
		switch {
		case err == nil:
			continue
		case err != nil && !os.IsNotExist(err):
			return "", err
		}

		err = renamePath(src, candidate)
		switch {
		case err == nil:
			return candidate, nil
		case isCrossDeviceError(err):
			if err := copyPath(src, candidate); err != nil {
				return "", err
			}
			if err := os.RemoveAll(src); err != nil {
				return "", err
			}
			return candidate, nil
		case os.IsExist(err):
			continue
		default:
			return "", err
		}
	}
}

func isCrossDeviceError(err error) bool {
	if errors.Is(err, syscall.EXDEV) {
		return true
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return errors.Is(linkErr.Err, syscall.EXDEV)
	}
	return false
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink copy is not supported for %q", src)
	}

	if info.IsDir() {
		return copyDir(src, dst, info)
	}
	return copyFile(src, dst, info)
}

func copyDir(src, dst string, rootInfo fs.FileInfo) error {
	if err := os.Mkdir(dst, dirPerm(rootInfo.Mode())); err != nil {
		return err
	}
	if !rootInfo.ModTime().IsZero() {
		_ = os.Chtimes(dst, time.Now(), rootInfo.ModTime())
	}

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == src {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		entryInfo, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			if err := os.Mkdir(target, dirPerm(entryInfo.Mode())); err != nil {
				return err
			}
			if !entryInfo.ModTime().IsZero() {
				_ = os.Chtimes(target, time.Now(), entryInfo.ModTime())
			}
			return nil
		}

		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink copy is not supported for %q", path)
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type %q", path)
		}
		return copyFile(path, target, entryInfo)
	})
	if err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return nil
}

func copyFile(src, dst string, info fs.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm(info.Mode()))
	if err != nil {
		return err
	}

	buf := make([]byte, copyBufferSize)
	_, copyErr := io.CopyBuffer(out, in, buf)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if syncErr != nil {
		_ = os.Remove(dst)
		return syncErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	if !info.ModTime().IsZero() {
		_ = os.Chtimes(dst, time.Now(), info.ModTime())
	}
	return nil
}

func filePerm(mode fs.FileMode) fs.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0o644
	}
	return perm
}

func dirPerm(mode fs.FileMode) fs.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0o755
	}
	return perm
}
