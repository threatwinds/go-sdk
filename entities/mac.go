package entities

import (
	"strings"
)

// ValidateMAC validates if a given string is a valid MAC address and returns the MAC address in uppercase
// and its SHA3-256 hash.
func ValidateMAC(value string) (string, string, error) {
	v := strings.ToUpper(value)

	e := ValidateRegEx(`^([0-9A-F]{2,2}[-]){5,5}([0-9A-F]{2,2})$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
