package entities

import (
	"testing"
)

func TestValidateInteger(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{
			name:     "valid integer",
			input:    42,
			expected: 42,
		},
		{
			name:     "valid float",
			input:    314,
			expected: 314,
		},
		{
			name:     "valid negative integer",
			input:    -10,
			expected: -10,
		},
	}

	invalidCases := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "invalid string",
			input: "not an integer",
		},
		{
			name:  "invalid boolean",
			input: true,
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateInteger(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %d, but got %d", tc.expected, actual)
			}
		})
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ValidateInteger(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
