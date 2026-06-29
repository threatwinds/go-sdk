package utils

import (
	"fmt"
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
			return fmt.Errorf("path contains an invalid character: path=%s, invalid=%s", path, c)
		}
	}

	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return fmt.Errorf("path starts with an invalid character: path=%s, invalid=%s", path, p)
		}
	}

	return nil
}
