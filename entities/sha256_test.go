package entities

import (
	"testing"
)

func TestValidateSHA256(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid sha-256",
			input:    "d1e3a5c3b4f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d01e",
			expected: "d1e3a5c3b4f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d01e",
		},
		{
			name:     "valid sha-256",
			input:    "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a112",
			expected: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a112",
		},
		{
			name:     "valid sha-256",
			input:    "f0e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a11234",
			expected: "f0e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a11234",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid sha-256",
			input: "d1e3a5c3b4f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0",
		},
		{
			name:  "invalid sha-256",
			input: "d1e3a5c3b4f5d6e7f8a9b0c1d2g3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0ee",
		},
		{
			name:  "invalid sha-256",
			input: "d1e3a5c3b4f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0eeff",
		},
		{
			name:  "invalid sha-256",
			input: "d1e3a5c3b4f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e!",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateSHA256(tc.input)
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
			_, _, err := ValidateSHA256(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
