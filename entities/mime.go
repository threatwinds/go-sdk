package entities

import (
	"strings"
)

// ValidateMime validates if a given string is a valid MIME type and returns the validated string, its SHA3-256 hash and an error if any.
func ValidateMime(value string) (string, string, error) {
	v := strings.ToLower(value)

	e := ValidateRegEx(`^([a-z]+)[/]([a-z0-9]+[a-z0-9+-.][a-z0-9]+)+$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
