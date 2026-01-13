package entities

import (
	"strings"
)

// ValidateSHA256 validates that a given value is a valid SHA256 hash.
// It takes an interface{} value and returns the validated value as a string,
// the SHA3256 hash of the value as a string, and an error if the value is not a valid SHA256 hash.
func ValidateSHA256(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{64}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
