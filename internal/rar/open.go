package rar

import (
	"errors"
	"io"

	"github.com/nwaples/rardecode/v2"
)

type archiveReadCloser interface {
	archiveReader
	io.Closer
	Volumes() []string
}

type openReaderFunc func(path string, opts ...rardecode.Option) (archiveReadCloser, error)

func openArchiveReader(path string, opts ...rardecode.Option) (archiveReadCloser, error) {
	return rardecode.OpenReader(path, opts...)
}

// OpenSettings contains decoder options used to open and extract an archive.
type OpenSettings struct {
	MaxDictionaryBytes int64
	Password           string
	AllowSymlinks      bool
}

// DecodeOptions converts settings into rardecode options.
func (s OpenSettings) DecodeOptions() []rardecode.Option {
	opts := make([]rardecode.Option, 0, 2)
	if s.MaxDictionaryBytes > 0 {
		opts = append(opts, rardecode.MaxDictionarySize(s.MaxDictionaryBytes))
	}
	if s.Password != "" {
		opts = append(opts, rardecode.Password(s.Password))
	}
	return opts
}

// IsPasswordError reports whether err indicates that archive decryption
// credentials are required or incorrect.
func IsPasswordError(err error) bool {
	return errors.Is(err, rardecode.ErrArchiveEncrypted) ||
		errors.Is(err, rardecode.ErrArchivedFileEncrypted) ||
		errors.Is(err, rardecode.ErrBadPassword)
}
