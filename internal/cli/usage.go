package cli

import (
	"fmt"
	"strings"

	"github.com/arodd/go-unrarall/internal/hooks"
)

// Usage renders command usage text.
func Usage(program string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Usage: %s [options] <DIRECTORY>\n", program)
	fmt.Fprintf(&b, "       %s --help\n", program)
	fmt.Fprintf(&b, "       %s --version\n\n", program)

	b.WriteString("Options:\n")
	b.WriteString("  -h, --help               Show this help message and exit.\n")
	b.WriteString("      --version            Show version information and exit.\n")
	b.WriteString("  -v, --verbose            Enable verbose logging (ignored when --quiet is set).\n")
	b.WriteString("  -q, --quiet              Suppress command output.\n")
	b.WriteString("  -d, --dry                Dry-run mode (log actions, no file writes).\n")
	b.WriteString("  -f, --force              Continue when SFV/extraction checks fail; run clean hooks after failures.\n")
	b.WriteString("      --allow-failures     Return success when some extractions succeed.\n")
	b.WriteString("  -s, --disable-cksfv      Disable SFV verification for <stem>.sfv.\n")
	b.WriteString("      --clean=SPEC         none|all|hook1,hook2 (default: none).\n")
	b.WriteString("      --full-path          Preserve full archive paths while extracting.\n")
	b.WriteString("      --allow-symlinks     Allow symlink entries with in-tree target validation.\n")
	b.WriteString("  -o, --output DIR         Output directory (must already exist).\n")
	b.WriteString("      --log-file FILE      Append command output to FILE while still writing to console.\n")
	b.WriteString("      --depth N            Nested recursion depth budget (default: 4; top-level scan is unbounded).\n")
	b.WriteString("      --skip-if-exists     Skip extraction when files already exist.\n")
	b.WriteString("      --password-file FILE Password file path (default: ~/.unrar_passwords).\n")
	b.WriteString("      --max-dict BYTES     Max allowed RAR dictionary bytes (default: 1073741824).\n")
	b.WriteString("\n")

	b.WriteString("Clean Hooks:\n")
	for _, hook := range hooks.Docs() {
		fmt.Fprintf(&b, "  %s: %s\n", hook.Name, hook.Help)
	}
	b.WriteString("  all: Run all hooks in default order.\n")
	b.WriteString("  none: Disable cleanup hooks.\n")

	return b.String()
}
