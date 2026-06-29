package entities

import (
	"strings"
)

// ValidateSHA3512 validates if a given string is a valid SHA3-512 hash and returns the hash in lowercase and its SHA3-256 hash.
func ValidateSHA3512(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{128}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
