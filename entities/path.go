package entities

import (
	"fmt"
	"strings"
)

// ValidatePath validates if the given value is a valid path and returns the path in lowercase and its SHA3-256 hash.
// If the value is not a string or contains "://" it returns an error.
func ValidatePath(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	v = strings.ToLower(v)

	if strings.Contains(v, "://") {
		return "", "", fmt.Errorf("value is not valid path: %v", value)
	}

	return v, GenerateSHA3256(v), nil
}
