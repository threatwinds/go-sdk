package entities

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// ValidateHexadecimal validates if the given value is a valid hexadecimal string.
// It returns the hexadecimal string in lowercase format, its SHA3-256 hash and an error if any.
func ValidateHexadecimal(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	v = strings.ToLower(v)

	h, err := hex.DecodeString(v)
	if err != nil {
		return "", "", err
	}

	hstr := hex.EncodeToString(h)

	return hstr, GenerateSHA3256(hstr), nil
}
