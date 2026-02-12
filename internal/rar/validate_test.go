package rar

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHasRarSignatureAtStart(t *testing.T) {
	t.Parallel()

	path := writeFixture(t, append([]byte{}, rar5Signature...))
	ok, err := HasRarSignature(path)
	if err != nil {
		t.Fatalf("HasRarSignature returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected RAR5 signature to be detected")
	}
}

func TestHasRarSignatureWithinSFXWindow(t *testing.T) {
	t.Parallel()

	prefix := bytes.Repeat([]byte{0xAA}, maxSFXBytes-len(rar4Signature))
	content := append(prefix, rar4Signature...)
	path := writeFixture(t, content)

	ok, err := HasRarSignature(path)
	if err != nil {
		t.Fatalf("HasRarSignature returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected RAR4 signature in SFX window to be detected")
	}
}

func TestHasRarSignatureBeyondSFXWindow(t *testing.T) {
	t.Parallel()

	prefix := bytes.Repeat([]byte{0xBB}, maxSFXBytes+32)
	content := append(prefix, rar5Signature...)
	path := writeFixture(t, content)

	ok, err := HasRarSignature(path)
	if err != nil {
		t.Fatalf("HasRarSignature returned error: %v", err)
	}
	if ok {
		t.Fatal("did not expect signature beyond SFX window to be detected")
	}
}

func TestHasRarSignatureMissing(t *testing.T) {
	t.Parallel()

	path := writeFixture(t, []byte("not a rar archive"))
	ok, err := HasRarSignature(path)
	if err != nil {
		t.Fatalf("HasRarSignature returned error: %v", err)
	}
	if ok {
		t.Fatal("did not expect missing signature to be detected")
	}
}

func writeFixture(t *testing.T, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fixture.bin")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}
