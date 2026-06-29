package entities

import (
	"testing"
)

func TestValidateMD5(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid md5",
			input:    "d41d8cd98f00b204e9800998ecf8427e",
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "valid md5",
			input:    "098f6bcd4621d373cade4e832627b4f6",
			expected: "098f6bcd4621d373cade4e832627b4f6",
		},
		{
			name:     "valid md5",
			input:    "5eb63bbbe01eeed093cb22bb8f5acdc3",
			expected: "5eb63bbbe01eeed093cb22bb8f5acdc3",
		},
		{
			name:     "valid md5",
			input:    "d1e3a5c6f7d8e9b0a1234567890abcde",
			expected: "d1e3a5c6f7d8e9b0a1234567890abcde",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid md5",
			input:    "not a md5",
			expected: "",
		},
		{
			name:     "invalid md5",
			input:    "1234567890abcdeg",
			expected: "",
		},
		{
			name:     "invalid md5",
			input:    "d1e3a5c6f7d8e9b0g1234567890abcde",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateMD5(tc.input)
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
			_, _, err := ValidateMD5(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
