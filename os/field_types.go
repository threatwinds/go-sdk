package os

// FieldTypeCategory represents a category of OpenSearch field types
type FieldTypeCategory int

const (
	CategoryUnknown     FieldTypeCategory = iota
	CategoryText                          // Full-text searchable types (text, match_only_text)
	CategoryKeyword                       // Exact-match string types (keyword, constant_keyword)
	CategoryNumeric                       // Numeric types (integer, long, float, etc.)
	CategoryDate                          // Date types (date, date_nanos)
	CategoryBoolean                       // Boolean type
	CategoryBinary                        // Binary data type
	CategoryIP                            // IP address type
	CategoryGeo                           // Geographic types (geo_point, geo_shape) - require geo queries
	CategoryCartesian                     // Cartesian types (xy_point, xy_shape) - require geo queries
	CategoryRange                         // Range types (integer_range, date_range, etc.)
	CategoryObject                        // Object/nested types
	CategoryVector                        // Vector types for ML/k-NN
	CategoryCompletion                    // Autocomplete types
	CategorySpecialized                   // Specialized types (percolator, rank, etc.)
	CategoryAlias                         // Alias type
)

// FieldTypeInfo contains metadata about a field type
type FieldTypeInfo struct {
	Name         string            // OpenSearch type name
	Category     FieldTypeCategory // Category this type belongs to
	AllowsMatch  bool              // Can use match/full-text queries
	AllowsTerm   bool              // Can use term/exact queries
	AllowsRange  bool              // Can use range queries
	AllowsSort   bool              // Can be used for sorting
	AllowsAgg    bool              // Can be used for aggregations
	NeedsKeyword bool              // Needs .keyword for term queries (text types)
	Priority     int               // Priority for conflict resolution
}

// fieldTypeRegistry is the centralized registry of all field types
var fieldTypeRegistry = map[string]FieldTypeInfo{
	// === TEXT TYPES ===
	"text": {
		Name: "text", Category: CategoryText,
		AllowsMatch: true, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: true, Priority: 100,
	},
	"match_only_text": {
		Name: "match_only_text", Category: CategoryText,
		AllowsMatch: true, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: true, Priority: 99,
	},
	"search_as_you_type": {
		Name: "search_as_you_type", Category: CategoryCompletion,
		AllowsMatch: true, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 98,
	},

	// === KEYWORD TYPES ===
	"keyword": {
		Name: "keyword", Category: CategoryKeyword,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 90,
	},
	"constant_keyword": {
		Name: "constant_keyword", Category: CategoryKeyword,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: false,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 89,
	},
	"wildcard": {
		Name: "wildcard", Category: CategoryKeyword,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: false,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 88,
	},
	"version": {
		Name: "version", Category: CategoryKeyword,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 87,
	},

	// === NUMERIC TYPES ===
	"long": {
		Name: "long", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 80,
	},
	"integer": {
		Name: "integer", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 78,
	},
	"short": {
		Name: "short", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 76,
	},
	"byte": {
		Name: "byte", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 74,
	},
	"unsigned_long": {
		Name: "unsigned_long", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 82,
	},
	"double": {
		Name: "double", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 72,
	},
	"float": {
		Name: "float", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 70,
	},
	"half_float": {
		Name: "half_float", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 68,
	},
	"scaled_float": {
		Name: "scaled_float", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 71,
	},
	"token_count": {
		Name: "token_count", Category: CategoryNumeric,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 66,
	},

	// === DATE TYPES ===
	"date": {
		Name: "date", Category: CategoryDate,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 50,
	},
	"date_nanos": {
		Name: "date_nanos", Category: CategoryDate,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 51,
	},

	// === BOOLEAN TYPE ===
	"boolean": {
		Name: "boolean", Category: CategoryBoolean,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: false,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 40,
	},

	// === IP TYPE ===
	"ip": {
		Name: "ip", Category: CategoryIP,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: true,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 35,
	},

	// === BINARY TYPE ===
	"binary": {
		Name: "binary", Category: CategoryBinary,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 5,
	},

	// === GEO TYPES (require special geo queries) ===
	"geo_point": {
		Name: "geo_point", Category: CategoryGeo,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: true, AllowsAgg: true, NeedsKeyword: false, Priority: 30,
	},
	"geo_shape": {
		Name: "geo_shape", Category: CategoryGeo,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 29,
	},

	// === CARTESIAN TYPES (require special queries) ===
	"xy_point": {
		Name: "xy_point", Category: CategoryCartesian,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 28,
	},
	"xy_shape": {
		Name: "xy_shape", Category: CategoryCartesian,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 27,
	},

	// === RANGE TYPES ===
	"integer_range": {
		Name: "integer_range", Category: CategoryRange,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: true,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 24,
	},
	"long_range": {
		Name: "long_range", Category: CategoryRange,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: true,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 25,
	},
	"float_range": {
		Name: "float_range", Category: CategoryRange,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: true,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 22,
	},
	"double_range": {
		Name: "double_range", Category: CategoryRange,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: true,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 23,
	},
	"date_range": {
		Name: "date_range", Category: CategoryRange,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: true,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 26,
	},
	"ip_range": {
		Name: "ip_range", Category: CategoryRange,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: true,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 21,
	},

	// === OBJECT TYPES ===
	"object": {
		Name: "object", Category: CategoryObject,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 10,
	},
	"nested": {
		Name: "nested", Category: CategoryObject,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 11,
	},
	"flat_object": {
		Name: "flat_object", Category: CategoryObject,
		AllowsMatch: false, AllowsTerm: true, AllowsRange: false,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 12,
	},
	"join": {
		Name: "join", Category: CategoryObject,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: true, NeedsKeyword: false, Priority: 9,
	},

	// === VECTOR TYPES (require k-NN queries) ===
	"knn_vector": {
		Name: "knn_vector", Category: CategoryVector,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 20,
	},
	"sparse_vector": {
		Name: "sparse_vector", Category: CategoryVector,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 19,
	},

	// === COMPLETION TYPE ===
	"completion": {
		Name: "completion", Category: CategoryCompletion,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 15,
	},

	// === SPECIALIZED TYPES ===
	"semantic": {
		Name: "semantic", Category: CategorySpecialized,
		AllowsMatch: true, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 95,
	},
	"rank_feature": {
		Name: "rank_feature", Category: CategorySpecialized,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 8,
	},
	"rank_features": {
		Name: "rank_features", Category: CategorySpecialized,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 7,
	},
	"percolator": {
		Name: "percolator", Category: CategorySpecialized,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 6,
	},

	// === ALIAS TYPE ===
	"alias": {
		Name: "alias", Category: CategoryAlias,
		AllowsMatch: false, AllowsTerm: false, AllowsRange: false,
		AllowsSort: false, AllowsAgg: false, NeedsKeyword: false, Priority: 1,
	},
}

// GetFieldTypeInfo returns metadata about a field type
func GetFieldTypeInfo(fieldType string) (FieldTypeInfo, bool) {
	info, ok := fieldTypeRegistry[fieldType]
	return info, ok
}

// GetFieldTypeCategory returns the category for a field type
func GetFieldTypeCategory(fieldType string) FieldTypeCategory {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.Category
	}
	return CategoryUnknown
}

// IsTextType returns true if the type supports full-text match queries
func IsTextType(fieldType string) bool {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.AllowsMatch
	}
	return false
}

// IsTermType returns true if the type supports term/exact queries
func IsTermType(fieldType string) bool {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.AllowsTerm
	}
	return false
}

// IsRangeType returns true if the type supports range queries
func IsRangeType(fieldType string) bool {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.AllowsRange
	}
	return false
}

// IsSortable returns true if the type supports sorting
func IsSortable(fieldType string) bool {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.AllowsSort
	}
	return false
}

// IsAggregatable returns true if the type supports aggregations
func IsAggregatable(fieldType string) bool {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.AllowsAgg
	}
	return false
}

// NeedsKeywordSubfield returns true if the type needs .keyword for term queries
func NeedsKeywordSubfield(fieldType string) bool {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.NeedsKeyword
	}
	return false
}

// GetTypePriority returns the priority for conflict resolution
func GetTypePriority(fieldType string) int {
	if info, ok := fieldTypeRegistry[fieldType]; ok {
		return info.Priority
	}
	return 0
}

// GetMostPermissiveType returns the type with highest priority from a list
func GetMostPermissiveType(types []string) string {
	if len(types) == 0 {
		return ""
	}

	mostPermissive := types[0]
	maxPriority := GetTypePriority(mostPermissive)

	for _, typ := range types[1:] {
		priority := GetTypePriority(typ)
		if priority > maxPriority {
			maxPriority = priority
			mostPermissive = typ
		}
	}

	return mostPermissive
}

// RegisterFieldType allows registering custom field types
func RegisterFieldType(info FieldTypeInfo) {
	fieldTypeRegistry[info.Name] = info
}

// GetAllFieldTypes returns all registered field type names
func GetAllFieldTypes() []string {
	types := make([]string, 0, len(fieldTypeRegistry))
	for name := range fieldTypeRegistry {
		types = append(types, name)
	}
	return types
}

// String returns a human-readable string for a category
func (c FieldTypeCategory) String() string {
	switch c {
	case CategoryText:
		return "Text"
	case CategoryKeyword:
		return "Keyword"
	case CategoryNumeric:
		return "Numeric"
	case CategoryDate:
		return "Date"
	case CategoryBoolean:
		return "Boolean"
	case CategoryBinary:
		return "Binary"
	case CategoryIP:
		return "IP"
	case CategoryGeo:
		return "Geo"
	case CategoryCartesian:
		return "Cartesian"
	case CategoryRange:
		return "Range"
	case CategoryObject:
		return "Object"
	case CategoryVector:
		return "Vector"
	case CategoryCompletion:
		return "Completion"
	case CategorySpecialized:
		return "Specialized"
	case CategoryAlias:
		return "Alias"
	default:
		return "Unknown"
	}
}
