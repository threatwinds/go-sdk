package entities

import (
	"strings"

	"github.com/google/uuid"
)

// ValidateUUID validates if a given value is a valid UUID string and returns the UUID, its SHA3-256 hash and an error if any.
func ValidateUUID(value string) (uuid.UUID, string, error) {
	u, err := uuid.Parse(strings.ToLower(value))
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return u, GenerateSHA3256(u.String()), nil
}
