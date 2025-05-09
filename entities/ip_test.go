package entities

import (
	"testing"
)

func TestValidateIP(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid ipv4",
			input:    "192.0.2.1",
			expected: "192.0.2.1",
		},
		{
			name:     "valid ipv6",
			input:    "2001:db8::68",
			expected: "2001:db8::68",
		},
	}

	invalidCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "invalid ip",
			input:    "invalid",
			expected: "",
		},
		{
			name:     "private ip",
			input:    "10.0.0.1",
			expected: "",
		},
		{
			name:     "interface local multicast ip",
			input:    "ff01::1",
			expected: "",
		},
		{
			name:     "link local multicast ip",
			input:    "ff02::1",
			expected: "",
		},
		{
			name:     "link local unicast ip",
			input:    "fe80::1",
			expected: "",
		},
		{
			name:     "loopback ip",
			input:    "127.0.0.1",
			expected: "",
		},
		{
			name:     "multicast ip",
			input:    "224.0.0.1",
			expected: "",
		},
		{
			name:     "unspecified ip",
			input:    "::",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateIP(tc.input)
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
			_, _, err := ValidateIP(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
