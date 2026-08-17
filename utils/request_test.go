package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/threatwinds/go-sdk/catcher"
)

// Upstream error ids are only adopted when they are canonical UUIDs — the
// shape catcher itself mints — so the fixtures below are real UUIDs rather
// than convenient labels. See TestSdkErrorFromResponse_MalformedErrorIDIsNotAdopted
// for the rejection half.
const (
	upstreamErrorID  = "5d08563a-9053-4e8e-ae8e-22781939a12b"
	upstreamErrorID2 = "8c2f0f5c-1c8f-4a7e-9a5f-0a1b2c3d4e5f"
)

func TestSdkErrorFromResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Header: http.Header{
			"x-error":    []string{"connection timeout"},
			"x-error-id": []string{upstreamErrorID},
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
	assert.Equal(t, upstreamErrorID, sdkErr.ErrorID)
	assert.Contains(t, sdkErr.Args, "error_id")
	assert.Equal(t, upstreamErrorID, sdkErr.Args["error_id"])

	// Code is the error's *type*, not this occurrence: an md5 of the message,
	// identical for every remote error this helper builds. It must therefore
	// be exactly what catcher itself derives for that message, and must not be
	// the upstream's id — putting a per-occurrence id in Code is the
	// conflation catcher.GinError was fixed to remove.
	assert.Equal(t, catcher.New("remote service error", nil, nil).Code, sdkErr.Code)
	assert.NotEqual(t, upstreamErrorID, sdkErr.Code)

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
			"x-error-id": []string{upstreamErrorID},
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
	assert.Equal(t, upstreamErrorID, unmarshaled.ErrorID)
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
		resp.Header.Set("x-error-id", upstreamErrorID)

		sdkErr := SdkErrorFromResponse(resp)

		assert.Equal(t, "upstream refused", *sdkErr.Cause)
		assert.Equal(t, upstreamErrorID, sdkErr.ErrorID)
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

// Msg is load-bearing twice over: catcher.SecureString returns it — and only
// it — for a status >= 500, so anything upstream-derived in Msg crosses the
// service boundary to the end client; and Code is an md5 of Msg, so anything
// per-occurrence in Msg makes Code per-occurrence too, destroying the one
// property Code exists for. Both are pinned here.
func TestSdkErrorFromResponse_MessageAndCodeAreConstant(t *testing.T) {
	first := SdkErrorFromResponse(&http.Response{
		StatusCode: 500,
		Header:     http.Header{"x-error": []string{`{"error":{"message":"pq: password authentication failed"}}`}},
	})
	second := SdkErrorFromResponse(&http.Response{
		StatusCode: 503,
		Header:     http.Header{"x-error": []string{"a completely different failure"}},
	})

	assert.Equal(t, remoteErrorMessage, first.Msg)
	assert.Equal(t, remoteErrorMessage, second.Msg)
	assert.Equal(t, first.Code, second.Code,
		"Code is the error's type; two remote failures must group under one Code")

	// The >= 500 disclosure path: SecureString is exactly what GinError writes
	// into the x-error response header and the JSON body.
	assert.Equal(t, remoteErrorMessage, first.SecureString())
	assert.NotContains(t, first.SecureString(), "password authentication failed")
}

// Adoption is the one path by which a foreign, remote-controlled value reaches
// ErrorID, a field this service then echoes in its own x-error-id header and
// every log line about the failure. Only a canonical UUID — the shape catcher
// itself mints — is accepted; anything else falls through to a freshly minted
// id, so a buggy or hostile endpoint cannot make two unrelated incidents share
// an id, nor plant a multi-megabyte string in our headers.
func TestSdkErrorFromResponse_MalformedErrorIDIsNotAdopted(t *testing.T) {
	tests := map[string]string{
		"not a uuid":       "auth-api-error-123",
		"empty":            "",
		"uuid with prefix": "id-" + upstreamErrorID,
		"braced uuid":      "{" + upstreamErrorID + "}",
		"urn uuid":         "urn:uuid:" + upstreamErrorID,
		"undashed uuid":    strings.ReplaceAll(upstreamErrorID, "-", ""),
		"oversized":        strings.Repeat("a", 1<<20),
	}

	for name, id := range tests {
		t.Run(name, func(t *testing.T) {
			resp := &http.Response{StatusCode: 500, Header: http.Header{}}
			resp.Header.Set("x-error-id", id)

			sdkErr := SdkErrorFromResponse(resp)

			assert.NotEqual(t, id, sdkErr.ErrorID, "a malformed upstream id must not be adopted")
			assert.NotContains(t, sdkErr.Args, "error_id")
			assert.NoError(t, uuid.Validate(sdkErr.ErrorID), "a fresh UUID must be minted instead")
		})
	}

	t.Run("canonical uuid is adopted", func(t *testing.T) {
		resp := &http.Response{StatusCode: 500, Header: http.Header{}}
		resp.Header.Set("x-error-id", upstreamErrorID)

		assert.Equal(t, upstreamErrorID, SdkErrorFromResponse(resp).ErrorID)
	})
}

// Args["status"] is what catcher.GinError answers the client with, so a status
// the relaying service cannot honestly send must never reach it. A 0 — from a
// hand-built response, a mock, a cached response or a RoundTripper that never
// set StatusCode — is the sharp case: gin's WriteHeader ignores a non-positive
// code, so the failure would go out as a 200 and every client-side
// `if status >= 400` check would wave it through.
func TestSdkErrorFromResponse_ClampsUnrelayableStatus(t *testing.T) {
	for _, status := range []int{0, 100, 200, 204, 304, 399, 600, 1000} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			sdkErr := SdkErrorFromResponse(&http.Response{StatusCode: status, Header: http.Header{}})

			assert.Equal(t, http.StatusInternalServerError, sdkErr.Args["status"],
				"status %d is not one this service can relay", status)
			// Severity still describes what actually came back.
			assert.Equal(t, calcSeverityFromStatus(status), sdkErr.Severity)
		})
	}

	for _, status := range []int{400, 401, 429, 500, 503, 599} {
		t.Run("relayable "+strconv.Itoa(status), func(t *testing.T) {
			sdkErr := SdkErrorFromResponse(&http.Response{StatusCode: status, Header: http.Header{}})
			assert.Equal(t, status, sdkErr.Args["status"])
		})
	}
}

func TestSdkErrorFromResponse_BoundsUpstreamDetail(t *testing.T) {
	// http.Transport.MaxResponseHeaderBytes allows 10MB of headers, and
	// whatever lands in Cause is rendered back out into the relaying service's
	// own x-error response header for any status < 500.
	huge := strings.Repeat("x", 64*1024)
	resp := &http.Response{StatusCode: 400, Header: http.Header{}}
	resp.Header.Set("x-error", huge)

	sdkErr := SdkErrorFromResponse(resp)

	assert.NotNil(t, sdkErr.Cause)
	assert.Less(t, len(*sdkErr.Cause), maxErrorDetailSize+64)
	assert.True(t, strings.HasSuffix(*sdkErr.Cause, "… (truncated)"))
}

// DoReq's non-2xx path is what lets an error id survive a service-to-service
// call: the id a downstream service (e.g. auth-api) minted and sent back in
// x-error-id must reach the caller (e.g. ai-api) so it can adopt it instead of
// minting a second, unrelated one for the same failure — without DoReq
// hijacking the error the caller builds around it.

func TestDoReq_NonOKResponse_ReturnsPlainRemoteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", upstreamErrorID)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	_, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Error(t, err)

	// Not an *SdkError: catcher.Error short-circuits on one, discarding the
	// wrapping call's message, args and status. See RemoteError's doc.
	assert.Nil(t, catcher.ToSdkError(err))

	var remote *RemoteError
	assert.True(t, errors.As(err, &remote))
	assert.Equal(t, http.StatusInternalServerError, remote.Status)
	assert.Equal(t, upstreamErrorID, remote.ID)
	assert.Equal(t, `{"error":"boom"}`, remote.Detail)
	assert.Equal(t, `error response (status=500): {"error":"boom"}`, err.Error())
}

// The point of reading x-error-id at all: the id survives into the error the
// caller builds, while everything else about that error stays the caller's.
func TestDoReq_WrapKeepsCallerContextAndInheritsUpstreamID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", upstreamErrorID)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	wrapped := catcher.New("failed to validate session with auth-api", err,
		map[string]any{"status": http.StatusBadGateway, "upstream": "auth-api"})

	// The caller's message, args and status all survive — this is the wrap
	// every service in the org already writes.
	assert.Equal(t, "failed to validate session with auth-api", wrapped.Msg)
	assert.Equal(t, http.StatusBadGateway, wrapped.Args["status"])
	assert.Equal(t, "auth-api", wrapped.Args["upstream"])
	assert.Equal(t, "CRITICAL", wrapped.Severity)

	// ...and the upstream's occurrence id is inherited rather than replaced by
	// a second, unrelated one.
	assert.Equal(t, upstreamErrorID, wrapped.ErrorID)

	// The upstream error is still reachable for a caller that wants its status.
	var remote *RemoteError
	assert.True(t, errors.As(wrapped, &remote))
	assert.Equal(t, http.StatusUnauthorized, remote.Status)
}

func TestDoReq_WrapMintsIDWhenUpstreamSuppliesNone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	assert.Equal(t, http.StatusBadGateway, status)

	wrapped := catcher.New("calling auth-api failed", err, map[string]any{"status": 500})
	assert.NoError(t, uuid.Validate(wrapped.ErrorID))
}

// An explicitly supplied error_id still wins over an inherited one.
func TestDoReq_WrapPrefersExplicitErrorIDOverInherited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", upstreamErrorID)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	wrapped := catcher.New("calling auth-api failed", err, map[string]any{"error_id": upstreamErrorID2})
	assert.Equal(t, upstreamErrorID2, wrapped.ErrorID)
}

// Code is the error's *type*: an operator grouping by it to count occurrences
// of "upstream returned 500" must get one Code, not one per upstream body.
func TestDoReq_WrapKeepsMessageAndCodeStableAcrossBodies(t *testing.T) {
	newServer := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(body))
		}))
	}

	first := newServer(`{"error":"user 1 not found"}`)
	defer first.Close()
	second := newServer(`{"error":"user 2 not found"}`)
	defer second.Close()

	_, _, errOne := DoReq[map[string]any](first.URL, nil, "GET", nil, false)
	_, _, errTwo := DoReq[map[string]any](second.URL, nil, "GET", nil, false)

	wrapOne := catcher.New("upstream call failed", errOne, map[string]any{"status": 500})
	wrapTwo := catcher.New("upstream call failed", errTwo, map[string]any{"status": 500})

	assert.Equal(t, "upstream call failed", wrapOne.Msg)
	assert.Equal(t, "upstream call failed", wrapTwo.Msg)
	assert.Equal(t, wrapOne.Code, wrapTwo.Code)
	assert.NotEqual(t, wrapOne.ErrorID, wrapTwo.ErrorID, "occurrence ids must still differ")
}

// What GinError writes into the x-error response header and the JSON body for
// a status >= 500 is exactly SecureString(), which is Msg alone. The upstream's
// body must not be in it.
func TestDoReq_WrappedErrorDoesNotDiscloseUpstreamBody(t *testing.T) {
	const secret = `{"error":{"message":"pq: password authentication failed for user \"casework\""}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error", `dial tcp 10.128.0.9:5432 connect refused`)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(secret))
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	wrapped := catcher.New("calling casework-api failed", err, map[string]any{"status": 500})

	assert.Equal(t, "calling casework-api failed", wrapped.SecureString())
	assert.NotContains(t, wrapped.SecureString(), "password authentication failed")
	assert.NotContains(t, wrapped.SecureString(), "10.128.0.9")

	// The detail is not lost, only kept off the wire: it is in Cause, which is
	// logged locally and which SecureString suppresses at >= 500.
	assert.Contains(t, *wrapped.Cause, "password authentication failed")
}

// An unbounded error body becomes an unbounded outbound header once a relaying
// service renders it, and it compounds per hop.
func TestDoReq_ErrorBodyIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat("x", 512*1024)))
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	var remote *RemoteError
	assert.True(t, errors.As(err, &remote))
	assert.Less(t, len(remote.Detail), maxErrorDetailSize+64)
	assert.True(t, strings.HasSuffix(remote.Detail, "… (truncated)"))

	// And the rendered string a caller logs or relays is bounded with it.
	assert.Less(t, len(err.Error()), maxErrorDetailSize+128)
}

// One call site must not log two different shapes depending on how the request
// failed: a wrap around a transport failure and a wrap around an HTTP failure
// keep the caller's message and args identically.
func TestDoReq_TransportAndHTTPFailuresWrapAlike(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, _, httpErr := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	// A listener that is already closed: client.Do fails before any response,
	// which is DoReq's other failure branch.
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()
	_, _, transportErr := DoReq[map[string]any](closedURL, nil, "GET", nil, false)

	assert.Error(t, httpErr)
	assert.Error(t, transportErr)

	args := func() map[string]any {
		return map[string]any{"type": "entity", "status": 500}
	}
	fromHTTP := catcher.New("sendEntity failed after retries", httpErr, args())
	fromTransport := catcher.New("sendEntity failed after retries", transportErr, args())

	assert.Equal(t, fromTransport.Msg, fromHTTP.Msg)
	assert.Equal(t, "entity", fromHTTP.Args["type"])
	assert.Equal(t, "entity", fromTransport.Args["type"])
	assert.Equal(t, 500, fromHTTP.Args["status"])
	assert.Equal(t, 500, fromTransport.Args["status"])
	assert.Equal(t, fromTransport.Code, fromHTTP.Code)
	assert.Equal(t, fromTransport.Severity, fromHTTP.Severity)
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

// The success path still reads the whole body — only the failure path is
// bounded, because only the failure path renders its body into an error string.
func TestDoReq_OKResponse_LargeBodyStillReadInFull(t *testing.T) {
	value := strings.Repeat("y", maxErrorDetailSize*4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"` + value + `"}`))
	}))
	defer server.Close()

	result, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, value, result["value"])
}

// A relaying caller that deliberately opts into the upstream's status gets it
// from SdkErrorFromResponse; DoReq's own error never imposes one.
func TestDoReq_UpstreamStatusDoesNotDictateCallerStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, status, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)
	assert.Equal(t, http.StatusUnauthorized, status)

	wrapped := catcher.New("auth check failed", err, map[string]any{"status": http.StatusInternalServerError})

	// The handler asked to answer 500; an upstream 401 must not turn that into
	// "your credentials are bad" for a client whose credentials were fine.
	assert.Equal(t, http.StatusInternalServerError, wrapped.Args["status"])
	assert.Equal(t, "ERROR", wrapped.Severity)
}

// The tests below cover the other half of the rule: a ThreatWinds peer's id is
// adopted when it sends one, and every path that has no id to adopt — because
// no response arrived, or because the endpoint is a third party that never
// sends one — generates one rather than leaving the failure unidentifiable.

// A ThreatWinds peer that sent no id: the failure still has exactly one, and
// the caller's wrap uses that one rather than minting a second.
func TestDoReq_RemoteErrorCarriesGeneratedIDWhenUpstreamSuppliesNone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	var remote *RemoteError
	assert.True(t, errors.As(err, &remote))

	// ID keeps meaning "what the upstream told us", which was nothing...
	assert.Empty(t, remote.ID)
	// ...but the failure is still identified, and identified only once.
	assert.NoError(t, uuid.Validate(remote.CatcherErrorID()))

	wrapped := catcher.New("calling auth-api failed", err, map[string]any{"status": 500})
	assert.Equal(t, remote.CatcherErrorID(), wrapped.ErrorID)
}

// An adopted id still wins over the generated fallback.
func TestDoReq_AdoptedIDBeatsGeneratedFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", upstreamErrorID)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	var remote *RemoteError
	assert.True(t, errors.As(err, &remote))
	assert.Equal(t, upstreamErrorID, remote.ID)
	assert.Equal(t, upstreamErrorID, remote.CatcherErrorID())
}

// A failure with no response at all has nothing to adopt from, and used to
// reach the caller carrying nothing to grep by.
func TestDoReq_TransportFailureCarriesGeneratedID(t *testing.T) {
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()

	_, _, err := DoReq[map[string]any](closedURL, nil, "GET", nil, false)
	assert.Error(t, err)

	var carrier catcher.ErrorIDCarrier
	assert.True(t, errors.As(err, &carrier))
	assert.NoError(t, uuid.Validate(carrier.CatcherErrorID()))

	// The id is in the rendered text too, so a caller that only logs the error
	// still has the value the caller that wraps it would log.
	assert.Contains(t, err.Error(), carrier.CatcherErrorID())

	wrapped := catcher.New("sendEntity failed", err, map[string]any{"status": 500})
	assert.Equal(t, carrier.CatcherErrorID(), wrapped.ErrorID)
}

// A body that is not the type the caller asked for is a failure of this call,
// not of the upstream, so it gets an id of its own.
func TestDoReq_UnmarshalFailureCarriesGeneratedID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	_, _, err := DoReq[map[string]any](server.URL, nil, "GET", nil, false)
	assert.Error(t, err)

	var carrier catcher.ErrorIDCarrier
	assert.True(t, errors.As(err, &carrier))
	assert.NoError(t, uuid.Validate(carrier.CatcherErrorID()))
}

// Two failures are two occurrences. An id shared between them would be worse
// than none: grepping it would return both.
func TestDoReq_DistinctFailuresGetDistinctIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, _, first := DoReq[map[string]any](server.URL, nil, "GET", nil, false)
	_, _, second := DoReq[map[string]any](server.URL, nil, "GET", nil, false)

	var a, b catcher.ErrorIDCarrier
	assert.True(t, errors.As(first, &a))
	assert.True(t, errors.As(second, &b))
	assert.NotEqual(t, a.CatcherErrorID(), b.CatcherErrorID())
}

// The original error stays reachable underneath the id wrapper, so a caller
// matching on a sentinel or a concrete transport type is unaffected.
func TestGeneratedIDErrorKeepsTheChainWalkable(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := withGeneratedErrorID(fmt.Errorf("context: %w", sentinel))

	assert.True(t, errors.Is(err, sentinel))
	assert.Contains(t, err.Error(), "context: sentinel")
}

// Nil in every position: a nil error must not become a non-nil one, and a
// typed-nil wrapper reached by errors.As must not panic.
func TestGeneratedIDErrorNilSafety(t *testing.T) {
	assert.Nil(t, withGeneratedErrorID(nil))

	var nilWrapper *generatedIDError
	assert.NotPanics(t, func() {
		assert.Equal(t, "<nil>", nilWrapper.Error())
		assert.Nil(t, nilWrapper.Unwrap())
		assert.Empty(t, nilWrapper.CatcherErrorID())
	})

	// A wrapper around a nil error still renders, rather than dereferencing it.
	empty := &generatedIDError{id: upstreamErrorID}
	assert.Contains(t, empty.Error(), upstreamErrorID)

	// And catcher survives being handed the typed nil as a cause.
	assert.NotPanics(t, func() {
		assert.NoError(t, uuid.Validate(catcher.New("typed nil cause", nilWrapper, nil).ErrorID))
	})
}

// DownloadStream targets arbitrary third-party URLs. It must not read
// x-error-id off one: catcher.GinError is what emits that header, and nothing
// outside this org runs it, so adopting it would let any endpoint name one of
// this org's occurrences.
func TestDownloadStream_DoesNotAdoptThirdPartyErrorID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", upstreamErrorID)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := DownloadStream(server.URL)
	assert.Error(t, err)

	var carrier catcher.ErrorIDCarrier
	assert.True(t, errors.As(err, &carrier))

	// Generated, not adopted.
	assert.NotEqual(t, upstreamErrorID, carrier.CatcherErrorID())
	assert.NoError(t, uuid.Validate(carrier.CatcherErrorID()))
	assert.NotContains(t, err.Error(), upstreamErrorID)

	// Still traceable: the id is in the text and inherited by a catcher wrap.
	assert.Contains(t, err.Error(), carrier.CatcherErrorID())
	wrapped := catcher.New("failed to download feed", err, map[string]any{"status": 502})
	assert.Equal(t, carrier.CatcherErrorID(), wrapped.ErrorID)
	assert.Equal(t, "failed to download feed", wrapped.Msg)
}

func TestDownloadStream_TransportFailureCarriesGeneratedID(t *testing.T) {
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()

	_, err := DownloadStream(closedURL)
	assert.Error(t, err)

	var carrier catcher.ErrorIDCarrier
	assert.True(t, errors.As(err, &carrier))
	assert.NoError(t, uuid.Validate(carrier.CatcherErrorID()))
}

// Download returns DownloadStream's error unchanged, so one failed download is
// one occurrence, not two.
func TestDownload_PropagatesTheStreamsID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	err := Download(server.URL, filepath.Join(t.TempDir(), "out.bin"))
	assert.Error(t, err)

	var carrier catcher.ErrorIDCarrier
	assert.True(t, errors.As(err, &carrier))
	assert.NoError(t, uuid.Validate(carrier.CatcherErrorID()))
	assert.Contains(t, err.Error(), carrier.CatcherErrorID())
}

// CheckConnectivity probes an arbitrary URL, so the same rule applies as for
// DownloadStream.
func TestCheckConnectivity_GeneratesIDAndDoesNotAdopt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-error-id", upstreamErrorID)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := CheckConnectivity(server.URL, 5*time.Second)
	assert.Error(t, err)

	var carrier catcher.ErrorIDCarrier
	assert.True(t, errors.As(err, &carrier))
	assert.NotEqual(t, upstreamErrorID, carrier.CatcherErrorID())
	assert.NoError(t, uuid.Validate(carrier.CatcherErrorID()))

	wrapped := catcher.New("upstream unreachable", err, map[string]any{"status": 502})
	assert.Equal(t, carrier.CatcherErrorID(), wrapped.ErrorID)
}

func TestCheckConnectivity_SuccessIsStillNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	assert.NoError(t, CheckConnectivity(server.URL, 5*time.Second))
}

// One response yields one occurrence id, whichever of the two error types is
// derived from it.
func TestOneResponseYieldsOneErrorID(t *testing.T) {
	t.Run("upstream supplied none", func(t *testing.T) {
		resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{}}

		remote := remoteErrorFrom(resp, nil)
		sdkErr := sdkErrorFrom(remote)

		assert.Empty(t, remote.ID)
		assert.NoError(t, uuid.Validate(remote.CatcherErrorID()))
		assert.Equal(t, remote.CatcherErrorID(), sdkErr.ErrorID)
		// Args records only what the upstream actually told us, which was
		// nothing — the generated id is this process's, not the upstream's.
		assert.NotContains(t, sdkErr.Args, "error_id")
	})

	t.Run("upstream supplied one", func(t *testing.T) {
		header := http.Header{}
		header.Set("x-error-id", upstreamErrorID)
		resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: header}

		remote := remoteErrorFrom(resp, nil)
		sdkErr := sdkErrorFrom(remote)

		assert.Equal(t, upstreamErrorID, remote.ID)
		assert.Equal(t, upstreamErrorID, sdkErr.ErrorID)
		assert.Equal(t, upstreamErrorID, sdkErr.Args["error_id"])
	})
}
