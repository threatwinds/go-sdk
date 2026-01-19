package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCELFlexibleOverloads(t *testing.T) {
	cache := NewCELCache("flexible_test")

	// Test case data
	rawJSON := `{
		"string_num": "100",
		"int_num": 100,
		"float_num": 100.0,
		"bool_val": true,
		"text": "Hello World",
		"ip": "192.168.1.1",
		"status": "active",
		"items": ["a", "b", "c"],
		"version": "v1.2.3"
	}`

	tests := []struct {
		name       string
		expression string
		want       bool
	}{
		// Existence
		{"exists_true", `exists("string_num")`, true},
		{"exists_false", `exists("non_existent")`, false},

		// Safe (Returns default if missing)
		{"safe_string_present", `safe("status", "pending") == "active"`, true},
		{"safe_string_missing", `safe("missing", "pending") == "pending"`, true},
		{"safe_num_present", `safe("int_num", 0.0) == 100.0`, true},
		{"safe_num_from_string", `safe("string_num", 0.0) == 100.0`, true},
		{"safe_bool_present", `safe("bool_val", false) == true`, true},

		// Equals (Flexible)
		{"equals_string_string", `equals("string_num", "100")`, true},
		{"equals_string_int", `equals("string_num", 100)`, true},
		{"equals_int_string", `equals("int_num", "100")`, true},
		{"equals_int_float", `equals("int_num", 100.0)`, true},
		{"equals_float_int", `equals("float_num", 100)`, true},
		{"equals_text", `equals("text", "Hello World")`, true},

		// EqualsIgnoreCase
		{"equalsIgnoreCase_true", `equalsIgnoreCase("text", "hello world")`, true},
		{"equalsIgnoreCase_false", `equalsIgnoreCase("text", "bye")`, false},

		// Contains
		{"contains_string_true", `contains("text", "Hello")`, true},
		{"contains_list_true", `contains("text", ["Hello", "Bye"])`, true},
		{"contains_list_false", `contains("text", ["Bye", "Adios"])`, false},

		// ContainsAll
		{"containsAll_true", `containsAll("text", ["Hello", "World"])`, true},
		{"containsAll_false", `containsAll("text", ["Hello", "Bye"])`, false},

		// OneOf (Hybrid)
		{"oneOf_string_match", `oneOf("status", ["active", "inactive"])`, true},
		{"oneOf_numeric_match_string_to_int", `oneOf("string_num", [100, 200])`, true},
		{"oneOf_numeric_match_int_to_string", `oneOf("int_num", ["100", "200"])`, true},
		{"oneOf_numeric_match_float", `oneOf("float_num", [100.0, 300.5])`, true},
		{"oneOf_mixed_match", `oneOf("int_num", ["active", 100])`, true},
		{"oneOf_false", `oneOf("status", ["pending", "deleted"])`, false},

		// StartsWith / EndsWith
		{"startsWith_string", `startsWith("text", "Hello")`, true},
		{"startsWith_list", `startsWith("text", ["Hi", "Hello"])`, true},
		{"endsWith_string", `endsWith("text", "World")`, true},
		{"endsWith_list", `endsWith("text", ["Earth", "World"])`, true},

		// Regex
		{"regexMatch_true", `regexMatch("version", "v[0-9]\\.[0-9]\\.[0-9]")`, true},
		{"regexMatch_false", `regexMatch("version", "v[a-z]")`, false},

		// Comparison (Flexible)
		{"greaterThan_true", `greaterThan("string_num", 50)`, true},
		{"greaterThan_float", `greaterThan("int_num", 99.9)`, true},
		{"lessThan_true", `lessThan("float_num", 200)`, true},
		{"lessOrEqual_true", `lessOrEqual("string_num", 100)`, true},
		{"greaterOrEqual_true", `greaterOrEqual("int_num", 100.0)`, true},

		// Network
		{"inCIDR_ipv4_true", `inCIDR("ip", "192.168.1.0/24")`, true},
		{"inCIDR_ipv4_false", `inCIDR("ip", "10.0.0.0/8")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := cache.Evaluate(&rawJSON, tt.expression)
			assert.NoError(t, err, "Expression: %s", tt.expression)
			assert.Equal(t, tt.want, res, "Expression: %s", tt.expression)
		})
	}
}

func TestCELSafeComplex(t *testing.T) {
	cache := NewCELCache("safe_test")
	data := `{"val": "123.45"}`

	// Test if safe can recover a float from a string
	res, err := cache.Evaluate(&data, `safe("val", 0.0) == 123.45`)
	assert.NoError(t, err)
	assert.True(t, res)

	// Test if safe returns default for invalid numeric string
	data2 := `{"val": "not_a_number"}`
	res2, err := cache.Evaluate(&data2, `safe("val", 9.9) == 9.9`)
	assert.NoError(t, err)
	assert.True(t, res2)
}

func TestCELRobustness(t *testing.T) {
	cache := NewCELCache("robustness_test")

	// Testing with messy or invalid data
	rawJSON := `{
		"ip": "invalid_ip",
		"status": null,
		"meta": {"tags": ["a", "b"]},
		"val": "not_a_number",
		"empty": "",
		"nested": {
			"key": "value"
		}
	}`

	tests := []struct {
		name       string
		expression string
		want       bool
	}{
		// Pathing errors (fields that don't exist at any level)
		{"non_existent_top", `exists("missing")`, false},
		{"non_existent_nested", `exists("nested.missing")`, false},
		{"safe_on_missing", `safe("missing.field", "default") == "default"`, true},
		{"equals_on_missing", `equals("missing.field", "value")`, false},

		// Null handling (field exists but is null)
		{"null_exists", `exists("status")`, true},
		{"null_equals_string", `equals("status", "null")`, false},
		{"null_equals_int", `equals("status", 0)`, false},
		{"safe_on_null", `safe("status", "default") == "default"`, true},

		// Invalid inputs for specific overloads
		{"ip_invalid_format_inCIDR", `inCIDR("ip", "192.168.1.0/24")`, false},
		{"cidr_invalid_format", `inCIDR("empty", "invalid_cidr")`, false},
		{"regex_malformed", `regexMatch("version", "[")`, false},
		{"regex_on_empty", `regexMatch("empty", ".*")`, true},

		// Type Mismatches & Confusion
		{"greaterThan_on_text", `greaterThan("val", 10)`, false},
		{"safe_num_on_text", `safe("val", 0.0) == 0.0`, true},   // "not_a_number" cannot be float
		{"oneOf_on_object", `oneOf("meta", ["a", "b"])`, false}, // meta is an object, not a string
		{"contains_on_object", `contains("meta", "tags")`, false},

		// Empty string scenarios
		{"equals_empty", `equals("empty", "")`, true},
		{"startsWith_empty", `startsWith("empty", "")`, true},
		{"oneOf_with_empty", `oneOf("empty", ["", "other"])`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := cache.Evaluate(&rawJSON, tt.expression)
			assert.NoError(t, err, "Expression: %s", tt.expression)
			assert.Equal(t, tt.want, res, "Expression: %s", tt.expression)
		})
	}
}
