package experimental

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// subprocess_test.go — E1 process isolation tests (Foundation Phase 3).
//
// These prove the isolation guarantees with a REAL child process: the test
// binary re-execs itself in one of several "fake child" modes selected by an
// env var (the standard Go subprocess-test pattern). The guarantees:
//   - echo round-trip works (the byte transport is correct);
//   - a HUNG child is contained by the request timeout (ErrTimeout), not a hang;
//   - a CRASHED/exited child surfaces as a contained error, not a gateway crash;
//   - Stop force-kills a child that ignores EOF (it can't hang shutdown forever).

const childModeEnv = "LINTASAN_EXP_TEST_CHILD"

// TestMain dispatches to a fake-child behavior when the harness launches this
// same binary with childModeEnv set; otherwise it runs the normal test suite.
func TestMain(m *testing.M) {
	switch os.Getenv(childModeEnv) {
	case "echo":
		runEchoChild()
		return
	case "hang":
		// Read one line, then sleep far past any test timeout without replying.
		bufio.NewReader(os.Stdin).ReadString('\n')
		time.Sleep(30 * time.Second)
		return
	case "crash":
		// Exit immediately with a non-zero code (simulates a crash on start).
		os.Exit(3)
	case "exit-after-one":
		// Answer one request, then exit — the next request must be contained.
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		fmt.Fprintf(os.Stdout, "got:%s", line) // includes the trailing newline
		os.Exit(0)
	case "ignore-eof":
		// Never exit on stdin EOF; only a kill stops it. Tests Stop's force-kill.
		for {
			time.Sleep(time.Second)
		}
	}
	os.Exit(m.Run())
}

func runEchoChild() {
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			// Echo back uppercased, newline-terminated.
			fmt.Fprintf(w, "%s\n", strings.ToUpper(strings.TrimRight(line, "\n")))
			w.Flush()
		}
		if err != nil {
			return
		}
	}
}

// childConfig returns a Config that re-execs THIS test binary in the given mode.
func childConfig(t *testing.T, mode string, reqTimeout, stopTimeout time.Duration) Config {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return Config{
		Name:           "test-child-" + mode,
		Path:           exe,
		Env:            append(os.Environ(), childModeEnv+"="+mode),
		RequestTimeout: reqTimeout,
		StopTimeout:    stopTimeout,
	}
}

// TestE1_EchoRoundTrip proves the byte transport works end-to-end with a real
// out-of-process child.
func TestE1_EchoRoundTrip(t *testing.T) {
	s := New(childConfig(t, "echo", 5*time.Second, 2*time.Second))
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	if !s.Running() {
		t.Fatal("expected Running() true after Start")
	}
	resp, err := s.Request(context.Background(), []byte("hello"))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if string(resp) != "HELLO" {
		t.Fatalf("echo round-trip: got %q want %q", resp, "HELLO")
	}
	// A second request on the same child also works (serialized).
	resp2, err := s.Request(context.Background(), []byte("world"))
	if err != nil {
		t.Fatalf("request 2: %v", err)
	}
	if string(resp2) != "WORLD" {
		t.Fatalf("second round-trip: got %q want %q", resp2, "WORLD")
	}
}

// TestE1_TimeoutIsContained proves a HUNG child is contained by the request
// timeout — the call returns ErrTimeout, it does NOT block the caller forever.
func TestE1_TimeoutIsContained(t *testing.T) {
	s := New(childConfig(t, "hang", 200*time.Millisecond, 1*time.Second))
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	start := time.Now()
	_, err := s.Request(context.Background(), []byte("please-hang"))
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout from a hung child, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("timeout was not enforced promptly: %v", elapsed)
	}
}

// TestE1_CrashOnStartIsContained proves a child that exits immediately surfaces
// as a contained error on the first request, not a gateway crash.
func TestE1_CrashOnStartIsContained(t *testing.T) {
	s := New(childConfig(t, "crash", 1*time.Second, 1*time.Second))
	// Start itself may succeed (launch is fast); the crash is observed by the
	// reaper. Give it a moment, then a request must be contained.
	if err := s.Start(context.Background()); err != nil {
		// Some platforms may report the failure at Start — also acceptable.
		return
	}
	// Allow the reaper to observe the exit.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && s.Running() {
		time.Sleep(10 * time.Millisecond)
	}
	_, err := s.Request(context.Background(), []byte("anyone-there"))
	if err == nil {
		t.Fatal("expected a contained error from a crashed child, got nil")
	}
}

// TestE1_ExitAfterOneIsContained proves that after a child exits mid-session,
// the next request is contained (no panic, no hang).
func TestE1_ExitAfterOneIsContained(t *testing.T) {
	s := New(childConfig(t, "exit-after-one", 1*time.Second, 1*time.Second))
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	resp, err := s.Request(context.Background(), []byte("first"))
	if err != nil {
		t.Fatalf("first request should succeed: %v", err)
	}
	if !strings.Contains(string(resp), "got:first") {
		t.Fatalf("unexpected first response: %q", resp)
	}
	// Child has now exited. Let the reaper see it, then the next request must be
	// a contained error.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && s.Running() {
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := s.Request(context.Background(), []byte("second")); err == nil {
		t.Fatal("expected contained error after child exit, got nil")
	}
}

// TestE1_StopForceKillsHungChild proves Stop cannot hang forever: a child that
// ignores stdin EOF is force-killed after StopTimeout.
func TestE1_StopForceKillsHungChild(t *testing.T) {
	s := New(childConfig(t, "ignore-eof", 1*time.Second, 200*time.Millisecond))
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	start := time.Now()
	_ = s.Stop()
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("Stop did not force-kill promptly: %v", elapsed)
	}
	if s.Running() {
		t.Fatal("child should not be Running after Stop")
	}
}

// TestE1_LifecycleGuards covers the not-started / double-start / stop-idempotent
// paths.
func TestE1_LifecycleGuards(t *testing.T) {
	s := New(childConfig(t, "echo", 1*time.Second, 1*time.Second))

	// Request before Start → ErrNotStarted.
	if _, err := s.Request(context.Background(), []byte("x")); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("expected ErrNotStarted before Start, got %v", err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Double Start → ErrAlreadyStarted.
	if err := s.Start(context.Background()); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("expected ErrAlreadyStarted, got %v", err)
	}
	// Stop twice is safe.
	if err := s.Stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("second stop should be safe, got %v", err)
	}
}

// TestE1_MissingPath proves a misconfigured launch is a contained error.
func TestE1_MissingPath(t *testing.T) {
	s := New(Config{Name: "no-path"})
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("expected error for empty Path, got nil")
	}
}
