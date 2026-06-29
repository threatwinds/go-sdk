package entities

import (
	"testing"
)

func TestValidateSHA1(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid SHA1",
			input:    "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
			expected: "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		},
		{
			name:     "valid SHA1",
			input:    "da39a3ee5e6b4b0d3255bfef95601890afd80709",
			expected: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:     "valid SHA1",
			input:    "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
			expected: "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid SHA1",
			input:    "2fd4e1c67a2d28fced849ee1bb76e7391b93eb1",
			expected: "",
		},
		{
			name:     "invalid SHA1",
			input:    "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12a",
			expected: "",
		},
		{
			name:     "invalid SHA1",
			input:    "2fd4e1c67a2d28fced849ee1bb76e7391b93eb1g",
			expected: "",
		},
		{
			name:     "invalid SHA1",
			input:    "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 ",
			expected: "",
		},
		{
			name:     "invalid SHA1",
			input:    " 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateSHA1(tc.input)
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
			_, _, err := ValidateSHA1(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
