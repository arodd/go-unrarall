package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/austin/go-unrarall/internal/cli"
	"github.com/austin/go-unrarall/internal/finder"
	"github.com/austin/go-unrarall/internal/log"
	"github.com/austin/go-unrarall/internal/rar"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	opts, err := cli.ParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprint(os.Stderr, cli.Usage(filepath.Base(args[0])))
		return 1
	}

	program := filepath.Base(args[0])
	if opts.ShowHelp {
		fmt.Print(cli.Usage(program))
		return 0
	}
	if opts.ShowVersion {
		fmt.Printf("%s %s\n", program, version)
		return 0
	}

	logger := log.New(opts.Quiet, opts.Verbose)

	candidates, err := finder.Scan(opts.Dir, opts.Depth)
	if err != nil {
		logger.Errorf("Failed to scan directory %q: %v", opts.Dir, err)
		return 1
	}
	if len(candidates) == 0 {
		logger.Infof("No candidate archives found in %s.", opts.Dir)
		return 0
	}

	validCandidates := 0
	for _, candidate := range candidates {
		ok, err := rar.HasRarSignature(candidate.Path)
		if err != nil {
			logger.Errorf("Failed to inspect archive %q: %v", candidate.Path, err)
			continue
		}
		if !ok {
			logger.Verbosef("Skipping %q: missing RAR signature.", candidate.Path)
			continue
		}
		validCandidates++
		logger.Verbosef("Validated candidate archive %q.", candidate.Path)
	}

	logger.Infof("Found %d candidate archive(s), %d signature-validated.", len(candidates), validCandidates)
	return 0
}
