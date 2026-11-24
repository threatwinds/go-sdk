package plugins

import (
	"reflect"

	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"github.com/threatwinds/go-sdk/jsondoc"
	"github.com/tidwall/gjson"
)

// lookup reads a field from the document an overload was handed: either the event's JSON text or a
// parsed jsondoc.Doc. Both answer with the same gjson result, so the overloads do not care which.
func lookup(data ref.Val, key string) gjson.Result {
	switch d := data.Value().(type) {
	case string:
		return gjson.Get(d, key)
	case *jsondoc.Doc:
		return d.Get(key)
	}

	return gjson.Result{}
}

var docType = types.NewOpaqueType("jsondoc.Doc")

// docVal lets a *jsondoc.Doc travel through CEL as the `_data_` argument of the overloads.
type docVal struct{ d *jsondoc.Doc }

func (v docVal) ConvertToNative(reflect.Type) (any, error) { return v.d, nil }

func (v docVal) ConvertToType(ref.Type) ref.Val {
	return types.NewErr("a jsondoc.Doc cannot be converted to another type")
}

func (v docVal) Equal(other ref.Val) ref.Val {
	o, ok := other.(docVal)
	return types.Bool(ok && o.d == v.d)
}

func (v docVal) Type() ref.Type { return docType }

func (v docVal) Value() any { return v.d }
