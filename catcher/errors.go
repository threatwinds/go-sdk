package catcher

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

// errorParam represents a single error field in the JSON response.
type errorParam struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	ErrorID string `json:"error_id"`
	Param   string `json:"param,omitempty"`
}

// errorResponse is the JSON envelope returned to the client on error.
type errorResponse struct {
	Error errorParam `json:"error"`
}

// SdkError is a struct that implements the Go error interface.
type SdkError struct {
	Timestamp string `json:"timestamp"`
	Code      string `json:"code"`
	// ErrorID identifies this error *occurrence*, as opposed to Code, which
	// identifies the error's *type* (an md5 of the message, shared by every
	// occurrence of the same message anywhere). ErrorID is meant to be
	// carried unchanged by every service that handles the same failure as it
	// propagates — gateway to ai-api to auth-api, say — so an operator can
	// grep one value across every service's logs and find every log line
	// that pertains to one specific failure. See build's args["error_id"]
	// handling for how it is populated, and the short-circuit branch of
	// build for why it stays fixed once set.
	ErrorID  string         `json:"error_id"`
	Trace    []string       `json:"trace,omitempty"`
	Msg      string         `json:"msg"`
	Cause    *string        `json:"cause,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
	Severity string         `json:"severity"`

	// cause holds the original error this SdkError was built from, so that
	// Unwrap can expose it to errors.Is/errors.As. It is deliberately
	// unexported: the wire format (JSON tags above) is unchanged, so every
	// existing consumer that reads .Cause as a string, or round-trips an
	// SdkError through JSON, keeps working exactly as before. An SdkError
	// decoded from JSON (e.g. received from another service) has a nil
	// cause and Unwraps to nil, which is correct — the original error value
	// was never transmitted, only its rendered text in Cause.
	cause error

	// logged records whether Log has already emitted this error's line, so
	// a second call — from Error's own construction-time log, an explicit
	// Log() call at a handling boundary, or GinError's boundary log — is a
	// no-op. Deliberately unexported for the same reason as cause: the wire
	// format above is unchanged, and an SdkError decoded from JSON has a nil
	// logged pointer regardless of whether the original was ever logged,
	// which is fine — decoding never runs Log. See Log's doc comment for
	// what a nil pointer means for that case.
	//
	// This is a *atomic.Bool, not a plain bool. SdkError is handed around
	// and returned by value-receiver methods (Error, JSON, SecureString,
	// GinError) on purpose — those signatures cannot change without
	// breaking every existing caller — which means the struct is copied
	// constantly: GinError(c) takes a copy to satisfy its value receiver,
	// dereferencing a *SdkError to log a %v of it copies, storing an
	// SdkError (not *SdkError) in a slice copies. A plain bool copies with
	// the struct, so each copy's flag is independent and "already logged"
	// stops being true anywhere else the moment the struct is duplicated —
	// that independence is the bug this field exists to close. A pointer
	// field copies too, but copying a pointer does not copy its pointee:
	// every copy of the struct still shares the one flag the pointer refers
	// to, so a value-receiver method can genuinely observe and flip
	// "logged" for every other copy in existence, not just its own. The
	// pointee also needs its own synchronization, because two goroutines
	// racing to log the same *SdkError is a real, intended use (see
	// TestLogConcurrentGoroutinesLogOnce). An embedded, by-value sync.Once
	// would also serialize the check-and-set and would also be shared
	// across copies for the same reason a pointer is — but embedding a lock
	// type directly in SdkError trips go vet's copylocks check the moment
	// any value-receiver method below (Error, JSON, SecureString, GinError)
	// copies the struct; confirmed by running `go vet ./catcher/...` against
	// that version rather than assuming. A *pointer* to a lock does not
	// trip copylocks, because copylocks only inspects values that directly
	// embed a Locker-shaped type — a pointer field, whatever it points to,
	// is just a pointer. So the field has to be a pointer either way,
	// leaving the choice of what it points to: *atomic.Bool over *sync.Once
	// because the operation being guarded is a plain check-and-set, which
	// atomic.Bool's CompareAndSwap expresses directly, without Once's
	// one-shot-function machinery.
	//
	// See Log's doc comment for what a nil pointer here means (an SdkError
	// that never went through build — a zero value or one decoded from
	// JSON).
	logged *atomic.Bool
}

// Error returns the error message.
func (e SdkError) Error() string {
	args, _ := json.Marshal(e.Args)

	causeText := unknownCause
	if e.Cause != nil {
		causeText = *e.Cause
	}

	return fmt.Sprintf("%s: %s. Args: %s", e.Msg, causeText, args)
}

// Unwrap returns the original error this SdkError was constructed from, if
// any, so errors.Is and errors.As can traverse beneath it.
//
// This uses a pointer receiver, unlike Error, JSON and SecureString below
// (which stay on value receivers for backwards compatibility — changing
// them would stop a bare SdkError value, as opposed to *SdkError, from
// satisfying the error interface). errors.Is/errors.As call Unwrap on every
// node of a chain without a nil check first. A value receiver has to
// dereference the receiver to build its copy, which panics immediately when
// called on a typed-nil *SdkError — and this package already produces
// exactly that value in the wild (see ToSdkError's typed-nil handling
// below). A pointer receiver instead returns nil for a nil receiver,
// matching Unwrap's documented contract of "no further error", so a routine
// errors.Is/As call can never turn into a panic. Since Error() (below)
// returns *SdkError everywhere in this package, *SdkError's method set —
// which includes both this pointer-receiver method and the value-receiver
// ones — is what every real caller's error chain actually carries.
func (e *SdkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Log emits this error's log line — exactly the line Error() emits at
// construction — and records that it has done so. A second call is a
// no-op, whether it arrives via the same *SdkError, a pointer taken to a
// copy of that same struct (e.g. GinError's local copy — see its doc
// comment), or a concurrent goroutine racing another call on the same
// pointer: this idempotency is the whole basis of a safe migration away
// from "log at construction". During the transition an error may end up
// logged at construction (Error), at an explicit handling boundary (Log),
// or at the HTTP boundary (GinError) — sometimes more than one of those
// will run against what turns out to be the same logical error — and it
// must never produce two lines for one error.
//
// Log takes a pointer receiver, unlike Error, JSON, SecureString and
// GinError below (which stay on value receivers for backwards
// compatibility). What actually makes a second call a no-op, though, is
// not the receiver — it's that logged is a *atomic.Bool: every SdkError
// built by build() (i.e. by Error or New) gets one allocated at
// construction, and every copy taken of that struct afterward — by a
// value-receiver method, by dereferencing the pointer, by storing an
// SdkError value somewhere — carries a pointer to that same atomic.Bool,
// not a copy of its value. A check-and-set through any copy's pointer is
// therefore visible to every other copy, including the caller's original.
// That also makes concurrent Log() calls on the same error safe without an
// external lock: CompareAndSwap(false, true) guarantees exactly one caller
// observes the false→true transition, so exactly one of them logs no
// matter how many goroutines call at once (see
// TestLogConcurrentGoroutinesLogOnce, run with -race).
//
// A nil logged pointer means this SdkError never went through build — a
// zero-value SdkError{} literal, or one decoded from JSON (decoding never
// calls Log, and the field is unexported so JSON tags can't touch it
// either). That is not an error state: Log treats "no flag yet" as "not
// logged yet", lazily allocating one — under mu, so two goroutines making
// their first call on the same nil-flag pointer don't race to install
// different flags — before doing the check-and-set. That makes repeated
// Log() calls on that *same* pointer idempotent, exactly as above, because
// the second call sees the flag the first call installed. What it cannot
// fix is GinError specifically: GinError has a value receiver, so when
// called on an SdkError whose logged pointer is nil, it is already holding
// an independent copy of the whole struct — nil pointer included — before
// Log ever runs. The flag Log lazily allocates for that call is real and
// makes that one call idempotent against itself, but a value receiver has
// no way to write it back to the caller's original (see GinError's doc
// comment), so a later Log() on the original pointer, or a second GinError
// call, starts from its own independent nil flag and logs again. This gap
// is unreachable through Error or New, which always allocate the flag in
// build() before any copy is ever made; it is only reachable by code that
// builds an SdkError{} literal directly, or decodes one from JSON, and
// then hands the same value to GinError (or Log) more than once.
//
// Safe to call on a nil *SdkError: it returns without doing anything,
// rather than panicking, matching the nil-safety already established for
// Unwrap and ToSdkError in this file.
func (e *SdkError) Log() {
	if e == nil {
		return
	}

	if !e.loggedFlag().CompareAndSwap(false, true) {
		return
	}

	// Snapshotted under mu because Configure writes beauty under mu, and a
	// read racing that write is a data race however benign the value looks.
	// Log is the one config read on this path that was left unguarded; Log
	// in log.go has always taken the lock for exactly this (see log.go).
	mu.Lock()
	b := beauty
	mu.Unlock()

	if b {
		printLog(fmt.Sprint(GetSeverityIcon(e.Severity), " ", e.JSON()), e.Severity)
	} else {
		printLog(e.JSON(), e.Severity)
	}
}

// loggedFlag returns e's logged flag, lazily allocating it under mu if e
// was never constructed via build (see the field doc on logged, and the
// "nil logged pointer" section of Log's doc comment above). The lock is
// only around the allocate-if-nil check: it exists so two goroutines
// making their first Log() call on the same nil-flag pointer install
// exactly one *atomic.Bool between them, rather than each allocating their
// own and one write clobbering the other.
func (e *SdkError) loggedFlag() *atomic.Bool {
	mu.Lock()
	defer mu.Unlock()
	if e.logged == nil {
		e.logged = &atomic.Bool{}
	}
	return e.logged
}

// JSON returns the JSON string representation of the SdkError.
func (e SdkError) JSON() string {
	jLog, _ := json.Marshal(e)
	return string(jLog)
}

// privateArgs are the Args keys that must never reach a consumer. They are
// operator context describing this platform's internals, not anything about
// the caller's own request.
//
// The leak this exists to close: gdk attaches the URL it relayed a request to,
// so an anonymous caller of the gateway was handed auth-api's internal address
// —
//
//	"url": "https://auth-<project-number>.us-central1.run.app/api/auth/v2/keypair"
//
// — and because GinError writes SecureString into the x-error header as well
// as the body, and the next service adopts that header as its own Cause, one
// hop's args became permanent text in every message downstream of it. Filtering
// here rather than at each call site means the redaction also applies to the
// text a relaying service inherits.
//
// Args are otherwise still rendered, because some of them genuinely are the
// caller's guidance: a rejected search field is answered with the list of
// accepted ones, a failed validation names the offending field. Those must
// survive. This set is therefore a denylist, which is the weaker of the two
// possible designs — a new key holding internal detail is exposed until it is
// added here. Prefer putting consumer-facing text in the message, and treat
// this list as the floor rather than the whole guarantee.
var privateArgs = map[string]struct{}{
	"url": {}, "host": {}, "hostname": {}, "endpoint": {}, "address": {},
	"upstream": {}, "target": {}, "target_url": {}, "base_url": {},
	"api-key": {}, "api_key": {}, "api-secret": {}, "api_secret": {},
	"secret": {}, "token": {}, "password": {}, "authorization": {},
}

// consumerArgs returns args without the keys privateArgs names. It returns a
// copy: e.Args is what Log serializes, and the operator must keep every key.
// A nil result marshals to "null" the same way an empty map would not, so an
// error with only private args renders "Args: {}" rather than losing the shape
// callers and tests expect.
func consumerArgs(args map[string]any) map[string]any {
	out := make(map[string]any, len(args))
	for k, v := range args {
		if _, private := privateArgs[strings.ToLower(k)]; private {
			continue
		}
		out[k] = v
	}
	return out
}

// SecureString renders this error for a consumer who is not this platform's
// operator: the response body and the x-error header GinError puts on the wire.
//
// At >= 500 it returns Msg alone — a server-side failure is not the caller's to
// diagnose, and Cause at that point is this service's own internals.
//
// Below 500 it returns the full description, minus the Args that privateArgs
// names. A 4xx is about something in the caller's own request, so explaining it
// is the point; disclosing the platform's internal topology while doing so is
// not.
//
// The log line is unaffected either way: Log serializes the struct itself (see
// JSON) and never goes through this method, so operators keep every key.
func (e SdkError) SecureString() string {
	status, ok := e.Args["status"]
	if ok {
		if castInt(status) >= 500 {
			return e.Msg
		}
	}

	args, _ := json.Marshal(consumerArgs(e.Args))

	causeText := unknownCause
	if e.Cause != nil {
		causeText = *e.Cause
	}

	return fmt.Sprintf("%s: %s. Args: %s", e.Msg, causeText, args)
}

// unknownCause is the placeholder build stores in Cause when an error was
// constructed without one. Cause is a *string that build always sets, so
// "no cause" is this sentinel rather than a nil pointer, and any code
// deciding whether there is something to explain must compare against it.
const unknownCause = "unknown cause"

// ErrorIDCarrier is implemented by an error that already carries a catcher
// error id minted somewhere else — by utils.RemoteError and client.APIError,
// which read it from an upstream ThreatWinds service's x-error-id response
// header, and by *SdkError itself.
//
// It exists so a failure can keep one occurrence id across a service boundary
// without the transport helper having to return an *SdkError to achieve it.
// Returning an *SdkError would work, but at a price: build short-circuits on
// an *SdkError cause and hands it straight back, discarding the wrapping
// call's msg, args and status override. A plain error implementing this
// interface lets the caller build its *own* error — its message, its args, its
// status — that happens to inherit the upstream's id.
//
// build consults it only when the cause is not itself an *SdkError (that case
// short-circuits before this is reached) and the call supplied no
// args["error_id"], so an explicitly supplied id always wins over an inherited
// one, and an inherited one always wins over a freshly minted UUID.
//
// An implementation returning "" means "no id to donate" and is treated as if
// the interface were not implemented at all.
//
// # Why the method is not called ErrorID()
//
// The obvious spelling for this method is ErrorID() string, matching the field
// it exposes. It cannot be: SdkError already has an ErrorID *field*, and Go
// forbids a type from declaring a field and a method of the same name, so
// *SdkError — the type most in need of satisfying this interface uniformly —
// could never implement it. The field is the half that cannot move. It is
// serialized as "error_id" (see its json tag), read out of log lines and off
// the wire by consumers, and written into the x-error-id response header by
// GinError; renaming it would break every one of those readers, to buy nothing.
// So the method takes the disambiguating prefix instead, and every
// implementation in the SDK spells it the same way.
//
// The dependency stays inverted through this interface: catcher imports
// neither utils nor client, and never will. An error type opts in by declaring
// the method; nothing else about it has to change.
type ErrorIDCarrier interface {
	CatcherErrorID() string
}

// CatcherErrorID implements ErrorIDCarrier, returning this error's occurrence
// id — the same value as the ErrorID field, which stays the serialized,
// consumer-visible one. See ErrorIDCarrier's doc for why the method carries a
// prefix the field does not.
//
// build never reaches this on an *SdkError cause: ToSdkError short-circuits
// first, so an existing *SdkError is returned by identity rather than donating
// its id to a second error. The method exists so that code outside this package
// can ask "which occurrence is this?" of any error the SDK produces — an
// *SdkError and a transport-level carrier alike — without type-switching over
// every error type in the SDK.
//
// Pointer receiver, and nil-safe, for the same reason as Unwrap and Log: this
// package produces typed-nil *SdkError values in the wild (see ToSdkError), and
// errors.As matches on type alone, so an interface method can be reached on a
// nil receiver by a routine errors.As walk.
func (e *SdkError) CatcherErrorID() string {
	if e == nil {
		return ""
	}

	return e.ErrorID
}

var _ ErrorIDCarrier = (*SdkError)(nil)

// AdoptErrorID guards the one path by which a foreign, remote-controlled value
// enters the field this package otherwise mints itself. It returns id when id
// may be adopted as an occurrence id, and "" when it may not — in which case
// the caller falls through to build's own uuid.NewString, so the error still
// gets an id and only the false correlation is lost.
//
// Only a canonical 36-character UUID is accepted: the shape this package always
// mints, and therefore the only shape a ThreatWinds service can legitimately
// send in x-error-id. Without the guard, a buggy or hostile endpoint can hand
// over an id already in use by an unrelated in-flight failure — which breaks
// the single property ErrorID exists to provide, that grepping one id finds the
// lines of exactly one incident — or an id megabytes long, since
// http.Transport.MaxResponseHeaderBytes allows 10MB of headers. The adopted
// value is echoed straight back out in the adopting service's own x-error-id
// response header and in every log line about the failure, so neither is
// hypothetical.
//
// It lives here, rather than once per transport package, because this package
// owns the id's format: it is the only thing that mints one, and the only thing
// that has to keep every id in the logs the same shape.
//
// # Where it is actually enforced
//
// build applies it to every id that reaches an *SdkError from outside: the
// caller-supplied args["error_id"] (the idiom build's own comment recommends
// for a service re-raising an upstream failure, and therefore the usual way an
// inbound x-error-id header is plumbed in by hand) and every id donated by an
// ErrorIDCarrier cause. That is what makes the guarantee true rather than
// advisory — no *SdkError this package constructs can carry an id it did not
// either mint or vet, whatever a caller or a foreign carrier hands it.
//
// The transport packages (utils.DoReq, client.APIError) apply the same rule to
// decide what they report and donate, but they do it with their own six-line
// copy rather than by importing this package: catcher links gin, and utils is
// linked by binaries — the ETL fetchers — that have no business carrying an
// HTTP framework or this package's logging goroutine. Their copies are not
// load-bearing for the invariant above (build vets their output anyway) and
// each is pinned to this function by a parity test in its own package, so the
// three cannot drift apart without a test failing.
func AdoptErrorID(id string) string {
	// uuid.Parse also accepts the urn:, braced and undashed forms; requiring
	// exactly 36 characters restricts it to the canonical one, so the value
	// written back out is the same shape every other id in the logs has.
	if len(id) != 36 {
		return ""
	}

	if _, err := uuid.Parse(id); err != nil {
		return ""
	}

	return id
}

// inheritedErrorID returns the first usable occurrence id donated by cause or
// by anything cause wraps, and "" when there is none to inherit. Callers of
// this package are unaffected unless their error type opts in by implementing
// ErrorIDCarrier.
//
// # Why this is a walk and not one errors.As
//
// ErrorIDCarrier's contract says an implementation returning "" means "no id to
// donate" and is treated as if the interface were not implemented at all. A
// single errors.As cannot honour that: it stops at the *first* node whose type
// matches, so a carrier with nothing to donate ends the search and shadows a
// real id deeper in the chain — the wrapping call then mints a fresh UUID and
// the two services' log lines for one failure stop being correlatable, which is
// the exact failure this whole mechanism exists to prevent.
//
// That was harmless while RemoteError was the only implementer. It is not now:
// *SdkError, *APIError, *RemoteError and *generatedIDError are all carriers, so
// a chain holding more than one is ordinary — and *APIError donates nothing on
// precisely the common case (an upstream that sent no x-error-id, or one the
// guard refused), which is the case that would do the shadowing. So each node is
// asked in turn and an empty answer continues the walk rather than ending it.
//
// The walk mirrors errors.As otherwise: it follows both Unwrap() error and
// Unwrap() []error (a join is searched depth-first, left to right, the order
// errors.As uses), and it honours a node's own As method so a type that reports
// itself as a carrier through that hook is still found.
func inheritedErrorID(cause error) string {
	for err := cause; err != nil; {
		if id := donatedErrorID(err); id != "" {
			return id
		}

		switch u := err.(type) {
		case interface{ Unwrap() error }:
			err = u.Unwrap()
		case interface{ Unwrap() []error }:
			for _, branch := range u.Unwrap() {
				if id := inheritedErrorID(branch); id != "" {
					return id
				}
			}

			return ""
		default:
			return ""
		}
	}

	return ""
}

// donatedErrorID asks one node of an error chain — not the chain beneath it —
// for an id it is willing to donate, returning "" when it has none or is not a
// carrier at all.
//
// The typed-nil check matters: a type match says nothing about the value, so a
// nil *Whatever boxed in a non-nil error interface satisfies the carrier
// interface and would have its method called on a nil receiver — the same trap
// ToSdkError already guards for *SdkError, but here the type is foreign, so
// whether that panics is not this package's to decide. Error construction is on
// every failure path in every service; it must not be able to turn a failure
// into a panic.
//
// The donated value goes through AdoptErrorID for the same reason a header does:
// a carrier is usually a transport error type holding a value some other process
// chose. An id that is not the shape this package mints is not adopted, and the
// walk continues as if this node had donated nothing.
func donatedErrorID(err error) string {
	var carrier ErrorIDCarrier

	switch v := err.(type) {
	case ErrorIDCarrier:
		carrier = v
	case interface{ As(any) bool }:
		if !v.As(&carrier) {
			return ""
		}
	default:
		return ""
	}

	if carrier == nil || isNilPointer(carrier) {
		return ""
	}

	return AdoptErrorID(carrier.CatcherErrorID())
}

// isNilPointer reports whether v is a non-nil interface holding a nil pointer
// (or another nil-able kind), which is not detectable by comparing the
// interface itself to nil.
func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}

// build implements the construction shared by Error and New: the
// short-circuit when cause is already an *SdkError, trace capture, the code
// hash, error-id adoption/generation, and severity derivation from
// args["status"]. It never logs — Error and New each decide independently
// whether the result gets logged, and how. Keeping this in one place is what
// stops the two constructors from drifting apart.
//
// skip is passed to runtime.Callers so the recorded trace starts at the
// caller of Error/New rather than at build or at Error/New itself; both
// exported constructors are exactly one frame above build, so both pass 3.
func build(msg string, cause error, args map[string]any, skip int) *SdkError {
	if err := ToSdkError(cause); err != nil {
		// Short-circuit: err already carries the error id it was built
		// with. args["error_id"] on *this* call is deliberately never
		// consulted here — see ErrorID's field doc and Error's doc comment
		// on the short-circuit contract.
		return err
	}

	// error_id is reserved, alongside status: when the caller supplies one
	// (e.g. a downstream service re-raising a failure it received from an
	// upstream call, carrying forward the id that upstream already minted),
	// it is adopted so the same occurrence keeps the same id across every
	// service that touches it. Failing that, a cause that carries an id of
	// its own donates it (see ErrorIDCarrier) — the same propagation,
	// without the call site having to plumb the id by hand. When neither is
	// present, a fresh UUID is generated here so every *SdkError this
	// package ever constructs has one, whether or not any caller asked for
	// it.
	//
	// Both inbound routes go through AdoptErrorID, and that is the point at
	// which the guard is real rather than advisory. The caller-supplied key
	// is not a trusted local value: the idiom this comment recommends is a
	// relay plumbing an inbound x-error-id header into args, so it is a
	// remote-controlled string, and without the check a 10MB header value or
	// one colliding with an unrelated in-flight failure would be written
	// straight into this error's ErrorID, back out through GinError's
	// x-error-id header and into every log line about the failure. A value
	// that does not pass is not an occurrence id; the error still gets one,
	// generated below, and only the false correlation is lost.
	suppliedID, _ := args["error_id"].(string)

	errorID := AdoptErrorID(suppliedID)
	if errorID == "" {
		errorID = inheritedErrorID(cause)
	}
	if errorID == "" {
		errorID = uuid.NewString()
	}

	// Snapshotted under mu for the same reason as beauty in Log above:
	// Configure writes noTrace under mu, so reading it unguarded from a
	// goroutine that is constructing an error races that write.
	mu.Lock()
	nt := noTrace
	mu.Unlock()

	var trace []string
	if !nt {
		pc := make([]uintptr, 25)
		n := runtime.Callers(skip, pc)
		frames := runtime.CallersFrames(pc[:n])

		trace = make([]string, 0, 10)
		for {
			frame, more := frames.Next()

			trace = append(trace, fmt.Sprint(frame.Function, " ", frame.Line))
			if !more {
				break
			}
		}
	}
	sum := md5.Sum([]byte(msg))

	// effectiveCause normalizes cause to nil in the two cases that must
	// not participate in the chain: a genuinely nil cause, and a
	// non-nil `error` interface wrapping a nil *SdkError (see
	// ToSdkError). Both render as "unknown cause" in the Cause string
	// below, and for the same reason both must Unwrap to nil rather
	// than to a nil pointer: calling cause.Error() on a typed-nil
	// *SdkError would dereference a nil receiver (Error() has a value
	// receiver) and panic, and handing that same nil pointer out via
	// Unwrap would only defer the panic to whatever errors.Is/As caller
	// eventually calls a method on it.
	effectiveCause := cause
	if sdkCause, ok := cause.(*SdkError); ok && sdkCause == nil {
		effectiveCause = nil
	}

	causeText := unknownCause
	if effectiveCause != nil {
		causeText = effectiveCause.Error()
	}

	err := &SdkError{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Code:      hex.EncodeToString(sum[:]),
		ErrorID:   errorID,
		Trace:     trace,
		Args:      args,
		Msg:       msg,
		Cause:     pointerOf(causeText),
		cause:     effectiveCause,
		// Allocated here, once, so every later copy of this struct — by a
		// value-receiver method, by dereferencing the pointer, by storing
		// an SdkError value — shares this same flag instead of getting an
		// independent one. See the field doc on logged for why that
		// sharing is the whole point.
		logged: &atomic.Bool{},
	}

	statusCode, ok := args["status"]
	if !ok {
		err.Severity = "ERROR"
	} else {
		err.Severity = calculateSeverity(statusCode)
	}

	return err
}

// Error tries to cast the cause as an SdkError, if it is not an SdkError, it creates a new SdkError with the given parameters.
// It logs the error message and returns the error.
// If cause is nil, it will store a blank string in the Cause field.
// The field Code is a hash of the message and trace. It is used to identify the recurrence of an error.
// Params:
// msg: the error message.
// cause: the error that caused this error.
// args: a map of additional information. Recognized keys when used with GinError():
//
//	"status" (int)         → HTTP status code (default 500)
//	"retry" (int)          → Retry-After header value in seconds
//	"code_override" (string) → overrides the error code in the JSON body
//	"param" (string)       → included as "param" in the JSON error detail
//	"error_id" (string)    → identifies this error occurrence; adopted if
//	                         supplied and it passes AdoptErrorID (a
//	                         canonical 36-character UUID, the only shape
//	                         this package mints), otherwise inherited from
//	                         a cause implementing ErrorIDCarrier,
//	                         otherwise generated as a UUID. Meant to be
//	                         carried unchanged by every service that
//	                         re-raises the same failure so it can be
//	                         traced across services. Ignored in the
//	                         short-circuit case below, same as msg and
//	                         every other arg.
//
// # This package logs at construction, not at handling
//
// If cause is already an *SdkError, Error does NOT construct a new error and
// does NOT log anything new: it returns cause unchanged, exactly as built,
// and calls Log() on it — a no-op if that pointer was already logged
// (which every existing call site's cause is, since Error has always logged
// at construction), but not a no-op if it came from New(), which does not
// log. msg, args, and any args["status"] override are silently ignored in
// the short-circuit case — the returned error's identity, Code, Severity
// and Args must stay stable as it propagates back up through however many
// call sites already have it, and re-logging it once per layer on the way
// up is not something this function does.
//
// This means passing an existing *SdkError as cause is never the right way
// to add logged context. To log context about an error you already have,
// pass nil as cause and fold the error text into args instead:
//
//	sdkErr := catcher.ToSdkError(err)
//	catcher.Error("context about the failure", nil, map[string]any{"cause": sdkErr.Error()})
//
// New is the preferred constructor for new call sites migrating away from
// logging at construction; see its doc comment.
//
// Returns:
// *SdkError: the error. This type implements the Go error interface.
func Error(msg string, cause error, args map[string]any) *SdkError {
	err := build(msg, cause, args, 3)
	err.Log()
	return err
}

// New constructs an *SdkError exactly like Error — same short-circuit when
// cause is already an *SdkError, same trace capture, same code hash, same
// severity derivation from args["status"] — but it does not log.
//
// New is the preferred constructor. The error is logged once it reaches a
// boundary: automatically, the first time GinError writes it to an HTTP
// response, or explicitly, by calling Log() on it. Log is idempotent, so
// it is safe to call from both places during a migration, or from neither
// if GinError will handle it. What is not safe is constructing with New and
// then discarding the result without ever reaching a boundary or calling
// Log(): unlike Error, a swallowed *SdkError from New is never logged
// anywhere, by anyone — if you catch it and decide not to propagate it,
// call Log() yourself before you let it go.
func New(msg string, cause error, args map[string]any) *SdkError {
	return build(msg, cause, args, 3)
}

func calculateSeverity(value interface{}) string {
	statusCode := castInt(value)

	if statusCode >= 100 && statusCode < 200 {
		return "DEBUG"
	} else if statusCode >= 200 && statusCode < 300 {
		return "INFO"
	} else if statusCode >= 300 && statusCode < 400 {
		return "NOTICE"
	} else if statusCode >= 400 && statusCode < 500 {
		return "WARNING"
	} else if statusCode >= 500 && statusCode < 502 {
		return "ERROR"
	} else if statusCode >= 502 && statusCode < 509 {
		return "CRITICAL"
	} else if statusCode >= 509 && statusCode < 511 {
		return "ALERT"
	}

	return "ERROR"
}

// ToSdkError tries to cast an error to a SdkError.
// If the error isn't an SdkError, it returns nil.
//
// It also returns nil when err's dynamic value is a nil *SdkError boxed in a
// non-nil error interface — e.g. a function declared to return *SdkError that
// returns nil, then gets assigned to an `error` variable. errors.As matches
// that case (the concrete type is right) and sets the target to that nil
// pointer; returning it as-is would hand callers a *SdkError that is not
// == nil as a concrete pointer but panics the moment any value-receiver
// method (Error, JSON, SecureString) is called on it.
func ToSdkError(err error) *SdkError {
	if err == nil {
		return nil
	}

	var sdkError *SdkError
	if errors.As(err, &sdkError) && sdkError != nil {
		return sdkError
	}

	return nil
}

// GinError is a helper function to return an error to the client using Gin framework context.
// It sets the headers x-error and x-error-id with the error message and the
// error's UUID (ErrorID) respectively — restoring this doc comment's
// long-standing claim, which the code had drifted away from by sending Code
// (a hash of the message, shared by every occurrence of that message) in
// x-error-id instead of a per-occurrence id. It also writes a JSON error
// body — which now includes error_id alongside the existing code, for a
// caller that reads the body rather than headers — and sets the status
// code.
//
// Before writing anything, it logs the error if it has not already been
// logged — the same line Log() would emit. This gives every HTTP path a
// guaranteed boundary log with no handler changes required: an *SdkError
// built with New() (which does not log) is still logged the moment it
// reaches GinError, and one built with Error() or already Log()'d is not
// logged a second time.
//
// GinError keeps its value receiver for backwards compatibility (see
// Error, JSON and SecureString above), so it only ever holds a *copy* of
// the SdkError, not the caller's pointer. That copy still genuinely marks
// the error logged for every other holder of it, because logged is a
// *atomic.Bool (see the field doc and Log's doc comment): copying the
// struct copies the pointer, not the flag it points to, so GinError's
// local copy shares the exact same atomic.Bool the caller's original
// pointer does. (&e).Log() below does its check-and-set on that shared
// flag, so the write is visible through the caller's original pointer too
// — a later err.Log() call, or a second GinError(c) call, on that same
// original pointer sees logged already true and stays silent. This is
// what makes it safe for both "New() built it, GinError logs it" and
// "Error() already logged it, GinError must not log it again" to share
// one code path.
//
// The one case this does not cover is an SdkError whose logged pointer is
// nil to begin with — a zero-value SdkError{} literal, or one decoded from
// JSON, that was never built by build(). There, GinError's copy has its
// own independent nil flag before Log ever runs, Log lazily allocates a
// flag for that copy alone, and — because a value receiver cannot write
// that allocation back to the caller — a later call against the original
// starts from its own independent nil flag and logs again. See the "nil
// logged pointer" section of Log's doc comment for the full explanation;
// it is unreachable through Error or New, which always allocate the flag
// before any copy is made.
// Additional Args keys:
//   - "retry": N — sets the Retry-After header to N seconds.
//   - "code_override": string — overrides the error code in the response body.
//   - "param": string — included as "param" in the response error detail.
//   - "error_id": string — see build's doc comment; reflected in the
//     x-error-id header and the response body's error_id, not the code.
func (e SdkError) GinError(c *gin.Context) {
	(&e).Log()

	secureMsg := e.SecureString()
	c.Header("x-error-id", e.ErrorID)
	c.Header("x-error", secureMsg)

	if retryVal, ok := e.Args["retry"]; ok {
		retry := castInt(retryVal)
		if retry > 0 {
			c.Header("Retry-After", strconv.Itoa(retry))
		}
	}

	resp := errorResponse{
		Error: errorParam{
			Message: secureMsg,
			Type:    e.Severity,
			Code:    e.Code,
			ErrorID: e.ErrorID,
		},
	}

	if codeOverride, ok := e.Args["code_override"]; ok {
		resp.Error.Code, _ = codeOverride.(string)
	}
	if param, ok := e.Args["param"]; ok {
		if p, ok := param.(string); ok {
			resp.Error.Param = p
		}
	}

	statusCode := http.StatusInternalServerError
	if status, ok := e.Args["status"]; ok {
		statusCode = castInt(status)
	}
	c.AbortWithStatusJSON(statusCode, resp)
}

func pointerOf[t any](s t) *t {
	return &s
}

func castInt(value interface{}) int {
	if value == nil {
		return 500
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		val, err := strconv.Atoi(v)
		if err != nil {
			return 500
		}
		return val
	default:
		return 500
	}
}
