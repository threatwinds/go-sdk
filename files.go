package go_sdk

import (
	"strings"
)

// ValidateFilePath validates the file path to ensure it does not contain any invalid characters or directory traversal attempts
func ValidateFilePath(path string) error {
	var contains []string = []string{
		"..",
		"~",
	}

	var prefixes []string = []string{
		"/",
	}

	for _, c := range contains {
		if strings.Contains(path, c) {
			return Error(Trace(), map[string]interface{}{
				"error":   "path contains an invalid character",
				"path":    path,
				"invalid": c,
			})
		}
	}

	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return Error(Trace(), map[string]interface{}{
				"error":   "path starts with an invalid character",
				"path":    path,
				"invalid": p,
			})
		}
	}

	return nil
}
