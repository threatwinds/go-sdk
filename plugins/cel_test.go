package plugins

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Address struct {
	Street *string `json:"street"`
	City   *string `json:"city"`
	Zip    *string `json:"zip"`
}

type Contact struct {
	Type  *string `json:"type"`
	Value *string `json:"value"`
}

type TestData struct {
	Name     *string        `json:"name"`
	Age      *int           `json:"age"`
	Score    *float64       `json:"score"`
	Address  *Address       `json:"address"`
	Contacts []*Contact     `json:"contacts"`
	Metadata map[string]any `json:"metadata"`
	Tags     []string       `json:"tags"`
	IP       *string        `json:"ip"`
}

func TestEvaluate(t *testing.T) {
	name := "Alice"
	age := 30
	score := 90.5

	// Address fields
	street := "123 Main St"
	city := "New York"
	zip := "10001"

	// Contact fields
	emailType := "email"
	emailValue := "alice@example.com"
	phoneType := "phone"
	phoneValue := "555-123-4567"
	ipAddr := "192.168.0.1"

	// Tags
	tags := []string{"student", "premium", "active"}

	// Create a complete test data with nested structures
	completeData := &TestData{
		Name:  &name,
		Age:   &age,
		Score: &score,
		Address: &Address{
			Street: &street,
			City:   &city,
			Zip:    &zip,
		},
		Contacts: []*Contact{
			{
				Type:  &emailType,
				Value: &emailValue,
			},
			{
				Type:  &phoneType,
				Value: &phoneValue,
			},
		},
		Metadata: map[string]any{
			"isActive":    true,
			"yearJoined":  2020,
			"avgGrade":    85.5,
			"department":  "Computer Science",
			"preferences": map[string]any{"theme": "dark", "notifications": true},
			"nullValue":   nil,
			"emptyMap":    map[string]any{},
			"mapWithNil":  map[string]any{"key": nil},
		},
		Tags: tags,
		IP:   &ipAddr,
	}

	// Create data with nil fields for testing nil handling
	var nilStreet, nilCity, nilZip *string
	var nilEmailType, nilEmailValue *string

	dataWithNils := &TestData{
		Name:  &name,
		Age:   &age,
		Score: nil,
		Address: &Address{
			Street: nilStreet,
			City:   nilCity,
			Zip:    nilZip,
		},
		Contacts: []*Contact{
			{
				Type:  nilEmailType,
				Value: nilEmailValue,
			},
			nil,
		},
		Metadata: map[string]any{
			"hasNilValues": true,
			"nilMap":       nil,
		},
		Tags: []string{},
	}

	tests := []struct {
		name       string
		data       *TestData
		expression string
		want       bool
		expectErr  bool
	}{
		{
			name: "valid expression true",
			data: &TestData{
				Name:  &name,
				Age:   &age,
				Score: &score,
			},
			expression: "age > double(20)",
			want:       true,
			expectErr:  false,
		},
		{
			name: "valid expression false",
			data: &TestData{
				Name:  &name,
				Age:   &age,
				Score: &score,
			},
			expression: "age < double(20)",
			want:       false,
			expectErr:  false,
		},
		{
			name: "complex expression true",
			data: &TestData{
				Name:  &name,
				Age:   &age,
				Score: &score,
			},
			expression: `age > double(20) && score > 85.0`,
			want:       true,
			expectErr:  false,
		},
		{
			name: "complex expression false",
			data: &TestData{
				Name:  &name,
				Age:   &age,
				Score: &score,
			},
			expression: `age > double(40) || score < 50.0`,
			want:       false,
			expectErr:  false,
		},
		{
			name: "missing field in expression",
			data: &TestData{
				Name: &name,
				Age:  &age,
			},
			expression: `score > 80.0`,
			want:       false,
			expectErr:  true,
		},
		{
			name: "invalid expression syntax",
			data: &TestData{
				Name:  &name,
				Age:   &age,
				Score: &score,
			},
			expression: `age > `,
			want:       false,
			expectErr:  true,
		},
		{
			name:       "nil data",
			data:       nil,
			expression: `age > 30`,
			want:       false,
			expectErr:  true,
		},
		{
			name: "empty expression",
			data: &TestData{
				Name:  &name,
				Age:   &age,
				Score: &score,
			},
			expression: ``,
			want:       false,
			expectErr:  true,
		},
		{
			name:       "map value access",
			data:       completeData,
			expression: `metadata.isActive == true`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "nested map value access",
			data:       completeData,
			expression: `metadata.preferences.theme == "dark"`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "numeric comparison in map",
			data:       completeData,
			expression: `metadata.yearJoined >= 2020 && metadata.avgGrade > 80.0`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "array length check",
			data:       completeData,
			expression: `size(contacts) == 2`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "check value in array",
			data:       completeData,
			expression: `"premium" in tags`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "string operations",
			data:       completeData,
			expression: `name.startsWith("Al") && name.endsWith("ce") && name.contains("lic")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "map key existence check",
			data:       completeData,
			expression: `"isActive" in metadata && !("nonExistent" in metadata)`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "multiple array elements check",
			data:       completeData,
			expression: `"student" in tags && "premium" in tags && "active" in tags && !("inactive" in tags)`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "complex nested map access",
			data:       completeData,
			expression: `metadata.preferences.theme == "dark" && metadata.preferences.notifications == true`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "string concatenation and comparison",
			data:       completeData,
			expression: `(name + " Smith") == "Alice Smith"`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "nil value in map",
			data:       completeData,
			expression: `metadata.nullValue == null`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "empty map in map",
			data:       completeData,
			expression: `size(metadata.emptyMap) == 0`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "nil value in nested map",
			data:       completeData,
			expression: `metadata.mapWithNil.key == null`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "empty array check",
			data:       dataWithNils,
			expression: `size(tags) == 0`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "nil value in map with condition",
			data:       dataWithNils,
			expression: `metadata.hasNilValues == true && metadata.nilMap == null`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "array size check",
			data:       completeData,
			expression: `size(contacts) == 2 && size(tags) == 3`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "array element presence check",
			data:       completeData,
			expression: `"student" in tags && "premium" in tags && "active" in tags`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "array element absence check",
			data:       completeData,
			expression: `!("nonexistent" in tags)`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "combined array and map operations",
			data:       completeData,
			expression: `size(contacts) == 2 && metadata.yearJoined > 2019 && "premium" in tags`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "inCIDR true",
			data:       completeData,
			want:       true,
			expression: `inCIDR("ip", "192.168.0.0/24")`,
			expectErr:  false,
		},
		{
			name:       "equal - field equals literal string",
			data:       completeData,
			expression: `equal("name", "Alice")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - field not equals literal string",
			data:       completeData,
			expression: `equal("address.city", "Los Angeles")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "equal - comparing metadata string values",
			data:       completeData,
			expression: `equal("metadata.department", "Computer Science")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - one field doesn't exist returns false",
			data:       completeData,
			expression: `equal("nonexistent.field", "name")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "equal - type mismatch number vs string returns false",
			data:       completeData,
			expression: `equal("age", "name")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "equal - type mismatch boolean vs string returns false",
			data:       completeData,
			expression: `equal("metadata.isActive", "name")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "equal - different string values",
			data:       completeData,
			expression: `equal("name", "address.city")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "equal - complex expression with AND",
			data:       completeData,
			expression: `equal("address.city", "New York") && equal("address.zip", "10001")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - complex expression with OR",
			data:       completeData,
			expression: `equal("address.city", "Boston") || equal("address.city", "New York")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - combined with exists",
			data:       completeData,
			expression: `exists("address.city") && equal("address.city", "New York")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - numeric strings comparison",
			data:       completeData,
			expression: `equal("address.zip", "10001")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - comparing street with literal",
			data:       completeData,
			expression: `equal("address.street", "123 Main St")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - comparing contact types",
			data:       completeData,
			expression: `equal("contacts.0.type", "email")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "equal - array type returns false",
			data:       completeData,
			expression: `equal("tags", "tags")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - exact match same case",
			data:       completeData,
			expression: `lowerEqual("name", "Alice")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - match different case lowercase",
			data:       completeData,
			expression: `lowerEqual("name", "alice")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - match different case uppercase",
			data:       completeData,
			expression: `lowerEqual("name", "ALICE")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - match different case mixed",
			data:       completeData,
			expression: `lowerEqual("name", "aLiCe")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - no match different values",
			data:       completeData,
			expression: `lowerEqual("name", "Bob")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - city match case insensitive",
			data:       completeData,
			expression: `lowerEqual("address.city", "new york")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - city match uppercase",
			data:       completeData,
			expression: `lowerEqual("address.city", "NEW YORK")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - department match case insensitive",
			data:       completeData,
			expression: `lowerEqual("metadata.department", "computer science")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - department match uppercase",
			data:       completeData,
			expression: `lowerEqual("metadata.department", "COMPUTER SCIENCE")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - field doesn't exist returns false",
			data:       completeData,
			expression: `lowerEqual("nonexistent.field", "value")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - type mismatch number vs string returns false",
			data:       completeData,
			expression: `lowerEqual("age", "30")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - type mismatch boolean vs string returns false",
			data:       completeData,
			expression: `lowerEqual("metadata.isActive", "true")`,
			want:       false,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - complex expression with AND",
			data:       completeData,
			expression: `lowerEqual("address.city", "NEW YORK") && lowerEqual("name", "alice")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - complex expression with OR",
			data:       completeData,
			expression: `lowerEqual("address.city", "BOSTON") || lowerEqual("address.city", "new york")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - combined with equal",
			data:       completeData,
			expression: `equal("address.zip", "10001") && lowerEqual("address.city", "NEW YORK")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - email case insensitive",
			data:       completeData,
			expression: `lowerEqual("contacts.0.value", "ALICE@EXAMPLE.COM")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - street address mixed case",
			data:       completeData,
			expression: `lowerEqual("address.street", "123 main st")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - contact type case insensitive",
			data:       completeData,
			expression: `lowerEqual("contacts.0.type", "EMAIL")`,
			want:       true,
			expectErr:  false,
		},
		{
			name:       "lowerEqual - combined with exists",
			data:       completeData,
			expression: `exists("name") && lowerEqual("name", "ALICE")`,
			want:       true,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.data)
			assert.NoError(t, err)
			strData := string(data)
			got, err := Evaluate(&strData, tt.expression)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
