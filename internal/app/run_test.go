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
			name:          "allow failures with skipped archives",
			stats:         Stats{ArchivesSkipped: 1, Failures: 2},
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
			if depth != -1 {
				t.Fatalf("top-level scan depth=%d, want -1 (unbounded)", depth)
			}
			return []finder.Candidate{{Path: topArchive, Stem: "top"}}, nil
		case topTmpDir:
			if depth != -1 {
				t.Fatalf("nested scan depth=%d, want -1 (unbounded)", depth)
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
		_ bool,
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
		_ bool,
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

func TestCollectExtractedArtifactsRejectsSymlinkWhenDisabled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Symlink("file.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("skipping symlink test: %v", err)
	}

	_, _, err := collectExtractedArtifacts(root, false)
	if err == nil {
		t.Fatal("expected symlink rejection error")
	}
}

func TestCollectExtractedArtifactsAllowsSymlinkWhenEnabled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Symlink("file.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("skipping symlink test: %v", err)
	}

	files, _, err := collectExtractedArtifacts(root, true)
	if err != nil {
		t.Fatalf("collectExtractedArtifacts returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 extracted items (file + symlink), got %d (%v)", len(files), files)
	}
}

func TestRunDepthZeroStillProcessesDeepCandidates(t *testing.T) {
	root := t.TempDir()
	deepDir := filepath.Join(root, "level1", "level2")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("mkdir deep dir: %v", err)
	}
	deepArchive := filepath.Join(deepDir, "deep.rar")
	if err := os.WriteFile(deepArchive, []byte("x"), 0o644); err != nil {
		t.Fatalf("write deep archive: %v", err)
	}

	restore := stubRunDependencies()
	defer restore()

	validateRarSignature = func(path string) (bool, error) {
		return true, nil
	}
	extractArchiveWithRetries = func(
		archivePath string,
		tmpDir string,
		_ bool,
		_ int64,
		_ bool,
		_ string,
	) (PasswordExtractionResult, error) {
		if archivePath != deepArchive {
			return PasswordExtractionResult{}, errors.New("unexpected archive path")
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "payload.txt"), []byte("ok"), 0o644); err != nil {
			return PasswordExtractionResult{}, err
		}
		return PasswordExtractionResult{Volumes: []string{archivePath}}, nil
	}

	opts := cli.Options{
		Dir:          root,
		Depth:        0,
		CKSFV:        false,
		CleanHooks:   []string{"none"},
		MaxDictBytes: 1 << 20,
		PasswordFile: filepath.Join(root, "passwords.txt"),
	}

	stats, err := Run(opts, log.New(true, false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if stats.ArchivesFound != 1 {
		t.Fatalf("ArchivesFound=%d, want 1", stats.ArchivesFound)
	}
	if stats.ArchivesExtracted != 1 {
		t.Fatalf("ArchivesExtracted=%d, want 1", stats.ArchivesExtracted)
	}
	if stats.Failures != 0 {
		t.Fatalf("Failures=%d, want 0", stats.Failures)
	}

	if _, err := os.Stat(filepath.Join(deepDir, "payload.txt")); err != nil {
		t.Fatalf("expected payload moved into deep archive directory, stat err=%v", err)
	}
}

func TestRunDryRunBypassesSkipIfExists(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "release.rar")
	if err := os.WriteFile(archivePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	skipChecks := 0

	restore := stubRunDependencies()
	defer restore()

	scanCandidates = func(_ string, _ int) ([]finder.Candidate, error) {
		return []finder.Candidate{{Path: archivePath, Stem: "release"}}, nil
	}
	validateRarSignature = func(path string) (bool, error) {
		return true, nil
	}
	checkAlreadyExtracted = func(_ string, _ string, _ bool, _ ...rardecode.Option) (bool, error) {
		skipChecks++
		return true, nil
	}
	extractArchiveWithRetries = func(
		_ string,
		_ string,
		_ bool,
		_ int64,
		_ bool,
		_ string,
	) (PasswordExtractionResult, error) {
		t.Fatal("extractArchiveWithRetries should not be called in dry-run mode")
		return PasswordExtractionResult{}, nil
	}

	opts := cli.Options{
		Dir:          root,
		Depth:        0,
		DryRun:       true,
		SkipIfExists: true,
		CKSFV:        false,
		CleanHooks:   []string{"none"},
		MaxDictBytes: 1 << 20,
		PasswordFile: filepath.Join(root, "passwords.txt"),
	}

	stats, err := Run(opts, log.New(true, false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if skipChecks != 0 {
		t.Fatalf("skip checks=%d, want 0 in dry-run mode", skipChecks)
	}
	if stats.ArchivesExtracted != 1 {
		t.Fatalf("ArchivesExtracted=%d, want 1", stats.ArchivesExtracted)
	}
	if stats.ArchivesSkipped != 0 {
		t.Fatalf("ArchivesSkipped=%d, want 0", stats.ArchivesSkipped)
	}
}

func TestRunSkipIfExistsChecksArchiveDirWhenOutputDirSet(t *testing.T) {
	root := t.TempDir()
	outputDir := t.TempDir()
	archivePath := filepath.Join(root, "release.rar")
	if err := os.WriteFile(archivePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	restore := stubRunDependencies()
	defer restore()

	scanCandidates = func(_ string, _ int) ([]finder.Candidate, error) {
		return []finder.Candidate{{Path: archivePath, Stem: "release"}}, nil
	}
	validateRarSignature = func(path string) (bool, error) {
		return true, nil
	}
	checkAlreadyExtracted = func(_ string, destRoot string, _ bool, _ ...rardecode.Option) (bool, error) {
		if got, want := destRoot, filepath.Dir(archivePath); got != want {
			t.Fatalf("skip check destination=%q, want archive directory %q", got, want)
		}
		return true, nil
	}
	extractArchiveWithRetries = func(
		_ string,
		_ string,
		_ bool,
		_ int64,
		_ bool,
		_ string,
	) (PasswordExtractionResult, error) {
		t.Fatal("extractArchiveWithRetries should not run when skip-if-exists succeeds")
		return PasswordExtractionResult{}, nil
	}

	opts := cli.Options{
		Dir:          root,
		OutputDir:    outputDir,
		Depth:        0,
		SkipIfExists: true,
		CKSFV:        false,
		CleanHooks:   []string{"none"},
		MaxDictBytes: 1 << 20,
		PasswordFile: filepath.Join(root, "passwords.txt"),
	}

	stats, err := Run(opts, log.New(true, false))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if stats.ArchivesFound != 1 {
		t.Fatalf("ArchivesFound=%d, want 1", stats.ArchivesFound)
	}
	if stats.ArchivesSkipped != 1 {
		t.Fatalf("ArchivesSkipped=%d, want 1", stats.ArchivesSkipped)
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
