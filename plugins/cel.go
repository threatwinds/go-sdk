package plugins

import (
	"github.com/google/cel-go/cel"
	"github.com/threatwinds/go-sdk/catcher"

	"reflect"
	"strings"
)

// Evaluate evaluates a CEL expression against the given data and returns the boolean result if successful.
// Returns true/false or an error in case of failure during evaluation or invalid output type.
func Evaluate[Data any](data *Data, expression string) (bool, error) {
	if data == nil {
		return false, catcher.Error("data is nil", nil, map[string]any{})
	}

	value := reflect.ValueOf(data).Elem()

	if value.IsZero() {
		return false, nil
	}

	typ := value.Type()

	values := make(map[string]interface{})

	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		tag := field.Tag.Get("json")
		tagParts := strings.Split(tag, ",")
		jsonName := tagParts[0]

		if fieldValue.IsZero() {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Interface:
			values[jsonName] = fieldValue.Elem().Interface()
		case reflect.Pointer:
			values[jsonName] = fieldValue.Elem().Interface()
		default:
			values[jsonName] = fieldValue.Interface()
		}
	}

	vars := make([]cel.EnvOption, 0, 3)

	for k := range values {
		vars = append(vars, cel.Variable(k, cel.DynType))
	}

	celEnv, err := cel.NewEnv(vars...)
	if err != nil {
		return false, catcher.Error("failed to start CEL environment", err, map[string]any{"variables": vars})
	}

	ast, issues := celEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, catcher.Error("failed to compile expression", nil, map[string]any{"expression": expression, "issues": issues.Errors()})
	}

	prg, err := celEnv.Program(ast)
	if err != nil {
		return false, catcher.Error("failed to create program", err, map[string]any{
			"expression": expression,
			"variables":  vars,
		})
	}

	out, _, err := prg.Eval(values)
	if err != nil {
		return false, catcher.Error("failed to evaluate program", err, map[string]any{
			"expression": expression,
			"variables":  vars,
		})
	}

	if out.Type() == cel.BoolType {
		return out.Value().(bool), nil
	}

	return false, catcher.Error("output type is not boolean", err, map[string]any{
		"expression": expression,
		"variables":  vars,
	})
}
