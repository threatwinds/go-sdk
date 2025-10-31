package entities

import "testing"

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
