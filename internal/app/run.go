package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/austin/go-unrarall/internal/cli"
	"github.com/austin/go-unrarall/internal/finder"
	"github.com/austin/go-unrarall/internal/fsutil"
	"github.com/austin/go-unrarall/internal/log"
	"github.com/austin/go-unrarall/internal/rar"
	"github.com/austin/go-unrarall/internal/sfv"
)

var (
	scanCandidates            = finder.Scan
	validateRarSignature      = rar.HasRarSignature
	createExtractionTempDir   = fsutil.CreateTempDir
	extractArchiveWithRetries = ExtractArchiveWithPasswords
	checkAlreadyExtracted     = AlreadyExtracted
	safeMovePath              = fsutil.SafeMove
	runCleanupSelection       = runCleanupHooks
)

// Stats tracks extraction outcomes across a run.
type Stats struct {
	ArchivesFound     int
	ArchivesExtracted int
	ArchivesSkipped   int
	Failures          int
}

func (s *Stats) add(other Stats) {
	s.ArchivesFound += other.ArchivesFound
	s.ArchivesExtracted += other.ArchivesExtracted
	s.ArchivesSkipped += other.ArchivesSkipped
	s.Failures += other.Failures
}

// ExitCode computes the process exit code using script-parity behavior.
func ExitCode(stats Stats, allowFailures bool) int {
	if stats.Failures == 0 {
		return 0
	}
	if allowFailures && stats.ArchivesExtracted > 0 {
		return 0
	}
	return 1
}

type runner struct {
	opts cli.Options
	log  *log.Logger
}

// Run executes archive extraction orchestration for opts.Dir.
func Run(opts cli.Options, logger *log.Logger) (Stats, error) {
	r := &runner{
		opts: opts,
		log:  logger,
	}

	stats, err := r.runDirectory(opts.Dir, opts.Depth)
	if err != nil {
		return stats, err
	}

	r.logSummary(stats)
	return stats, nil
}

func (r *runner) runDirectory(dir string, depth int) (Stats, error) {
	candidates, err := scanCandidates(dir, depth)
	if err != nil {
		return Stats{}, err
	}

	var stats Stats
	for _, candidate := range candidates {
		candidateStats, err := r.processCandidate(candidate, depth)
		stats.add(candidateStats)
		if err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func (r *runner) processCandidate(candidate finder.Candidate, depth int) (Stats, error) {
	stats := Stats{ArchivesFound: 1}

	ok, err := validateRarSignature(candidate.Path)
	if err != nil {
		r.log.Errorf("Failed to inspect archive %q: %v", candidate.Path, err)
		stats.Failures++
		return stats, nil
	}
	if !ok {
		r.log.Errorf("Skipping file %q because it does not appear to be a valid rar file.", candidate.Path)
		stats.Failures++
		return stats, nil
	}

	rarDir := filepath.Dir(candidate.Path)
	destRoot := destinationRoot(r.opts.OutputDir, rarDir)

	sfvErr := r.verifySFVIfPresent(rarDir, candidate.Stem)
	if sfvErr != nil && !r.opts.Force {
		r.log.Errorf("SFV verification failed for %q: %v", candidate.Path, sfvErr)
		stats.Failures++
		return stats, nil
	}
	if sfvErr != nil && r.opts.Force {
		r.log.Errorf("SFV verification failed for %q, continuing due to --force: %v", candidate.Path, sfvErr)
	}

	if r.opts.SkipIfExists && !r.opts.Force && sfvErr == nil {
		skip, err := checkAlreadyExtracted(candidate.Path, destRoot, r.opts.FullPath)
		if err != nil {
			r.log.Verbosef("Skip-if-exists check failed for %q: %v", candidate.Path, err)
		} else if skip {
			r.log.Infof("File %q appears to have already been extracted, skipping.", candidate.Path)
			stats.ArchivesSkipped++
			return stats, nil
		}
	}

	if r.opts.DryRun {
		r.log.Infof("Dry-run: would extract %q to %q", candidate.Path, destRoot)
		if shouldRunHooks(r.opts.CleanHooks) {
			if err := runCleanupSelection(r.opts.CleanHooks, destRoot, rarDir, candidate.Stem, true, r.log); err != nil {
				r.log.Errorf("Cleanup hooks failed for %q: %v", candidate.Path, err)
				stats.Failures++
				return stats, nil
			}
		}
		stats.ArchivesExtracted++
		return stats, nil
	}

	tmpDir, err := createExtractionTempDir(rarDir)
	if err != nil {
		return stats, fmt.Errorf("create temp directory for %q: %w", candidate.Path, err)
	}

	extractResult, extractErr := extractArchiveWithRetries(
		candidate.Path,
		tmpDir,
		r.opts.FullPath,
		r.opts.MaxDictBytes,
		r.opts.AllowSymlinks,
		r.opts.PasswordFile,
	)
	if extractErr == nil {
		if extractResult.UsedPassword {
			r.log.Verbosef("Extraction of %q succeeded using password %q", candidate.Path, extractResult.Password)
		}
		r.log.Verbosef("Extracted %q using volumes: %v", candidate.Path, extractResult.Volumes)
	}

	var nestedStats Stats
	if extractErr == nil {
		nestedStats, extractErr = r.runRecursive(tmpDir, depth-1)
		stats.add(nestedStats)
	}

	if err := moveExtractedArtifacts(tmpDir, destRoot, r.opts.AllowSymlinks); err != nil {
		return stats, fmt.Errorf("move extracted artifacts for %q: %w", candidate.Path, err)
	}
	if err := os.RemoveAll(tmpDir); err != nil {
		return stats, fmt.Errorf("remove temp directory %q: %w", tmpDir, err)
	}

	if shouldRunHooks(r.opts.CleanHooks) {
		if extractErr == nil || r.opts.Force {
			if err := runCleanupSelection(r.opts.CleanHooks, destRoot, rarDir, candidate.Stem, false, r.log); err != nil {
				r.log.Errorf("Cleanup hooks failed for %q: %v", candidate.Path, err)
				if extractErr == nil {
					extractErr = err
				}
			}
		} else {
			r.log.Errorf("Couldn't run cleanup hooks for %q because extraction failed. Use --force to override.", candidate.Path)
		}
	}

	if extractErr != nil {
		r.log.Errorf("Extraction failed for %q: %v", candidate.Path, extractErr)
		stats.Failures++
		return stats, nil
	}

	stats.ArchivesExtracted++
	return stats, nil
}

func (r *runner) verifySFVIfPresent(rarDir, stem string) error {
	if !r.opts.CKSFV {
		return nil
	}

	sfvPath := filepath.Join(rarDir, stem+".sfv")
	file, err := os.Open(sfvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	entries, err := sfv.Parse(file)
	if err != nil {
		return err
	}
	return sfv.Verify(rarDir, entries)
}

func (r *runner) logSummary(stats Stats) {
	if stats.ArchivesExtracted > 0 {
		if shouldRunHooks(r.opts.CleanHooks) {
			r.log.Infof("%d rar file(s) found, extracted, and cleaned.", stats.ArchivesExtracted)
		} else {
			r.log.Infof("%d rar file(s) found and extracted.", stats.ArchivesExtracted)
		}
	} else {
		r.log.Infof("no rar files extracted")
	}

	if stats.Failures > 0 {
		r.log.Errorf("%d failure(s)", stats.Failures)
		if r.opts.AllowFailures && stats.ArchivesExtracted > 0 {
			r.log.Infof("%d success(es)", stats.ArchivesExtracted)
		}
	}
}

func destinationRoot(outputDir, rarDir string) string {
	if outputDir != "" {
		return outputDir
	}
	return rarDir
}

func moveExtractedArtifacts(tmpDir, destRoot string, allowSymlinks bool) error {
	files, emptyDirs, err := collectExtractedArtifacts(tmpDir, allowSymlinks)
	if err != nil {
		return err
	}

	for _, rel := range files {
		srcPath := filepath.Join(tmpDir, rel)
		dstPath := filepath.Join(destRoot, rel)

		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		if _, err := safeMovePath(srcPath, dstPath); err != nil {
			return err
		}
	}

	for _, rel := range emptyDirs {
		if err := os.MkdirAll(filepath.Join(destRoot, rel), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func collectExtractedArtifacts(tmpDir string, allowSymlinks bool) ([]string, []string, error) {
	files := make([]string, 0, 16)
	emptyDirs := make([]string, 0, 16)

	err := filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == tmpDir {
			return nil
		}

		rel, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				emptyDirs = append(emptyDirs, rel)
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			if !allowSymlinks {
				return fmt.Errorf("unsupported extracted symlink %q", path)
			}
			files = append(files, rel)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported extracted file type %q", path)
		}

		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Strings(files)
	sort.Strings(emptyDirs)
	return files, emptyDirs, nil
}
