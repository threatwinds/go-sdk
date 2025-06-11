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
