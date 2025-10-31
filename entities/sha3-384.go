package entities

import (
	"strings"
)

// ValidateSHA3256 validates if a given string is a valid SHA3-256 hash and returns the hash in lowercase and its SHA3-256 hash.
func ValidateSHA3384(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{96}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
