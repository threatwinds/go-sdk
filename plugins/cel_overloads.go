package plugins

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/tidwall/gjson"
)

// Helper to check if two numeric values are effectively equal
func numericEqual(a, b float64) bool {
	return a == b
}

// Helper to parse a numeric value from gjson Result
func getNumeric(v gjson.Result) (float64, bool) {
	if v.Type == gjson.Number {
		return v.Float(), true
	}
	if v.Type == gjson.String {
		f, err := strconv.ParseFloat(v.String(), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// Helper to parse a numeric value from ref.Val
func valToFloat(v ref.Val) (float64, bool) {
	switch val := v.Value().(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func (c *CELCache) celExists() cel.EnvOption {
	return cel.Function("exists",
		cel.Overload("string_exists_bool",
			[]*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
			cel.BinaryBinding(func(data ref.Val, key ref.Val) ref.Val {
				v := gjson.Get(data.Value().(string), key.Value().(string))
				return types.Bool(v.Exists())
			}),
		),
	)
}

func (c *CELCache) safe() cel.EnvOption {
	return cel.Function("safe",
		cel.Overload("string_string_safe_string", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.StringType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for safe(string, string, string)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				def := args[2].Value().(string)
				v := gjson.Get(data, key)
				if v.Exists() && v.Type == gjson.String {
					return types.String(v.String())
				}
				return types.String(def)
			}),
		),
		cel.Overload("string_num_safe_num", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.DoubleType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for safe(string, string, double)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				def := args[2].Value().(float64)
				v := gjson.Get(data, key)
				if v.Exists() {
					if f, ok := getNumeric(v); ok {
						return types.Double(f)
					}
				}
				return types.Double(def)
			}),
		),
		cel.Overload("string_bool_safe_bool", []*cel.Type{cel.StringType, cel.StringType, cel.BoolType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for safe(string, string, bool)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				def := args[2].Value().(bool)
				v := gjson.Get(data, key)
				if v.Exists() && v.IsBool() {
					return types.Bool(v.Bool())
				}
				return types.Bool(def)
			}),
		),
	)
}

func (c *CELCache) inCIDR() cel.EnvOption {
	return cel.Function("inCIDR", cel.Overload("string_string_inCIDR_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for inCIDR(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			network := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				_, subnet, err := net.ParseCIDR(network)
				if err != nil {
					return types.False
				}
				ip := net.ParseIP(v.String())
				if ip == nil {
					return types.False
				}
				return types.Bool(subnet.Contains(ip))
			}
			return types.False
		}),
	))
}

func (c *CELCache) equals() cel.EnvOption {
	return cel.Function("equals",
		cel.Overload("string_string_equals_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for equals(string, string, string)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				val := args[2].Value().(string)
				v := gjson.Get(data, key)
				if !v.Exists() {
					return types.False
				}

				// Flexible match for string literal "1" against number 1 or string "1"
				if v.Type == gjson.String && v.String() == val {
					return types.True
				}
				if f1, ok1 := getNumeric(v); ok1 {
					if f2, err := strconv.ParseFloat(val, 64); err == nil {
						return types.Bool(numericEqual(f1, f2))
					}
				}

				return types.Bool(v.String() == val)
			}),
		),
		cel.Overload("string_int_equals_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for equals(string, string, int)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				val := args[2].Value().(int64)
				v := gjson.Get(data, key)
				if !v.Exists() {
					return types.False
				}

				if f, ok := getNumeric(v); ok {
					return types.Bool(numericEqual(f, float64(val)))
				}
				return types.False
			}),
		),
		cel.Overload("string_float_equals_bool", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for equals(string, string, double)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				val := args[2].Value().(float64)
				v := gjson.Get(data, key)
				if !v.Exists() {
					return types.False
				}

				if f, ok := getNumeric(v); ok {
					return types.Bool(numericEqual(f, val))
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) equalsIgnoreCase() cel.EnvOption {
	return cel.Function("equalsIgnoreCase", cel.Overload("string_string_equalsIgnoreCase_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for equalsIgnoreCase(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() && !v.IsObject() && !v.IsArray() {
				return types.Bool(strings.EqualFold(v.String(), val))
			}
			return types.False
		}),
	))
}

func (c *CELCache) contains() cel.EnvOption {
	return cel.Function("contains",
		cel.Overload("string_string_contains_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for contains(string, string, string)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				val := args[2].Value().(string)
				v := gjson.Get(data, key)
				if v.Exists() && !v.IsObject() && !v.IsArray() {
					return types.Bool(strings.Contains(v.String(), val))
				}
				return types.False
			}),
		),
		cel.Overload("string_list_contains_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for contains(string, string, list)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				listVal := args[2].Value().([]ref.Val)
				v := gjson.Get(data, key)
				if v.Exists() && !v.IsObject() && !v.IsArray() {
					for _, item := range listVal {
						if strings.Contains(v.String(), strings.TrimSpace(item.Value().(string))) {
							return types.Bool(true)
						}
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) containsAll() cel.EnvOption {
	return cel.Function("containsAll", cel.Overload("string_list_containsAll_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for containsAll(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() {
				for _, item := range listVal {
					if !strings.Contains(v.String(), strings.TrimSpace(item.Value().(string))) {
						return types.Bool(false)
					}
				}
				return types.Bool(true)
			}
			return types.False
		}),
	))
}

func (c *CELCache) oneOf() cel.EnvOption {
	return cel.Function("oneOf", cel.Overload("string_dyn_list_oneOf_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.DynType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for oneOf(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if !v.Exists() {
				return types.False
			}

			f1, ok1 := getNumeric(v)
			s1 := v.String()

			for _, item := range listVal {
				// Try numeric comparison if field can be numeric
				if ok1 {
					if f2, ok2 := valToFloat(item); ok2 {
						if numericEqual(f1, f2) {
							return types.Bool(true)
						}
					}
				}

				// Fallback: compare formatted string representation
				s2 := fmt.Sprintf("%v", item.Value())
				if s1 == strings.TrimSpace(s2) {
					return types.Bool(true)
				}
			}
			return types.False
		}),
	))
}

func (c *CELCache) startsWith() cel.EnvOption {
	return cel.Function("startsWith",
		cel.Overload("string_string_startsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for startsWith(string, string, string)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				prefix := args[2].Value().(string)
				v := gjson.Get(data, key)
				if v.Exists() && !v.IsObject() && !v.IsArray() {
					return types.Bool(strings.HasPrefix(v.String(), prefix))
				}
				return types.False
			}),
		),
		cel.Overload("string_list_startsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for startsWith(string, string, list)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				listVal := args[2].Value().([]ref.Val)
				v := gjson.Get(data, key)
				if v.Exists() && !v.IsObject() && !v.IsArray() {
					s := v.String()
					for _, item := range listVal {
						if strings.HasPrefix(s, strings.TrimSpace(item.Value().(string))) {
							return types.Bool(true)
						}
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) endsWith() cel.EnvOption {
	return cel.Function("endsWith",
		cel.Overload("string_string_endsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for endsWith(string, string, string)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				suffix := args[2].Value().(string)
				v := gjson.Get(data, key)
				if v.Exists() && !v.IsObject() && !v.IsArray() {
					return types.Bool(strings.HasSuffix(v.String(), suffix))
				}
				return types.False
			}),
		),
		cel.Overload("string_list_endsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if len(args) != 3 {
					return types.NewErr("invalid number of arguments for endsWith(string, string, list)")
				}
				data := args[0].Value().(string)
				key := args[1].Value().(string)
				listVal := args[2].Value().([]ref.Val)
				v := gjson.Get(data, key)
				if v.Exists() && !v.IsObject() && !v.IsArray() {
					s := v.String()
					for _, item := range listVal {
						if strings.HasSuffix(s, strings.TrimSpace(item.Value().(string))) {
							return types.Bool(true)
						}
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) regexMatch() cel.EnvOption {
	return cel.Function("regexMatch", cel.Overload("string_string_regexMatch_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for regexMatch(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			pattern := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() {
				re, err := rCache.Get(pattern)
				if err != nil {
					return types.False
				}
				return types.Bool(re.MatchString(v.String()))
			}
			return types.False
		}),
	))
}

func (c *CELCache) lessThan() cel.EnvOption {
	return cel.Function("lessThan",
		cel.Overload("string_string_lessThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 < f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_int_lessThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 < f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_double_lessThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 < f2)
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) greaterThan() cel.EnvOption {
	return cel.Function("greaterThan",
		cel.Overload("string_string_greaterThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 > f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_int_greaterThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 > f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_double_greaterThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 > f2)
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) lessOrEqual() cel.EnvOption {
	return cel.Function("lessOrEqual",
		cel.Overload("string_string_lessOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 <= f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_int_lessOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 <= f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_double_lessOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 <= f2)
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) greaterOrEqual() cel.EnvOption {
	return cel.Function("greaterOrEqual",
		cel.Overload("string_string_greaterOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 >= f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_int_greaterOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 >= f2)
					}
				}
				return types.False
			}),
		),
		cel.Overload("string_double_greaterOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.BoolType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				if f1, ok1 := getNumeric(gjson.Get(args[0].Value().(string), args[1].Value().(string))); ok1 {
					if f2, ok2 := valToFloat(args[2]); ok2 {
						return types.Bool(f1 >= f2)
					}
				}
				return types.False
			}),
		),
	)
}

func (c *CELCache) isHour() cel.EnvOption {
	return cel.Function("isHour", cel.Overload("string_string_int_isHour_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			target := args[2].Value().(int64)
			v := gjson.Get(data, key)
			if v.Exists() {
				t, err := time.Parse(time.RFC3339, v.String())
				if err != nil {
					return types.False
				}
				return types.Bool(int64(t.Hour()) == target)
			}
			return types.False
		}),
	))
}

func (c *CELCache) isMinute() cel.EnvOption {
	return cel.Function("isMinute", cel.Overload("string_string_int_isMinute_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			target := args[2].Value().(int64)
			v := gjson.Get(data, key)
			if v.Exists() {
				t, err := time.Parse(time.RFC3339, v.String())
				if err != nil {
					return types.False
				}
				return types.Bool(int64(t.Minute()) == target)
			}
			return types.False
		}),
	))
}

func (c *CELCache) isDayOfWeek() cel.EnvOption {
	return cel.Function("isDayOfWeek", cel.Overload("string_string_int_isDayOfWeek_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			target := args[2].Value().(int64)
			v := gjson.Get(data, key)
			if v.Exists() {
				t, err := time.Parse(time.RFC3339, v.String())
				if err != nil {
					return types.False
				}
				return types.Bool(int64(t.Weekday()) == target)
			}
			return types.False
		}),
	))
}

func (c *CELCache) isWeekend() cel.EnvOption {
	return cel.Function("isWeekend", cel.Overload("string_string_isWeekend_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() {
				t, err := time.Parse(time.RFC3339, v.String())
				if err != nil {
					return types.False
				}
				w := t.Weekday()
				return types.Bool(w == time.Saturday || w == time.Sunday)
			}
			return types.False
		}),
	))
}

func (c *CELCache) isWorkDay() cel.EnvOption {
	return cel.Function("isWorkDay", cel.Overload("string_string_isWorkDay_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() {
				t, err := time.Parse(time.RFC3339, v.String())
				if err != nil {
					return types.False
				}
				w := t.Weekday()
				return types.Bool(w >= time.Monday && w <= time.Friday)
			}
			return types.False
		}),
	))
}

func (c *CELCache) isBetweenTime() cel.EnvOption {
	return cel.Function("isBetweenTime", cel.Overload("string_string_string_string_isBetweenTime_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			startStr := args[2].Value().(string)
			endStr := args[3].Value().(string)

			v := gjson.Get(data, key)
			if !v.Exists() {
				return types.False
			}

			t, err := time.Parse(time.RFC3339, v.String())
			if err != nil {
				return types.False
			}

			startT, err := time.Parse("15:04", startStr)
			if err != nil {
				return types.False
			}

			endT, err := time.Parse("15:04", endStr)
			if err != nil {
				return types.False
			}

			// Current time in comparable normalized format
			current := t.Hour()*60 + t.Minute()
			start := startT.Hour()*60 + startT.Minute()
			end := endT.Hour()*60 + endT.Minute()

			if start <= end {
				return types.Bool(current >= start && current <= end)
			}
			// Overnight range (e.g., 22:00 to 06:00)
			return types.Bool(current >= start || current <= end)
		}),
	))
}
