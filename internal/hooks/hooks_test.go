package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRARRemovesMatchingVolumes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	rarDir := filepath.Join(root, "rar")
	if err := os.MkdirAll(rarDir, 0o755); err != nil {
		t.Fatalf("mkdir rar dir: %v", err)
	}

	create := func(name string) {
		path := filepath.Join(rarDir, name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	create("release.sfv")
	create("release.001")
	create("release.r00")
	create("release.part01.rar")
	create("release.RAR")
	create("release.txt")
	create("other.rar")

	err := Run([]string{"rar"}, Context{
		ExtractRoot: root,
		RarDir:      rarDir,
		Stem:        "release",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, name := range []string{
		"release.sfv",
		"release.001",
		"release.r00",
		"release.part01.rar",
		"release.RAR",
	} {
		if _, err := os.Stat(filepath.Join(rarDir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected %q to be removed, stat err=%v", name, err)
		}
	}
	for _, name := range []string{"release.txt", "other.rar"} {
		if _, err := os.Stat(filepath.Join(rarDir, name)); err != nil {
			t.Fatalf("expected %q to remain, stat err=%v", name, err)
		}
	}
}

func TestRunSampleVideosRootOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	makeFile := func(path string) {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	makeFile(filepath.Join(root, "sample.release.mkv"))
	makeFile(filepath.Join(root, "sample-other.txt"))
	makeFile(filepath.Join(subDir, "sample.release.mkv"))

	err := Run([]string{"sample_videos"}, Context{
		ExtractRoot: root,
		RarDir:      root,
		Stem:        "release",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "sample.release.mkv")); !os.IsNotExist(err) {
		t.Fatalf("expected root sample video to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "sample-other.txt")); err != nil {
		t.Fatalf("expected non-matching root file to remain, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(subDir, "sample.release.mkv")); err != nil {
		t.Fatalf("expected nested sample video to remain, stat err=%v", err)
	}
}

func TestRunNamedFolderHooks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, rel := range []string{
		filepath.Join("nested", "covers", "cover.jpg"),
		filepath.Join("nested", "proof", "proof.png"),
		filepath.Join("nested", "sample", "clip.mkv"),
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	err := Run([]string{"covers_folders", "proof_folders", "sample_folders"}, Context{
		ExtractRoot: root,
		RarDir:      root,
		Stem:        "release",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, rel := range []string{
		filepath.Join("nested", "covers"),
		filepath.Join("nested", "proof"),
		filepath.Join("nested", "sample"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected %q to be removed, stat err=%v", rel, err)
		}
	}
}

func TestRunEmptyFoldersRemovesNestedEmptyDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	rarDir := filepath.Join(root, "rar")
	if err := os.MkdirAll(filepath.Join(rarDir, "a", "b"), 0o755); err != nil {
		t.Fatalf("mkdir nested empty dirs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rarDir, "has-files"), 0o755); err != nil {
		t.Fatalf("mkdir has-files: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rarDir, "has-files", "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}

	err := Run([]string{"empty_folders"}, Context{
		ExtractRoot: root,
		RarDir:      rarDir,
		Stem:        "release",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(rarDir, "a")); !os.IsNotExist(err) {
		t.Fatalf("expected empty subtree to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(rarDir, "has-files")); err != nil {
		t.Fatalf("expected non-empty dir to remain, stat err=%v", err)
	}
}
