package rar

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nwaples/rardecode/v2"
)

type fakeArchiveEntry struct {
	header rardecode.FileHeader
	data   []byte
}

type fakeArchiveReader struct {
	entries []fakeArchiveEntry
	index   int
	current *bytes.Reader
}

func (r *fakeArchiveReader) Next() (*rardecode.FileHeader, error) {
	if r.index >= len(r.entries) {
		return nil, io.EOF
	}

	entry := r.entries[r.index]
	r.index++
	r.current = bytes.NewReader(entry.data)

	headerCopy := entry.header
	return &headerCopy, nil
}

func (r *fakeArchiveReader) Read(p []byte) (int, error) {
	if r.current == nil {
		return 0, io.EOF
	}
	return r.current.Read(p)
}

func TestExtractFromArchiveReaderFullPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modTime := time.Now().Add(-time.Hour).Truncate(time.Second)

	reader := &fakeArchiveReader{
		entries: []fakeArchiveEntry{
			{header: rardecode.FileHeader{Name: "nested", IsDir: true, ModificationTime: modTime}},
			{header: rardecode.FileHeader{Name: "nested/file.txt", ModificationTime: modTime}, data: []byte("hello")},
		},
	}

	if err := extractFromArchiveReader(reader, root, true); err != nil {
		t.Fatalf("extractFromArchiveReader returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("extracted content=%q, want %q", string(data), "hello")
	}

	info, err := os.Stat(filepath.Join(root, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("stat extracted file: %v", err)
	}
	if got := info.ModTime().Truncate(time.Second); !got.Equal(modTime) {
		t.Fatalf("modtime=%v, want %v", got, modTime)
	}
}

func TestExtractFromArchiveReaderFlatten(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	reader := &fakeArchiveReader{
		entries: []fakeArchiveEntry{
			{header: rardecode.FileHeader{Name: "nested/file.txt"}, data: []byte("flatten")},
			{header: rardecode.FileHeader{Name: "nested/ignored-dir", IsDir: true}},
		},
	}

	if err := extractFromArchiveReader(reader, root, false); err != nil {
		t.Fatalf("extractFromArchiveReader returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "file.txt"))
	if err != nil {
		t.Fatalf("read flattened file: %v", err)
	}
	if got := string(data); got != "flatten" {
		t.Fatalf("flattened content=%q, want %q", got, "flatten")
	}
	if _, err := os.Stat(filepath.Join(root, "nested")); !os.IsNotExist(err) {
		t.Fatalf("did not expect nested directory in flatten mode, stat err=%v", err)
	}
}

func TestExtractFromArchiveReaderRejectsUnsafePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	reader := &fakeArchiveReader{
		entries: []fakeArchiveEntry{
			{header: rardecode.FileHeader{Name: "../escape.txt"}, data: []byte("boom")},
		},
	}

	err := extractFromArchiveReader(reader, root, true)
	if err == nil {
		t.Fatal("expected unsafe path error")
	}
	if !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractFromArchiveReaderRejectsSymlinkEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	reader := &fakeArchiveReader{
		entries: []fakeArchiveEntry{
			{
				header: rardecode.FileHeader{
					Name:       "link",
					HostOS:     rardecode.HostOSUnix,
					Attributes: 0xA000 | 0o777,
				},
			},
		},
	}

	err := extractFromArchiveReader(reader, root, true)
	if err == nil {
		t.Fatal("expected symlink rejection error")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}
