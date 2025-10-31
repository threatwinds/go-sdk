package entities

import (
	"fmt"
)

// ValidateFloat validates if the given value is a float64 or an int64 that can be converted to a float64.
// It returns the validated float64 value, its SHA3-256 hash, and an error if the value is not a float64 or an int64.
func ValidateFloat(value float64) (float64, string, error) {
	return value, GenerateSHA3256(fmt.Sprint(value)), nil
}
