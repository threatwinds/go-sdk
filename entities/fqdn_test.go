package entities

import (
	"testing"
)

func TestValidateFQDN(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid fqdn",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "valid fqdn with subdomain",
			input:    "www.example.com",
			expected: "www.example.com",
		},
		{
			name:     "valid fqdn with multiple subdomains",
			input:    "www.subdomain.example.com",
			expected: "www.subdomain.example.com",
		},
		{
			name:     "valid fqdn with hyphens",
			input:    "sub-domain.example.com",
			expected: "sub-domain.example.com",
		},
		{
			name:     "valid fqdn with multiple hyphens",
			input:    "sub-domain.sub-sub-domain.example.com",
			expected: "sub-domain.sub-sub-domain.example.com",
		},
		{
			name:     "valid fqdn with long TLD",
			input:    "example.technology",
			expected: "example.technology",
		},
		{
			name:     "valid fqdn with short TLD",
			input:    "example.co",
			expected: "example.co",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid fqdn with spaces",
			input: "example .com",
		},
		{
			name:  "invalid fqdn with special characters",
			input: "example.com!",
		},
		{
			name:  "invalid fqdn with underscore",
			input: "example_com",
		},
		{
			name:  "invalid fqdn with leading dot",
			input: ".example.com",
		},
		{
			name:  "invalid fqdn with trailing dot",
			input: "example.com.",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateFQDN(tc.input)
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
			_, _, err := ValidateFQDN(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
