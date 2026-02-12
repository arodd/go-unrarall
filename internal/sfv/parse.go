package sfv

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// Entry represents a single SFV file checksum line.
type Entry struct {
	Name string
	CRC  uint32
}

// Parse parses SFV contents into entries.
func Parse(r io.Reader) ([]Entry, error) {
	scanner := bufio.NewScanner(r)
	entries := make([]Entry, 0, 16)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		entry, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("sfv line %d: %w", lineNo, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func parseLine(line string) (Entry, error) {
	last := len(line) - 1
	for last >= 0 && unicode.IsSpace(rune(line[last])) {
		last--
	}
	if last < 0 {
		return Entry{}, fmt.Errorf("empty line")
	}

	crcEnd := last + 1
	for last >= 0 && isHexDigit(line[last]) {
		last--
	}

	crcStart := last + 1
	if crcEnd-crcStart != 8 {
		return Entry{}, fmt.Errorf("invalid crc field")
	}
	if last < 0 || !unicode.IsSpace(rune(line[last])) {
		return Entry{}, fmt.Errorf("missing separator before crc")
	}

	name := strings.TrimSpace(line[:last+1])
	if name == "" {
		return Entry{}, fmt.Errorf("missing filename")
	}

	crcValue, err := strconv.ParseUint(line[crcStart:crcEnd], 16, 32)
	if err != nil {
		return Entry{}, fmt.Errorf("invalid crc %q: %w", line[crcStart:crcEnd], err)
	}

	return Entry{
		Name: name,
		CRC:  uint32(crcValue),
	}, nil
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
