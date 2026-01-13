package entities

import (
	"encoding/base64"
)

// ValidateBase64 validates if a given string is a valid base64 encoded string.
// It returns the original string, its SHA3-256 hash and an error if the validation fails.
func ValidateBase64(value string) (string, string, error) {
	_, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", "", err
	}

	return value, GenerateSHA3256(value), nil
}
