package entities

import (
	"testing"
)

func TestValidateHexadecimal(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid hexadecimal",
			input:    "48656c6c6f20576f726c64",
			expected: "48656c6c6f20576f726c64",
		},
		{
			name:     "valid hexadecimal with uppercase letters",
			input:    "48656C6C6F20576F726C64",
			expected: "48656c6c6f20576f726c64",
		},
		{
			name:     "valid hexadecimal with odd number of characters",
			input:    "48656c6c6f20576f726c",
			expected: "48656c6c6f20576f726c",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid hexadecimal",
			input: "hello",
		},
		{
			name:  "invalid hexadecimal with special characters",
			input: "48@56c6c6f20576f726c",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateHexadecimal(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, actual)
			}
		})
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ValidateHexadecimal(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
