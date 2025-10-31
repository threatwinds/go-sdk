package entities

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid email",
			input:    "test@example.com",
			expected: "test@example.com",
		},
		{
			name:     "valid email with uppercase",
			input:    "Test@Example.com",
			expected: "test@example.com",
		},
		{
			name:     "valid email with spaces",
			input:    " test@example.com ",
			expected: "test@example.com",
		},
		{
			name:     "valid email with plus sign",
			input:    "test+123@example.com",
			expected: "test+123@example.com",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid email",
			input: "notanemail",
		},
		{
			name:  "invalid email with missing domain",
			input: "test@",
		},
		{
			name:  "invalid email with missing username",
			input: "@example.com",
		},
		{
			name:  "invalid email with whitespace",
			input: "test @example.com",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateEmail(tc.input)
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
			_, _, err := ValidateEmail(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
