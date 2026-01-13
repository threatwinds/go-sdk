package entities

import (
	"testing"
)

func TestValidateSHA512(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid SHA-512 hash",
			input:    "d1e959eb179c911faea4624c60c5c7029b309fcf44a08d1a4c6a801375befb2b1e8327e2a0c3b1f1e3e7b5c4a5a8addcbbd3b234dce144361a4fbee8f2f8fd51",
			expected: "d1e959eb179c911faea4624c60c5c7029b309fcf44a08d1a4c6a801375befb2b1e8327e2a0c3b1f1e3e7b5c4a5a8addcbbd3b234dce144361a4fbee8f2f8fd51",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid SHA-512 hash",
			input:    "invalid",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateSHA512(tc.input)
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
			_, _, err := ValidateSHA512(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
