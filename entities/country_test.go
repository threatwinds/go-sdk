package entities

import (
	"testing"
)

func TestValidateCountry(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
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
}
