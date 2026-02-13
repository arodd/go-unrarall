package hooks

import (
	"reflect"
	"testing"
)

func TestResolveNamesAllMatchesScriptOrder(t *testing.T) {
	t.Parallel()

	got := resolveNames([]string{"all"})
	want := []string{
		"covers_folders",
		"nfo",
		"osx_junk",
		"proof_folders",
		"rar",
		"sample_folders",
		"sample_videos",
		"windows_junk",
		"empty_folders",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveNames(all)=%v, want %v", got, want)
	}
}

func TestResolveNamesPreservesExplicitSelectionOrder(t *testing.T) {
	t.Parallel()

	got := resolveNames([]string{"sample_videos", "nfo", "sample_videos", "rar"})
	want := []string{"sample_videos", "nfo", "rar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveNames(explicit)=%v, want %v", got, want)
	}
}
