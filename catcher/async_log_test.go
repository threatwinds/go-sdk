package catcher

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// AsyncEnabled reports whether async logging is on right now.
//
// It is declared in a _test.go file, so it is part of the test binary and not of
// the package's public API. It exists for the external test package in
// exit_log_test.go, which drives a subprocess to check that the subprocess
// really did start in catcher's shipped configuration; that check has to read
// package state the external package cannot see, and inferring the state from
// what the subprocess prints is exactly the circular argument this file's tests
// exist to remove.
func AsyncEnabled() bool {
	mu.Lock()
	defer mu.Unlock()

	return async
}

// TestAsyncWriterDeliversLogLines is the package's only proof that the async
// writer delivers anything at all.
//
// Every other test that inspects output goes through captureStdout, which
// switches async off for the duration of the capture — necessarily, see its doc
// comment — so before this test existed the whole suite verified only the
// synchronous tail of printLog. That is not the path catcher ships: init enables
// async unless CATCHER_ASYNC is literally "false", and printLog routes every
// severity below ERROR through the channel, so DEBUG, INFO, NOTICE and WARNING —
// the overwhelming majority of the fleet's log volume — are delivered by the
// goroutine none of those tests could observe. Dropping every queued line on the
// floor was a green test run.
func TestAsyncWriterDeliversLogLines(t *testing.T) {
	cases := []struct {
		name     string
		log      func()
		msg      string
		severity string
	}{
		{
			name:     "info",
			log:      func() { Info("delivered by the async writer", nil) },
			msg:      "delivered by the async writer",
			severity: "INFO",
		},
		{
			name:     "warning",
			log:      func() { Warn("also delivered by the async writer", nil) },
			msg:      "also delivered by the async writer",
			severity: "WARNING",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureStdoutAsync(t, tc.log)

			if strings.TrimSpace(out) == "" {
				t.Fatalf("async writer never delivered the %s line, got %q: with async on — catcher's default — every line at this severity is queued to logChan and written by the writer goroutine, so nothing arriving means nothing in the fleet's logs either", tc.severity, out)
			}

			parsed := lastJSONLine(t, out)

			if parsed["msg"] != tc.msg {
				t.Fatalf("expected msg %q, got %v", tc.msg, parsed["msg"])
			}
			if parsed["severity"] != tc.severity {
				t.Fatalf("expected severity %q, got %v", tc.severity, parsed["severity"])
			}
		})
	}
}

// TestInfoIsNotWrittenSynchronously guards the direction opposite to the
// exit-log regression tests: fatal severities are written synchronously so they
// survive an os.Exit, and everything else must keep the channel, because routing
// all of it synchronously would "fix" the lost-fatal-log bug by deleting async
// logging.
//
// The check is a blocking one, because delivery alone cannot tell the paths
// apart — both eventually write the same bytes to the same descriptor. So
// os.Stdout is left with no room: a synchronous write parks until something
// drains it, while a queued one returns immediately, since the queue is a
// 10,000-slot channel and the only thing blocked is the writer draining it.
//
// This replaces the subprocess test that used to cover this direction. That one
// asserted nothing: a race it could not control decided whether the INFO line
// appeared, so it either passed or called t.Skip, and t.Skip is invisible
// without -v. It stayed green with the production fix reverted.
func TestInfoIsNotWrittenSynchronously(t *testing.T) {
	c := beginAsyncCapture(t)
	defer c.finish()

	fillPipe(t, c.w)

	returned := make(chan struct{})
	go func() {
		defer close(returned)
		Info("info must not block on a stalled stdout", nil)
	}()

	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("Info blocked on a stdout with no room left: INFO is being written synchronously. Only ERROR and above need that guarantee — sending the high-volume severities down the same path turns every log call into a write the caller waits on.")
	}

	out := c.finish()
	if !strings.Contains(out, "info must not block on a stalled stdout") {
		t.Fatalf("the queued INFO line never reached stdout once the pipe drained; got %d bytes", len(out))
	}
}

// TestDisablingAsyncWaitsForTheWriter pins the wait in Configure's disable
// branch, which is the invariant every capture in this package rests on and the
// one thing here that a reader can mistake for a pointless latency cost.
//
// Deleting it is invisible to `go test`: the failure it produces is a race
// between a goroutine still flushing and a caller that has already been told
// logging is synchronous, and that race is only reported under -race — which
// this repository's CI never runs, because .github holds a dependabot config and
// no workflows at all.
//
// So the test makes the loss observable without the detector. Nothing reads the
// pipe while the lines are queued, so the writer fills the 64 KiB kernel buffer
// and stops there with the rest of the backlog still in the channel; finish then
// disables async and closes the pipe. With the wait, Configure returns only once
// the last line is out and all of them are counted here. Without it, Configure
// returns while thousands of lines are still queued, the close discards them,
// and the count comes up short.
func TestDisablingAsyncWaitsForTheWriter(t *testing.T) {
	c := beginAsyncCapture(t)
	defer c.finish()

	// Enough lines, and long enough ones, that the backlog cannot fit in the
	// pipe's buffer — but well inside logChan's 10,000 slots, so none of them
	// take printLog's channel-full fallback instead of the path under test.
	const lines = 4000
	payload := strings.Repeat("x", 200)

	queued := make(chan struct{})
	go func() {
		defer close(queued)
		for i := 0; i < lines; i++ {
			printLog(fmt.Sprintf("drain-%04d %s", i, payload), "INFO")
		}
	}()

	select {
	case <-queued:
	case <-time.After(30 * time.Second):
		// Queuing is instant; writing is not, because nothing is draining the
		// pipe yet. Reporting that here beats letting the package's own
		// timeout kill the binary, which takes every other test's result with
		// it. The deferred finish starts the drain and releases the writer.
		t.Fatalf("queueing %d INFO lines blocked: they are reaching stdout directly instead of the channel, so this test can no longer say anything about the drain", lines)
	}

	out := c.finish()

	if got := strings.Count(out, "drain-"); got != lines {
		t.Fatalf("disabling async returned before the writer had drained: %d of %d queued lines reached stdout. Configure's disable branch must wait on logDone, or a caller that switches to synchronous logging and then redirects or closes stdout loses everything still in flight.", got, lines)
	}
}

// TestCapturesAreSerialized holds captureStdout to the rule its header states:
// two captures never hold os.Stdout at once.
//
// os.Stdout is one variable for the whole process, so overlapping captures
// interleave — one of them collects nothing and another collects a line it never
// wrote, which is the same cross-delivery this branch removed from the serial
// suite, reintroduced by a single t.Parallel() anywhere in the package. Worse,
// the loser restores *its* idea of the original stdout, which is another
// capture's pipe, so the process is left writing every subsequent log line into
// a closed descriptor — silently, since catcher discards write errors.
//
// The overlap here is arranged rather than hoped for: the second capture is
// started while the first is provably still inside fn, and the first logs only
// after the second has had a wide window to replace os.Stdout. Serialization
// makes that ordering impossible, so each capture sees exactly its own line;
// without it, the first capture's line lands in the second's pipe every time
// rather than on the occasions the scheduler happens to arrange it.
func TestCapturesAreSerialized(t *testing.T) {
	const overlap = 200 * time.Millisecond

	inFirst := make(chan struct{})
	secondOut := make(chan string, 1)

	go func() {
		<-inFirst
		secondOut <- captureStdout(t, func() {
			Info("second-capture", nil)
			time.Sleep(overlap)
		})
	}()

	firstOut := captureStdout(t, func() {
		close(inFirst)
		// Long enough that an unserialized second capture has certainly
		// replaced os.Stdout by the time the line below is written.
		time.Sleep(overlap / 2)
		Info("first-capture", nil)
	})

	second := <-secondOut

	if !strings.Contains(firstOut, "first-capture") {
		t.Fatalf("the first capture did not collect its own line; it went to the other capture's pipe or to the terminal.\ngot: %q", firstOut)
	}
	if strings.Contains(firstOut, "second-capture") {
		t.Fatalf("the first capture collected the second one's line: the two captures overlapped.\ngot: %q", firstOut)
	}
	if !strings.Contains(second, "second-capture") {
		t.Fatalf("the second capture did not collect its own line.\ngot: %q", second)
	}
	if strings.Contains(second, "first-capture") {
		t.Fatalf("the second capture collected the first one's line: the two captures overlapped.\ngot: %q", second)
	}
}

// TestCapturesInParallelTestsDoNotCrossDeliver is the scenario that motivates
// the serialization above, written the way it would actually appear: subtests
// that call t.Parallel and capture. Each capture holds os.Stdout for a wide
// window, so unserialized captures interleave rather than merely being able to.
func TestCapturesInParallelTestsDoNotCrossDeliver(t *testing.T) {
	names := []string{"alpha", "bravo", "charlie", "delta"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			marker := "parallel-" + name
			out := captureStdout(t, func() {
				Info(marker, nil)
				time.Sleep(50 * time.Millisecond)
				Info(marker, nil)
			})

			if strings.TrimSpace(out) == "" {
				t.Fatalf("capture %q came back empty: its lines went to another capture's pipe, or to the terminal", name)
			}

			for _, other := range names {
				if other == name {
					continue
				}
				if strings.Contains(out, "parallel-"+other) {
					t.Fatalf("capture %q also received %q's line; captures are overlapping", name, other)
				}
			}

			if parsed := lastJSONLine(t, out); parsed["msg"] != marker {
				t.Fatalf("expected msg %q, got %v", marker, parsed["msg"])
			}
		})
	}
}
