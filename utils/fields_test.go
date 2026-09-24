package utils

import "testing"

// SanitizeField must preserve underscored field names (standard JSON keys such as
// audit_verb, ip_permission, correlation_candidate_*). Before the fix it stripped
// every character outside [a-zA-Z0-9.], silently renaming fields so that rules
// referencing the original name could never match.
func TestSanitizeField_PreservesUnderscores(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"underscore kept", "log.correlationCandidate.aws_ecs_credential_theft", "log.correlationCandidate.aws_ecs_credential_theft"},
		{"nested underscore", "log.ip_permission", "log.ip_permission"},
		{"dotted underscore", "log.requestParameters.ipRanges", "log.requestParameters.ipRanges"},
		{"plain unchanged", "origin.ip", "origin.ip"},
		{"camelCase unchanged", "log.sourceIPAddress", "log.sourceIPAddress"},
		{"reserved dot kept", "log.tlsDetails.tlsVersion", "log.tlsDetails.tlsVersion"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			SanitizeField(&got)
			if got != tt.want {
				t.Errorf("SanitizeField(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Characters that are genuinely unsafe in a gjson/proto path must still be
// removed, so the relaxation is not a free-for-all.
func TestSanitizeField_StillStripsUnsafeChars(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"space stripped", "log.ip permission", "log.ippermission"},
		{"hyphen stripped", "log.tls-version", "log.tlsversion"},
		{"leading hyphen stripped", "log.x-amz-acl", "log.xamzacl"},
		{"slash stripped", "log/a/b", "logab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			SanitizeField(&got)
			if got != tt.want {
				t.Errorf("SanitizeField(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
