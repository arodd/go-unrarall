package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arodd/go-unrarall/internal/app"
	"github.com/arodd/go-unrarall/internal/cli"
	logpkg "github.com/arodd/go-unrarall/internal/log"
)

func TestRunWithIOHelpWritesToConsoleAndLogFile(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "unrarall.log")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runWithIO([]string{"unrarall", "--help", "--log-file", logPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("runWithIO exit code=%d, want 0", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	if got, want := string(logBytes), stdout.String(); got != want {
		t.Fatalf("log output mismatch\nlog: %q\nstdout: %q", got, want)
	}
	if !strings.Contains(string(logBytes), "--log-file FILE") {
		t.Fatalf("expected help output to include --log-file, got %q", string(logBytes))
	}
}

func TestRunWithIOParseErrorWritesToLogFileWhenProvided(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "parse.log")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runWithIO(
		[]string{"unrarall", "--not-a-flag", "--log-file", logPath},
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("runWithIO exit code=%d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Error:") {
		t.Fatalf("expected parse error output, got %q", errOutput)
	}
	if !strings.Contains(errOutput, "Usage: unrarall") {
		t.Fatalf("expected usage output, got %q", errOutput)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read parse log file: %v", err)
	}
	logOutput := string(logBytes)
	if !strings.Contains(logOutput, "Error:") {
		t.Fatalf("expected parse error in log output, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "Usage: unrarall") {
		t.Fatalf("expected usage in log output, got %q", logOutput)
	}
}

func TestRunWithIOVersionAppendsAcrossInvocations(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "append.log")

	var stdoutA bytes.Buffer
	var stderrA bytes.Buffer
	if exitCode := runWithIO([]string{"first", "--version", "--log-file", logPath}, &stdoutA, &stderrA); exitCode != 0 {
		t.Fatalf("first runWithIO exit code=%d, want 0", exitCode)
	}

	var stdoutB bytes.Buffer
	var stderrB bytes.Buffer
	if exitCode := runWithIO([]string{"second", "--version", "--log-file", logPath}, &stdoutB, &stderrB); exitCode != 0 {
		t.Fatalf("second runWithIO exit code=%d, want 0", exitCode)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read append log file: %v", err)
	}

	want := "first " + version + "\nsecond " + version + "\n"
	if got := string(logBytes); got != want {
		t.Fatalf("append log output=%q, want %q", got, want)
	}
}

func TestRunWithIORuntimeMessagesTeeToLogFile(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "runtime.log")

	originalRunApp := runApp
	defer func() {
		runApp = originalRunApp
	}()
	runApp = func(_ cli.Options, logger *logpkg.Logger) (app.Stats, error) {
		logger.Infof("runtime info")
		return app.Stats{}, errors.New("runtime boom")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runWithIO([]string{"unrarall", "--log-file", logPath, root}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("runWithIO exit code=%d, want 1", exitCode)
	}

	if !strings.Contains(stdout.String(), "runtime info") {
		t.Fatalf("expected runtime info on stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Run failed: runtime boom") {
		t.Fatalf("expected runtime error on stderr, got %q", stderr.String())
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read runtime log file: %v", err)
	}
	logOutput := string(logBytes)
	if !strings.Contains(logOutput, "runtime info") {
		t.Fatalf("expected runtime info in log file, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "Run failed: runtime boom") {
		t.Fatalf("expected runtime error in log file, got %q", logOutput)
	}
}
