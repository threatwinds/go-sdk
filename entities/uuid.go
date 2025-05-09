package entities

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidateUUID validates if a given value is a valid UUID string and returns the UUID, its SHA3-256 hash and an error if any.
func ValidateUUID(value interface{}) (uuid.UUID, string, error) {
	v, ok := value.(string)
	if !ok {
		return uuid.UUID{}, "", fmt.Errorf("value is not string: %v", value)
	}

	u, err := uuid.Parse(strings.ToLower(v))
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return u, GenerateSHA3256(u.String()), nil
}
