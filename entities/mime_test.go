package entities

import (
	"testing"
)

func TestValidateMime(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid mime type",
			input:    "application/json",
			expected: "application/json",
		},
		{
			name:     "valid mime type",
			input:    "text/html",
			expected: "text/html",
		},
		{
			name:     "valid mime type",
			input:    "image/png",
			expected: "image/png",
		},
		{
			name:     "valid mime type",
			input:    "audio/mpeg",
			expected: "audio/mpeg",
		},
		{
			name:     "valid mime type",
			input:    "audio/3mu",
			expected: "audio/3mu",
		},
		{
			name:     "valid mime type",
			input:    "video/mp4",
			expected: "video/mp4",
		},
	}

	invalidCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid mime type",
			input:    "application",
			expected: "",
		},
		{
			name:     "invalid mime type",
			input:    "application/json-",
			expected: "",
		},
		{
			name:     "invalid mime type",
			input:    "te-xt/text",
			expected: "",
		},
		{
			name:     "invalid mime type",
			input:    "image/png--png",
			expected: "",
		},
		{
			name:     "invalid mime type",
			input:    "audio/-mp3",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateMime(tc.input)
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
			_, _, err := ValidateMime(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
