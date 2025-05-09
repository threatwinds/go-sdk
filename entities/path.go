package entities

import (
	"fmt"
	"strings"
)

// ValidatePath validates if the given value is a valid path and returns the path in lowercase and its SHA3-256 hash.
// If the value is not a string or contains "://" it returns an error.
func ValidatePath(value string) (string, string, error) {
	v := strings.ToLower(value)
	if strings.Contains(v, "://") {
		return "", "", fmt.Errorf("value is not valid path: %v", value)
	}
	return v, GenerateSHA3256(v), nil
}
