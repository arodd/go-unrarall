package finder

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestIsFirstVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		ok       bool
		stem     string
	}{
		{name: "plain rar", filename: "release.rar", ok: true, stem: "release"},
		{name: "part 01 rar", filename: "release.part01.rar", ok: true, stem: "release"},
		{name: "part 001 rar mixed case", filename: "release.PART001.RAR", ok: true, stem: "release"},
		{name: "part 2 rar", filename: "release.part02.rar", ok: false, stem: ""},
		{name: "001 first volume", filename: "release.001", ok: true, stem: "release"},
		{name: "002 not first volume", filename: "release.002", ok: false, stem: ""},
		{name: "r00 sidecar", filename: "release.r00", ok: false, stem: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok, stem := IsFirstVolume(tc.filename)
			if ok != tc.ok || stem != tc.stem {
				t.Fatalf("IsFirstVolume(%q) = (%v, %q), want (%v, %q)", tc.filename, ok, stem, tc.ok, tc.stem)
			}
		})
	}
}

func TestScanRespectsDepthAndFilters(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustTouch(t, filepath.Join(root, "movie.rar"))
	mustTouch(t, filepath.Join(root, "movie.r00"))
	mustTouch(t, filepath.Join(root, "series.part01.rar"))
	mustTouch(t, filepath.Join(root, "series.part02.rar"))
	mustTouch(t, filepath.Join(root, "pack.001"))
	mustTouch(t, filepath.Join(root, "pack.002"))

	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	mustTouch(t, filepath.Join(nested, "deep.part1.rar"))

	deepNested := filepath.Join(nested, "deeper")
	if err := os.MkdirAll(deepNested, 0o755); err != nil {
		t.Fatalf("mkdir deep nested: %v", err)
	}
	mustTouch(t, filepath.Join(deepNested, "too-deep.rar"))

	candidatesDepth0, err := Scan(root, 0)
	if err != nil {
		t.Fatalf("Scan depth 0 returned error: %v", err)
	}
	gotDepth0 := candidateNames(candidatesDepth0)
	wantDepth0 := []string{
		"movie.rar",
		"pack.001",
		"series.part01.rar",
	}
	if !reflect.DeepEqual(gotDepth0, wantDepth0) {
		t.Fatalf("depth 0 candidates = %v, want %v", gotDepth0, wantDepth0)
	}

	candidatesDepth1, err := Scan(root, 1)
	if err != nil {
		t.Fatalf("Scan depth 1 returned error: %v", err)
	}
	gotDepth1 := candidateNames(candidatesDepth1)
	wantDepth1 := []string{
		"deep.part1.rar",
		"movie.rar",
		"pack.001",
		"series.part01.rar",
	}
	if !reflect.DeepEqual(gotDepth1, wantDepth1) {
		t.Fatalf("depth 1 candidates = %v, want %v", gotDepth1, wantDepth1)
	}

	candidatesUnbounded, err := Scan(root, -1)
	if err != nil {
		t.Fatalf("Scan unbounded returned error: %v", err)
	}
	gotUnbounded := candidateNames(candidatesUnbounded)
	wantUnbounded := []string{
		"deep.part1.rar",
		"movie.rar",
		"pack.001",
		"series.part01.rar",
		"too-deep.rar",
	}
	if !reflect.DeepEqual(gotUnbounded, wantUnbounded) {
		t.Fatalf("unbounded candidates = %v, want %v", gotUnbounded, wantUnbounded)
	}
}

func mustTouch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func candidateNames(candidates []Candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, filepath.Base(candidate.Path))
	}
	sort.Strings(out)
	return out
}
