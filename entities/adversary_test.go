package entities

import (
	"testing"
)

func TestValidateAdversary(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid adversary",
			input:    "hacker",
			expected: "hacker",
		},
		{
			name:     "valid adversary",
			input:    "cybercriminal",
			expected: "cybercriminal",
		},
		{
			name:     "valid adversary",
			input:    "phisher",
			expected: "phisher",
		},
		{
			name:     "valid adversary",
			input:    "malware author",
			expected: "malware author",
		},
		{
			name:     "valid adversary",
			input:    "botnet operator",
			expected: "botnet operator",
		},
		{
			name:     "valid adversary",
			input:    "script kiddie",
			expected: "script kiddie",
		},
		{
			name:     "valid adversary",
			input:    "nation state actor",
			expected: "nation state actor",
		},
		{
			name:     "valid adversary",
			input:    "insider threat",
			expected: "insider threat",
		},
		{
			name:     "valid adversary",
			input:    "hacktivist",
			expected: "hacktivist",
		},
		{
			name:     "valid adversary",
			input:    "cyber terrorist",
			expected: "cyber terrorist",
		},
		{
			name:     "valid adversary",
			input:    "hacker",
			expected: "hacker",
		},
		{
			name:     "valid adversary",
			input:    "cybercriminal",
			expected: "cybercriminal",
		},
	}

	invalidCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid domain",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "valid email domain",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "valid email address",
			input:    "test@example.com",
			expected: "test@example.com",
		},
		{
			name:     "valid url",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "valid uuid",
			input:    "123e4567-e89b-12d3-a456-426655440000",
			expected: "123e4567-e89b-12d3-a456-426655440000",
		},
		{
			name:     "valid ip",
			input:    "192.0.2.1",
			expected: "192.0.2.1",
		},
		{
			name:     "valid phone",
			input:    "+1-202-555-0155",
			expected: "+1-202-555-0155",
		},
		{
			name:     "valid fqdn",
			input:    "example.com",
			expected: "example.com",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateAdversary(tc.input)
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
			_, _, err := ValidateAdversary(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
