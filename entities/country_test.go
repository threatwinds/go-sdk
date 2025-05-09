package entities

import (
	"testing"
)

func TestValidateCountry(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid country",
			input:    "united states",
			expected: "United States",
		},
		{
			name:     "valid country",
			input:    "united kingdom",
			expected: "United Kingdom",
		},
		{
			name:     "valid country",
			input:    "south africa",
			expected: "South Africa",
		},
		{
			name:     "valid country",
			input:    "new zealand",
			expected: "New Zealand",
		},
		{
			name:     "valid country",
			input:    "saudi arabia",
			expected: "Saudi Arabia",
		},
	}

	invalidCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "invalid country",
			input:    123,
			expected: "",
		},
		{
			name:     "invalid country",
			input:    true,
			expected: "",
		},
		{
			name:     "invalid country",
			input:    []string{"united states"},
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateCountry(tc.input)
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
			_, _, err := ValidateCountry(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
