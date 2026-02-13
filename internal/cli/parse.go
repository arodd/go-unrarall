package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/austin/go-unrarall/internal/hooks"
)

// Options contains parsed command-line options.
type Options struct {
	Dir           string
	OutputDir     string
	LogFile       string
	Depth         int
	SkipIfExists  bool
	FullPath      bool
	AllowSymlinks bool
	Force         bool
	DryRun        bool
	Quiet         bool
	Verbose       bool
	AllowFailures bool

	CKSFV        bool
	PasswordFile string

	CleanHooks   []string
	MaxDictBytes int64

	ShowHelp    bool
	ShowVersion bool
}

// ParseArgs parses and validates command-line arguments.
func ParseArgs(args []string) (Options, error) {
	opts := defaultOptions()
	fs := flag.NewFlagSet("unrarall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		disableCK bool
		cleanSpec string
		logFile   requiredPathFlag
	)

	fs.BoolVar(&opts.Verbose, "verbose", false, "")
	fs.BoolVar(&opts.Verbose, "v", false, "")
	fs.BoolVar(&opts.Quiet, "quiet", false, "")
	fs.BoolVar(&opts.Quiet, "q", false, "")
	fs.BoolVar(&opts.DryRun, "dry", false, "")
	fs.BoolVar(&opts.DryRun, "d", false, "")
	fs.BoolVar(&opts.Force, "force", false, "")
	fs.BoolVar(&opts.Force, "f", false, "")
	fs.BoolVar(&opts.AllowFailures, "allow-failures", false, "")
	fs.BoolVar(&disableCK, "disable-cksfv", false, "")
	fs.BoolVar(&disableCK, "s", false, "")
	fs.BoolVar(&opts.FullPath, "full-path", false, "")
	fs.BoolVar(&opts.AllowSymlinks, "allow-symlinks", false, "")
	fs.IntVar(&opts.Depth, "depth", 4, "")
	fs.BoolVar(&opts.SkipIfExists, "skip-if-exists", false, "")
	fs.StringVar(&opts.OutputDir, "output", "", "")
	fs.StringVar(&opts.OutputDir, "o", "", "")
	fs.Var(&logFile, "log-file", "")
	fs.StringVar(&opts.PasswordFile, "password-file", opts.PasswordFile, "")
	fs.StringVar(&cleanSpec, "clean", "none", "")
	fs.Int64Var(&opts.MaxDictBytes, "max-dict", 1<<30, "")
	fs.BoolVar(&opts.ShowVersion, "version", false, "")
	fs.BoolVar(&opts.ShowHelp, "help", false, "")
	fs.BoolVar(&opts.ShowHelp, "h", false, "")

	if err := fs.Parse(args[1:]); err != nil {
		return Options{}, err
	}

	if opts.Quiet {
		// Match script parity: quiet suppresses output even if verbose is also set.
		opts.Verbose = false
	}
	if opts.Depth < 0 {
		return Options{}, fmt.Errorf("--depth must be >= 0")
	}
	if opts.MaxDictBytes <= 0 {
		return Options{}, fmt.Errorf("--max-dict must be > 0")
	}

	hooks, err := parseCleanHooks(cleanSpec)
	if err != nil {
		return Options{}, err
	}
	opts.CleanHooks = hooks
	opts.CKSFV = !disableCK

	if logFile.set {
		opts.LogFile = logFile.value
	}
	if opts.LogFile != "" {
		opts.LogFile, err = filepath.Abs(opts.LogFile)
		if err != nil {
			return Options{}, fmt.Errorf("failed to resolve log file path: %w", err)
		}
	}

	if opts.ShowHelp || opts.ShowVersion {
		return opts, nil
	}

	if fs.NArg() != 1 {
		return Options{}, fmt.Errorf("expected exactly one DIRECTORY argument")
	}

	opts.Dir, err = filepath.Abs(fs.Arg(0))
	if err != nil {
		return Options{}, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	if opts.OutputDir != "" {
		opts.OutputDir, err = filepath.Abs(opts.OutputDir)
		if err != nil {
			return Options{}, fmt.Errorf("failed to resolve output directory path: %w", err)
		}
	}

	if err := validatePaths(opts); err != nil {
		return Options{}, err
	}

	return opts, nil
}

func defaultOptions() Options {
	return Options{
		Depth:         4,
		CKSFV:         true,
		CleanHooks:    []string{"none"},
		MaxDictBytes:  1 << 30,
		PasswordFile:  defaultPasswordFile(),
		ShowHelp:      false,
		ShowVersion:   false,
		AllowFailures: false,
	}
}

func validatePaths(opts Options) error {
	info, err := os.Stat(opts.Dir)
	if err != nil {
		return fmt.Errorf("directory %q: %w", opts.Dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", opts.Dir)
	}

	if opts.OutputDir == "" {
		return nil
	}

	outInfo, err := os.Stat(opts.OutputDir)
	if err != nil {
		return fmt.Errorf("output directory %q: %w", opts.OutputDir, err)
	}
	if !outInfo.IsDir() {
		return fmt.Errorf("output path %q is not a directory", opts.OutputDir)
	}
	return nil
}

func parseCleanHooks(spec string) ([]string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("clean up hooks must be specified when using --clean=")
	}

	parts := strings.Split(spec, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		hook := strings.ToLower(strings.TrimSpace(part))
		if hook == "" {
			return nil, fmt.Errorf("--clean contains an empty hook name")
		}
		if !isKnownHook(hook) {
			return nil, fmt.Errorf("unknown clean hook %q", hook)
		}
		if _, ok := seen[hook]; ok {
			continue
		}
		seen[hook] = struct{}{}
		out = append(out, hook)
	}

	if len(out) > 1 && slices.Contains(out, "none") {
		return nil, fmt.Errorf("--clean=none cannot be combined with other hooks")
	}
	if len(out) > 1 && slices.Contains(out, "all") {
		return nil, fmt.Errorf("--clean=all cannot be combined with other hooks")
	}
	return out, nil
}

func isKnownHook(name string) bool {
	return hooks.IsKnown(name)
}

func defaultPasswordFile() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".unrar_passwords"
	}
	return filepath.Join(home, ".unrar_passwords")
}

type requiredPathFlag struct {
	set   bool
	value string
}

func (f *requiredPathFlag) String() string {
	return f.value
}

func (f *requiredPathFlag) Set(value string) error {
	f.set = true
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--log-file requires FILE")
	}
	f.value = value
	return nil
}
