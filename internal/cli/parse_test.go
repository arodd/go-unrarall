package cli

import (
	"path/filepath"
	"strings"
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

func TestParseArgsResolvesLogFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	logPath := filepath.Join("logs", "unrarall.log")

	opts, err := ParseArgs([]string{"unrarall", "--log-file", logPath, root})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	absLogPath, err := filepath.Abs(logPath)
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}
	if opts.LogFile != absLogPath {
		t.Fatalf("LogFile=%q, want %q", opts.LogFile, absLogPath)
	}
}

func TestParseArgsResolvesLogFileForHelpMode(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join("logs", "help.log")
	opts, err := ParseArgs([]string{"unrarall", "--help", "--log-file", logPath})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	absLogPath, err := filepath.Abs(logPath)
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}
	if opts.LogFile != absLogPath {
		t.Fatalf("LogFile=%q, want %q", opts.LogFile, absLogPath)
	}
}

func TestParseArgsRejectsEmptyLogFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := ParseArgs([]string{"unrarall", "--log-file=", root})
	if err == nil {
		t.Fatal("expected log-file validation error")
	}
	if !strings.Contains(err.Error(), "--log-file requires FILE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseArgsRejectsMissingLogFileValue(t *testing.T) {
	t.Parallel()

	_, err := ParseArgs([]string{"unrarall", "--log-file"})
	if err == nil {
		t.Fatal("expected missing log-file value error")
	}
}

func TestParseArgsQuietOverridesVerbose(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "verbose then quiet",
			args: []string{"unrarall", "--verbose", "--quiet", root},
		},
		{
			name: "quiet then verbose",
			args: []string{"unrarall", "--quiet", "--verbose", root},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts, err := ParseArgs(tc.args)
			if err != nil {
				t.Fatalf("ParseArgs returned error: %v", err)
			}
			if !opts.Quiet {
				t.Fatal("expected Quiet=true")
			}
			if opts.Verbose {
				t.Fatal("expected Verbose=false when quiet is enabled")
			}
		})
	}
}

func TestParseArgsRejectsEmptyCleanSpec(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := ParseArgs([]string{"unrarall", "--clean=", root})
	if err == nil {
		t.Fatal("expected empty --clean error")
	}
	if !strings.Contains(err.Error(), "clean up hooks must be specified when using --clean=") {
		t.Fatalf("unexpected error: %v", err)
	}
}
