package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arodd/go-unrarall/internal/rar"
)

// PasswordExtractionResult captures password retry metadata for a successful
// extraction attempt.
type PasswordExtractionResult struct {
	Volumes      []string
	UsedPassword bool
	Password     string
}

// PasswordRequiredError indicates that an archive is encrypted and no password
// was available to successfully open it.
type PasswordRequiredError struct {
	ArchivePath  string
	PasswordFile string
	Cause        error
}

func (e *PasswordRequiredError) Error() string {
	if e.PasswordFile == "" {
		return fmt.Sprintf("archive %q is encrypted and requires a password", e.ArchivePath)
	}
	return fmt.Sprintf("archive %q is encrypted; add password(s) to %q", e.ArchivePath, e.PasswordFile)
}

func (e *PasswordRequiredError) Unwrap() error {
	return e.Cause
}

type archiveExtractorWithSettings func(
	archivePath string,
	tmpDir string,
	fullPath bool,
	settings rar.OpenSettings,
) ([]string, error)

// ExtractArchiveWithPasswords extracts archivePath into tmpDir, retrying with
// passwords from passwordFile if the archive is encrypted.
func ExtractArchiveWithPasswords(
	archivePath string,
	tmpDir string,
	fullPath bool,
	maxDictBytes int64,
	allowSymlinks bool,
	passwordFile string,
) (PasswordExtractionResult, error) {
	return extractArchiveWithPasswords(
		rar.ExtractToDirWithSettings,
		archivePath,
		tmpDir,
		fullPath,
		maxDictBytes,
		allowSymlinks,
		passwordFile,
	)
}

func extractArchiveWithPasswords(
	extract archiveExtractorWithSettings,
	archivePath string,
	tmpDir string,
	fullPath bool,
	maxDictBytes int64,
	allowSymlinks bool,
	passwordFile string,
) (PasswordExtractionResult, error) {
	settings := rar.OpenSettings{
		MaxDictionaryBytes: maxDictBytes,
		AllowSymlinks:      allowSymlinks,
	}

	volumes, err := extract(archivePath, tmpDir, fullPath, settings)
	if err == nil {
		return PasswordExtractionResult{Volumes: volumes}, nil
	}
	if !rar.IsPasswordError(err) {
		return PasswordExtractionResult{}, err
	}

	passwords, loadErr := readPasswordFile(passwordFile)
	if loadErr != nil {
		return PasswordExtractionResult{}, &PasswordRequiredError{
			ArchivePath:  archivePath,
			PasswordFile: passwordFile,
			Cause:        loadErr,
		}
	}
	if len(passwords) == 0 {
		return PasswordExtractionResult{}, &PasswordRequiredError{
			ArchivePath:  archivePath,
			PasswordFile: passwordFile,
			Cause:        errors.New("password file is empty"),
		}
	}

	lastErr := err
	for _, password := range passwords {
		settings.Password = password

		volumes, tryErr := extract(archivePath, tmpDir, fullPath, settings)
		if tryErr == nil {
			return PasswordExtractionResult{
				Volumes:      volumes,
				UsedPassword: true,
				Password:     password,
			}, nil
		}
		if !rar.IsPasswordError(tryErr) {
			return PasswordExtractionResult{}, tryErr
		}
		lastErr = tryErr
	}

	return PasswordExtractionResult{}, lastErr
}

func readPasswordFile(path string) ([]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("password file path is empty")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	passwords := make([]string, 0, 8)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		passwords = append(passwords, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return passwords, nil
}
