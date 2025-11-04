package plugins

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"reflect"
	"time"
)

// Evaluate evaluates a CEL expression against the given data and returns the boolean result if successful.
// Returns true/false or an error in case of failure during evaluation or invalid output type.
func Evaluate(data *string, expression string, envOption ...cel.EnvOption) (bool, error) {
	if data == nil {
		return false, catcher.Error("data is nil", nil, map[string]any{})
	}

	var valuesMap map[string]interface{}

	err := json.Unmarshal([]byte(*data), &valuesMap)
	if err != nil {
		return false, catcher.Error("cannot unmarshal data", err, map[string]any{})
	}

	envOptions := []cel.EnvOption{
		celExists(data),
		safeBool(data),
		safeString(data),
		safeNum(data),
		inCIDR(data),
		equal(data),
		lowerEqual(data),
		contain(data),
		inList(data),
		startWith(data),
		endWith(data),
	}

	// Add the provided environment options first (including cel.Types)
	envOptions = append(envOptions, envOption...)

	for k, v := range valuesMap {
		envOptions = append(envOptions, cel.Variable(k, valueToCelType(v)))
	}

	celEnv, err := cel.NewEnv(envOptions...)
	if err != nil {
		return false, catcher.Error("failed to start CEL environment", err, map[string]any{})
	}

	ast, issues := celEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, catcher.Error("failed to compile expression", nil, map[string]any{"expression": expression, "issues": issues.Errors()})
	}

	prg, err := celEnv.Program(ast)
	if err != nil {
		return false, catcher.Error("failed to create program", err, map[string]any{
			"expression": expression,
		})
	}

	out, _, err := prg.Eval(valuesMap)
	if err != nil {
		return false, catcher.Error("failed to evaluate program", err, map[string]any{
			"expression": expression,
		})
	}

	if out.Type() == cel.BoolType {
		return out.Value().(bool), nil
	}

	return false, catcher.Error("output type is not boolean", err, map[string]any{
		"expression": expression,
	})
}

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

func equal(s *string) cel.EnvOption {
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

func inList(s *string) cel.EnvOption {
	return cel.Function("inList", cel.Overload("string_string_inList_bool", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
		cel.BinaryBinding(func(key ref.Val, listVal ref.Val) ref.Val {
			v := gjson.Get(*s, key.Value().(string))
			if v.Exists() && v.Type == gjson.String {
				list := strings.Split(listVal.Value().(string), ",")
				for _, item := range list {
					if v.String() == strings.TrimSpace(item) {
						return types.Bool(true)
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

func valueToCelType(value interface{}) *cel.Type {
	switch value.(type) {
	case bool:
		return cel.BoolType
	case string:
		return cel.StringType
	case int, int32, int64:
		return cel.IntType
	case uint, uint32, uint64:
		return cel.UintType
	case float32, float64:
		return cel.DoubleType
	case []byte:
		return cel.BytesType
	case time.Time:
		return cel.TimestampType
	case map[string]interface{}:
		return cel.MapType(cel.StringType, cel.DynType)
	case map[string]*structpb.Value:
		return cel.MapType(cel.StringType, cel.DynType)
	case []interface{}:
		return cel.ListType(cel.DynType)
	case nil:
		return cel.NullType
	case proto.Message:
		return cel.DynType
	default:
		t := reflect.TypeOf(value)
		return cel.ObjectType(t.String())
	}
}
