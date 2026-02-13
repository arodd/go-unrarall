package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arodd/go-unrarall/internal/rar"
)

func TestAlreadyExtractedFromListedFullPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "clip.mkv"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "info.nfo"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write root file: %v", err)
	}

	ok, err := alreadyExtractedFromListed(root, []rar.ListedFile{
		{Name: "nested/clip.mkv"},
		{Name: "info.nfo"},
	}, true)
	if err != nil {
		t.Fatalf("alreadyExtractedFromListed returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected all files to exist in full-path mode")
	}
}

func TestAlreadyExtractedFromListedFlatten(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "clip.mkv"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ok, err := alreadyExtractedFromListed(root, []rar.ListedFile{
		{Name: "nested/clip.mkv"},
	}, false)
	if err != nil {
		t.Fatalf("alreadyExtractedFromListed returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected flatten mode basename existence check to pass")
	}
}

func TestAlreadyExtractedFromListedMissingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ok, err := alreadyExtractedFromListed(root, []rar.ListedFile{
		{Name: "missing.bin"},
	}, true)
	if err != nil {
		t.Fatalf("alreadyExtractedFromListed returned error: %v", err)
	}
	if ok {
		t.Fatal("expected missing file to report not already extracted")
	}
}

func TestAlreadyExtractedFromListedUnsafePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ok, err := alreadyExtractedFromListed(root, []rar.ListedFile{
		{Name: "../escape.txt"},
	}, true)
	if err != nil {
		t.Fatalf("alreadyExtractedFromListed returned error: %v", err)
	}
	if ok {
		t.Fatal("expected unsafe relative path to report not already extracted")
	}
}
