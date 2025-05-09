package entities

import (
	"fmt"
	"strings"
)

// ValidateSHA1 validates if a given value is a valid SHA1 hash.
// It receives a value of any type and returns the validated SHA1 hash as a string,
// its SHA3-256 hash as a string and an error if the value is not a string or if it is not a valid SHA1 hash.
func ValidateSHA1(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	v = strings.ToLower(v)
	e := ValidateRegEx(`^[0-9a-f]{40}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
