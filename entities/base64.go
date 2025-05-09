package entities

import (
	"encoding/base64"
	"fmt"
)

// ValidateBase64 validates if a given string is a valid base64 encoded string.
// It returns the original string, its SHA3-256 hash and an error if the validation fails.
func ValidateBase64(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}
	_, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", "", err
	}

	return v, GenerateSHA3256(v), nil
}
