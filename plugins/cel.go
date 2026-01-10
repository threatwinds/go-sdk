package plugins

import (
	"encoding/json"

	"github.com/google/cel-go/cel"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"reflect"
	"time"
)

var rCache = new(utils.RegexpCache)

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
		equalString(data),
		equalInt(data),
		equalFloat(data),
		lowerEqual(data),
		contain(data),
		containAny(data),
		containAll(data),
		oneOf(data),
		oneOfInt(data),
		oneOfDouble(data),
		startWith(data),
		startWithList(data),
		endWithList(data),
		endWith(data),
		regexMatch(data),
		lessThan(data),
		greaterThan(data),
		lessOrEqual(data),
		greaterOrEqual(data),
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
