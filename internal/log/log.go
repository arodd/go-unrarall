package log

import (
	"fmt"
	"io"
	"os"
)

// Logger provides simple leveled logging controls.
type Logger struct {
	quiet       bool
	verbose     bool
	infoWriter  io.Writer
	errorWriter io.Writer
}

// New creates a new logger.
func New(quiet, verbose bool) *Logger {
	return NewWithWriters(quiet, verbose, os.Stdout, os.Stderr)
}

// NewWithWriters creates a logger that writes info/verbose and error output to custom sinks.
func NewWithWriters(quiet, verbose bool, infoWriter, errorWriter io.Writer) *Logger {
	if infoWriter == nil {
		infoWriter = io.Discard
	}
	if errorWriter == nil {
		errorWriter = io.Discard
	}

	return &Logger{
		quiet:       quiet,
		verbose:     verbose,
		infoWriter:  infoWriter,
		errorWriter: errorWriter,
	}
}

// Infof logs a standard informational message.
func (l *Logger) Infof(format string, args ...any) {
	if l.quiet {
		return
	}
	fmt.Fprintf(l.infoWriter, format+"\n", args...)
}

// Verbosef logs details that should only appear in verbose mode.
func (l *Logger) Verbosef(format string, args ...any) {
	if l.quiet || !l.verbose {
		return
	}
	fmt.Fprintf(l.infoWriter, format+"\n", args...)
}

// Errorf logs errors to stderr.
func (l *Logger) Errorf(format string, args ...any) {
	if l.quiet {
		return
	}
	fmt.Fprintf(l.errorWriter, format+"\n", args...)
}
