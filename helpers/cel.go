package helpers

import "github.com/google/cel-go/cel"

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
