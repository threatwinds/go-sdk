package entities

import (
	"testing"
)

func TestValidatePort(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid port with tcp",
			input:    "555/TCP",
			expected: "555/tcp",
		},
		{
			name:     "valid port with udp",
			input:    "555/UDP",
			expected: "555/udp",
		},
	}

	invalidCases := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "invalid port with spaces",
			input: "555 udp",
		},
		{
			name:  "inverted",
			input: "tcp/555",
		},
		{
			name:  "letters",
			input: "55a/tcp",
		},
		{
			name:  "invalid protocol",
			input: "555/inva",
		},
		{
			name:  "over range",
			input: "65536/tcp",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidatePort(tc.input)
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
			_, _, err := ValidatePort(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
