package entities

import (
	"fmt"
	"strings"
)

// ValidateSHA512224 validates if a given value is a valid SHA512/224 hash.
// It receives a value of any type and returns the validated hash as a string,
// the hash generated using SHA3-256 and an error if the value is not a string or
// if it doesn't match the expected format.
func ValidateSHA512224(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}
	v = strings.ToLower(v)
	e := ValidateRegEx(`^[0-9a-f]{56}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
