package cli

import (
	"fmt"
	"strings"
)

type hookDoc struct {
	Name string
	Help string
}

var hookDocs = []hookDoc{
	{Name: "nfo", Help: "Remove <stem>.nfo from the extraction root."},
	{Name: "rar", Help: "Remove RAR volumes and matching SFV files next to the archive."},
	{Name: "osx_junk", Help: "Remove .DS_Store from the extraction root."},
	{Name: "windows_junk", Help: "Remove Thumbs.db from the extraction root."},
	{Name: "covers_folders", Help: "Remove directories named covers recursively from the extraction root."},
	{Name: "proof_folders", Help: "Remove directories named proof recursively from the extraction root."},
	{Name: "sample_folders", Help: "Remove directories named sample recursively from the extraction root."},
	{Name: "sample_videos", Help: "Remove root sample video files related to the archive stem."},
	{Name: "empty_folders", Help: "Remove empty directories recursively from the archive directory."},
}

// Usage renders command usage text.
func Usage(program string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Usage: %s [options] <DIRECTORY>\n", program)
	fmt.Fprintf(&b, "       %s --help\n", program)
	fmt.Fprintf(&b, "       %s --version\n\n", program)

	b.WriteString("Options:\n")
	b.WriteString("  -h, --help               Show this help message and exit.\n")
	b.WriteString("      --version            Show version information and exit.\n")
	b.WriteString("  -v, --verbose            Enable verbose logging.\n")
	b.WriteString("  -q, --quiet              Suppress command output.\n")
	b.WriteString("  -d, --dry                Dry-run mode (planned behavior; no writes).\n")
	b.WriteString("  -f, --force              Force extraction when checks fail (planned behavior).\n")
	b.WriteString("      --allow-failures     Return success when some extractions succeed.\n")
	b.WriteString("  -s, --disable-cksfv      Skip SFV verification (planned behavior).\n")
	b.WriteString("      --clean=SPEC         none|all|hook1,hook2 (default: none).\n")
	b.WriteString("      --full-path          Preserve full archive paths while extracting.\n")
	b.WriteString("  -o, --output DIR         Output directory (must already exist).\n")
	b.WriteString("      --depth N            Recursive scan depth (default: 4).\n")
	b.WriteString("      --skip-if-exists     Skip extraction when files already exist.\n")
	b.WriteString("      --password-file FILE Password file path (default: ~/.unrar_passwords).\n")
	b.WriteString("      --max-dict BYTES     Max allowed RAR dictionary bytes (default: 1073741824).\n")
	b.WriteString("\n")

	b.WriteString("Clean Hooks:\n")
	for _, hook := range hookDocs {
		fmt.Fprintf(&b, "  %s: %s\n", hook.Name, hook.Help)
	}
	b.WriteString("  all: Run all hooks in default order.\n")
	b.WriteString("  none: Disable cleanup hooks.\n")

	return b.String()
}
