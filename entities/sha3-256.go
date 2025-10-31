package entities

import (
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"
)

// ValidateSHA3256 validates if the given value is a valid SHA3-256 hash.
func ValidateSHA3256(value string) (string, string, error) {
	v := strings.ToLower(value)
	e := ValidateRegEx(`^[0-9a-f]{64}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}

// GenerateSHA3256 generates a SHA3-256 hash from the given value.
func GenerateSHA3256[T string | int64 | float64 | bool](value T) string {
	v := fmt.Sprintf("%v", value)
	sum := sha3.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
