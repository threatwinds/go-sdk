package entities

import (
	"github.com/threatwinds/go-sdk/utils"
	"reflect"
	"testing"
)

func TestAttributes_GetAttribute(t *testing.T) {
	// Create a test Attributes instance with some values
	attrs := Attributes{
		Domain:    utils.PointerOf("example.com"),
		IP:        utils.PointerOf("192.168.1.1"),
		Port:      utils.PointerOf("443/tcp"),
		Latitude:  utils.PointerOf(40.7128),
		Longitude: utils.PointerOf(-74.0060),
	}

	// Test cases
	tests := []struct {
		name       string
		tagName    string
		wantValue  interface{}
		wantExists bool
	}{
		{
			name:       "existing string attribute",
			tagName:    "domain",
			wantValue:  "example.com",
			wantExists: true,
		},
		{
			name:       "existing string attribute with dash",
			tagName:    "ip",
			wantValue:  "192.168.1.1",
			wantExists: true,
		},
		{
			name:       "existing int attribute",
			tagName:    "port",
			wantValue:  "443/tcp",
			wantExists: true,
		},
		{
			name:       "existing float64 attribute",
			tagName:    "latitude",
			wantValue:  40.7128,
			wantExists: true,
		},
		{
			name:       "non-existent attribute",
			tagName:    "nonexistent",
			wantValue:  nil,
			wantExists: false,
		},
		{
			name:       "nil attribute",
			tagName:    "email-address", // This field exists but is nil in our test instance
			wantValue:  nil,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotExists := attrs.GetAttribute(tt.tagName)
			if gotExists != tt.wantExists {
				t.Errorf("GetAttribute() exists = %v, want %v", gotExists, tt.wantExists)
			}
			if gotExists && !reflect.DeepEqual(gotValue, tt.wantValue) {
				t.Errorf("GetAttribute() value = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestAttributes_ToMap(t *testing.T) {
	// Create a test Attributes instance with some values
	attrs := Attributes{
		Domain:    utils.PointerOf("example.com"),
		IP:        utils.PointerOf("192.168.1.1"),
		Port:      utils.PointerOf("443/tcp"),
		Latitude:  utils.PointerOf(40.7128),
		Longitude: utils.PointerOf(-74.0060),
	}

	// Call ToMap
	result := attrs.ToMap()

	// Verify the result
	expectedKeys := []string{"domain", "ip", "port", "latitude", "longitude"}
	for _, key := range expectedKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("ToMap() result missing key %s", key)
		}
	}

	// Check specific values
	if val, ok := result["domain"]; !ok || val != *attrs.Domain {
		t.Errorf("ToMap() domain = %v, want %v", val, *attrs.Domain)
	}
	if val, ok := result["ip"]; !ok || val != *attrs.IP {
		t.Errorf("ToMap() ip = %v, want %v", val, *attrs.IP)
	}
	if val, ok := result["port"]; !ok || val != *attrs.Port {
		t.Errorf("ToMap() port = %v, want %v", val, *attrs.Port)
	}
	if val, ok := result["latitude"]; !ok || val != *attrs.Latitude {
		t.Errorf("ToMap() latitude = %v, want %v", val, *attrs.Latitude)
	}
	if val, ok := result["longitude"]; !ok || val != *attrs.Longitude {
		t.Errorf("ToMap() longitude = %v, want %v", val, *attrs.Longitude)
	}

	// Verify that nil fields are not included
	if _, exists := result["email-address"]; exists {
		t.Errorf("ToMap() should not include nil fields, but found email-address")
	}
}

func TestAttributes_SetAttribute(t *testing.T) {
	// Create an empty Attributes instance
	attrs := Attributes{}

	// Test cases
	tests := []struct {
		name      string
		tagName   string
		value     interface{}
		wantSet   bool
		checkFunc func(t *testing.T, attrs *Attributes)
	}{
		{
			name:    "set string attribute",
			tagName: "domain",
			value:   "example.com",
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.Domain == nil {
					t.Errorf("SetAttribute() failed to set domain")
					return
				}
				if *attrs.Domain != "example.com" {
					t.Errorf("SetAttribute() domain = %v, want %v", *attrs.Domain, "example.com")
				}
			},
		},
		{
			name:    "set string attribute with dash",
			tagName: "ip",
			value:   "192.168.1.1",
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.IP == nil {
					t.Errorf("SetAttribute() failed to set IP")
					return
				}
				if *attrs.IP != "192.168.1.1" {
					t.Errorf("SetAttribute() IP = %v, want %v", *attrs.IP, "192.168.1.1")
				}
			},
		},
		{
			name:    "set int attribute",
			tagName: "port",
			value:   "443/tcp",
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.Port == nil {
					t.Errorf("SetAttribute() failed to set port")
					return
				}
				if *attrs.Port != "443/tcp" {
					t.Errorf("SetAttribute() port = %v, want %v", *attrs.Port, 443)
				}
			},
		},
		{
			name:    "set float64 attribute",
			tagName: "latitude",
			value:   40.7128,
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.Latitude == nil {
					t.Errorf("SetAttribute() failed to set latitude")
					return
				}
				if *attrs.Latitude != 40.7128 {
					t.Errorf("SetAttribute() latitude = %v, want %v", *attrs.Latitude, 40.7128)
				}
			},
		},
		{
			name:    "set int as string",
			tagName: "port",
			value:   "8080/tcp",
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.Port == nil {
					t.Errorf("SetAttribute() failed to set port")
					return
				}
				if *attrs.Port != "8080/tcp" {
					t.Errorf("SetAttribute() port = %v, want %v", *attrs.Port, 8080)
				}
			},
		},
		{
			name:    "set float as string",
			tagName: "longitude",
			value:   "-74.0060",
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.Longitude == nil {
					t.Errorf("SetAttribute() failed to set longitude")
					return
				}
				if *attrs.Longitude != -74.0060 {
					t.Errorf("SetAttribute() longitude = %v, want %v", *attrs.Longitude, -74.0060)
				}
			},
		},
		{
			name:    "set attribute to nil",
			tagName: "domain",
			value:   nil,
			wantSet: true,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				if attrs.Domain != nil {
					t.Errorf("SetAttribute() domain = %v, want nil", *attrs.Domain)
				}
			},
		},
		{
			name:    "non-existent attribute",
			tagName: "nonexistent",
			value:   "value",
			wantSet: false,
			checkFunc: func(t *testing.T, attrs *Attributes) {
				// No changes should be made
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSet := attrs.SetAttribute(tt.tagName, tt.value)
			if gotSet != tt.wantSet {
				t.Errorf("SetAttribute() success = %v, want %v", gotSet, tt.wantSet)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, &attrs)
			}
		})
	}
}
