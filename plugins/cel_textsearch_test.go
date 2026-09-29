package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCELTextSearchOnNonScalars pins the split behavior of the text-search
// CEL functions.
//
// contains/containsAll/startsWith/endsWith search the raw JSON text of objects
// and arrays (and their #() query results) instead of returning false. This is
// the v1.1.34 fix that un-killed detection rules like O365
// contains("log.Parameters", "...").
//
// regexMatch is deliberately different: it is a SCALAR-ONLY string-type guard.
// Filters use regexMatch("log.X", ".+") to mean "X is a non-empty string", so
// an object/array value must evaluate to false (not be stringified and
// matched). This keeps malformed vendor fields (e.g. userIdentity.arn = {})
// from being coerced into downstream string values.
func TestCELTextSearchOnNonScalars(t *testing.T) {
	data := `{
		"log": {
			"Parameters": [{"Name": "ForwardTo", "Value": "attacker@evil.com"}, {"Name": "Enabled", "Value": "True"}],
			"Members": ["user@corp.com"],
			"subject": "urgent invoice",
			"count": 5,
			"nested": {"deep": {"cidr": "0.0.0.0/0"}},
			"emptyObj": {},
			"emptyArr": []
		}
	}`
	tests := []struct {
		name string
		expr string
		want bool
	}{
		// --- contains-family: text-search over a whole array/object ---
		{"contains_array", `contains("log.Parameters", "ForwardTo")`, true},
		{"contains_array_false", `contains("log.Parameters", "Nope")`, false},
		{"contains_object", `contains("log.nested", "0.0.0.0/0")`, true},
		{"containsArray_list_overload", `contains("log.Parameters", ["ForwardTo", "Enabled"])`, true},
		{"containsArray_list_overload_false", `contains("log.Parameters", ["Nope"])`, false},
		{"containsAll_array", `containsAll("log.Parameters", ["ForwardTo", "Enabled"])`, true},
		{"containsAll_array_false", `containsAll("log.Parameters", ["ForwardTo", "Nope"])`, false},
		{"startsWith_array", `startsWith("log.Parameters", "[")`, true},
		{"endsWith_object", `endsWith("log.nested", "}")`, true},
		{"containsQuery_first_match", `contains("log.Parameters.#(Name==ForwardTo).Value", "evil.com")`, true},
		// --- regexMatch: scalar-only type guard, objects/arrays are false ---
		{"regexMatch_array_false", `regexMatch("log.Parameters", "attacker@.*")`, false},
		{"regexMatch_object_false", `regexMatch("log.nested", "cidr")`, false},
		{"regexMatch_emptyObj_false", `regexMatch("log.emptyObj", ".+")`, false},
		{"regexMatch_emptyArr_false", `regexMatch("log.emptyArr", ".+")`, false},
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
