package entities

import (
	"fmt"
	"reflect"
)

// ValidateInteger validates if a value is an integer and returns its int64 representation,
// its SHA3-256 hash and an error if the value is not an integer.
func ValidateInteger(value interface{}) (int64, string, error) {
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Float64 {
		value = int64(value.(float64))
	}
	if t.Kind() == reflect.Int {
		value = int64(value.(int))
	}
	v, ok := value.(int64)
	if !ok {
		return 0, "", fmt.Errorf("value is not integer: %v", value)
	}

	return int64(v), GenerateSHA3256(fmt.Sprint(int(v))), nil
}
