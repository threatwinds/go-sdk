package entities

import (
	"testing"
)

func TestValidateCIDR(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid CIDR",
			input:    "1.1.0.0/16",
			expected: "1.1.0.0/16",
		},
		{
			name:     "valid CIDR",
			input:    "8.8.8.0/24",
			expected: "8.8.8.0/24",
		},
		{
			name:     "valid CIDR",
			input:    "34.0.0.0/8",
			expected: "34.0.0.0/8",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid CIDR",
			input: "192.168.0.0/16",
		},
		{
			name:  "invalid CIDR",
			input: "10.0.0.0/0",
		},
		{
			name:  "invalid CIDR",
			input: "10.0.0.0/24",
		},
		{
			name:  "invalid CIDR",
			input: "not a CIDR",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateCIDR(tc.input)
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
			_, _, err := ValidateCIDR(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
