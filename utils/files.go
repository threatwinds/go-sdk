package utils

import (
	"github.com/threatwinds/go-sdk/catcher"
	"strings"
)

// ValidateFilePath validates the file path to ensure it does not contain any invalid characters or directory traversal attempts
func ValidateFilePath(path string) error {
	var contains = []string{
		"..",
		"~",
	}

	var prefixes = []string{
		"/",
	}

	for _, c := range contains {
		if strings.Contains(path, c) {
			return catcher.Error("path contains an invalid character", nil, map[string]any{
				"path":    path,
				"invalid": c,
			})
		}
	}

	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return catcher.Error("path starts with an invalid character", nil, map[string]any{
				"path":    path,
				"invalid": p,
			})
		}
	}

	return nil
}
