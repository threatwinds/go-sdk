package utils

import (
	"os"
	"path/filepath"
)

type Folder string

// MkdirJoin creates a directory structure specified by the joined path of given parts and returns the complete path.
// It uses `os.MkdirAll` with a default permission of 0755 for creating directories.
// Returns the resulting path and an error if any occurs during directory creation.
func MkdirJoin(f ...string) (Folder, error) {
	address := filepath.Join(f...)
	err := os.MkdirAll(address, 0755)
	return Folder(address), err
}

// FileJoin concatenates folder and file components into a single path using platform-specific path separators.
func (folder *Folder) FileJoin(file string) string {
	return filepath.Join(folder.String(), file)
}

// String returns the string representation of the Folder type.
func (folder *Folder) String() string {
	return string(*folder)
}
