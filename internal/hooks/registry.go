package hooks

import "slices"

type definition struct {
	Name string
	Help string
	Run  func(Context) error
}

var definitions = []definition{
	{
		Name: "nfo",
		Help: "Remove <stem>.nfo from the extraction root.",
		Run:  runNFO,
	},
	{
		Name: "rar",
		Help: "Remove RAR volumes and matching SFV files next to the archive.",
		Run:  runRAR,
	},
	{
		Name: "osx_junk",
		Help: "Remove .DS_Store from the extraction root.",
		Run:  runOSXJunk,
	},
	{
		Name: "windows_junk",
		Help: "Remove Thumbs.db from the extraction root.",
		Run:  runWindowsJunk,
	},
	{
		Name: "covers_folders",
		Help: "Remove directories named covers recursively from the extraction root.",
		Run:  runCoversFolders,
	},
	{
		Name: "proof_folders",
		Help: "Remove directories named proof recursively from the extraction root.",
		Run:  runProofFolders,
	},
	{
		Name: "sample_folders",
		Help: "Remove directories named sample recursively from the extraction root.",
		Run:  runSampleFolders,
	},
	{
		Name: "sample_videos",
		Help: "Remove root sample video files related to the archive stem.",
		Run:  runSampleVideos,
	},
	{
		Name: "empty_folders",
		Help: "Remove empty directories recursively from the archive directory.",
		Run:  runEmptyFolders,
	},
}

// Doc describes a cleanup hook for usage/help rendering.
type Doc struct {
	Name string
	Help string
}

// Docs returns the available cleanup hooks in execution order.
func Docs() []Doc {
	out := make([]Doc, 0, len(definitions))
	for _, def := range definitions {
		out = append(out, Doc{
			Name: def.Name,
			Help: def.Help,
		})
	}
	return out
}

// IsKnown reports whether name is a recognized cleanup hook token.
func IsKnown(name string) bool {
	if name == "none" || name == "all" {
		return true
	}

	for _, def := range definitions {
		if def.Name == name {
			return true
		}
	}
	return false
}

func isVirtual(name string) bool {
	return name == "none" || name == "all"
}

func resolveNames(selection []string) []string {
	if len(selection) == 0 {
		return nil
	}
	if len(selection) == 1 && selection[0] == "none" {
		return nil
	}
	if len(selection) == 1 && selection[0] == "all" {
		out := make([]string, 0, len(definitions))
		for _, def := range definitions {
			out = append(out, def.Name)
		}
		return out
	}

	out := make([]string, 0, len(selection))
	for _, name := range selection {
		if isVirtual(name) {
			continue
		}
		if slices.Contains(out, name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func lookup(name string) (definition, bool) {
	for _, def := range definitions {
		if def.Name == name {
			return def, true
		}
	}
	return definition{}, false
}
