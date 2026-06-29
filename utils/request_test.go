package utils

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threatwinds/go-sdk/catcher"
)

func TestSdkErrorFromResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Header: http.Header{
			"x-error":    []string{"connection timeout"},
			"x-error-id": []string{"abc123def456"},
		},
	}

	sdkErr := SdkErrorFromResponse(resp)

	assert.NotNil(t, sdkErr)
	assert.Equal(t, "abc123def456", sdkErr.Code)
	assert.Equal(t, "remote service error", sdkErr.Msg)
	assert.NotNil(t, sdkErr.Cause)
	assert.Equal(t, "connection timeout", *sdkErr.Cause)
	assert.Equal(t, "ERROR", sdkErr.Severity)
	assert.Nil(t, sdkErr.Trace)
	assert.Contains(t, sdkErr.Args, "status")
	assert.Equal(t, float64(500), sdkErr.Args["status"])
	assert.Contains(t, sdkErr.Args, "error_code")
	assert.Equal(t, "abc123def456", sdkErr.Args["error_code"])
	assert.NotEmpty(t, sdkErr.Timestamp)

	// Verify it implements error via ToSdkError
	assert.NotNil(t, catcher.ToSdkError(sdkErr))
}

func TestSdkErrorFromResponse_MissingHeaders(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Header:     make(http.Header),
	}

	sdkErr := SdkErrorFromResponse(resp)

	assert.NotNil(t, sdkErr)
	assert.Equal(t, "remote service error", sdkErr.Msg)
	assert.NotNil(t, sdkErr.Cause)
	assert.Equal(t, "unknown cause", *sdkErr.Cause)
	assert.Equal(t, "ERROR", sdkErr.Severity)
	assert.Nil(t, sdkErr.Trace)
	assert.NotEmpty(t, sdkErr.Code)
}

func TestSdkErrorFromResponse_PartialHeaders(t *testing.T) {
	resp := &http.Response{
		StatusCode: 503,
		Header: http.Header{
			"x-error": []string{"service unavailable"},
		},
	}

	sdkErr := SdkErrorFromResponse(resp)

	assert.NotNil(t, sdkErr)
	assert.Equal(t, "service unavailable", *sdkErr.Cause)
	assert.Equal(t, "CRITICAL", sdkErr.Severity)
	assert.NotEmpty(t, sdkErr.Code)
	assert.Equal(t, "service unavailable", sdkErr.Args["error_code"])
}

func TestSdkErrorFromResponse_SeverityMapping(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{100, "DEBUG"},
		{200, "INFO"},
		{301, "NOTICE"},
		{400, "WARNING"},
		{404, "WARNING"},
		{500, "ERROR"},
		{501, "ERROR"},
		{502, "CRITICAL"},
		{503, "CRITICAL"},
		{508, "CRITICAL"},
		{509, "ALERT"},
		{510, "ALERT"},
		{600, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.status,
				Header:     make(http.Header),
			}
			sdkErr := SdkErrorFromResponse(resp)
			assert.Equal(t, tt.expected, sdkErr.Severity, "status %d should map to %s", tt.status, tt.expected)
		})
	}
}

func TestCalcSeverityFromStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{100, "DEBUG"},
		{199, "DEBUG"},
		{200, "INFO"},
		{299, "INFO"},
		{300, "NOTICE"},
		{399, "NOTICE"},
		{400, "WARNING"},
		{499, "WARNING"},
		{500, "ERROR"},
		{501, "ERROR"},
		{502, "CRITICAL"},
		{503, "CRITICAL"},
		{508, "CRITICAL"},
		{509, "ALERT"},
		{510, "ALERT"},
		{999, "ERROR"},
		{0, "ERROR"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := calcSeverityFromStatus(tt.status)
			assert.Equal(t, tt.expected, result, "calcSeverityFromStatus(%d) = %s, want %s", tt.status, result, tt.expected)
		})
	}
}

func TestSdkErrorFromResponse_JsonMarshallable(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Header: http.Header{
			"x-error":    []string{"db connection failed"},
			"x-error-id": []string{"test-code-99"},
		},
	}

	sdkErr := SdkErrorFromResponse(resp)

	// SdkError should be JSON marshalable
	jBytes, err := json.Marshal(sdkErr)
	assert.NoError(t, err)
	assert.NotEmpty(t, jBytes)

	var unmarshaled catcher.SdkError
	err = json.Unmarshal(jBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, sdkErr.Code, unmarshaled.Code)
}
