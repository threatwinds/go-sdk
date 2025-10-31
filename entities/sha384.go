package entities

import (
	"strings"
)

// ValidateSHA384 validates a string value as a SHA384 hash and returns the hash value, its SHA3256 hash, and an error if any.
func ValidateSHA384(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{96}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
