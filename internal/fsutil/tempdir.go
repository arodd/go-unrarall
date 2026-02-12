package fsutil

import (
	"fmt"
	"os"
	"strings"
)

// CreateTempDir creates an extraction temp directory under parent.
func CreateTempDir(parent string) (string, error) {
	if strings.TrimSpace(parent) == "" {
		return "", fmt.Errorf("temp parent directory is required")
	}
	return os.MkdirTemp(parent, ".unrarall-")
}
