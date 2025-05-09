package entities

import (
	"fmt"
	"strings"
)

// ValidateMD5 validates if a given string is a valid MD5 hash.
// It receives a value of type interface{} and returns the validated string, its SHA3-256 hash and an error.
func ValidateMD5(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	v = strings.ToLower(v)

	if len([]rune(v)) == 32 {
		e := ValidateRegEx(`^[0-9a-f]{32}$`, v)
		if e != nil {
			return "", "", e
		}
	} else {
		e := ValidateRegEx(`^[0-9a-f]{16}$`, v)
		if e != nil {
			return "", "", e
		}
	}

	return v, GenerateSHA3256(v), nil
}
