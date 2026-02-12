package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/austin/go-unrarall/internal/app"
	"github.com/austin/go-unrarall/internal/cli"
	"github.com/austin/go-unrarall/internal/log"
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
	stats, err := app.Run(opts, logger)
	if err != nil {
		logger.Errorf("Run failed: %v", err)
		return 1
	}

	return app.ExitCode(stats, opts.AllowFailures)
}
