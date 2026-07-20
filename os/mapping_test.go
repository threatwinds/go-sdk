package os

import "testing"

// Tests for mapping.go helper functions (delegates to field_types.go)

func TestMappingIsTextType(t *testing.T) {
	// Verify the delegated function works correctly
	if !isTextType("text") {
		t.Error("isTextType('text') should be true")
	}
	if !isTextType("match_only_text") {
		t.Error("isTextType('match_only_text') should be true")
	}
	if isTextType("keyword") {
		t.Error("isTextType('keyword') should be false")
	}
	if isTextType("integer") {
		t.Error("isTextType('integer') should be false")
	}
}

func TestMappingIsTermType(t *testing.T) {
	// Core types
	if !isTermType("keyword") {
		t.Error("isTermType('keyword') should be true")
	}
	if !isTermType("integer") {
		t.Error("isTermType('integer') should be true")
	}
	if !isTermType("long") {
		t.Error("isTermType('long') should be true")
	}
	if !isTermType("date") {
		t.Error("isTermType('date') should be true")
	}
	if !isTermType("boolean") {
		t.Error("isTermType('boolean') should be true")
	}
	if !isTermType("ip") {
		t.Error("isTermType('ip') should be true")
	}

	// New types added via registry
	if !isTermType("short") {
		t.Error("isTermType('short') should be true")
	}
	if !isTermType("byte") {
		t.Error("isTermType('byte') should be true")
	}
	if !isTermType("unsigned_long") {
		t.Error("isTermType('unsigned_long') should be true")
	}
	if !isTermType("half_float") {
		t.Error("isTermType('half_float') should be true")
	}
	if !isTermType("scaled_float") {
		t.Error("isTermType('scaled_float') should be true")
	}
	if !isTermType("date_nanos") {
		t.Error("isTermType('date_nanos') should be true")
	}
	if !isTermType("constant_keyword") {
		t.Error("isTermType('constant_keyword') should be true")
	}
	if !isTermType("version") {
		t.Error("isTermType('version') should be true")
	}

	// Text types should not be term types
	if isTermType("text") {
		t.Error("isTermType('text') should be false")
	}

	// Geo types require special queries
	if isTermType("geo_point") {
		t.Error("isTermType('geo_point') should be false")
	}
}

func TestMappingGetMostPermissiveType(t *testing.T) {
	result := getMostPermissiveType([]string{"keyword", "text"})
	if result != "text" {
		t.Errorf("getMostPermissiveType returned %q, want 'text'", result)
	}

	result = getMostPermissiveType([]string{"integer", "long"})
	if result != "long" {
		t.Errorf("getMostPermissiveType returned %q, want 'long'", result)
	}

	result = getMostPermissiveType([]string{"float", "double"})
	if result != "double" {
		t.Errorf("getMostPermissiveType returned %q, want 'double'", result)
	}

	// New types
	result = getMostPermissiveType([]string{"date", "date_nanos"})
	if result != "date_nanos" {
		t.Errorf("getMostPermissiveType returned %q, want 'date_nanos'", result)
	}
}

func TestContainsHelper(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("contains should find 'a' in slice")
	}
	if !contains(slice, "b") {
		t.Error("contains should find 'b' in slice")
	}
	if !contains(slice, "c") {
		t.Error("contains should find 'c' in slice")
	}
	if contains(slice, "x") {
		t.Error("contains should not find 'x' in slice")
	}
	if contains(slice, "") {
		t.Error("contains should not find empty string in slice")
	}
	if contains([]string{}, "a") {
		t.Error("contains should not find anything in empty slice")
	}
}

func TestQueryTypeString(t *testing.T) {
	tests := []struct {
		qt       QueryType
		expected string
	}{
		{QueryTypeTerm, "Term"},
		{QueryTypeTerms, "Terms"},
		{QueryTypeMatch, "Match"},
		{QueryTypeMatchPhrase, "MatchPhrase"},
		{QueryTypeRange, "Range"},
		{QueryTypeSort, "Sort"},
		{QueryTypeAggregation, "Aggregation"},
		{QueryTypeExists, "Exists"},
		{QueryTypeWildcard, "Wildcard"},
		{QueryTypeFuzzy, "Fuzzy"},
		{QueryTypeRegexp, "Regexp"},
		{QueryTypeMatchPhrasePrefix, "MatchPhrasePrefix"},
		{QueryTypePrefix, "Prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := queryTypeString(tt.qt); got != tt.expected {
				t.Errorf("queryTypeString(%v) = %q, want %q", tt.qt, got, tt.expected)
			}
		})
	}
}

func TestQueryTypeStringUnknown(t *testing.T) {
	// Test unknown query type
	unknown := QueryType(999)
	result := queryTypeString(unknown)
	if result != "Unknown" {
		t.Errorf("queryTypeString(999) = %q, want 'Unknown'", result)
	}
}

// Test that mapping helper functions work correctly with new field types
func TestMappingHelperFunctions_NewTypes(t *testing.T) {
	// All new numeric types should be term types
	numericTypes := []string{"short", "byte", "half_float", "scaled_float", "unsigned_long", "token_count"}
	for _, typ := range numericTypes {
		if !isTermType(typ) {
			t.Errorf("isTermType(%q) = false, want true (new numeric type)", typ)
		}
		if isTextType(typ) {
			t.Errorf("isTextType(%q) = true, want false (numeric type)", typ)
		}
	}

	// New text types should be text types
	textLikeTypes := []string{"match_only_text", "search_as_you_type", "semantic"}
	for _, typ := range textLikeTypes {
		if !isTextType(typ) {
			t.Errorf("isTextType(%q) = false, want true (text-like type)", typ)
		}
	}

	// Keyword-like types should be term types
	keywordLikeTypes := []string{"constant_keyword", "wildcard", "version"}
	for _, typ := range keywordLikeTypes {
		if !isTermType(typ) {
			t.Errorf("isTermType(%q) = false, want true (keyword-like type)", typ)
		}
	}

	// Geo types should not be text or term types
	geoTypes := []string{"geo_point", "geo_shape", "xy_point", "xy_shape"}
	for _, typ := range geoTypes {
		if isTextType(typ) {
			t.Errorf("isTextType(%q) = true, want false (geo type)", typ)
		}
		if isTermType(typ) {
			t.Errorf("isTermType(%q) = true, want false (geo type)", typ)
		}
	}

	// Vector types should not be text or term types
	vectorTypes := []string{"knn_vector", "sparse_vector"}
	for _, typ := range vectorTypes {
		if isTextType(typ) {
			t.Errorf("isTextType(%q) = true, want false (vector type)", typ)
		}
		if isTermType(typ) {
			t.Errorf("isTermType(%q) = true, want false (vector type)", typ)
		}
	}
}
