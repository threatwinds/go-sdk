package entities

import (
	"testing"
)

func TestValidateDate(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid date",
			input:    "2022-12-31",
			expected: "2022-12-31",
		},
		{
			name:     "valid date",
			input:    "2022-01-01",
			expected: "2022-01-01",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateDate(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, actual)
			}
		})
	}
}

func TestValidateDatetime(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid datetime",
			input:    "2022-12-31T23:59:59.999999999Z",
			expected: "2022-12-31T23:59:59.999999999Z",
		},
		{
			name:     "valid datetime",
			input:    "2022-01-01T00:00:00Z",
			expected: "2022-01-01T00:00:00Z",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateDatetime(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, actual)
			}
		})
	}
}
