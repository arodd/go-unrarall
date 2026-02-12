package hooks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/austin/go-unrarall/internal/log"
)

var sampleVideoPattern = `(?i)^sample.*%s\.(asf|avi|mkv|mp4|m4v|mov|mpg|mpeg|ogg|webm|wmv)$`

// Context carries archive metadata and execution settings for cleanup hooks.
type Context struct {
	ExtractRoot string
	RarDir      string
	Stem        string
	DryRun      bool
	Log         *log.Logger
}

// Run executes cleanup hooks in deterministic order based on selection.
func Run(selection []string, ctx Context) error {
	names := resolveNames(selection)
	if len(names) == 0 {
		return nil
	}

	var errs []error
	for _, name := range names {
		def, ok := lookup(name)
		if !ok {
			errs = append(errs, fmt.Errorf("unknown clean hook %q", name))
			continue
		}
		if ctx.Log != nil {
			ctx.Log.Verbosef("Running clean hook %q", name)
		}
		if err := def.Run(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

func runNFO(ctx Context) error {
	return removeFile(filepath.Join(ctx.ExtractRoot, ctx.Stem+".nfo"), ctx)
}

func runRAR(ctx Context) error {
	entries, err := os.ReadDir(ctx.RarDir)
	if err != nil {
		return err
	}

	pattern := regexp.MustCompile(
		fmt.Sprintf(`(?i)^%s\.(sfv|[0-9]+|[r-z][0-9]+|rar|part[0-9]+\.rar)$`, regexp.QuoteMeta(ctx.Stem)),
	)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !pattern.MatchString(entry.Name()) {
			continue
		}
		if err := removeFile(filepath.Join(ctx.RarDir, entry.Name()), ctx); err != nil {
			return err
		}
	}
	return nil
}

func runOSXJunk(ctx Context) error {
	return removeFile(filepath.Join(ctx.ExtractRoot, ".DS_Store"), ctx)
}

func runWindowsJunk(ctx Context) error {
	return removeFile(filepath.Join(ctx.ExtractRoot, "Thumbs.db"), ctx)
}

func runCoversFolders(ctx Context) error {
	return removeNamedDirectories(ctx.ExtractRoot, "covers", ctx)
}

func runProofFolders(ctx Context) error {
	return removeNamedDirectories(ctx.ExtractRoot, "proof", ctx)
}

func runSampleFolders(ctx Context) error {
	return removeNamedDirectories(ctx.ExtractRoot, "sample", ctx)
}

func runSampleVideos(ctx Context) error {
	entries, err := os.ReadDir(ctx.ExtractRoot)
	if err != nil {
		return err
	}

	expr := regexp.MustCompile(fmt.Sprintf(sampleVideoPattern, regexp.QuoteMeta(ctx.Stem)))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !expr.MatchString(name) {
			continue
		}
		if err := removeFile(filepath.Join(ctx.ExtractRoot, name), ctx); err != nil {
			return err
		}
	}
	return nil
}

func runEmptyFolders(ctx Context) error {
	_, err := pruneEmptyDirectories(ctx.RarDir, true, ctx)
	return err
}

func removeNamedDirectories(root, targetName string, ctx Context) error {
	matches := make([]string, 0, 8)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		if !strings.EqualFold(d.Name(), targetName) {
			return nil
		}
		matches = append(matches, path)
		return nil
	})
	if err != nil {
		return err
	}

	sort.Slice(matches, func(i, j int) bool {
		return len(matches[i]) > len(matches[j])
	})
	for _, match := range matches {
		if err := removeDirectoryTree(match, ctx); err != nil {
			return err
		}
	}
	return nil
}

func removeFile(path string, ctx Context) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}

	if ctx.DryRun {
		if ctx.Log != nil {
			ctx.Log.Verbosef("Dry-run: remove file %q", path)
		}
		return nil
	}
	return os.Remove(path)
}

func removeDirectoryTree(path string, ctx Context) error {
	if ctx.DryRun {
		if ctx.Log != nil {
			ctx.Log.Verbosef("Dry-run: remove directory tree %q", path)
		}
		return nil
	}
	return os.RemoveAll(path)
}

func pruneEmptyDirectories(path string, isRoot bool, ctx Context) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	empty := true
	for _, entry := range entries {
		if !entry.IsDir() {
			empty = false
			continue
		}

		childPath := filepath.Join(path, entry.Name())
		childEmpty, err := pruneEmptyDirectories(childPath, false, ctx)
		if err != nil {
			return false, err
		}
		if !childEmpty {
			empty = false
		}
	}

	if isRoot || !empty {
		return empty, nil
	}

	if ctx.DryRun {
		if ctx.Log != nil {
			ctx.Log.Verbosef("Dry-run: remove empty directory %q", path)
		}
		return true, nil
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return true, nil
}
