package go_sdk

import (
	"errors"
	"strings"
)

// Helper function to validate file paths security.
func ValidateFilePath(path string) error {
	// Add validation logic for file paths
	// Check for directory traversal attempts
	if strings.Contains(path, "..") {
		return errors.New("path contains invalid characters")
	}

	return nil
}
