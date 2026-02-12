package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSafeMoveRenamesFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "source.txt")
	dst := filepath.Join(root, "dest.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	final, err := SafeMove(src, dst)
	if err != nil {
		t.Fatalf("SafeMove returned error: %v", err)
	}
	if final != dst {
		t.Fatalf("SafeMove final path=%q, want %q", final, dst)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists after move: %v", err)
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("destination content=%q, want %q", string(content), "hello")
	}
}

func TestSafeMoveAddsSuffixWhenDestinationExists(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "source.txt")
	dst := filepath.Join(root, "dest.txt")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("write existing dest: %v", err)
	}

	final, err := SafeMove(src, dst)
	if err != nil {
		t.Fatalf("SafeMove returned error: %v", err)
	}
	want := dst + ".1"
	if final != want {
		t.Fatalf("SafeMove final=%q, want %q", final, want)
	}
	content, err := os.ReadFile(final)
	if err != nil {
		t.Fatalf("read suffixed destination: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("destination content=%q, want %q", string(content), "new")
	}
}

func TestSafeMoveFallsBackToCopyOnEXDEV(t *testing.T) {
	originalRename := renamePath
	renamePath = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}
	t.Cleanup(func() {
		renamePath = originalRename
	})

	root := t.TempDir()
	src := filepath.Join(root, "source.txt")
	dst := filepath.Join(root, "dest.txt")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	final, err := SafeMove(src, dst)
	if err != nil {
		t.Fatalf("SafeMove returned error: %v", err)
	}
	if final != dst {
		t.Fatalf("SafeMove final path=%q, want %q", final, dst)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists after fallback move: %v", err)
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(content) != "payload" {
		t.Fatalf("destination content=%q, want %q", string(content), "payload")
	}
}

func TestSafeMoveDirectoryFallbackOnEXDEV(t *testing.T) {
	originalRename := renamePath
	renamePath = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}
	t.Cleanup(func() {
		renamePath = originalRename
	})

	root := t.TempDir()
	src := filepath.Join(root, "srcdir")
	dst := filepath.Join(root, "destdir")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir source tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("dir-data"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	final, err := SafeMove(src, dst)
	if err != nil {
		t.Fatalf("SafeMove returned error: %v", err)
	}
	if final != dst {
		t.Fatalf("SafeMove final path=%q, want %q", final, dst)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source directory still exists: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dst, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read copied nested file: %v", err)
	}
	if got := string(content); got != "dir-data" {
		t.Fatalf("nested file content=%q, want %q", got, "dir-data")
	}
}

func TestSafeMoveSourceMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := SafeMove(filepath.Join(root, "missing.txt"), filepath.Join(root, "dest.txt"))
	if err == nil {
		t.Fatal("expected error for missing source")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "no such file") {
		t.Fatalf("expected missing file error, got %v", err)
	}
}
