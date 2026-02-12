package sfv

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Mismatch describes a checksum mismatch for an SFV entry.
type Mismatch struct {
	Name     string
	Expected uint32
	Actual   uint32
}

// VerificationError captures all missing and mismatched SFV entries.
type VerificationError struct {
	Missing    []string
	Mismatches []Mismatch
}

// Error implements the error interface.
func (e *VerificationError) Error() string {
	return fmt.Sprintf(
		"sfv verification failed: %d missing, %d mismatched",
		len(e.Missing),
		len(e.Mismatches),
	)
}

// Verify checks all entries against files under baseDir.
func Verify(baseDir string, entries []Entry) error {
	missing := make([]string, 0)
	mismatches := make([]Mismatch, 0)

	for _, entry := range entries {
		target := filepath.Join(baseDir, normalizeEntryName(entry.Name))
		actual, err := fileCRC32(target)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, entry.Name)
				continue
			}
			return fmt.Errorf("verify %q: %w", entry.Name, err)
		}

		if actual != entry.CRC {
			mismatches = append(mismatches, Mismatch{
				Name:     entry.Name,
				Expected: entry.CRC,
				Actual:   actual,
			})
		}
	}

	if len(missing) > 0 || len(mismatches) > 0 {
		return &VerificationError{
			Missing:    missing,
			Mismatches: mismatches,
		}
	}
	return nil
}

func normalizeEntryName(name string) string {
	normalized := strings.ReplaceAll(name, "\\", "/")
	return filepath.FromSlash(normalized)
}

func fileCRC32(path string) (uint32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(h, file); err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}
