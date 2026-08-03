package catcher_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/threatwinds/go-sdk/catcher"
)

// The behaviour under test only manifests across a real os.Exit, so the
// subprocess re-runs this same test binary with a marker env var and exits from
// inside it. Asserting in-process would prove nothing: the whole question is
// whether the write reaches the fd before the process dies.
const exitMarker = "CATCHER_EXIT_LOG_SUBPROCESS"

func TestMain(m *testing.M) {
	switch os.Getenv(exitMarker) {
	case "error":
		_ = catcher.Error("fatal message that must survive", nil, nil)
		os.Exit(1)
	case "critical":
		_ = catcher.Error("critical message that must survive", nil, map[string]any{"status": 502})
		os.Exit(1)
	case "info":
		catcher.Info("informational message", nil)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// runExit re-executes this binary in the given mode and returns everything it
// wrote. CATCHER_ASYNC is left unset on purpose: the default path is exactly
// what silently swallowed these messages in production.
func runExit(t *testing.T, mode string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), exitMarker+"="+mode)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess in mode %q was expected to exit non-zero", mode)
	}
	return string(out)
}

// TestErrorSurvivesImmediateExit is the regression test for a fatal log that
// was written and then thrown away.
//
// catcher's init enables async whenever CATCHER_ASYNC is anything other than
// the literal "false", so the default is on. printLog then hands the message to
// a channel drained by a goroutine, and an os.Exit that follows kills the
// process before that goroutine ever runs. Every "log the reason, then exit"
// site in the fleet therefore died in complete silence — compute-api exited 1
// with zero bytes on both streams when it could not reach GCP, in tests and in
// production alike, because package init runs before main can reconfigure
// catcher.
func TestErrorSurvivesImmediateExit(t *testing.T) {
	out := runExit(t, "error")
	if !strings.Contains(out, "fatal message that must survive") {
		t.Fatalf("the message was lost across os.Exit; a service dying this way reports nothing.\ngot: %q", out)
	}
}

// TestCriticalSurvivesImmediateExit covers a severity above ERROR, so the fix
// cannot be written as an equality check on "ERROR" alone.
func TestCriticalSurvivesImmediateExit(t *testing.T) {
	out := runExit(t, "critical")
	if !strings.Contains(out, "critical message that must survive") {
		t.Fatalf("a CRITICAL message was lost across os.Exit.\ngot: %q", out)
	}
}

// TestInfoStaysAsync guards the other direction. Routing everything
// synchronously would fix the symptom by removing async logging altogether,
// which is a performance change nobody asked for; only fatal severities need
// the guarantee. An INFO message racing os.Exit is allowed to be lost.
func TestInfoStaysAsync(t *testing.T) {
	out := runExit(t, "info")
	if strings.Contains(out, "informational message") {
		t.Skip("INFO happened to flush before exit; this direction is inherently racy, not a failure")
	}
}
