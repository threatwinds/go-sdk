package entities

import (
	"fmt"
	"strings"
)

// ValidateSHA384 validates a string value as a SHA384 hash and returns the hash value, its SHA3256 hash, and an error if any.
func ValidateSHA384(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}
	v = strings.ToLower(v)
	e := ValidateRegEx(`^[0-9a-f]{96}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
