package entities

import (
	"fmt"
	"strings"
)

// ValidateMAC validates if a given string is a valid MAC address and returns the MAC address in uppercase
// and its SHA3-256 hash.
func ValidateMAC(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	v = strings.ToUpper(v)

	e := ValidateRegEx(`^([0-9A-F]{2,2}[-]){5,5}([0-9A-F]{2,2})$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
