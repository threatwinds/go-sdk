package helpers

import (
	"fmt"
	"strconv"
)

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
			return 0
		}
		return val
	default:
		return 0
	}
}

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
			return 0
		}
		return val
	default:
		return 0
	}
}

func CastBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		val, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return val
	default:
		return false
	}
}

func CastString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}