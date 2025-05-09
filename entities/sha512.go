package entities

import (
	"fmt"
	"strings"
)

// ValidateSHA512 validates if a given value is a valid SHA512 hash.
// It receives a value of any type and returns the validated hash as a string,
// its SHA3256 hash as a string and an error if the value is not a valid SHA512 hash.
func ValidateSHA512(value interface{}) (string, string, error) {
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
