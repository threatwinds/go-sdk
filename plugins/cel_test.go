package plugins

import (
	"github.com/threatwinds/go-sdk/utils"
	"testing"
)

func TestEvaluate(t *testing.T) {
	t.Run("Equals", func(t *testing.T) {
		w := Where{
			Variables: []*Variable{
				{
					Get:    "field1",
					As:     "field1",
					OfType: "string",
				},
				{
					Get:    "field2",
					As:     "field2",
					OfType: "string",
				},
			},
			Expression: "field1 == field2",
		}

		equal, _ := w.Evaluate(utils.PointerOf(`{"field1": "value1", "field2": "value1"}`))

		if !equal {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("Not Equals", func(t *testing.T) {
		w := Where{
			Variables: []*Variable{
				{
					Get:    "field1",
					As:     "field1",
					OfType: "string",
				},
				{
					Get:    "field2",
					As:     "field2",
					OfType: "string",
				},
			},
			Expression: "field1 == field2",
		}

		equal, _ := w.Evaluate(utils.PointerOf(`{"field1": "value0", "field2": "value1"}`))

		if equal {
			t.Errorf("Expected false, got true")
		}
	})

	t.Run("StartsWith", func(t *testing.T) {
		w := Where{
			Variables: []*Variable{
				{
					Get:    "log.field",
					As:     "field",
					OfType: "string",
				},
			},
			Expression: "field.startsWith('8.1.0')",
		}

		startsWith, _ := w.Evaluate(utils.PointerOf(`{"log": {"field": "8.1.0-20241029"}}`))

		if !startsWith {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("Eval exists", func(t *testing.T) {
		w := Where{
			Variables: []*Variable{
				{
					Get:    "field2",
					As:     "field",
					OfType: "string",
				},
			},
			Expression: "field_ok==true",
		}

		exists, _ := w.Evaluate(utils.PointerOf(`{"field1": "value"}`))

		if exists {
			t.Errorf("Expected false, got true")
		}
	})
}
