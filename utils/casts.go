package utils

import (
	"fmt"
	"github.com/threatwinds/go-sdk/catcher"
	"strconv"
)

// CastInt64 attempts to cast a given interface{} value to an int64.
// It supports the following types: int, int64, float64, and string.
// If the value is a string, it tries to parse it as an int64.
// If the value cannot be cast to int64, it returns 0.
//
// Parameters:
//   - value: The value to be cast to int64.
//
// Returns:
//   - The int64 representation of the value, or 0 if the value cannot be cast.
func CastInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			_ = catcher.Error("failed to cast string to int64", err, map[string]any{"value": v})
			return 0
		}
		return val
	default:
		return 0
	}
}

// CastFloat64 attempts to cast an interface{} to a float64.
// It supports the following types: int, int64, float64, and string.
// If the value is a string, it tries to parse it as a float64.
// If the conversion is not possible, it returns 0.
//
// Parameters:
//   - value: The value to be cast to float64.
//
// Returns:
//   - The float64 representation of the value, or 0 if the value cannot be cast.
func CastFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case string:
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			_ = catcher.Error("failed to cast string to float64", err, map[string]any{"value": v})
			return 0
		}
		return val
	default:
		return 0
	}
}

// CastBool attempts to cast an interface{} to a bool. It supports the following types:
// - bool: returns the value directly.
// - int, int64, float64: returns true if the value is non-zero, false otherwise.
// - string: attempts to parse the string as a boolean using strconv.ParseBool.
// For any other type or if parsing fails, it returns false.
//
// Parameters:
//   - value: The value to be cast to bool.
//
// Returns:
//   - The bool representation of the value, or false if the value cannot be cast.
func CastBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		val, err := strconv.ParseBool(v)
		if err != nil {
			_ = catcher.Error("failed to cast string to bool", err, map[string]any{"value": v})
			return false
		}
		return val
	default:
		return false
	}
}

// CastString attempts to cast an interface{} to a string.
// If the value is already a string, it returns the value directly.
// Otherwise, it converts the value to a string using fmt.Sprintf.
//
// Parameters:
//   - value: The interface{} value to be cast to a string.
//
// Returns:
//   - A string representation of the input value.
func CastString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
