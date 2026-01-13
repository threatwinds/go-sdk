package entities

import (
	"testing"
)

func TestValidateFloat(t *testing.T) {
	validCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "valid float",
			input:    3.14,
			expected: 3.14,
		},
		{
			name:     "valid float",
			input:    42.0,
			expected: 42.0,
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateFloat(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %v, but got %v", tc.expected, actual)
			}
		})
	}
}
