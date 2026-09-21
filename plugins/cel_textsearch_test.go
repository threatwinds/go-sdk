package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCELTextSearchOnNonScalars pins the behavior that contains/containsAll/
// startsWith/endsWith/regexMatch search the raw JSON text of objects and
// arrays (and their #() query results) instead of returning false.
//
// Before the fix these returned false for any value whose gjson type is not
// String, which silently killed detection rules (e.g. O365
// contains("log.Parameters", "...")) and forced lossy cast-to-string filter
// steps. Scalar behavior must be unchanged.
func TestCELTextSearchOnNonScalars(t *testing.T) {
	data := `{
		"log": {
			"Parameters": [{"Name": "ForwardTo", "Value": "attacker@evil.com"}, {"Name": "Enabled", "Value": "True"}],
			"Members": ["user@corp.com"],
			"subject": "urgent invoice",
			"count": 5,
			"nested": {"deep": {"cidr": "0.0.0.0/0"}}
		}
	}`
	tests := []struct {
		name string
		expr string
		want bool
	}{
		// --- the bug: text-search over a whole array/object ---
		{"contains_array", `contains("log.Parameters", "ForwardTo")`, true},
		{"contains_array_false", `contains("log.Parameters", "Nope")`, false},
		{"contains_object", `contains("log.nested", "0.0.0.0/0")`, true},
		{"containsArray_list_overload", `contains("log.Parameters", ["ForwardTo", "Enabled"])`, true},
		{"containsArray_list_overload_false", `contains("log.Parameters", ["Nope"])`, false},
		{"containsAll_array", `containsAll("log.Parameters", ["ForwardTo", "Enabled"])`, true},
		{"containsAll_array_false", `containsAll("log.Parameters", ["ForwardTo", "Nope"])`, false},
		{"startsWith_array", `startsWith("log.Parameters", "[")`, true},
		{"endsWith_object", `endsWith("log.nested", "}")`, true},
		{"regexMatch_array", `regexMatch("log.Parameters", "attacker@.*")`, true},
		{"regexMatch_object", `regexMatch("log.nested", "cidr")`, true},
		{"containsQuery_first_match", `contains("log.Parameters.#(Name==ForwardTo).Value", "evil.com")`, true},
		// --- scalar behavior must be unchanged ---
		{"contains_scalar", `contains("log.subject", "invoice")`, true},
		{"contains_scalar_false", `contains("log.subject", "nope")`, false},
		{"startsWith_scalar", `startsWith("log.subject", "urgent")`, true},
		{"endsWith_scalar", `endsWith("log.subject", "invoice")`, true},
		{"regexMatch_scalar", `regexMatch("log.subject", "urg[ae]nt")`, true},
		// --- null/missing must stay false, not error ---
		{"contains_missing", `contains("log.missing", "x")`, false},
		{"regexMatch_missing", `regexMatch("log.missing", "x")`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strData := data
			got, err := cCache.Evaluate(&strData, tt.expr)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
