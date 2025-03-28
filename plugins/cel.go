package plugins

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/tidwall/gjson"
)

// GetCelType returns a pointer to a cel.Type based on the provided string type identifier.
// Supported type identifiers include:
// - "string": returns cel.StringType
// - "int": returns cel.IntType
// - "double": returns cel.DoubleType
// - "bool": returns cel.BoolType
// - "bytes": returns cel.BytesType
// - "uint": returns cel.UintType
// - "timestamp": returns cel.TimestampType
// - "duration": returns cel.DurationType
// - "type": returns cel.TypeType
// - "null": returns cel.NullType
// - "any": returns cel.AnyType
// - "[]string": returns cel.ListType(cel.StringType)
// - "[]int": returns cel.ListType(cel.IntType)
// - "[]double": returns cel.ListType(cel.DoubleType)
// - "[]bool": returns cel.ListType(cel.BoolType)
// - "[]bytes": returns cel.ListType(cel.BytesType)
// - "[]uint": returns cel.ListType(cel.UintType)
// - "[]timestamp": returns cel.ListType(cel.TimestampType)
// - "[]duration": returns cel.ListType(cel.DurationType)
// - "[]type": returns cel.ListType(cel.TypeType)
// - "[]null": returns cel.ListType(cel.NullType)
// - "[]any": returns cel.ListType(cel.AnyType)
// - "map[string]string": returns cel.MapType(cel.StringType, cel.StringType)
// - "map[string]int": returns cel.MapType(cel.StringType, cel.IntType)
// - "map[string]double": returns cel.MapType(cel.StringType, cel.DoubleType)
// - "map[string]bool": returns cel.MapType(cel.StringType, cel.BoolType)
// - "map[string]bytes": returns cel.MapType(cel.StringType, cel.BytesType)
// - "map[string]uint": returns cel.MapType(cel.StringType, cel.UintType)
// - "map[string]timestamp": returns cel.MapType(cel.StringType, cel.TimestampType)
// - "map[string]duration": returns cel.MapType(cel.StringType, cel.DurationType)
// - "map[string]type": returns cel.MapType(cel.StringType, cel.TypeType)
// - "map[string]null": returns cel.MapType(cel.StringType, cel.NullType)
// - "map[string]any": returns cel.MapType(cel.StringType, cel.AnyType)
// If the provided type identifier does not match any of the supported types, cel.AnyType is returned.
func GetCelType(t string) *cel.Type {
	switch t {
	case "string":
		return cel.StringType
	case "int":
		return cel.IntType
	case "double":
		return cel.DoubleType
	case "bool":
		return cel.BoolType
	case "bytes":
		return cel.BytesType
	case "uint":
		return cel.UintType
	case "timestamp":
		return cel.TimestampType
	case "duration":
		return cel.DurationType
	case "type":
		return cel.TypeType
	case "null":
		return cel.NullType
	case "any":
		return cel.AnyType
	case "[]string":
		return cel.ListType(cel.StringType)
	case "[]int":
		return cel.ListType(cel.IntType)
	case "[]double":
		return cel.ListType(cel.DoubleType)
	case "[]bool":
		return cel.ListType(cel.BoolType)
	case "[]bytes":
		return cel.ListType(cel.BytesType)
	case "[]uint":
		return cel.ListType(cel.UintType)
	case "[]timestamp":
		return cel.ListType(cel.TimestampType)
	case "[]duration":
		return cel.ListType(cel.DurationType)
	case "[]type":
		return cel.ListType(cel.TypeType)
	case "[]null":
		return cel.ListType(cel.NullType)
	case "[]any":
		return cel.ListType(cel.AnyType)
	case "map[string]string":
		return cel.MapType(cel.StringType, cel.StringType)
	case "map[string]int":
		return cel.MapType(cel.StringType, cel.IntType)
	case "map[string]double":
		return cel.MapType(cel.StringType, cel.DoubleType)
	case "map[string]bool":
		return cel.MapType(cel.StringType, cel.BoolType)
	case "map[string]bytes":
		return cel.MapType(cel.StringType, cel.BytesType)
	case "map[string]uint":
		return cel.MapType(cel.StringType, cel.UintType)
	case "map[string]timestamp":
		return cel.MapType(cel.StringType, cel.TimestampType)
	case "map[string]duration":
		return cel.MapType(cel.StringType, cel.DurationType)
	case "map[string]type":
		return cel.MapType(cel.StringType, cel.TypeType)
	case "map[string]null":
		return cel.MapType(cel.StringType, cel.NullType)
	case "map[string]any":
		return cel.MapType(cel.StringType, cel.AnyType)
	default:
		return cel.AnyType
	}
}

// Evaluate evaluates a given event against the defined expression in the Where struct.
// It uses the CEL (Common Expression Language) library to compile and evaluate the expression.
//
// Parameters:
//   - event: A pointer to a string representing the event to be evaluated.
//
// Returns:
//   - bool: Returns true if the event satisfies the expression, otherwise false.
//   - error: Returns an error if there are any issues during the evaluation process.
//
// The function performs the following steps:
//  1. Initializes CEL environment options and a map to hold variable values.
//  2. Iterates over the Variables in the Where struct, setting up CEL variables and extracting values from the event.
//  3. Creates a new CEL environment with the defined variables.
//  4. Compiles the expression in the Where struct.
//  5. If there are any compilation issues, logs the error and returns false.
//  6. Creates a CEL program from the compiled AST.
//  7. If there are any errors creating the program, logs the error and returns false.
//  8. Evaluates the program with the extracted values.
//  9. If there are any evaluation errors, logs the error and returns false.
//  10. Checks if the output type is a boolean and returns its value. Otherwise, returns false.
func (def *Where) Evaluate(event *string) (bool, error) {
	vars := make([]cel.EnvOption, 0, 3)
	values := make(map[string]any)
	for _, variable := range def.Variables {
		vars = append(vars, cel.Variable(variable.As, GetCelType(variable.OfType)))
		vars = append(vars, cel.Variable(fmt.Sprintf("%s_ok", variable.As), GetCelType("bool")))

		value := gjson.Get(*event, variable.Get)
		values[variable.As] = value.Value()

		if value.Exists() {
			values[fmt.Sprintf("%s_ok", variable.As)] = true
		} else {
			values[fmt.Sprintf("%s_ok", variable.As)] = false
		}
	}

	celEnv, err := cel.NewEnv(vars...)
	if err != nil {
		return false, catcher.Error("failed to start CEL environment", err, map[string]any{"variables": vars})
	}

	ast, issues := celEnv.Compile(def.Expression)
	if issues != nil && issues.Err() != nil {
		return false, catcher.Error("failed to compile expression", err, map[string]any{"expression": def.Expression})
	}

	prg, err := celEnv.Program(ast)
	if err != nil {
		return false, catcher.Error("failed to create program", err, map[string]any{
			"expression": def.Expression,
			"variables":  vars,
		})
	}

	out, _, err := prg.Eval(values)
	if err != nil {
		return false, catcher.Error("failed to evaluate program", err, map[string]any{
			"expression": def.Expression,
			"variables":  vars,
		})
	}

	if out.Type() == cel.BoolType {
		return out.Value().(bool), nil
	}

	return false, catcher.Error("output type is not boolean", err, map[string]any{
		"expression": def.Expression,
		"variables":  vars,
	})
}
