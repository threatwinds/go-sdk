package entities

import (
	"reflect"
	"strings"
	"testing"
)

// Every declared type must have an attribute field of the same name.
//
// ingest-api rejects an entity whose Attributes carry no value under a key
// matching its Type (`GetAttribute(t.Type)` in controllers/entity.go), and
// Attributes is a struct, so a type with no matching field can never satisfy
// that — the value is silently dropped at JSON unmarshal and the request 400s.
//
// "password" was in this state: a fully declared type, served in the catalogue
// and offered to analysts, that could not be reported. Nothing caught it,
// because the checks above only ask whether each definition is filled in, not
// whether it is reachable.
// Asks the struct type, not an instance. GetAttribute cannot answer this: it
// returns (nil, false) both when no such field exists and when the field exists
// but is nil, so probing a zero-valued Attributes reports every field missing.
func TestEveryTypeHasAnAttributeField(t *testing.T) {
	fields := make(map[string]bool)
	typ := reflect.TypeOf(Attributes{})
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			fields[tag] = true
		}
	}

	for _, def := range Definitions {
		if !fields[def.Type] {
			t.Errorf(
				"type %q has no matching field in Attributes, so it can be offered but never reported",
				def.Type,
			)
		}
	}
}

func TestValidateDefinitions(t *testing.T) {
	for _, def := range Definitions {
		if def.Type == "" {
			t.Error("Type is empty: ", def)
		}
		if def.Label == "" {
			t.Error("Label is empty: ", def)
		}
		if def.Description == "" {
			t.Error("Description is empty: ", def)
		}
		if def.DataType == "" {
			t.Error("DataType is empty: ", def)
		}
		if def.Example == nil {
			t.Log("Example is nil: ", def)
		}
		if len(def.Associations) == 0 {
			t.Log("Associations are empty: ", def)
		}
		if len(def.Attributes) == 0 {
			t.Log("Attributes are empty: ", def)
		}
		if len(def.Tags) == 0 {
			t.Log("Tags are empty: ", def)
		}
		if len(def.Correlate) == 0 {
			t.Log("Correlate is empty: ", def)
		}
	}
}
