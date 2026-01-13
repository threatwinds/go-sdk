package entities

import (
	"testing"
)

func TestValidateSHA384(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid sha-384",
			input:    "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3",
			expected: "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3",
		},
		{
			name:     "valid sha-384",
			input:    "b7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3",
			expected: "b7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid sha-384",
			input:    "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a",
			expected: "",
		},
		{
			name:     "invalid sha-384",
			input:    "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3z",
			expected: "",
		},
		{
			name:     "invalid sha-384",
			input:    "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3g",
			expected: "",
		},
		{
			name:     "invalid sha-384",
			input:    "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6",
			expected: "",
		},
		{
			name:     "invalid sha-384",
			input:    "a7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a3d7c6a",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateSHA384(tc.input)
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
			_, _, err := ValidateSHA384(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
