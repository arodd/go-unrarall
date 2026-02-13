package app

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/arodd/go-unrarall/internal/rar"
	"github.com/nwaples/rardecode/v2"
)

// AlreadyExtracted returns true when every non-directory entry in archivePath
// already exists in destRoot according to fullPath mode.
func AlreadyExtracted(archivePath, destRoot string, fullPath bool, opts ...rardecode.Option) (bool, error) {
	files, err := rar.ListFiles(archivePath, opts...)
	if err != nil {
		return false, err
	}
	return alreadyExtractedFromListed(destRoot, files, fullPath)
}

func alreadyExtractedFromListed(destRoot string, files []rar.ListedFile, fullPath bool) (bool, error) {
	for _, file := range files {
		if file.IsDir {
			continue
		}

		target, ok := skipTargetPath(destRoot, file.Name, fullPath)
		if !ok {
			return false, nil
		}

		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

func skipTargetPath(destRoot, archiveName string, fullPath bool) (string, bool) {
	normalized := strings.ReplaceAll(archiveName, "\\", "/")
	if !fullPath {
		base := strings.TrimSpace(path.Base(normalized))
		if base == "" || base == "." || base == "/" {
			return "", false
		}
		return filepath.Join(destRoot, filepath.FromSlash(base)), true
	}

	rel := filepath.Clean(filepath.FromSlash(normalized))
	if rel == "" || rel == "." || rel == ".." {
		return "", false
	}
	if filepath.IsAbs(rel) {
		return "", false
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Join(destRoot, rel), true
}
