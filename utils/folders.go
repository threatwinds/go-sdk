package utils

import (
	"os"
	"path/filepath"
)

// MkdirJoin creates a directory structure specified by the joined path of given parts and returns the complete path.
// It uses `os.MkdirAll` with a default permission of 0755 for creating directories.
// Returns the resulting path and an error if any occurs during directory creation.
func MkdirJoin(f ...string) (string, error) {
	address := filepath.Join(f...)
	err := os.MkdirAll(address, 0755)
	return address, err
}

// FileJoin concatenates multiple path components into a single path using platform-specific path separators.
func FileJoin(f ...string) string {
	return filepath.Join(f...)
}
