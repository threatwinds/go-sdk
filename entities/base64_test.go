package entities

import (
	"testing"
)

func TestValidateBase64(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid base64",
			input:    "dGVzdA==",
			expected: "dGVzdA==",
		},
		{
			name:     "valid base64",
			input:    "aGVsbG8gd29ybGQ=",
			expected: "aGVsbG8gd29ybGQ=",
		},
		{
			name:     "valid base64",
			input:    "YXNkZg==",
			expected: "YXNkZg==",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid base64",
			input:    "not base64",
			expected: "",
		},
		{
			name:     "invalid base64",
			input:    "dGVzdA",
			expected: "",
		},
		{
			name:     "invalid base64",
			input:    "dGVzdA===",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateBase64(tc.input)
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
			_, _, err := ValidateBase64(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
