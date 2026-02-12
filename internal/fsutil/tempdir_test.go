package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTempDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	dir, err := CreateTempDir(parent)
	if err != nil {
		t.Fatalf("CreateTempDir returned error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat temp dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("temp path %q is not a directory", dir)
	}

	rel, err := filepath.Rel(parent, dir)
	if err != nil {
		t.Fatalf("filepath.Rel failed: %v", err)
	}
	if strings.HasPrefix(rel, "..") {
		t.Fatalf("temp dir %q not created under parent %q", dir, parent)
	}
}

func TestCreateTempDirRequiresParent(t *testing.T) {
	t.Parallel()

	if _, err := CreateTempDir(""); err == nil {
		t.Fatal("expected error when parent is empty")
	}
}
