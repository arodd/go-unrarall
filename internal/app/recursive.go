package app

import "fmt"

func (r *runner) runRecursive(tmpDir string, depth int) (Stats, error) {
	if depth < 0 {
		return Stats{}, nil
	}

	nestedStats, err := r.runDirectory(tmpDir, depth)
	if err != nil {
		return nestedStats, err
	}
	if ExitCode(nestedStats, r.opts.AllowFailures) != 0 {
		return nestedStats, fmt.Errorf("nested extraction run failed")
	}
	return nestedStats, nil
}
