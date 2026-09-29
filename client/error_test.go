package client

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/threatwinds/go-sdk/catcher"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	err := newAPIError("GET", "/test", http.StatusNotFound, "not found", "err-123", "", []byte(`{}`))
	got := err.Error()
	if got != "404: GET /test: not found" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestAPIError_IsMethods(t *testing.T) {
	tests := []struct {
		status int
		fn     func(*APIError) bool
		name   string
	}{
		{400, (*APIError).IsValidationError, "validation"},
		{401, (*APIError).IsUnauthorized, "unauthorized"},
		{403, (*APIError).IsForbidden, "forbidden"},
		{404, (*APIError).IsNotFound, "not found"},
		{429, (*APIError).IsRateLimited, "rate limited"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newAPIError("GET", "/", tc.status, "msg", "id", "", nil)
			if !tc.fn(err) {
				t.Errorf("Is*() should return true for %d", tc.status)
			}
		})
	}
}

func TestAPIError_IsMethods_FalseForOtherStatus(t *testing.T) {
	err := newAPIError("GET", "/", http.StatusInternalServerError, "msg", "id", "", nil)
	if err.IsNotFound() || err.IsUnauthorized() || err.IsForbidden() || err.IsRateLimited() || err.IsValidationError() {
		t.Error("all Is*() should be false for 500")
	}
}

func TestAPIError_Fields(t *testing.T) {
	body := []byte(`{"detail":"test"}`)
	err := newAPIError("DELETE", "/item/5", http.StatusForbidden, "forbidden", "err-42", "", body)
	if err.StatusCode != 403 {
		t.Error("status wrong")
	}
	if err.Message != "forbidden" {
		t.Error("message wrong")
	}
	if err.ErrorID != "err-42" {
		t.Error("errorID wrong")
	}
	if string(err.Body) != string(body) {
		t.Error("body wrong")
	}
}

func TestSDKError(t *testing.T) {
	e := newSDKErr("test message")
	if e.Error() != "client: test message" {
		t.Errorf("unexpected: %q", e.Error())
	}
}

func TestAPIError_RetryAfter(t *testing.T) {
	err := newAPIError("GET", "/", 429, "rate limited", "id", "30", nil)
	if err.RetryAfter() != "30" {
		t.Errorf("RetryAfter() = %q, want %q", err.RetryAfter(), "30")
	}
	err2 := newAPIError("GET", "/", 429, "rate limited", "id", "", nil)
	if err2.RetryAfter() != "" {
		t.Errorf("RetryAfter() should be empty when not set")
	}
}

// The tests below cover error-id propagation: the id an upstream ThreatWinds
// service sends in x-error-id has to survive into whatever error the calling
// service builds, or one failure ends up with two unrelated ids and the two
// services' log lines cannot be correlated.

const (
	upstreamErrorID  = "5d08563a-9053-4e8e-ae8e-22781939a12b"
	upstreamErrorID2 = "8c2f0f5c-1c8f-4a7e-9a5f-0a1b2c3d4e5f"
)

func TestAPIError_DonatesUpstreamErrorID(t *testing.T) {
	err := newAPIError("GET", "/api/auth/v1/session", 401, "unauthorized", upstreamErrorID, "", nil)

	if got := err.CatcherErrorID(); got != upstreamErrorID {
		t.Errorf("expected the upstream id to be donated, got %q", got)
	}

	// The interface is what catcher's errors.As walk looks for.
	var carrier catcher.ErrorIDCarrier
	if !errors.As(error(err), &carrier) {
		t.Fatal("expected *APIError to satisfy catcher.ErrorIDCarrier")
	}
	if carrier.CatcherErrorID() != upstreamErrorID {
		t.Errorf("expected %q through the interface, got %q", upstreamErrorID, carrier.CatcherErrorID())
	}
}

// The wrap every service writes: its own message, its own args, its own status
// — and the upstream's occurrence id rather than a second one.
func TestAPIError_WrapInheritsUpstreamErrorID(t *testing.T) {
	err := newAPIError("GET", "/api/auth/v1/session", 401, "unauthorized", upstreamErrorID, "", nil)

	wrapped := catcher.New("failed to validate session with auth-api", err,
		map[string]any{"status": http.StatusBadGateway, "upstream": "auth-api"})

	if wrapped.ErrorID != upstreamErrorID {
		t.Errorf("expected the id to be inherited as %q, got %q", upstreamErrorID, wrapped.ErrorID)
	}
	if wrapped.Msg != "failed to validate session with auth-api" {
		t.Errorf("expected the caller's msg, got %q", wrapped.Msg)
	}
	if wrapped.Args["status"] != http.StatusBadGateway || wrapped.Args["upstream"] != "auth-api" {
		t.Errorf("expected the caller's args, got %v", wrapped.Args)
	}

	// The upstream error is still reachable for a caller that wants its status.
	var apiErr *APIError
	if !errors.As(error(wrapped), &apiErr) || apiErr.StatusCode != 401 {
		t.Error("expected the *APIError to stay reachable through the wrap")
	}
}

// Wrapping the wrap hits catcher's short-circuit, which returns the existing
// error by identity. The id must not change there either.
func TestAPIError_ErrorIDSurvivesShortCircuit(t *testing.T) {
	err := newAPIError("GET", "/api/billing/v1/quota", 503, "unavailable", upstreamErrorID, "", nil)

	inner := catcher.New("calling billing-api failed", err, map[string]any{"status": 502})
	outer := catcher.New("handler failed", inner, map[string]any{"status": 500})

	if inner.ErrorID != upstreamErrorID || outer.ErrorID != upstreamErrorID {
		t.Errorf("expected %q throughout, got %q then %q", upstreamErrorID, inner.ErrorID, outer.ErrorID)
	}
}

// Nothing to adopt: the failure names itself, so it is still traceable — it
// just traces to this service only, which is the truth. What it must not do is
// leave every wrap to mint its own id, which is what returning "" here did.
func TestAPIError_MintsItsOwnIDWhenUpstreamSuppliesNone(t *testing.T) {
	err := newAPIError("GET", "/api/auth/v1/session", 500, "boom", "", "", nil)

	donated := err.CatcherErrorID()
	if uuid.Validate(donated) != nil {
		t.Fatalf("expected a locally minted UUID to donate, got %q", donated)
	}
	if err.ErrorID != "" {
		t.Errorf("the field must keep reporting that the server sent nothing, got %q", err.ErrorID)
	}

	// One failure, wrapped at two layers — a handler building its own response
	// error while a middleware or caller wraps the same value — is one id.
	first := catcher.New("calling auth-api failed", err, map[string]any{"status": 500})
	second := catcher.New("session validation failed", err, map[string]any{"status": 502})

	if first.ErrorID != donated || second.ErrorID != donated {
		t.Errorf("expected both wraps to carry %q, got %q and %q", donated, first.ErrorID, second.ErrorID)
	}
}

// Two separate failures must not share an id, however they were minted.
func TestAPIError_LocalIDsAreOnePerFailure(t *testing.T) {
	first := newAPIError("GET", "/", 500, "boom", "", "", nil)
	second := newAPIError("GET", "/", 500, "boom", "", "", nil)

	if first.CatcherErrorID() == second.CatcherErrorID() {
		t.Errorf("expected different ids for two failures, both got %q", first.CatcherErrorID())
	}
}

// An id that is not the shape catcher mints is not adopted: it would either
// collide with an unrelated in-flight failure or blow up a log line. The field
// still reports what the server actually sent, and the failure still carries
// one id of its own.
func TestAPIError_MalformedUpstreamIDIsNotAdopted(t *testing.T) {
	for _, malformed := range []string{
		"c72b9698fa1927e1dd12d3cf26ed84b2",              // an md5 Code, as ai-api sends today
		"{5d08563a-9053-4e8e-ae8e-22781939a12b}",        // braced form
		"urn:uuid:5d08563a-9053-4e8e-ae8e-22781939a12b", // urn form
		strings.Repeat("a", 64*1024),                    // an id megabytes long
	} {
		err := newAPIError("GET", "/", 500, "boom", malformed, "", nil)

		donated := err.CatcherErrorID()
		if donated == malformed {
			t.Errorf("expected %.20q not to be donated", malformed)
		}
		if uuid.Validate(donated) != nil {
			t.Errorf("expected a locally minted UUID for %.20q, got %q", malformed, donated)
		}
		if err.ErrorID != malformed {
			t.Error("the field must still report what the server sent")
		}

		wrapped := catcher.New("calling upstream failed", err, nil)
		if wrapped.ErrorID != donated {
			t.Errorf("expected the wrap to carry %q for %.20q, got %q", donated, malformed, wrapped.ErrorID)
		}
	}
}

// The local guard is a copy of catcher's, kept out of the production build so
// this package links nothing from the SDK (see adoptErrorID's doc comment).
// Copies drift; this is what stops that.
func TestAdoptErrorIDMatchesCatcher(t *testing.T) {
	canonical := uuid.NewString()

	for _, in := range []string{
		canonical,
		"",
		"c72b9698fa1927e1dd12d3cf26ed84b2",
		"{5d08563a-9053-4e8e-ae8e-22781939a12b}",
		"urn:uuid:5d08563a-9053-4e8e-ae8e-22781939a12b",
		"5d08563a90534e8eae8e22781939a12b",
		strings.Repeat("z", 36),
		strings.Repeat("a", 64*1024),
		canonical + " ",
		"abc\r\nX-Injected: 1",
	} {
		if got, want := adoptErrorID(in), catcher.AdoptErrorID(in); got != want {
			t.Errorf("adoptErrorID(%.40q) = %q, catcher.AdoptErrorID = %q", in, got, want)
		}
	}
}

// The interface this package satisfies structurally, asserted where linking
// catcher costs nothing: a rename or a signature change in catcher must fail a
// test here rather than silently stop *APIError donating its id.
var _ catcher.ErrorIDCarrier = (*APIError)(nil)

// An explicitly supplied error_id still wins over an inherited one.
func TestAPIError_ExplicitErrorIDBeatsInherited(t *testing.T) {
	err := newAPIError("GET", "/", 500, "boom", upstreamErrorID, "", nil)

	wrapped := catcher.New("calling upstream failed", err, map[string]any{"error_id": upstreamErrorID2})
	if wrapped.ErrorID != upstreamErrorID2 {
		t.Errorf("expected the explicit id to win, got %q", wrapped.ErrorID)
	}
}

// errors.As matches on type alone, so a typed-nil *APIError reaches this method
// on a nil receiver.
func TestAPIError_CatcherErrorIDNilSafe(t *testing.T) {
	var nilErr *APIError

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a typed-nil *APIError must not panic: %v", r)
		}
	}()

	if got := nilErr.CatcherErrorID(); got != "" {
		t.Errorf("expected an empty id, got %q", got)
	}
	if got := catcher.New("typed nil cause", nilErr, nil).ErrorID; uuid.Validate(got) != nil {
		t.Errorf("expected a generated UUID, got %q", got)
	}
}

// SDKError is raised before any request, so it has no id to carry and must not
// claim otherwise.
func TestSDKError_IsNotAnErrorIDCarrier(t *testing.T) {
	var carrier catcher.ErrorIDCarrier
	if errors.As(error(newSDKErr("authentication required")), &carrier) {
		t.Error("SDKError must not advertise itself as carrying an upstream id")
	}
}
