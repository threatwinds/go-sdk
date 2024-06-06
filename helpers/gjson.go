package helpers

import (
	"strings"

	"github.com/tidwall/gjson"
)

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
