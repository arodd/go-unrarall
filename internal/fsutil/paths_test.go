package fsutil

import (
	"path/filepath"
	"testing"
)

func TestSanitizeRelPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		ok    bool
		want  string
	}{
		{
			name:  "simple relative",
			input: "file.txt",
			ok:    true,
			want:  "file.txt",
		},
		{
			name:  "nested path",
			input: "nested/path/file.bin",
			ok:    true,
			want:  filepath.Join("nested", "path", "file.bin"),
		},
		{
			name:  "backslashes normalized",
			input: "nested\\path\\file.bin",
			ok:    true,
			want:  filepath.Join("nested", "path", "file.bin"),
		},
		{
			name:  "clean dot segments",
			input: "./nested/../file.bin",
			ok:    true,
			want:  "file.bin",
		},
		{
			name:  "parent traversal denied",
			input: "../escape.bin",
			ok:    false,
		},
		{
			name:  "absolute unix path denied",
			input: "/etc/passwd",
			ok:    false,
		},
		{
			name:  "drive path denied",
			input: "C:/Windows/system32.dll",
			ok:    false,
		},
		{
			name:  "empty denied",
			input: "   ",
			ok:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := SanitizeRelPath(tc.input)
			if ok != tc.ok {
				t.Fatalf("SanitizeRelPath(%q) ok=%v, want %v", tc.input, ok, tc.ok)
			}
			if tc.ok && got != tc.want {
				t.Fatalf("SanitizeRelPath(%q)=%q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
