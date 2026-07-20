package os

import "testing"

func TestFieldTypeRegistry_AllTypesRegistered(t *testing.T) {
	expectedTypes := []string{
		// Text
		"text", "match_only_text", "search_as_you_type",
		// Keyword
		"keyword", "constant_keyword", "wildcard", "version",
		// Numeric
		"integer", "long", "short", "byte", "unsigned_long",
		"float", "double", "half_float", "scaled_float", "token_count",
		// Date
		"date", "date_nanos",
		// Other
		"boolean", "ip", "binary",
		// Geo
		"geo_point", "geo_shape", "xy_point", "xy_shape",
		// Range
		"integer_range", "long_range", "float_range", "double_range", "date_range", "ip_range",
		// Object
		"object", "nested", "flat_object", "join",
		// Vector
		"knn_vector", "sparse_vector",
		// Completion
		"completion",
		// Specialized
		"semantic", "rank_feature", "rank_features", "percolator",
		// Alias
		"alias",
	}

	for _, typeName := range expectedTypes {
		if _, ok := GetFieldTypeInfo(typeName); !ok {
			t.Errorf("Field type %q not registered", typeName)
		}
	}
}

func TestIsTextType(t *testing.T) {
	textTypes := []string{"text", "match_only_text", "search_as_you_type", "semantic"}
	nonTextTypes := []string{"keyword", "integer", "geo_point", "knn_vector", "nested"}

	for _, typ := range textTypes {
		if !IsTextType(typ) {
			t.Errorf("IsTextType(%q) = false, want true", typ)
		}
	}
	for _, typ := range nonTextTypes {
		if IsTextType(typ) {
			t.Errorf("IsTextType(%q) = true, want false", typ)
		}
	}
}

func TestIsTermType(t *testing.T) {
	termTypes := []string{
		"keyword", "constant_keyword", "wildcard", "version",
		"integer", "long", "short", "byte", "unsigned_long",
		"float", "double", "half_float", "scaled_float", "token_count",
		"date", "date_nanos", "boolean", "ip", "flat_object",
	}
	nonTermTypes := []string{
		"text", "match_only_text", "geo_point", "geo_shape",
		"knn_vector", "nested", "object", "binary", "percolator",
	}

	for _, typ := range termTypes {
		if !IsTermType(typ) {
			t.Errorf("IsTermType(%q) = false, want true", typ)
		}
	}
	for _, typ := range nonTermTypes {
		if IsTermType(typ) {
			t.Errorf("IsTermType(%q) = true, want false", typ)
		}
	}
}

func TestIsRangeType(t *testing.T) {
	rangeTypes := []string{
		"keyword", "version", "long", "integer", "short", "byte",
		"unsigned_long", "double", "float", "half_float", "scaled_float",
		"token_count", "date", "date_nanos", "ip",
		"integer_range", "long_range", "float_range", "double_range", "date_range", "ip_range",
	}
	nonRangeTypes := []string{
		"text", "boolean", "geo_point", "knn_vector", "nested", "binary",
	}

	for _, typ := range rangeTypes {
		if !IsRangeType(typ) {
			t.Errorf("IsRangeType(%q) = false, want true", typ)
		}
	}
	for _, typ := range nonRangeTypes {
		if IsRangeType(typ) {
			t.Errorf("IsRangeType(%q) = true, want false", typ)
		}
	}
}

func TestIsSortable(t *testing.T) {
	sortableTypes := []string{
		"keyword", "constant_keyword", "wildcard", "version",
		"long", "integer", "date", "boolean", "ip", "geo_point",
	}
	nonSortableTypes := []string{
		"text", "geo_shape", "nested", "knn_vector", "binary", "completion",
	}

	for _, typ := range sortableTypes {
		if !IsSortable(typ) {
			t.Errorf("IsSortable(%q) = false, want true", typ)
		}
	}
	for _, typ := range nonSortableTypes {
		if IsSortable(typ) {
			t.Errorf("IsSortable(%q) = true, want false", typ)
		}
	}
}

func TestIsAggregatable(t *testing.T) {
	aggregatableTypes := []string{
		"keyword", "long", "integer", "date", "boolean", "ip",
		"geo_point", "geo_shape", "nested", "flat_object", "join",
		"integer_range", "long_range",
	}
	nonAggregatableTypes := []string{
		"text", "binary", "knn_vector", "sparse_vector", "completion", "percolator",
	}

	for _, typ := range aggregatableTypes {
		if !IsAggregatable(typ) {
			t.Errorf("IsAggregatable(%q) = false, want true", typ)
		}
	}
	for _, typ := range nonAggregatableTypes {
		if IsAggregatable(typ) {
			t.Errorf("IsAggregatable(%q) = true, want false", typ)
		}
	}
}

func TestGeoTypesRequireSpecialQueries(t *testing.T) {
	geoTypes := []string{"geo_point", "geo_shape", "xy_point", "xy_shape"}

	for _, typ := range geoTypes {
		if IsTextType(typ) {
			t.Errorf("Geo type %q should not allow match queries", typ)
		}
		if IsTermType(typ) {
			t.Errorf("Geo type %q should not allow term queries", typ)
		}
	}
}

func TestVectorTypesRequireSpecialQueries(t *testing.T) {
	vectorTypes := []string{"knn_vector", "sparse_vector"}

	for _, typ := range vectorTypes {
		if IsTextType(typ) {
			t.Errorf("Vector type %q should not allow match queries", typ)
		}
		if IsTermType(typ) {
			t.Errorf("Vector type %q should not allow term queries", typ)
		}
		if IsRangeType(typ) {
			t.Errorf("Vector type %q should not allow range queries", typ)
		}
	}
}

func TestGetMostPermissiveType(t *testing.T) {
	tests := []struct {
		name     string
		types    []string
		expected string
	}{
		{"text_vs_keyword", []string{"keyword", "text"}, "text"},
		{"text_vs_all", []string{"integer", "keyword", "text", "boolean"}, "text"},
		{"long_vs_integer", []string{"integer", "long"}, "long"},
		{"unsigned_long_vs_long", []string{"long", "unsigned_long"}, "unsigned_long"},
		{"double_vs_float", []string{"float", "double"}, "double"},
		{"scaled_float_vs_float", []string{"float", "scaled_float"}, "scaled_float"},
		{"date_nanos_vs_date", []string{"date", "date_nanos"}, "date_nanos"},
		{"keyword_vs_constant_keyword", []string{"constant_keyword", "keyword"}, "keyword"},
		{"unknown_fallback", []string{"unknown1", "unknown2"}, "unknown1"},
		{"single_type", []string{"keyword"}, "keyword"},
		{"empty_slice", []string{}, ""},
		{"semantic_vs_keyword", []string{"keyword", "semantic"}, "semantic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMostPermissiveType(tt.types)
			if result != tt.expected {
				t.Errorf("GetMostPermissiveType(%v) = %q, want %q", tt.types, result, tt.expected)
			}
		})
	}
}

func TestNeedsKeywordSubfield(t *testing.T) {
	needsKeyword := []string{"text", "match_only_text"}
	doesNotNeed := []string{
		"keyword", "integer", "date", "boolean", "search_as_you_type",
		"geo_point", "knn_vector", "completion",
	}

	for _, typ := range needsKeyword {
		if !NeedsKeywordSubfield(typ) {
			t.Errorf("NeedsKeywordSubfield(%q) = false, want true", typ)
		}
	}
	for _, typ := range doesNotNeed {
		if NeedsKeywordSubfield(typ) {
			t.Errorf("NeedsKeywordSubfield(%q) = true, want false", typ)
		}
	}
}

func TestFieldTypeCategory(t *testing.T) {
	tests := []struct {
		typeName string
		category FieldTypeCategory
	}{
		{"text", CategoryText},
		{"match_only_text", CategoryText},
		{"keyword", CategoryKeyword},
		{"constant_keyword", CategoryKeyword},
		{"wildcard", CategoryKeyword},
		{"version", CategoryKeyword},
		{"long", CategoryNumeric},
		{"integer", CategoryNumeric},
		{"short", CategoryNumeric},
		{"byte", CategoryNumeric},
		{"unsigned_long", CategoryNumeric},
		{"double", CategoryNumeric},
		{"float", CategoryNumeric},
		{"half_float", CategoryNumeric},
		{"scaled_float", CategoryNumeric},
		{"token_count", CategoryNumeric},
		{"date", CategoryDate},
		{"date_nanos", CategoryDate},
		{"boolean", CategoryBoolean},
		{"ip", CategoryIP},
		{"binary", CategoryBinary},
		{"geo_point", CategoryGeo},
		{"geo_shape", CategoryGeo},
		{"xy_point", CategoryCartesian},
		{"xy_shape", CategoryCartesian},
		{"integer_range", CategoryRange},
		{"long_range", CategoryRange},
		{"float_range", CategoryRange},
		{"double_range", CategoryRange},
		{"date_range", CategoryRange},
		{"ip_range", CategoryRange},
		{"object", CategoryObject},
		{"nested", CategoryObject},
		{"flat_object", CategoryObject},
		{"join", CategoryObject},
		{"knn_vector", CategoryVector},
		{"sparse_vector", CategoryVector},
		{"completion", CategoryCompletion},
		{"search_as_you_type", CategoryCompletion},
		{"semantic", CategorySpecialized},
		{"rank_feature", CategorySpecialized},
		{"rank_features", CategorySpecialized},
		{"percolator", CategorySpecialized},
		{"alias", CategoryAlias},
		{"unknown", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if got := GetFieldTypeCategory(tt.typeName); got != tt.category {
				t.Errorf("GetFieldTypeCategory(%q) = %v, want %v", tt.typeName, got, tt.category)
			}
		})
	}
}

func TestGetTypePriority(t *testing.T) {
	// Text should have highest priority
	if GetTypePriority("text") <= GetTypePriority("keyword") {
		t.Error("text should have higher priority than keyword")
	}

	// Keyword should be higher than numeric
	if GetTypePriority("keyword") <= GetTypePriority("long") {
		t.Error("keyword should have higher priority than long")
	}

	// Long should be higher than integer
	if GetTypePriority("long") <= GetTypePriority("integer") {
		t.Error("long should have higher priority than integer")
	}

	// Unknown types should have priority 0
	if GetTypePriority("unknown_type") != 0 {
		t.Errorf("unknown type should have priority 0, got %d", GetTypePriority("unknown_type"))
	}
}

func TestCategoryString(t *testing.T) {
	tests := []struct {
		category FieldTypeCategory
		expected string
	}{
		{CategoryUnknown, "Unknown"},
		{CategoryText, "Text"},
		{CategoryKeyword, "Keyword"},
		{CategoryNumeric, "Numeric"},
		{CategoryDate, "Date"},
		{CategoryBoolean, "Boolean"},
		{CategoryBinary, "Binary"},
		{CategoryIP, "IP"},
		{CategoryGeo, "Geo"},
		{CategoryCartesian, "Cartesian"},
		{CategoryRange, "Range"},
		{CategoryObject, "Object"},
		{CategoryVector, "Vector"},
		{CategoryCompletion, "Completion"},
		{CategorySpecialized, "Specialized"},
		{CategoryAlias, "Alias"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.category.String(); got != tt.expected {
				t.Errorf("FieldTypeCategory(%d).String() = %q, want %q", tt.category, got, tt.expected)
			}
		})
	}
}

func TestGetAllFieldTypes(t *testing.T) {
	types := GetAllFieldTypes()

	// Should have at least 40 types
	if len(types) < 40 {
		t.Errorf("Expected at least 40 field types, got %d", len(types))
	}

	// Check that some core types are present
	coreTypes := []string{"text", "keyword", "long", "date", "boolean", "geo_point"}
	typeMap := make(map[string]bool)
	for _, typ := range types {
		typeMap[typ] = true
	}

	for _, core := range coreTypes {
		if !typeMap[core] {
			t.Errorf("Core type %q not found in GetAllFieldTypes()", core)
		}
	}
}

func TestRegisterFieldType(t *testing.T) {
	// Register a custom type
	customType := FieldTypeInfo{
		Name:         "custom_type",
		Category:     CategorySpecialized,
		AllowsMatch:  true,
		AllowsTerm:   true,
		AllowsRange:  false,
		AllowsSort:   true,
		AllowsAgg:    true,
		NeedsKeyword: false,
		Priority:     50,
	}

	RegisterFieldType(customType)

	// Verify it was registered
	info, ok := GetFieldTypeInfo("custom_type")
	if !ok {
		t.Fatal("Custom type was not registered")
	}

	if info.Name != "custom_type" {
		t.Errorf("Custom type name = %q, want %q", info.Name, "custom_type")
	}

	if !IsTextType("custom_type") {
		t.Error("Custom type should allow match queries")
	}

	if !IsTermType("custom_type") {
		t.Error("Custom type should allow term queries")
	}

	// Clean up - remove custom type
	delete(fieldTypeRegistry, "custom_type")
}

func TestGetFieldTypeInfo_UnknownType(t *testing.T) {
	info, ok := GetFieldTypeInfo("nonexistent_type")
	if ok {
		t.Error("Expected ok=false for unknown type")
	}
	if info.Name != "" {
		t.Errorf("Expected empty FieldTypeInfo for unknown type, got %+v", info)
	}
}
