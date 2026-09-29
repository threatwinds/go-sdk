package catcher_test

import (
	"fmt"
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
	case "config":
		// Reports the configuration this subprocess actually started in, so
		// the parent can prove the regression tests below are running against
		// the path that broke in production rather than against a synchronous
		// one that passes them trivially.
		fmt.Printf("CATCHER_ASYNC=%q async=%v\n", os.Getenv("CATCHER_ASYNC"), catcher.AsyncEnabled())
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// childEnv builds the environment for the subprocess: the parent's, minus every
// CATCHER_ variable, plus the mode marker.
//
// The stripping is what keeps the tests below meaningful. They assert that a
// message logged immediately before os.Exit survives, and that is only a
// question at all when logging is async — which is catcher's default, and the
// configuration that swallowed those messages in production. Inheriting
// CATCHER_ASYNC=false from a Taskfile, a CI env block or a developer's shell
// would make every one of them pass for the wrong reason: with logging
// synchronous, of course the message survives. The suite would report ok with
// the fix reverted, which is exactly what it did before this function existed.
func childEnv(mode string) []string {
	parent := os.Environ()
	env := make([]string, 0, len(parent)+1)

	for _, kv := range parent {
		if strings.HasPrefix(kv, "CATCHER_") {
			continue
		}
		env = append(env, kv)
	}

	return append(env, exitMarker+"="+mode)
}

// runExit re-executes this binary in the given mode and returns everything it
// wrote.
func runExit(t *testing.T, mode string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = childEnv(mode)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess in mode %q was expected to exit non-zero", mode)
	}
	return string(out)
}

// TestSubprocessStartsAsync makes the premise of the two regression tests below
// an assertion rather than an assumption, under an environment deliberately set
// to the value that would hollow them out.
//
// Without the stripping in childEnv, an ambient CATCHER_ASYNC=false reaches the
// subprocess, catcher's init leaves async off, and "the message survived an
// os.Exit" becomes a statement about a synchronous write that was never in
// doubt. This test fails in that case instead, naming the variable responsible.
func TestSubprocessStartsAsync(t *testing.T) {
	t.Setenv("CATCHER_ASYNC", "false")
	t.Setenv("CATCHER_BEAUTY", "false")

	out := runExit(t, "config")

	if !strings.Contains(out, `CATCHER_ASYNC=""`) {
		t.Fatalf("the subprocess inherited CATCHER_ASYNC from this process; the exit-log regression tests would then be asserting a synchronous write and would pass with the fix reverted.\ngot: %q", out)
	}
	if !strings.Contains(out, "async=true") {
		t.Fatalf("the subprocess did not start with async logging on, so the tests below no longer cover the path that lost these messages in production.\ngot: %q", out)
	}
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

// The other direction — that routing everything synchronously is not an
// acceptable fix, because only fatal severities need the guarantee — used to be
// covered here by a subprocess that logged an INFO line and exited. That test
// could not fail: whether the line appeared was decided by a race it had no way
// to control, so it either passed or called t.Skip, and it stayed green with the
// production fix reverted. It has been replaced by TestInfoIsNotWrittenSynchronously
// in the internal test package, which distinguishes the two paths by whether the
// caller blocks on a stalled stdout instead of by who wins a race.
