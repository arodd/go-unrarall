package rar

import (
	"bytes"
	"io"
	"os"
)

var (
	rar5Signature = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00}
	rar4Signature = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}
)

const maxSFXBytes = 1 << 20 // 1 MiB

// HasRarSignature checks whether a file contains a RAR4 or RAR5 signature
// in the searchable prefix window used for SFX archives.
func HasRarSignature(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	window := maxSFXBytes + len(rar5Signature)
	buf := make([]byte, window)
	readN, readErr := io.ReadFull(file, buf)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return false, readErr
	}
	buf = buf[:readN]

	if bytes.Contains(buf, rar5Signature) || bytes.Contains(buf, rar4Signature) {
		return true, nil
	}
	return false, nil
}
