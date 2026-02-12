package fsutil

import (
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeRelPath normalizes archive paths and ensures the result is a safe
// relative path that cannot escape a destination root.
func SanitizeRelPath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	normalized := strings.ReplaceAll(trimmed, "\\", "/")
	cleaned := path.Clean(normalized)

	if cleaned == "." || cleaned == ".." || cleaned == "/" {
		return "", false
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", false
	}
	if strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	if hasDrivePrefix(cleaned) {
		return "", false
	}

	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}
		if strings.ContainsRune(segment, 0) {
			return "", false
		}
	}

	rel := filepath.FromSlash(cleaned)
	if filepath.IsAbs(rel) {
		return "", false
	}
	if filepath.VolumeName(rel) != "" {
		return "", false
	}
	return rel, true
}

func hasDrivePrefix(pathValue string) bool {
	if len(pathValue) < 2 || pathValue[1] != ':' {
		return false
	}
	return unicode.IsLetter(rune(pathValue[0]))
}
