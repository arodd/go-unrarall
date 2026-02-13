package finder

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Scan walks root and returns first-volume candidate archives.
// A negative maxDepth means unbounded scanning.
func Scan(root string, maxDepth int) ([]Candidate, error) {
	candidates := make([]Candidate, 0, 16)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		depth, err := relativeDepth(root, path)
		if err != nil {
			return err
		}
		if maxDepth >= 0 && depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		isFirst, stem := IsFirstVolume(d.Name())
		if !isFirst {
			return nil
		}
		candidates = append(candidates, Candidate{
			Path: path,
			Stem: stem,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(candidates, func(i, j int) bool {
		return strings.ToLower(candidates[i].Path) < strings.ToLower(candidates[j].Path)
	})
	return candidates, nil
}

func relativeDepth(root, path string) (int, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0, err
	}
	if rel == "." {
		return 0, nil
	}
	segments := strings.Split(rel, string(filepath.Separator))
	return len(segments) - 1, nil
}
