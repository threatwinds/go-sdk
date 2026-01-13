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

func (c *CELCache) safeString() cel.EnvOption {
	return cel.Function("safe", cel.Overload("string_string_safe_string", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.StringType,
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
	))
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

func (c *CELCache) safeNum() cel.EnvOption {
	return cel.Function("safe", cel.Overload("string_num_safe_num", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.DoubleType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for safe(string, string, double)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			def := args[2].Value().(float64)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.Number {
				return types.Double(v.Float())
			}
			return types.Double(def)
		}),
	))
}

func (c *CELCache) safeBool() cel.EnvOption {
	return cel.Function("safe", cel.Overload("string_bool_safe_bool", []*cel.Type{cel.StringType, cel.StringType, cel.BoolType}, cel.BoolType,
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
	))
}

func (c *CELCache) equalStrings() cel.EnvOption {
	return cel.Function("equals", cel.Overload("string_string_equals_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for equals(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(v.String() == val)
			}
			return types.False
		}),
	))
}

func (c *CELCache) equalIntegers() cel.EnvOption {
	return cel.Function("equals", cel.Overload("string_int_equals_bool", []*cel.Type{cel.StringType, cel.StringType, cel.IntType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for equals(string, string, int)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(int64)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.Number {
				if v.Float() != float64(v.Int()) {
					return types.False
				}
				return types.Bool(v.Int() == val)
			}
			return types.False
		}),
	))
}

func (c *CELCache) equalFloats() cel.EnvOption {
	return cel.Function("equals", cel.Overload("string_float_equals_bool", []*cel.Type{cel.StringType, cel.StringType, cel.DoubleType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for equals(string, string, double)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(float64)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.Number {
				if v.Float() == float64(v.Int()) {
					return types.False
				}
				return types.Bool(v.Float() == val)
			}
			return types.False
		}),
	))
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
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.EqualFold(v.String(), val))
			}
			return types.False
		}),
	))
}

func (c *CELCache) contains() cel.EnvOption {
	return cel.Function("contains", cel.Overload("string_string_contains_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for contains(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.Contains(v.String(), val))
			}
			return types.False
		}),
	))
}

func (c *CELCache) containsAny() cel.EnvOption {
	return cel.Function("contains", cel.Overload("string_list_contains_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for contains(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				for _, item := range listVal {
					if strings.Contains(v.String(), strings.TrimSpace(item.Value().(string))) {
						return types.Bool(true)
					}
				}
			}
			return types.False
		}),
	))
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
			if v.Exists() && v.Type == gjson.String {
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
	return cel.Function("oneOf", cel.Overload("string_list_oneOf_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for oneOf(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				for _, item := range listVal {
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

func (c *CELCache) oneOfInt() cel.EnvOption {
	return cel.Function("oneOf", cel.Overload("string_listint_oneOf_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.IntType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for oneOf(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.Number {
				for _, item := range listVal {
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

func (c *CELCache) oneOfDouble() cel.EnvOption {
	return cel.Function("oneOf", cel.Overload("string_listfloat_oneOf_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.DoubleType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for oneOf(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.Number {
				for _, item := range listVal {
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

func (c *CELCache) startsWith() cel.EnvOption {
	return cel.Function("startsWith", cel.Overload("string_string_startsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for startsWith(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			prefix := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.HasPrefix(v.String(), prefix))
			}
			return types.False
		}),
	))
}

func (c *CELCache) startsWithList() cel.EnvOption {
	return cel.Function("startsWith", cel.Overload("string_list_startsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for startsWith(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				for _, item := range listVal {
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

func (c *CELCache) endsWith() cel.EnvOption {
	return cel.Function("endsWith", cel.Overload("string_string_endsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for endsWith(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			suffix := args[2].Value().(string)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				return types.Bool(strings.HasSuffix(v.String(), suffix))
			}
			return types.False
		}),
	))
}

func (c *CELCache) endsWithList() cel.EnvOption {
	return cel.Function("endsWith", cel.Overload("string_list_endsWith_bool", []*cel.Type{cel.StringType, cel.StringType, cel.ListType(cel.StringType)}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for endsWith(string, string, list)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			listVal := args[2].Value().([]ref.Val)
			v := gjson.Get(data, key)
			if v.Exists() && v.Type == gjson.String {
				for _, item := range listVal {
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
			if v.Exists() && v.Type == gjson.String {
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
	return cel.Function("lessThan", cel.Overload("string_string_lessThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for lessThan(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}
			f2, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return types.False
			}
			return types.Bool(v.Float() < f2)
		}),
	))
}

func (c *CELCache) greaterThan() cel.EnvOption {
	return cel.Function("greaterThan", cel.Overload("string_string_greaterThan_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for greaterThan(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}
			f2, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return types.False
			}
			return types.Bool(v.Float() > f2)
		}),
	))
}

func (c *CELCache) lessOrEqual() cel.EnvOption {
	return cel.Function("lessOrEqual", cel.Overload("string_string_lessOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for lessOrEqual(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}
			f2, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return types.False
			}
			return types.Bool(v.Float() <= f2)
		}),
	))
}

func (c *CELCache) greaterOrEqual() cel.EnvOption {
	return cel.Function("greaterOrEqual", cel.Overload("string_string_greaterOrEqual_bool", []*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.BoolType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 3 {
				return types.NewErr("invalid number of arguments for greaterOrEqual(string, string, string)")
			}
			data := args[0].Value().(string)
			key := args[1].Value().(string)
			val := args[2].Value().(string)
			v := gjson.Get(data, key)
			if !v.Exists() || v.Type != gjson.Number {
				return types.False
			}
			f2, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return types.False
			}
			return types.Bool(v.Float() >= f2)
		}),
	))
}
