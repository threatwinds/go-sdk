package entities

import (
	"fmt"
)

// ValidateBoolean validates if a given value is a boolean and generates a SHA3-256 hash of the value.
// Returns a boolean indicating if the value is a boolean, the SHA3-256 hash of the value and an error if any.
func ValidateBoolean(value interface{}) (bool, string, error) {
	v, ok := value.(bool)
	if !ok {
		return false, "", fmt.Errorf("value is not boolean: %v", value)
	}

	return v, GenerateSHA3256(fmt.Sprint(v)), nil
}
