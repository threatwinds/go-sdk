package relay

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
)

// Upstream error ids are only adopted when they are canonical UUIDs — the shape
// catcher itself mints — so the fixtures below are real UUIDs rather than
// convenient labels. See TestSdkErrorFromResponse_MalformedErrorIDIsNotAdopted
// for the rejection half.
const upstreamErrorID = "5d08563a-9053-4e8e-ae8e-22781939a12b"

// utilsMaxErrorDetailSize mirrors the bound utils applies to upstream-supplied
// text (its unexported maxErrorDetailSize). The assertions below only need to
// know that the detail is bounded near it, not to be exact.
const utilsMaxErrorDetailSize = 4 * 1024

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
	assert.Less(t, len(*sdkErr.Cause), utilsMaxErrorDetailSize+64)
	assert.True(t, strings.HasSuffix(*sdkErr.Cause, "… (truncated)"))
}

// One response yields one occurrence id, whichever of the two error types is
// derived from it.
func TestOneResponseYieldsOneErrorID(t *testing.T) {
	t.Run("upstream supplied none", func(t *testing.T) {
		resp := &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{}}

		remote := utils.RemoteErrorFromResponse(resp)
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

		remote := utils.RemoteErrorFromResponse(resp)
		sdkErr := sdkErrorFrom(remote)

		assert.Equal(t, upstreamErrorID, remote.ID)
		assert.Equal(t, upstreamErrorID, sdkErr.ErrorID)
		assert.Equal(t, upstreamErrorID, sdkErr.Args["error_id"])
	})
}
