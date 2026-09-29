package entities

import (
	"net/mail"
	"strings"
)

// ValidateEmail validates if a given string is a valid email address.
// It returns the email address, its SHA3-256 hash and an error if any.
func ValidateEmail(value string) (string, string, error) {
	addr, err := mail.ParseAddress(strings.ToLower(value))
	if err != nil {
		return "", "", err
	}

	return addr.Address, GenerateSHA3256(addr.Address), nil
}
