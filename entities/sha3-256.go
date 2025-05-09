package entities

import (
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"
)

// ValidateSHA3256 validates if the given value is a valid SHA3-256 hash.
func ValidateSHA3256(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}
	v = strings.ToLower(v)
	e := ValidateRegEx(`^[0-9a-f]{64}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}

// GenerateSHA3256 generates a SHA3-256 hash from the given string.
func GenerateSHA3256(value string) string {
	sum := sha3.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
