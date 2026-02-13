package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/arodd/go-unrarall/internal/rar"
	"github.com/nwaples/rardecode/v2"
)

type extractionAttempt struct {
	Password      string
	MaxDict       int64
	AllowSymlinks bool
}

func TestExtractArchiveWithPasswordsUnencrypted(t *testing.T) {
	t.Parallel()

	var attempts []extractionAttempt
	extract := func(_ string, _ string, _ bool, settings rar.OpenSettings) ([]string, error) {
		attempts = append(attempts, extractionAttempt{
			Password:      settings.Password,
			MaxDict:       settings.MaxDictionaryBytes,
			AllowSymlinks: settings.AllowSymlinks,
		})
		return []string{"release.rar"}, nil
	}

	result, err := extractArchiveWithPasswords(
		extract,
		"/archives/release.rar",
		t.TempDir(),
		true,
		1<<20,
		false,
		"/unused/passwords.txt",
	)
	if err != nil {
		t.Fatalf("extractArchiveWithPasswords returned error: %v", err)
	}

	if result.UsedPassword {
		t.Fatal("did not expect a password retry for unencrypted archive")
	}
	if result.Password != "" {
		t.Fatalf("result password=%q, want empty", result.Password)
	}
	if !reflect.DeepEqual(result.Volumes, []string{"release.rar"}) {
		t.Fatalf("volumes=%v, want %v", result.Volumes, []string{"release.rar"})
	}
	if !reflect.DeepEqual(attempts, []extractionAttempt{{Password: "", MaxDict: 1 << 20, AllowSymlinks: false}}) {
		t.Fatalf("attempts=%v", attempts)
	}
}

func TestExtractArchiveWithPasswordsRetriesPasswordFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	passwordFile := filepath.Join(root, "passwords.txt")
	if err := os.WriteFile(passwordFile, []byte("wrong\nsecret\n"), 0o644); err != nil {
		t.Fatalf("write password file: %v", err)
	}

	var attempts []extractionAttempt
	extract := func(_ string, _ string, _ bool, settings rar.OpenSettings) ([]string, error) {
		attempts = append(attempts, extractionAttempt{
			Password:      settings.Password,
			MaxDict:       settings.MaxDictionaryBytes,
			AllowSymlinks: settings.AllowSymlinks,
		})
		switch settings.Password {
		case "":
			return nil, rardecode.ErrArchiveEncrypted
		case "secret":
			return []string{"release.part01.rar", "release.part02.rar"}, nil
		default:
			return nil, rardecode.ErrBadPassword
		}
	}

	result, err := extractArchiveWithPasswords(
		extract,
		"/archives/release.part01.rar",
		t.TempDir(),
		false,
		1<<21,
		true,
		passwordFile,
	)
	if err != nil {
		t.Fatalf("extractArchiveWithPasswords returned error: %v", err)
	}

	if !result.UsedPassword {
		t.Fatal("expected password retry success")
	}
	if result.Password != "secret" {
		t.Fatalf("password=%q, want %q", result.Password, "secret")
	}
	wantVolumes := []string{"release.part01.rar", "release.part02.rar"}
	if !reflect.DeepEqual(result.Volumes, wantVolumes) {
		t.Fatalf("volumes=%v, want %v", result.Volumes, wantVolumes)
	}

	wantAttempts := []extractionAttempt{
		{Password: "", MaxDict: 1 << 21, AllowSymlinks: true},
		{Password: "wrong", MaxDict: 1 << 21, AllowSymlinks: true},
		{Password: "secret", MaxDict: 1 << 21, AllowSymlinks: true},
	}
	if !reflect.DeepEqual(attempts, wantAttempts) {
		t.Fatalf("attempts=%v, want %v", attempts, wantAttempts)
	}
}

func TestExtractArchiveWithPasswordsMissingPasswordFile(t *testing.T) {
	t.Parallel()

	extract := func(_ string, _ string, _ bool, _ rar.OpenSettings) ([]string, error) {
		return nil, rardecode.ErrArchiveEncrypted
	}

	_, err := extractArchiveWithPasswords(
		extract,
		"/archives/release.rar",
		t.TempDir(),
		true,
		1<<20,
		false,
		filepath.Join(t.TempDir(), "missing.txt"),
	)
	if err == nil {
		t.Fatal("expected password required error")
	}

	var passwordErr *PasswordRequiredError
	if !errors.As(err, &passwordErr) {
		t.Fatalf("error=%T, want *PasswordRequiredError", err)
	}
	if passwordErr.PasswordFile == "" {
		t.Fatal("expected password file path in error")
	}
}

func TestExtractArchiveWithPasswordsEmptyPasswordFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	passwordFile := filepath.Join(root, "passwords.txt")
	if err := os.WriteFile(passwordFile, []byte("\n\r\n"), 0o644); err != nil {
		t.Fatalf("write password file: %v", err)
	}

	extract := func(_ string, _ string, _ bool, _ rar.OpenSettings) ([]string, error) {
		return nil, rardecode.ErrArchiveEncrypted
	}

	_, err := extractArchiveWithPasswords(
		extract,
		"/archives/release.rar",
		t.TempDir(),
		true,
		1<<20,
		false,
		passwordFile,
	)
	if err == nil {
		t.Fatal("expected password required error")
	}

	var passwordErr *PasswordRequiredError
	if !errors.As(err, &passwordErr) {
		t.Fatalf("error=%T, want *PasswordRequiredError", err)
	}
	if passwordErr.Unwrap() == nil || !strings.Contains(passwordErr.Unwrap().Error(), "empty") {
		t.Fatalf("unexpected unwrap error: %v", passwordErr.Unwrap())
	}
}

func TestExtractArchiveWithPasswordsPassesThroughNonPasswordError(t *testing.T) {
	t.Parallel()

	expected := errors.New("corrupt archive")
	extract := func(_ string, _ string, _ bool, _ rar.OpenSettings) ([]string, error) {
		return nil, expected
	}

	_, err := extractArchiveWithPasswords(
		extract,
		"/archives/release.rar",
		t.TempDir(),
		true,
		1<<20,
		false,
		"/unused",
	)
	if !errors.Is(err, expected) {
		t.Fatalf("error=%v, want %v", err, expected)
	}
}

func TestReadPasswordFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "passwords.txt")
	if err := os.WriteFile(path, []byte("alpha\r\n\n beta \n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	passwords, err := readPasswordFile(path)
	if err != nil {
		t.Fatalf("readPasswordFile returned error: %v", err)
	}
	want := []string{"alpha", " beta "}
	if !reflect.DeepEqual(passwords, want) {
		t.Fatalf("passwords=%v, want %v", passwords, want)
	}
}
