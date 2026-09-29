package entities

import (
	"encoding/hex"
	"strings"
)

// ValidateHexadecimal validates if the given value is a valid hexadecimal string.
// It returns the hexadecimal string in lowercase format, its SHA3-256 hash and an error if any.
func ValidateHexadecimal(value string) (string, string, error) {
	v := strings.ToLower(value)

	h, err := hex.DecodeString(v)
	if err != nil {
		return "", "", err
	}

	hstr := hex.EncodeToString(h)

	return hstr, GenerateSHA3256(hstr), nil
}
