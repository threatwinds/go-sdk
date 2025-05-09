package entities

import (
	"fmt"
	"strings"
)

// ValidateSHA3512 validates if a given string is a valid SHA3-512 hash and returns the hash in lowercase and its SHA3-256 hash.
func ValidateSHA3512(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}
	v = strings.ToLower(v)
	e := ValidateRegEx(`^[0-9a-f]{128}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
