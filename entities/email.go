package entities

import (
	"fmt"
	"net/mail"
	"strings"
)

// ValidateEmail validates if a given string is a valid email address.
// It returns the email address, its SHA3-256 hash and an error if any.
func ValidateEmail(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	addr, err := mail.ParseAddress(strings.ToLower(v))
	if err != nil {
		return "", "", err
	}

	return addr.Address, GenerateSHA3256(addr.Address), nil
}
