package utils

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/threatwinds/go-sdk/catcher"
)

// Upstream error ids are only adopted when they are canonical UUIDs — the
// shape catcher itself mints — so the fixtures below are real UUIDs rather
// than convenient labels. See TestAdoptErrorIDRejectsWhatCatcherRejects for the
// rejection half.
const (
	upstreamErrorID  = "5d08563a-9053-4e8e-ae8e-22781939a12b"
	upstreamErrorID2 = "8c2f0f5c-1c8f-4a7e-9a5f-0a1b2c3d4e5f"
)

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


// The two error types this package returns satisfy catcher.ErrorIDCarrier
// structurally — the interface is one method, and satisfying it requires
// declaring that method, not importing the package that declares it. The
// assertion lives here rather than beside the types so that the production
// build of this package keeps importing nothing from catcher (see adoptErrorID
// for what that import would cost every binary that links utils), while a
// rename or a signature change in catcher still fails a test here rather than
// silently stopping these types donating their ids.
var (
	_ catcher.ErrorIDCarrier = (*RemoteError)(nil)
	_ catcher.ErrorIDCarrier = (*generatedIDError)(nil)
)

// adoptErrorID is a copy of catcher.AdoptErrorID, kept out of the production
// build for the same reason. Copies drift; this is what stops that.
func TestAdoptErrorIDMatchesCatcher(t *testing.T) {
	canonical := uuid.NewString()

	for _, in := range []string{
		canonical,
		"",
		"auth-api-error-123",
		"c72b9698fa1927e1dd12d3cf26ed84b2",
		"id-" + upstreamErrorID,
		"{" + upstreamErrorID + "}",
		"urn:uuid:" + upstreamErrorID,
		strings.ReplaceAll(upstreamErrorID, "-", ""),
		strings.Repeat("z", 36),
		strings.Repeat("a", 1<<20),
		canonical + " ",
		"abc\r\nX-Injected: 1",
	} {
		assert.Equal(t, catcher.AdoptErrorID(in), adoptErrorID(in),
			"adoptErrorID(%.40q) must match catcher.AdoptErrorID", in)
	}
}

// The rejection half of the adoption rule, at this package's own boundary: a
// malformed id is not reported as the upstream's, and the failure still carries
// exactly one locally minted id.
func TestAdoptErrorIDRejectsWhatCatcherRejects(t *testing.T) {
	for name, id := range map[string]string{
		"not a uuid":       "auth-api-error-123",
		"empty":            "",
		"uuid with prefix": "id-" + upstreamErrorID,
		"braced uuid":      "{" + upstreamErrorID + "}",
		"urn uuid":         "urn:uuid:" + upstreamErrorID,
		"undashed uuid":    strings.ReplaceAll(upstreamErrorID, "-", ""),
		"oversized":        strings.Repeat("a", 1<<20),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &http.Response{StatusCode: 500, Header: http.Header{}}
			resp.Header.Set("x-error-id", id)

			remote := RemoteErrorFromResponse(resp)

			assert.Empty(t, remote.ID, "a malformed upstream id must not be reported as adopted")
			assert.NoError(t, uuid.Validate(remote.CatcherErrorID()), "a fresh UUID must be minted instead")
		})
	}

	t.Run("canonical uuid is adopted", func(t *testing.T) {
		resp := &http.Response{StatusCode: 500, Header: http.Header{}}
		resp.Header.Set("x-error-id", upstreamErrorID)

		remote := RemoteErrorFromResponse(resp)

		assert.Equal(t, upstreamErrorID, remote.ID)
		assert.Equal(t, upstreamErrorID, remote.CatcherErrorID())
	})
}

// RemoteErrorFromResponse is the entry point relay.SdkErrorFromResponse builds
// on, and the one that must not touch a body it does not own.
func TestRemoteErrorFromResponse_NilSafetyAndBody(t *testing.T) {
	assert.NotPanics(t, func() {
		remote := RemoteErrorFromResponse(nil)
		assert.Equal(t, http.StatusInternalServerError, remote.Status)
		assert.NoError(t, uuid.Validate(remote.CatcherErrorID()))
	})

	assert.NotPanics(t, func() {
		remote := RemoteErrorFromResponse(&http.Response{StatusCode: 404})
		assert.Equal(t, 404, remote.Status)
		assert.Empty(t, remote.Detail)
	})

	resp := &http.Response{StatusCode: 500, Header: http.Header{}}
	resp.Header.Set("x-error", "upstream refused")

	assert.NotPanics(t, func() {
		assert.Equal(t, "upstream refused", RemoteErrorFromResponse(resp).Detail)
	})
	assert.Nil(t, resp.Body)
}
