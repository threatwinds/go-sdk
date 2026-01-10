package plugins

import (
	"net"
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/tidwall/gjson"
)

func celExists(s *string) cel.EnvOption {
	return cel.Function("exists",
		cel.Overload("string_exists_bool",
			[]*cel.Type{cel.StringType}, cel.BoolType,
			cel.UnaryBinding(func(key ref.Val) ref.Val {
				v := gjson.Get(*s, key.Value().(string))
				return types.Bool(v.Exists())
			}),
		),
	)
}

func safeString(s *string) cel.EnvOption {
	return cel.Function("safe", cel.Overload("string_string_safe_string", []*cel.Type{cel.StringType, cel.StringType}, cel.StringType,
		cel.BinaryBinding(func(key ref.Val, def ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				return types.String(v.String())
			}
			return def
		}),
	))
}

func inCIDR(s *string) cel.EnvOption {
	return cel.Function("inCIDR", cel.Overload("string_string_inCIDR_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, network ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				_, subnet, err := net.ParseCIDR(network.Value().(string))
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

func safeNum(s *string) cel.EnvOption {
	return cel.Function("safe", cel.Overload("string_num_safe_num", []*cel.Type{cel.StringType, cel.DoubleType}, cel.DoubleType,
		cel.BinaryBinding(func(key ref.Val, def ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.Number {
				return types.Double(v.Float())
			}
			return def
		}),
	))
}

func safeBool(s *string) cel.EnvOption {
	return cel.Function("safe", cel.Overload("string_bool_safe_bool", []*cel.Type{cel.StringType, cel.BoolType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, def ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.IsBool() {
				return types.Bool(v.Bool())
			}
			return def
		}),
	))
}

func equalString(s *string) cel.EnvOption {
	return cel.Function("equal", cel.Overload("string_string_equal_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(v.String() == val.Value())
			}
			return types.False
		}),
	))
}

func equalInt(s *string) cel.EnvOption {
	return cel.Function("equal", cel.Overload("string_int_equal_bool", []*cel.Type{cel.StringType, cel.IntType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.Number {
				if intVal, ok := val.Value().(int64); ok {
					if v.Float() != float64(v.Int()) {
						return types.False
					}
					return types.Bool(v.Int() == intVal)
				}
			}
			return types.False
		}),
	))
}

func equalFloat(s *string) cel.EnvOption {
	return cel.Function("equal", cel.Overload("string_float_equal_bool", []*cel.Type{cel.StringType, cel.DoubleType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.Number {
				if floatVal, ok := val.Value().(float64); ok {
					if v.Float() == float64(v.Int()) {
						return types.False
					}
					return types.Bool(v.Float() == floatVal)
				}
			}
			return types.False
		}),
	))
}

func lowerEqual(s *string) cel.EnvOption {
	return cel.Function("lowerEqual", cel.Overload("string_string_lowerEqual_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.EqualFold(v.String(), val.Value().(string)))
			}
			return types.False
		}),
	))
}

func contain(s *string) cel.EnvOption {
	return cel.Function("contain", cel.Overload("string_string_contain_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.Contains(v.String(), val.Value().(string)))
			}
			return types.False
		}),
	))
}

func containAny(s *string) cel.EnvOption {
	return cel.Function("contain", cel.Overload("string_list_contain_bool", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				items := val.Value().([]ref.Val)
				for _, item := range items {
					if strings.Contains(v.String(), strings.TrimSpace(item.Value().(string))) {
						return types.Bool(true)
					}
				}
			}
			return types.False
		}),
	))
}

func containAll(s *string) cel.EnvOption {
	return cel.Function("containAll", cel.Overload("string_list_containAll_bool", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				items := val.Value().([]ref.Val)
				for _, item := range items {
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

func oneOf(s *string) cel.EnvOption {
	return cel.Function("oneOf", cel.Overload("string_list_oneOf_bool", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, listVal ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				items := listVal.Value().([]ref.Val)
				for _, item := range items {
					if v.String() == strings.TrimSpace(item.Value().(string)) {
						return types.Bool(true)
					}
				}
				return types.Bool(false)
			}
			return types.False
		}),
	))
}

func oneOfInt(s *string) cel.EnvOption {
	return cel.Function("oneOf", cel.Overload("string_listint_oneOf_bool", []*cel.Type{cel.StringType, cel.ListType(cel.IntType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, listVal ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.Number {
				items := listVal.Value().([]ref.Val)
				for _, item := range items {
					if intVal, ok := item.Value().(int64); ok {
						if v.Float() == float64(v.Int()) && v.Int() == intVal {
							return types.Bool(true)
						}
					}
				}
				return types.Bool(false)
			}
			return types.False
		}),
	))
}

func oneOfDouble(s *string) cel.EnvOption {
	return cel.Function("oneOf", cel.Overload("string_listfloat_oneOf_bool", []*cel.Type{cel.StringType, cel.ListType(cel.DoubleType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, listVal ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.Number {
				items := listVal.Value().([]ref.Val)
				for _, item := range items {
					if floatVal, ok := item.Value().(float64); ok {
						if v.Float() != float64(v.Int()) && v.Float() == floatVal {
							return types.Bool(true)
						}
					}
				}
				return types.Bool(false)
			}
			return types.False
		}),
	))
}

func startWith(s *string) cel.EnvOption {
	return cel.Function("startWith", cel.Overload("string_string_startWith_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, prefix ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.HasPrefix(v.String(), prefix.Value().(string)))
			}
			return types.False
		}),
	))
}

func startWithList(s *string) cel.EnvOption {
	return cel.Function("startWith", cel.Overload("string_list_startWith_bool", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, listVal ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				items := listVal.Value().([]ref.Val)
				for _, item := range items {
					if strings.HasPrefix(v.String(), strings.TrimSpace(item.Value().(string))) {
						return types.Bool(true)
					}
				}
				return types.Bool(false)
			}
			return types.False
		}),
	))
}

func endWith(s *string) cel.EnvOption {
	return cel.Function("endWith", cel.Overload("string_string_endWith_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, suffix ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.HasSuffix(v.String(), suffix.Value().(string)))
			}
			return types.False
		}),
	))
}

func endWithList(s *string) cel.EnvOption {
	return cel.Function("endWith", cel.Overload("string_list_endWith_bool", []*cel.Type{cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, listVal ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				items := listVal.Value().([]ref.Val)
				for _, item := range items {
					if strings.HasSuffix(v.String(), strings.TrimSpace(item.Value().(string))) {
						return types.Bool(true)
					}
				}
				return types.Bool(false)
			}
			return types.False
		}),
	))
}

func regexMatch(s *string) cel.EnvOption {
	return cel.Function("regexMatch", cel.Overload("string_string_regexMatch_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, pattern ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				re, err := rCache.Get(pattern.Value().(string))
				if err != nil {
					return types.False
				}
				return types.Bool(re.MatchString(v.String()))
			}
			return types.False
		}),
	))
}

func lessThan(s *string) cel.EnvOption {
	return cel.Function("lessThan", cel.Overload("string_string_lessThan_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}

			f2, err := strconv.ParseFloat(val.Value().(string), 64)
			if err != nil {
				return types.False
			}

			return types.Bool(v.Float() < f2)
		}),
	))
}

func greaterThan(s *string) cel.EnvOption {
	return cel.Function("greaterThan", cel.Overload("string_string_greaterThan_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}

			f2, err := strconv.ParseFloat(val.Value().(string), 64)
			if err != nil {
				return types.False
			}

			return types.Bool(v.Float() > f2)
		}),
	))
}

func lessOrEqual(s *string) cel.EnvOption {
	return cel.Function("lessOrEqual", cel.Overload("string_string_lessOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}

			f2, err := strconv.ParseFloat(val.Value().(string), 64)
			if err != nil {
				return types.False
			}

			return types.Bool(v.Float() <= f2)
		}),
	))
}

func greaterOrEqual(s *string) cel.EnvOption {
	return cel.Function("greaterOrEqual", cel.Overload("string_string_greaterOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, val ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}

			f2, err := strconv.ParseFloat(val.Value().(string), 64)
			if err != nil {
				return types.False
			}

			return types.Bool(v.Float() >= f2)
		}),
	))
}
