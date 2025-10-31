package entities

import (
	"testing"
)

func TestValidatePath(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid path",
			input:    "/path/to/file",
			expected: "/path/to/file",
		},
		{
			name:     "valid path with spaces",
			input:    "/path/to file",
			expected: "/path/to file",
		},
		{
			name:     "valid path with dots",
			input:    "/path/to/file.txt",
			expected: "/path/to/file.txt",
		},
		{
			name:     "valid path with hyphens",
			input:    "/path/to-file",
			expected: "/path/to-file",
		},
		{
			name:     "valid path with underscores",
			input:    "/path/to_file",
			expected: "/path/to_file",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid path with protocol",
			input:    "https://example.com",
			expected: "",
		},
		{
			name:     "invalid path with spaces and protocol",
			input:    "https://example.com/path/tofile",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidatePath(tc.input)
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
			_, _, err := ValidatePath(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
