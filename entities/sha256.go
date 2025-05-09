package entities

import (
	"fmt"
	"strings"
)

// ValidateSHA256 validates that a given value is a valid SHA256 hash.
// It takes an interface{} value and returns the validated value as a string,
// the SHA3256 hash of the value as a string, and an error if the value is not a valid SHA256 hash.
func ValidateSHA256(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}
	v = strings.ToLower(v)
	e := ValidateRegEx(`^[0-9a-f]{64}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
