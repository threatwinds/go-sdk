package catcher

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
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
//     the real terminal, or another test's capture pipe.
//  2. Never assume the ambient configuration. The package starts every test
//     binary in the production defaults (beauty on, async on, tracing off, see
//     catcher's init), and a test that needs something else must ask for it and
//     give it back.
//  3. Never parse a captured line by hand. beauty prefixes a severity icon, and
//     whether it is on is exactly the kind of ambient state rule 2 forbids
//     depending on.

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
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

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
