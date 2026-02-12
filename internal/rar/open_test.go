package rar

import (
	"errors"
	"fmt"
	"testing"

	"github.com/nwaples/rardecode/v2"
)

func TestOpenSettingsDecodeOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings OpenSettings
		wantLen  int
	}{
		{
			name:     "empty settings",
			settings: OpenSettings{},
			wantLen:  0,
		},
		{
			name: "max dictionary only",
			settings: OpenSettings{
				MaxDictionaryBytes: 1 << 20,
			},
			wantLen: 1,
		},
		{
			name: "password only",
			settings: OpenSettings{
				Password: "secret",
			},
			wantLen: 1,
		},
		{
			name: "both settings",
			settings: OpenSettings{
				MaxDictionaryBytes: 1 << 20,
				Password:           "secret",
			},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := len(tc.settings.DecodeOptions()); got != tc.wantLen {
				t.Fatalf("DecodeOptions len=%d, want %d", got, tc.wantLen)
			}
		})
	}
}

func TestIsPasswordError(t *testing.T) {
	t.Parallel()

	if !IsPasswordError(rardecode.ErrArchiveEncrypted) {
		t.Fatal("expected ErrArchiveEncrypted to be classified as password error")
	}
	if !IsPasswordError(rardecode.ErrArchivedFileEncrypted) {
		t.Fatal("expected ErrArchivedFileEncrypted to be classified as password error")
	}
	if !IsPasswordError(rardecode.ErrBadPassword) {
		t.Fatal("expected ErrBadPassword to be classified as password error")
	}
	if !IsPasswordError(fmt.Errorf("wrapped: %w", rardecode.ErrBadPassword)) {
		t.Fatal("expected wrapped ErrBadPassword to be classified as password error")
	}
	if IsPasswordError(errors.New("other failure")) {
		t.Fatal("did not expect generic error to be classified as password error")
	}
}
