package go_sdk

import (
	"strings"

	"github.com/tidwall/gjson"
)

// GetValueOf returns the Go representation of a gjson.Result value.
// It converts the gjson.Result to the appropriate Go type based on its type:
// - For gjson.String, it returns a string.
// - For gjson.Number, it returns an int if the value is an integer, or a float if it contains a decimal point or a comma.
// - For gjson.True, it returns true.
// - For gjson.False, it returns false.
// - For gjson.JSON, it returns the raw JSON string.
// - For any other type, it returns an empty string.
func GetValueOf(value gjson.Result) interface{} {
	switch value.Type {
	case gjson.String:
		return value.String()
	case gjson.Number:
		if strings.Contains(value.String(), ".") {
			return value.Float()
		} else if strings.Contains(value.String(), ",") {
			return value.Float()
		} else {
			return value.Int()
		}
	case gjson.True:
		return true
	case gjson.False:
		return false
	case gjson.JSON:
		return value.Raw
	default:
		return ""
	}
}
