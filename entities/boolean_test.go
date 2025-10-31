package entities

import (
	"testing"
)

func TestValidateBoolean(t *testing.T) {
	validCases := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "valid boolean true",
			input:    true,
			expected: true,
		},
		{
			name:     "valid boolean false",
			input:    false,
			expected: false,
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateBoolean(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %v, but got %v", tc.expected, actual)
			}
		})
	}
}
