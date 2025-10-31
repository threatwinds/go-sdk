package entities

import (
	"testing"
)

func TestValidateMAC(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid MAC address",
			input:    "00-11-22-33-44-55",
			expected: "00-11-22-33-44-55",
		},
		{
			name:     "valid MAC address",
			input:    "AA-BB-CC-DD-EE-FF",
			expected: "AA-BB-CC-DD-EE-FF",
		},
		{
			name:     "valid MAC address",
			input:    "01-23-45-67-89-AB",
			expected: "01-23-45-67-89-AB",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid MAC address",
			input:    "00-11-22-33-44-5",
			expected: "",
		},
		{
			name:     "invalid MAC address",
			input:    "00-11-22-33-44-55-66",
			expected: "",
		},
		{
			name:     "invalid MAC address",
			input:    "00-11-22-33-44-5G",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateMAC(tc.input)
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
			_, _, err := ValidateMAC(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
