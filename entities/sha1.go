package entities

import (
	"strings"
)

// ValidateSHA1 validates if a given value is a valid SHA1 hash.
// It receives a value of any type and returns the validated SHA1 hash as a string,
// its SHA3-256 hash as a string and an error if the value is not a string or if it is not a valid SHA1 hash.
func ValidateSHA1(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{40}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
