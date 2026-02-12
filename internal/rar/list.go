package rar

import "github.com/nwaples/rardecode/v2"

// ListedFile is a listing entry from a RAR archive.
type ListedFile struct {
	Name            string
	IsDir           bool
	Encrypted       bool
	HeaderEncrypted bool
}

// ListFiles returns all files in an archive without extracting contents.
func ListFiles(path string, opts ...rardecode.Option) ([]ListedFile, error) {
	files, err := rardecode.List(path, opts...)
	if err != nil {
		return nil, err
	}

	out := make([]ListedFile, 0, len(files))
	for _, file := range files {
		out = append(out, ListedFile{
			Name:            file.Name,
			IsDir:           file.IsDir,
			Encrypted:       file.Encrypted,
			HeaderEncrypted: file.HeaderEncrypted,
		})
	}
	return out, nil
}
