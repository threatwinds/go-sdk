package entities

import (
	"fmt"
	"reflect"
)

// ValidateFloat validates if the given value is a float64 or an int64 that can be converted to a float64.
// It returns the validated float64 value, its SHA3-256 hash, and an error if the value is not a float64 or an int64.
func ValidateFloat(value interface{}) (float64, string, error) {
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Int64 {
		value = float64(value.(int64))
	}
	v, ok := value.(float64)
	if !ok {
		return 0, "", fmt.Errorf("value is not float: %v", value)
	}

	return v, GenerateSHA3256(fmt.Sprint(v)), nil
}
