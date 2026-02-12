package cli

import (
	"path/filepath"
	"testing"
)

func TestParseArgsSecurityDefaults(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	opts, err := ParseArgs([]string{"unrarall", root})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	if opts.MaxDictBytes != 1<<30 {
		t.Fatalf("MaxDictBytes=%d, want %d", opts.MaxDictBytes, int64(1<<30))
	}
	if opts.AllowSymlinks {
		t.Fatal("expected AllowSymlinks to default to false")
	}
}

func TestParseArgsAllowSymlinks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	opts, err := ParseArgs([]string{"unrarall", "--allow-symlinks", root})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if !opts.AllowSymlinks {
		t.Fatal("expected --allow-symlinks to set AllowSymlinks=true")
	}
}

func TestParseArgsRejectsNonPositiveMaxDict(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := ParseArgs([]string{"unrarall", "--max-dict", "0", root})
	if err == nil {
		t.Fatal("expected max-dict validation error")
	}
}

func TestParseArgsResolvesOutputDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	output := t.TempDir()

	opts, err := ParseArgs([]string{"unrarall", "--output", output, root})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if opts.OutputDir != filepath.Clean(output) {
		t.Fatalf("OutputDir=%q, want %q", opts.OutputDir, filepath.Clean(output))
	}
}
