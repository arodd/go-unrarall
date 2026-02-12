package sfv

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifySuccess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := []byte("payload")
	file := filepath.Join(root, "video.mkv")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	entries := []Entry{
		{
			Name: "video.mkv",
			CRC:  crc32.ChecksumIEEE(content),
		},
	}

	if err := Verify(root, entries); err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
}

func TestVerifyMissingAndMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	okData := []byte("ok")
	okFile := filepath.Join(root, "ok.bin")
	if err := os.WriteFile(okFile, okData, 0o644); err != nil {
		t.Fatalf("write ok file: %v", err)
	}

	badData := []byte("bad")
	badFile := filepath.Join(root, "bad.bin")
	if err := os.WriteFile(badFile, badData, 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	entries := []Entry{
		{Name: "ok.bin", CRC: crc32.ChecksumIEEE(okData)},
		{Name: "missing.bin", CRC: 0x12345678},
		{Name: "bad.bin", CRC: 0xDEADBEEF},
	}

	err := Verify(root, entries)
	if err == nil {
		t.Fatal("expected verification error")
	}

	var verificationErr *VerificationError
	if !errors.As(err, &verificationErr) {
		t.Fatalf("Verify returned %T, want *VerificationError", err)
	}

	if len(verificationErr.Missing) != 1 || verificationErr.Missing[0] != "missing.bin" {
		t.Fatalf("missing = %v", verificationErr.Missing)
	}
	if len(verificationErr.Mismatches) != 1 {
		t.Fatalf("mismatches length = %d, want 1", len(verificationErr.Mismatches))
	}
	if verificationErr.Mismatches[0].Name != "bad.bin" {
		t.Fatalf("mismatch entry = %+v", verificationErr.Mismatches[0])
	}
}

func TestVerifyNormalizesBackslashSeparators(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}

	content := []byte("nested")
	target := filepath.Join(sub, "clip.avi")
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	entries := []Entry{
		{Name: fmt.Sprintf("sub%[1]sclip.avi", "\\"), CRC: crc32.ChecksumIEEE(content)},
	}

	if err := Verify(root, entries); err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
}
