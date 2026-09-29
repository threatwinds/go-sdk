package entities

import (
	"strings"
)

// ValidateSHA512 validates if a given value is a valid SHA512 hash.
// It receives a value of any type and returns the validated hash as a string,
// its SHA3256 hash as a string and an error if the value is not a valid SHA512 hash.
func ValidateSHA512(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{128}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
