package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arodd/go-unrarall/internal/app"
	"github.com/arodd/go-unrarall/internal/cli"
	"github.com/arodd/go-unrarall/internal/log"
)

// version may be overridden at build time with:
// -ldflags "-X main.version=<version>"
var version = "1.0.1"

var runApp = app.Run

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	return runWithIO(args, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdout, stderr io.Writer) (exitCode int) {
	program := programName(args)

	opts, err := cli.ParseArgs(args)
	if err != nil {
		parseStderr := stderr
		if logFilePath, ok := findLogFilePath(args); ok {
			_, sinkStderr, logFile, sinkErr := appendSinks(stdout, stderr, logFilePath)
			if sinkErr != nil {
				fmt.Fprintf(stderr, "Error: %v\n", sinkErr)
			} else {
				parseStderr = sinkStderr
				defer func() {
					if closeErr := logFile.Close(); closeErr != nil {
						fmt.Fprintf(stderr, "Error: close log file %q: %v\n", logFilePath, closeErr)
					}
				}()
			}
		}

		fmt.Fprintf(parseStderr, "Error: %v\n\n", err)
		fmt.Fprint(parseStderr, cli.Usage(program))
		return 1
	}

	stdoutSink := stdout
	stderrSink := stderr
	if opts.LogFile != "" {
		var sinkErr error
		var logFile *os.File
		stdoutSink, stderrSink, logFile, sinkErr = appendSinks(stdout, stderr, opts.LogFile)
		if sinkErr != nil {
			fmt.Fprintf(stderr, "Error: %v\n", sinkErr)
			return 1
		}
		defer func() {
			if closeErr := logFile.Close(); closeErr != nil {
				fmt.Fprintf(stderr, "Error: close log file %q: %v\n", opts.LogFile, closeErr)
				if exitCode == 0 {
					exitCode = 1
				}
			}
		}()
	}

	if opts.ShowHelp {
		fmt.Fprint(stdoutSink, cli.Usage(program))
		return 0
	}
	if opts.ShowVersion {
		fmt.Fprintf(stdoutSink, "%s %s\n", program, version)
		return 0
	}

	logger := log.NewWithWriters(opts.Quiet, opts.Verbose, stdoutSink, stderrSink)
	stats, runErr := runApp(opts, logger)
	if runErr != nil {
		logger.Errorf("Run failed: %v", runErr)
		return 1
	}

	return app.ExitCode(stats, opts.AllowFailures)
}

func appendSinks(stdout, stderr io.Writer, logFilePath string) (io.Writer, io.Writer, *os.File, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open log file %q: %w", logFilePath, err)
	}

	return io.MultiWriter(stdout, logFile), io.MultiWriter(stderr, logFile), logFile, nil
}

func findLogFilePath(args []string) (string, bool) {
	var (
		rawPath string
		found   bool
	)

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}

		if strings.HasPrefix(arg, "--log-file=") || strings.HasPrefix(arg, "-log-file=") {
			value := arg[strings.IndexByte(arg, '=')+1:]
			if strings.TrimSpace(value) == "" {
				continue
			}
			rawPath = value
			found = true
			continue
		}

		if arg == "--log-file" || arg == "-log-file" {
			if i+1 >= len(args) {
				break
			}
			value := args[i+1]
			if strings.TrimSpace(value) != "" {
				rawPath = value
				found = true
			}
			i++
		}
	}

	if !found {
		return "", false
	}

	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", false
	}
	return absPath, true
}

func programName(args []string) string {
	if len(args) == 0 || args[0] == "" {
		return "unrarall"
	}
	return filepath.Base(args[0])
}
