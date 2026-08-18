package catcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// This file is the package's only test harness. Everything that captures log
// output or changes catcher's configuration for the duration of a test lives
// here, because all three of those concerns are package-level mutable state
// (beauty, async, noTrace, logChan and os.Stdout itself) and a second, private
// way of touching any of them is how this package's tests became order- and
// timing-dependent in the first place.
//
// The rules the helpers below enforce, and which no test should re-implement:
//
//  1. Never assign os.Stdout while catcher's async writer goroutine is alive.
//     That goroutine resolves os.Stdout at the moment it writes each queued
//     line, so the assignment is an unsynchronized write racing its read, and
//     the line lands wherever os.Stdout happens to point by then — which may be
//     the real terminal, or another test's capture pipe. Note that this is an
//     ordering rule, not a ban on capturing async output: stop the writer,
//     assign os.Stdout, start a writer again, and every line the new writer
//     delivers goes to the pipe with no race at all. That is exactly what
//     captureStdoutAsync does.
//  2. Never assume the ambient configuration. The package starts every test
//     binary in the production defaults (beauty on, async on, tracing off, see
//     catcher's init), and a test that needs something else must ask for it and
//     give it back.
//  3. Never parse a captured line by hand. beauty prefixes a severity icon, and
//     whether it is on is exactly the kind of ambient state rule 2 forbids
//     depending on.
//  4. Never hold os.Stdout from two places at once. os.Stdout is one global for
//     the whole process, so two overlapping captures interleave, cross-deliver
//     each other's lines and leave the loser's already-closed pipe installed as
//     stdout for every test that follows. captureMu enforces this rather than
//     leaving it to a convention that one t.Parallel() would break.

// captureMu serializes every capture in this package, so a capture may be
// started from a parallel test without the captures having to know about each
// other. It also covers the Configure calls each capture makes to park and
// restart the async writer: those write the same package-level state.
//
// It is deliberately not a nesting-tolerant lock. A capture inside another
// capture cannot be made to work — the inner one would restore the outer one's
// pipe as "the original stdout" on the way out — so lockCapture turns that case
// into a diagnosable failure instead of a hang.
var captureMu sync.Mutex

// lockCapture acquires captureMu, failing the test rather than blocking forever
// if it cannot. Waiting is normal and expected (another parallel test is
// capturing); waiting for 30 seconds means the holder is never going to let go,
// which in practice means the caller nested one capture inside another and is
// waiting on itself.
func lockCapture(t *testing.T) {
	t.Helper()

	if captureMu.TryLock() {
		return
	}

	deadline := time.After(30 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if captureMu.TryLock() {
				return
			}
		case <-deadline:
			t.Fatal("another capture has held os.Stdout for 30s: captures are serialized because os.Stdout is process-wide, and a capture nested inside another one waits on itself forever")
		}
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
//
// It first takes the package out of async mode for the duration of the
// capture, and restores it afterward. Without that, this helper cannot be
// deterministic for *any* test, whatever severity that test itself writes at:
// the async writer goroutine (started by catcher's init, since async defaults
// to on) resolves the package-level os.Stdout at the moment it writes each
// queued line, so a line queued by an *earlier* test can be flushed into a
// later test's pipe — landing inside a capture window belonging to code that
// never wrote it. Every assertion here that counts lines, or requires the
// output to be empty, is one stray flush away from failing, and the failure
// moves from test to test between runs because it depends only on when the
// goroutine happens to be scheduled.
//
// Configure(_, false, _) is what makes this work rather than merely narrow the
// window: its disable path stops the writer and waits for it to finish
// draining before returning (see catcher.go), so on return there is no
// goroutine left that could write to the os.Stdout this function is about to
// replace, and anything queued by an earlier test has already gone to the real
// stdout. Async is restored only after os.Stdout is put back, so the restored
// writer can never see the pipe either.
//
// The pipe is drained by a goroutine that runs *while* fn does, rather than
// read once fn has returned. An os.Pipe holds only what fits in the kernel
// buffer (64 KiB on Linux); a capture that reads afterwards deadlocks the
// moment fn logs more than that, which with tracing enabled is a couple of
// hundred lines. Restoration is deferred, so a panic or a t.Fatal inside fn
// gives os.Stdout and async mode back instead of stranding every later test on
// a closed pipe.
//
// The consequence worth stating plainly: what this helper captures is the
// *synchronous* write path. Anything it asserts about formatting or content is
// equally true of the async path, which shares every line of printLog above the
// branch — but a test that wants to prove the async writer delivers at all must
// use captureStdoutAsync instead, or it proves nothing about the configuration
// catcher actually ships with.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	lockCapture(t)
	// Released last of all: the deferred restores below run first (defers are
	// LIFO), so the next capture starts from a restored os.Stdout and a
	// restored async mode rather than from whatever this one had mid-flight.
	defer captureMu.Unlock()

	mu.Lock()
	b, wasAsync, nt := beauty, async, noTrace
	mu.Unlock()

	if wasAsync {
		Configure(b, false, nt)
		// Restored last, after os.Stdout is back and the write end is
		// closed: the new writer must never be able to observe the pipe.
		defer Configure(b, true, nt)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	var buf bytes.Buffer
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		defer func() { _ = r.Close() }()
		_, _ = buf.ReadFrom(r)
	}()

	original := os.Stdout
	os.Stdout = w

	func() {
		defer func() {
			os.Stdout = original
			_ = w.Close()
		}()
		fn()
	}()

	// Closing the write end above ends the ReadFrom; receiving here both
	// waits for the last byte and establishes the happens-before edge that
	// makes reading buf from this goroutine safe.
	<-drained

	return buf.String()
}

// asyncCapture captures stdout with catcher's async writer running, so what it
// collects is what that goroutine actually delivered rather than what a
// synchronous write produced.
//
// It exists because captureStdout cannot answer the question at all: it switches
// async off for the duration of the capture, on purpose and for good reasons, so
// a package whose only capture helper is that one has zero coverage of the path
// every DEBUG, INFO, NOTICE and WARNING line takes in production. Async is
// catcher's shipped default — init enables it unless CATCHER_ASYNC is literally
// "false" — so that is the majority of all log lines in the fleet.
//
// Rule 1 in this file's header is respected, not excepted. The sequence is:
// stop the writer, replace os.Stdout while no writer exists, start a writer
// (which therefore resolves the pipe, and nothing else), and on the way out stop
// the writer and wait for it before putting os.Stdout back. No assignment of
// os.Stdout ever overlaps a live writer.
//
// Callers use captureStdoutAsync unless they need to control *when* the pipe
// starts being drained; TestDisablingAsyncWaitsForTheWriter does, because a
// pipe nobody is reading is how it builds a backlog for the drain to work
// through.
type asyncCapture struct {
	t        *testing.T
	beauty   bool
	noTrace  bool
	wasAsync bool

	original *os.File
	r, w     *os.File

	buf     bytes.Buffer
	drained chan struct{}
	reading bool

	finished bool
	output   string
}

// beginAsyncCapture takes the capture lock, points os.Stdout at a fresh pipe and
// starts catcher's async writer against it. Every caller must pair it with
// finish, through a defer, or the lock and os.Stdout are both stranded.
func beginAsyncCapture(t *testing.T) *asyncCapture {
	t.Helper()

	lockCapture(t)

	mu.Lock()
	c := &asyncCapture{t: t, beauty: beauty, wasAsync: async, noTrace: noTrace}
	mu.Unlock()

	// Rule 1: the writer must be down before os.Stdout is touched.
	if c.wasAsync {
		Configure(c.beauty, false, c.noTrace)
	}

	r, w, err := os.Pipe()
	if err != nil {
		if c.wasAsync {
			Configure(c.beauty, true, c.noTrace)
		}
		captureMu.Unlock()
		t.Fatalf("failed to create pipe: %v", err)
	}

	c.r, c.w = r, w
	c.drained = make(chan struct{})

	c.original = os.Stdout
	os.Stdout = w

	// Only now, with the pipe installed, is a writer started: this one resolves
	// os.Stdout for the first time after the assignment, so every line it
	// delivers lands in the pipe.
	Configure(c.beauty, true, c.noTrace)

	return c
}

// read starts draining the pipe, and is idempotent.
//
// Until it is called the pipe holds only what fits in the kernel buffer (64 KiB
// on Linux) and the writer blocks once that is full — useful when a test wants a
// backlog, fatal when it does not. finish calls this before it does anything
// else, so a caller that only wants the output never has to think about it, as
// long as fn stays under 64 KiB.
func (c *asyncCapture) read() {
	if c.reading {
		return
	}
	c.reading = true

	go func() {
		defer close(c.drained)
		defer func() { _ = c.r.Close() }()
		_, _ = c.buf.ReadFrom(c.r)
	}()
}

// finish stops the writer, restores os.Stdout and returns everything the writer
// delivered. It is idempotent, so `defer c.finish()` and a final `c.finish()`
// compose.
func (c *asyncCapture) finish() string {
	if c.finished {
		return c.output
	}
	c.finished = true

	c.read()

	// Configure's disable path stops the writer and waits for it to drain, and
	// that wait is the invariant everything below depends on: without it the
	// assignment and the Close on the next two lines race a goroutine that is
	// still writing, and whatever it had not yet flushed is thrown away.
	Configure(c.beauty, false, c.noTrace)

	os.Stdout = c.original
	_ = c.w.Close()

	// Receiving both waits for the last byte and establishes the happens-before
	// edge that makes reading buf from this goroutine safe.
	<-c.drained

	if c.wasAsync {
		Configure(c.beauty, true, c.noTrace)
	}

	c.output = c.buf.String()
	captureMu.Unlock()

	return c.output
}

// captureStdoutAsync redirects os.Stdout for the duration of fn, with catcher's
// async writer running, and returns everything that writer delivered.
//
// It is the async counterpart of captureStdout: same guarantees about
// restoration on panic, same concurrent drain, but the lines it returns arrived
// through logChan and the writer goroutine rather than from printLog's
// synchronous tail.
func captureStdoutAsync(t *testing.T, fn func()) string {
	t.Helper()

	c := beginAsyncCapture(t)
	defer c.finish()

	c.read()
	fn()

	return c.finish()
}

// fillPipe writes to w until the kernel buffer behind it is full, so that the
// next write to it blocks.
//
// The fill is bounded by a write deadline rather than by an assumed buffer size:
// os.Pipe files are pollable, so a write that cannot proceed returns
// os.ErrDeadlineExceeded instead of parking, and that is the signal the buffer
// is full. The deadline is cleared afterwards, leaving w a perfectly ordinary
// blocking destination that happens to have no room left.
func fillPipe(t *testing.T, w *os.File) {
	t.Helper()

	chunk := make([]byte, 4096)
	for i := range chunk {
		chunk[i] = 'x'
	}

	for {
		if err := w.SetWriteDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			t.Fatalf("failed to set a write deadline on the pipe: %v", err)
		}

		if _, err := w.Write(chunk); err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("unexpected error while filling the pipe: %v", err)
			}
			break
		}
	}

	if err := w.SetWriteDeadline(time.Time{}); err != nil {
		t.Fatalf("failed to clear the write deadline on the pipe: %v", err)
	}
}

// lastJSONLine parses the last non-empty line of output as JSON. Every
// assertion in this package that needs log content produces exactly one line
// per captured call, but taking the last line (rather than assuming there is
// only one) is the defensive choice.
//
// beauty (package-level, on by default) may prefix the line with a severity
// icon and a space ahead of the JSON object; the prefix is stripped by
// locating the opening brace rather than by asserting beauty's current value,
// so this helper works regardless of what the configuration happens to be.
func lastJSONLine(t *testing.T, output string) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	last := lines[len(lines)-1]

	if i := strings.IndexByte(last, '{'); i > 0 {
		last = last[i:]
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(last), &parsed); err != nil {
		t.Fatalf("expected a JSON log line, got %q: %v", last, err)
	}
	return parsed
}

// withConfig applies a configuration for the duration of the calling test and
// restores whatever was in effect before it, afterward. It is the only way
// tests are allowed to call Configure: a test that sets configuration without
// giving it back changes the starting state of every test that runs after it,
// which is how a suite stops being order-independent.
//
// Cleanups run last-in-first-out, so nesting these composes correctly.
func withConfig(t *testing.T, b, a, nt bool) {
	t.Helper()

	mu.Lock()
	ob, oa, ont := beauty, async, noTrace
	mu.Unlock()

	Configure(b, a, nt)
	t.Cleanup(func() {
		Configure(ob, oa, ont)
	})
}

// withSyncLogging disables async logging for the duration of the calling test,
// leaving beauty and tracing alone. It exists because captureStdout only
// produces deterministic results for synchronous writes (see its doc comment);
// a test whose own writes are INFO severity — InfiniteLoop's termination log,
// for one — would otherwise race the async drain goroutine.
//
// Note that captureStdout already does this for the span of a capture. Use
// this when the whole test, not just the captured span, must be synchronous.
func withSyncLogging(t *testing.T) {
	t.Helper()

	mu.Lock()
	b, nt := beauty, noTrace
	mu.Unlock()

	withConfig(t, b, false, nt)
}

// withTrace turns stack-trace collection on for the duration of the calling
// test, leaving beauty and async alone.
//
// Tracing is off unless CATCHER_NO_TRACE is literally "false" (catcher's init),
// and that default is deliberate and documented — traces cost CPU and memory on
// every log line, and relay strips them precisely because they are normally
// absent. So a test that asserts a trace was captured is asserting a
// non-default configuration and has to establish it: reading the env var at
// that point is too late, since init has already run, and exporting the
// variable outside the process would move the dependency out of the test that
// has it. Configure is the only thing that works in-process.
//
// It is restored on cleanup so later tests keep seeing the default, which is
// what their captured lines are written against.
func withTrace(t *testing.T) {
	t.Helper()

	mu.Lock()
	b, a := beauty, async
	mu.Unlock()

	withConfig(t, b, a, false)
}

// devNull opens os.DevNull for *writing* and closes it when the benchmark
// ends. os.Open is read-only, so a benchmark that assigns its result to
// os.Stdout measures a write that fails with EBADF — discarded by the `_, _ =`
// on catcher's write paths — rather than a write that is thrown away by the
// kernel, and leaks the descriptor on top.
func devNull(b *testing.B) *os.File {
	b.Helper()

	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("failed to open %s: %v", os.DevNull, err)
	}
	b.Cleanup(func() { _ = f.Close() })
	return f
}

// benchToDevNull points catcher's output at /dev/null under the given
// configuration, and restores both when the benchmark ends.
//
// Restoring matters even though benchmarks look self-contained: Configure
// writes package-level state, `go test -bench` runs whichever benchmarks the
// pattern selected in source order, and a benchmark that leaves beauty or
// tracing flipped silently changes what the next one is measuring.
func benchToDevNull(b *testing.B, beautify, useAsync, disableTrace bool) {
	b.Helper()

	mu.Lock()
	ob, oa, ont := beauty, async, noTrace
	mu.Unlock()

	Configure(beautify, useAsync, disableTrace)

	original := os.Stdout
	os.Stdout = devNull(b)

	b.Cleanup(func() {
		// Stop the writer before touching os.Stdout, for the same reason
		// captureStdout does: it resolves os.Stdout at write time, so
		// reassigning while it is alive is a race whose loser is a log line
		// written to the wrong descriptor.
		Configure(beautify, false, disableTrace)
		os.Stdout = original
		Configure(ob, oa, ont)
	})
}
