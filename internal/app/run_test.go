package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/austin/go-unrarall/internal/cli"
	"github.com/austin/go-unrarall/internal/finder"
	"github.com/austin/go-unrarall/internal/log"
	"github.com/nwaples/rardecode/v2"
)

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		stats         Stats
		allowFailures bool
		want          int
	}{
		{
			name:  "success without failures",
			stats: Stats{ArchivesExtracted: 2, Failures: 0},
			want:  0,
		},
		{
			name:          "failures without allow failures",
			stats:         Stats{ArchivesExtracted: 2, Failures: 1},
			allowFailures: false,
			want:          1,
		},
		{
			name:          "allow failures with successes",
			stats:         Stats{ArchivesExtracted: 1, Failures: 2},
			allowFailures: true,
			want:          0,
		},
		{
			name:          "allow failures without successes",
			stats:         Stats{ArchivesExtracted: 0, Failures: 2},
			allowFailures: true,
			want:          1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := ExitCode(tc.stats, tc.allowFailures); got != tc.want {
				t.Fatalf("ExitCode()=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestRunRecursesIntoNestedArchives(t *testing.T) {
	root := t.TempDir()
	topArchive := filepath.Join(root, "top.rar")
	if err := os.WriteFile(topArchive, []byte("x"), 0o644); err != nil {
		t.Fatalf("write top archive: %v", err)
	}

	var topTmpDir string
	var nestedTmpDir string
	scanCalls := 0

	restore := stubRunDependencies()
	defer restore()

	validateRarSignature = func(path string) (bool, error) {
		return true, nil
	}
	scanCandidates = func(dir string, depth int) ([]finder.Candidate, error) {
		scanCalls++
		switch dir {
		case root:
			if depth != 1 {
				t.Fatalf("top-level depth=%d, want 1", depth)
			}
			return []finder.Candidate{{Path: topArchive, Stem: "top"}}, nil
		case topTmpDir:
			if depth != 0 {
				t.Fatalf("nested depth=%d, want 0", depth)
			}
			return []finder.Candidate{{Path: filepath.Join(topTmpDir, "nested.rar"), Stem: "nested"}}, nil
		default:
			return nil, nil
		}
	}
	createExtractionTempDir = func(parent string) (string, error) {
		switch parent {
		case root:
			topTmpDir = filepath.Join(root, ".tmp-top")
			if err := os.MkdirAll(topTmpDir, 0o755); err != nil {
				return "", err
			}
			return topTmpDir, nil
		case topTmpDir:
			nestedTmpDir = filepath.Join(topTmpDir, ".tmp-nested")
			if err := os.MkdirAll(nestedTmpDir, 0o755); err != nil {
				return "", err
			}
			return nestedTmpDir, nil
		default:
			return "", errors.New("unexpected temp parent")
		}
	}
	extractArchiveWithRetries = func(
		archivePath string,
		tmpDir string,
		_ bool,
		_ int64,
		_ string,
	) (PasswordExtractionResult, error) {
		switch archivePath {
		case topArchive:
			if err := os.WriteFile(filepath.Join(tmpDir, "nested.rar"), []byte("x"), 0o644); err != nil {
				return PasswordExtractionResult{}, err
			}
			if err := os.WriteFile(filepath.Join(tmpDir, "top.txt"), []byte("top"), 0o644); err != nil {
				return PasswordExtractionResult{}, err
			}
			return PasswordExtractionResult{Volumes: []string{archivePath}}, nil
		case filepath.Join(topTmpDir, "nested.rar"):
			if err := os.WriteFile(filepath.Join(tmpDir, "nested.txt"), []byte("nested"), 0o644); err != nil {
				return PasswordExtractionResult{}, err
			}
			return PasswordExtractionResult{Volumes: []string{archivePath}}, nil
		default:
			return PasswordExtractionResult{}, errors.New("unexpected archive path")
		}
	}
	checkAlreadyExtracted = func(_ string, _ string, _ bool, _ ...rardecode.Option) (bool, error) {
		return false, nil
	}
	runCleanupSelection = func(_ []string, _ string, _ string, _ string, _ bool, _ *log.Logger) error {
		return nil
	}

	opts := cli.Options{
		Dir:          root,
		Depth:        1,
		CKSFV:        false,
		CleanHooks:   []string{"none"},
		MaxDictBytes: 1 << 20,
		PasswordFile: filepath.Join(root, "passwords.txt"),
	}

	stats, err := Run(opts, log.New(true, false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if scanCalls < 2 {
		t.Fatalf("expected recursive scan calls, got %d", scanCalls)
	}
	if stats.ArchivesFound != 2 {
		t.Fatalf("ArchivesFound=%d, want 2", stats.ArchivesFound)
	}
	if stats.ArchivesExtracted != 2 {
		t.Fatalf("ArchivesExtracted=%d, want 2", stats.ArchivesExtracted)
	}
	if stats.Failures != 0 {
		t.Fatalf("Failures=%d, want 0", stats.Failures)
	}

	if _, err := os.Stat(filepath.Join(root, "top.txt")); err != nil {
		t.Fatalf("expected top payload moved into destination root, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "nested.txt")); err != nil {
		t.Fatalf("expected nested payload moved into destination root, stat err=%v", err)
	}
}

func TestRunForceStillRunsHooksAfterExtractionFailure(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "broken.rar")
	if err := os.WriteFile(archivePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	hookCalls := 0

	restore := stubRunDependencies()
	defer restore()

	scanCandidates = func(dir string, depth int) ([]finder.Candidate, error) {
		return []finder.Candidate{{Path: archivePath, Stem: "broken"}}, nil
	}
	validateRarSignature = func(path string) (bool, error) {
		return true, nil
	}
	createExtractionTempDir = func(parent string) (string, error) {
		return os.MkdirTemp(parent, ".tmp-")
	}
	extractArchiveWithRetries = func(
		archivePath string,
		tmpDir string,
		_ bool,
		_ int64,
		_ string,
	) (PasswordExtractionResult, error) {
		return PasswordExtractionResult{}, errors.New("decode failed")
	}
	runCleanupSelection = func(_ []string, _ string, _ string, _ string, _ bool, _ *log.Logger) error {
		hookCalls++
		return nil
	}

	opts := cli.Options{
		Dir:          root,
		Depth:        0,
		Force:        true,
		CKSFV:        false,
		CleanHooks:   []string{"nfo"},
		MaxDictBytes: 1 << 20,
		PasswordFile: filepath.Join(root, "passwords.txt"),
	}

	stats, err := Run(opts, log.New(true, false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if hookCalls != 1 {
		t.Fatalf("hook calls=%d, want 1", hookCalls)
	}
	if stats.Failures != 1 {
		t.Fatalf("Failures=%d, want 1", stats.Failures)
	}
	if stats.ArchivesExtracted != 0 {
		t.Fatalf("ArchivesExtracted=%d, want 0", stats.ArchivesExtracted)
	}
}

func stubRunDependencies() func() {
	oldScanCandidates := scanCandidates
	oldValidateRarSignature := validateRarSignature
	oldCreateExtractionTempDir := createExtractionTempDir
	oldExtractArchiveWithRetries := extractArchiveWithRetries
	oldCheckAlreadyExtracted := checkAlreadyExtracted
	oldSafeMovePath := safeMovePath
	oldRunCleanupSelection := runCleanupSelection

	return func() {
		scanCandidates = oldScanCandidates
		validateRarSignature = oldValidateRarSignature
		createExtractionTempDir = oldCreateExtractionTempDir
		extractArchiveWithRetries = oldExtractArchiveWithRetries
		checkAlreadyExtracted = oldCheckAlreadyExtracted
		safeMovePath = oldSafeMovePath
		runCleanupSelection = oldRunCleanupSelection
	}
}
