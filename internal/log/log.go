package log

import (
	"fmt"
	"os"
)

// Logger provides simple leveled logging controls.
type Logger struct {
	quiet   bool
	verbose bool
}

// New creates a new logger.
func New(quiet, verbose bool) *Logger {
	return &Logger{
		quiet:   quiet,
		verbose: verbose,
	}
}

// Infof logs a standard informational message.
func (l *Logger) Infof(format string, args ...any) {
	if l.quiet {
		return
	}
	fmt.Printf(format+"\n", args...)
}

// Verbosef logs details that should only appear in verbose mode.
func (l *Logger) Verbosef(format string, args ...any) {
	if l.quiet || !l.verbose {
		return
	}
	fmt.Printf(format+"\n", args...)
}

// Errorf logs errors to stderr.
func (l *Logger) Errorf(format string, args ...any) {
	if l.quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
