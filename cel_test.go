package go_sdk

import "testing"

func TestEvaluate(t *testing.T) {
	t.Run("Equals", func(t *testing.T) {
		w := Where{
			Variables: []*Variable{
				{
					Get: "field1",
					As: "field1",
					OfType: "string",
				},
				{
					Get: "field2",
					As: "field2",
					OfType: "string",
				},
			},
			Expression: "field1 == field2",
		}
	
		equal := w.Evaluate(PointerOf(`{"field1": "value1", "field2": "value1"}`))
	
		if !equal {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("Not Equals", func(t *testing.T) {
		w := Where{
			Variables: []*Variable{
				{
					Get: "field1",
					As: "field1",
					OfType: "string",
				},
				{
					Get: "field2",
					As: "field2",
					OfType: "string",
				},
			},
			Expression: "field1 == field2",
		}
	
		equal := w.Evaluate(PointerOf(`{"field1": "value0", "field2": "value1"}`))
	
		if equal {
			t.Errorf("Expected true, got false")
		}
	})
}