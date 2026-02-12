package finder

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var partVolumeRe = regexp.MustCompile(`(?i)^(.*)\.part([0-9]+)\.rar$`)

// Candidate describes a candidate first-volume archive.
type Candidate struct {
	Path string
	Stem string
}

// IsFirstVolume reports whether filename looks like the first volume of an archive set.
func IsFirstVolume(filename string) (bool, string) {
	lower := strings.ToLower(filename)

	if strings.HasSuffix(lower, ".001") {
		return true, filename[:len(filename)-len(".001")]
	}

	match := partVolumeRe.FindStringSubmatch(filename)
	if match != nil {
		partNum, err := strconv.Atoi(match[2])
		if err != nil || partNum != 1 {
			return false, ""
		}
		return true, match[1]
	}

	if strings.HasSuffix(lower, ".rar") {
		return true, filename[:len(filename)-len(filepath.Ext(filename))]
	}
	return false, ""
}
