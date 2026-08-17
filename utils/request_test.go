package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	assert.Equal(t, "remote service error", sdkErr.Msg)
	assert.NotNil(t, sdkErr.Cause)
	assert.Equal(t, "connection timeout", *sdkErr.Cause)
	assert.Equal(t, "ERROR", sdkErr.Severity)
	assert.Nil(t, sdkErr.Trace)
	assert.Contains(t, sdkErr.Args, "status")
	assert.Equal(t, 500, sdkErr.Args["status"])
	assert.NotEmpty(t, sdkErr.Timestamp)

	// The upstream's x-error-id is an *occurrence* id, so it lands in ErrorID
	// — the field an operator greps across every service's logs to reconstruct
	// one failure — and in Args["error_id"], the key catcher.build reserves
	// for exactly this adoption.
	assert.Equal(t, "abc123def456", sdkErr.ErrorID)
	assert.Contains(t, sdkErr.Args, "error_id")
	assert.Equal(t, "abc123def456", sdkErr.Args["error_id"])

	// Code is the error's *type*, not this occurrence: an md5 of the message,
	// identical for every remote error this helper builds. It must therefore
	// be exactly what catcher itself derives for that message, and must not be
	// the upstream's id — putting a per-occurrence id in Code is the
	// conflation catcher.GinError was fixed to remove.
	assert.Equal(t, catcher.New("remote service error", nil, nil).Code, sdkErr.Code)
	assert.NotEqual(t, "abc123def456", sdkErr.Code)

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

	// No id to adopt, so catcher mints one — every SdkError has an occurrence
	// id whether or not an upstream supplied it — and nothing is left in
	// Args under the reserved key to suggest otherwise.
	assert.NotEmpty(t, sdkErr.ErrorID)
	assert.NotContains(t, sdkErr.Args, "error_id")
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

	// x-error carries a human message, x-error-id an id. With only the former
	// present there is no id to adopt, so one is minted: an occurrence id is
	// never derived from error text — two unrelated services both answering
	// "service unavailable" would otherwise share an id and appear, to anyone
	// grepping it, to be one failure.
	assert.NotEmpty(t, sdkErr.ErrorID)
	assert.NotEqual(t, "service unavailable", sdkErr.ErrorID)
	assert.NotContains(t, sdkErr.Args, "error_id")
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
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
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
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
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

	// The occurrence id has to survive the wire too — it is the value the next
	// service re-raises and an operator quotes from a support ticket.
	assert.Equal(t, "test-code-99", unmarshaled.ErrorID)
}

func TestSdkErrorFromResponse_NilSafety(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		sdkErr := SdkErrorFromResponse(nil)

		assert.NotNil(t, sdkErr)
		assert.Equal(t, "remote service error", sdkErr.Msg)
		assert.Equal(t, "unknown cause", *sdkErr.Cause)
		// No response at all is a local failure; 500 is what catcher would
		// default to anyway, and keeps GinError from writing status 0.
		assert.Equal(t, http.StatusInternalServerError, sdkErr.Args["status"])
		assert.Equal(t, "ERROR", sdkErr.Severity)
		assert.NotEmpty(t, sdkErr.ErrorID)
	})

	t.Run("nil header map", func(t *testing.T) {
		sdkErr := SdkErrorFromResponse(&http.Response{StatusCode: 404})

		assert.NotNil(t, sdkErr)
		assert.Equal(t, "unknown cause", *sdkErr.Cause)
		assert.Equal(t, 404, sdkErr.Args["status"])
		assert.Equal(t, "WARNING", sdkErr.Severity)
		assert.NotEmpty(t, sdkErr.ErrorID)
	})

	t.Run("canonical header keys", func(t *testing.T) {
		// What a response that came off the wire actually carries: net/http
		// canonicalizes on read, so the keys are X-Error / X-Error-Id.
		resp := &http.Response{StatusCode: 502, Header: http.Header{}}
		resp.Header.Set("x-error", "upstream refused")
		resp.Header.Set("x-error-id", "canonical-id-7")

		sdkErr := SdkErrorFromResponse(resp)

		assert.Equal(t, "upstream refused", *sdkErr.Cause)
		assert.Equal(t, "canonical-id-7", sdkErr.ErrorID)
		assert.Equal(t, "CRITICAL", sdkErr.Severity)
	})
}

func TestSdkErrorFromResponse_DoesNotReadBody(t *testing.T) {
	// The body may already be consumed, or nil on a hand-built response; it is
	// the caller's to read and to close. Reading a nil Body here would panic.
	resp := &http.Response{StatusCode: 500, Header: make(http.Header)}

	assert.NotPanics(t, func() { _ = SdkErrorFromResponse(resp) })
	assert.Nil(t, resp.Body)
}

// DoReq's non-2xx path is what lets an error id survive a service-to-service
// call: the id a downstream service (e.g. auth-api) minted and sent back in
// x-error-id must come out the other end of DoReq attached to the
// *catcher.SdkError it returns, so the caller (e.g. ai-api) can adopt it
// instead of minting a second, unrelated one for the same failure.

func TestDoReq_NonOKResponse_AdoptsUpstreamErrorID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", "auth-api-error-123")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	_, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Error(t, err)

	sdkErr := catcher.ToSdkError(err)
	assert.NotNil(t, sdkErr)
	assert.Equal(t, "auth-api-error-123", sdkErr.ErrorID)
}

func TestDoReq_NonOKResponse_MissingErrorIDHeader_StillGeneratesOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	assert.Equal(t, http.StatusBadGateway, status)
	sdkErr := catcher.ToSdkError(err)
	assert.NotNil(t, sdkErr)
	assert.NotEmpty(t, sdkErr.ErrorID)
}

func TestDoReq_NonOKResponse_ErrorIDSurvivesCatcherErrorShortCircuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", "downstream-id-456")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)
	downstream := catcher.ToSdkError(err)

	// Mirrors the real call site: a caller wraps DoReq's error with
	// catcher.Error, which short-circuits on an *SdkError cause and hands
	// the exact same error back, id and all, rather than minting a new one.
	wrapped := catcher.Error("calling auth-api failed", err, map[string]any{"status": 999})

	assert.Same(t, downstream, wrapped)
	assert.Equal(t, "downstream-id-456", wrapped.ErrorID)
}

func TestDoReq_OKResponse_Unaffected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", "should-be-ignored")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	result, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, result["ok"])
}

func TestDoReq_NonOKResponse_StatusReachesArgsForSeverity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	_, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	sdkErr := catcher.ToSdkError(err)
	assert.NotNil(t, sdkErr)
	assert.Equal(t, status, sdkErr.Args["status"])
	assert.Equal(t, "WARNING", sdkErr.Severity)
}
