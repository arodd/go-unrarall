package log

import (
	"bytes"
	"testing"
)

func TestLoggerRoutesInfoAndErrorMessages(t *testing.T) {
	t.Parallel()

	var infoBuf bytes.Buffer
	var errBuf bytes.Buffer

	logger := NewWithWriters(false, false, &infoBuf, &errBuf)
	logger.Infof("info %d", 1)
	logger.Verbosef("verbose hidden")
	logger.Errorf("error %d", 2)

	if got, want := infoBuf.String(), "info 1\n"; got != want {
		t.Fatalf("info output=%q, want %q", got, want)
	}
	if got, want := errBuf.String(), "error 2\n"; got != want {
		t.Fatalf("error output=%q, want %q", got, want)
	}
}

func TestLoggerVerboseRouting(t *testing.T) {
	t.Parallel()

	var infoBuf bytes.Buffer
	var errBuf bytes.Buffer

	logger := NewWithWriters(false, true, &infoBuf, &errBuf)
	logger.Verbosef("verbose")

	if got, want := infoBuf.String(), "verbose\n"; got != want {
		t.Fatalf("info output=%q, want %q", got, want)
	}
	if got := errBuf.String(); got != "" {
		t.Fatalf("expected empty error output, got %q", got)
	}
}

func TestLoggerQuietSuppressesAllOutput(t *testing.T) {
	t.Parallel()

	var infoBuf bytes.Buffer
	var errBuf bytes.Buffer

	logger := NewWithWriters(true, true, &infoBuf, &errBuf)
	logger.Infof("info")
	logger.Verbosef("verbose")
	logger.Errorf("error")

	if got := infoBuf.String(); got != "" {
		t.Fatalf("expected no info output, got %q", got)
	}
	if got := errBuf.String(); got != "" {
		t.Fatalf("expected no error output, got %q", got)
	}
}
