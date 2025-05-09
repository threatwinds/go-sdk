package entities

import (
	"testing"
)

func TestValidateCity(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid city",
			input:    "new york",
			expected: "New York",
		},
		{
			name:     "valid city",
			input:    "san francisco",
			expected: "San Francisco",
		},
		{
			name:     "valid city",
			input:    "los angeles",
			expected: "Los Angeles",
		},
		{
			name:     "valid city",
			input:    "london",
			expected: "London",
		},
		{
			name:     "valid city",
			input:    "paris",
			expected: "Paris",
		},
	}

	invalidCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "invalid city",
			input:    123,
			expected: "",
		},
		{
			name:     "invalid city",
			input:    true,
			expected: "",
		},
		{
			name:     "invalid city",
			input:    []string{"new york"},
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateCity(tc.input)
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
			_, _, err := ValidateCity(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
