package entities

import (
	"strings"
)

// ValidatePort validates a port with protocol.
// It returns the validated PORT, its SHA3-256 hash, and an error if the validation fails.
func ValidatePort(value string) (string, string, error) {
	v := strings.ToLower(value)

	e := ValidateRegEx(`^(6553[0-5]|655[0-2]\d|65[0-4]\d\d|6[0-4]\d{3}|[1-5]\d{4}|[1-9]\d{0,3})(\/)(tcp|udp)$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
