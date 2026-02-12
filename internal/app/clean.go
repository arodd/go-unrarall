package app

import (
	"github.com/austin/go-unrarall/internal/hooks"
	"github.com/austin/go-unrarall/internal/log"
)

func shouldRunHooks(selected []string) bool {
	return !(len(selected) == 0 || (len(selected) == 1 && selected[0] == "none"))
}

func runCleanupHooks(
	selected []string,
	extractRoot string,
	rarDir string,
	stem string,
	dryRun bool,
	logger *log.Logger,
) error {
	return hooks.Run(selected, hooks.Context{
		ExtractRoot: extractRoot,
		RarDir:      rarDir,
		Stem:        stem,
		DryRun:      dryRun,
		Log:         logger,
	})
}
