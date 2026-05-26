package circuit

import (
	"sync"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	b := New(0, 0)
	if b.State() != StateClosed {
		t.Errorf("expected Closed, got %s", b.State())
	}
	if b.Failures() != 0 {
		t.Errorf("expected 0 failures, got %d", b.Failures())
	}
}

func TestNew_CustomValues(t *testing.T) {
	b := New(5, 10*time.Second)
	if b.State() != StateClosed {
		t.Errorf("expected Closed, got %s", b.State())
	}
	// Internal fields are private; test behavior instead.
}

func TestAllow_Closed(t *testing.T) {
	b := New(3, 30*time.Second)
	for i := 0; i < 100; i++ {
		if !b.Allow() {
			t.Fatalf("expected Allow()=true in Closed state (iteration %d)", i)
		}
	}
}

func TestOpen_AfterThreshold(t *testing.T) {
	b := New(3, 10*time.Second)

	// 3 consecutive failures should trip the breaker.
	for i := 0; i < 3; i++ {
		b.Failure()
	}

	if b.State() != StateOpen {
		t.Fatalf("expected Open, got %s", b.State())
	}

	// Requests should be rejected.
	if b.Allow() {
		t.Fatal("expected Allow()=false in Open state")
	}
}

func TestHalfOpen_AfterCooldown(t *testing.T) {
	b := New(3, 50*time.Millisecond)

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		b.Failure()
	}

	// Wait for cooldown.
	time.Sleep(60 * time.Millisecond)

	// First request should transition to HalfOpen and be allowed.
	if !b.Allow() {
		t.Fatal("expected Allow()=true after cooldown")
	}
	if b.State() != StateHalfOpen {
		t.Fatalf("expected HalfOpen, got %s", b.State())
	}
}

func TestHalfOpen_MaxProbes(t *testing.T) {
	b := New(3, 10*time.Millisecond)

	// Trip and wait for cooldown.
	for i := 0; i < 3; i++ {
		b.Failure()
	}
	time.Sleep(20 * time.Millisecond)

	// Allow up to halfOpenMax (2) requests.
	if !b.Allow() {
		t.Fatal("expected Allow()=true for probe 1")
	}
	if !b.Allow() {
		t.Fatal("expected Allow()=true for probe 2")
	}

	// Third request should be rejected.
	if b.Allow() {
		t.Fatal("expected Allow()=false after max probes")
	}
}

func TestHalfOpen_SuccessRecovery(t *testing.T) {
	b := New(3, 10*time.Millisecond)

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		b.Failure()
	}
	time.Sleep(20 * time.Millisecond)

	// Enter HalfOpen.
	b.Allow()

	// Success should transition to Closed.
	b.Success()

	if b.State() != StateClosed {
		t.Fatalf("expected Closed after Success in HalfOpen, got %s", b.State())
	}
	if b.Failures() != 0 {
		t.Fatalf("expected 0 failures after recovery, got %d", b.Failures())
	}

	// Subsequent requests should be allowed.
	if !b.Allow() {
		t.Fatal("expected Allow()=true after recovery")
	}
}

func TestHalfOpen_FailureBackToOpen(t *testing.T) {
	b := New(3, 10*time.Millisecond)

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		b.Failure()
	}
	time.Sleep(20 * time.Millisecond)

	// Enter HalfOpen.
	b.Allow()

	// Failure in HalfOpen should go back to Open.
	b.Failure()

	if b.State() != StateOpen {
		t.Fatalf("expected Open after Failure in HalfOpen, got %s", b.State())
	}

	// Requests should be rejected.
	if b.Allow() {
		t.Fatal("expected Allow()=false after falling back to Open")
	}
}

func TestReset(t *testing.T) {
	b := New(3, 10*time.Second)

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		b.Failure()
	}

	if b.State() != StateOpen {
		t.Fatalf("expected Open before reset, got %s", b.State())
	}

	b.Reset()

	if b.State() != StateClosed {
		t.Fatalf("expected Closed after reset, got %s", b.State())
	}
	if b.Failures() != 0 {
		t.Fatalf("expected 0 failures after reset, got %d", b.Failures())
	}
}

func TestConcurrency(t *testing.T) {
	b := New(3, 10*time.Millisecond)

	var wg sync.WaitGroup
	const goroutines = 50
	results := make(chan bool, goroutines)

	// Trip the breaker first.
	for i := 0; i < 3; i++ {
		b.Failure()
	}
	time.Sleep(20 * time.Millisecond)

	// Concurrent Allow calls.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- b.Allow()
		}()
	}

	wg.Wait()
	close(results)

	allowed := 0
	for r := range results {
		if r {
			allowed++
		}
	}

	// Only up to halfOpenMax (2) should be allowed.
	if allowed > 2 {
		t.Errorf("expected at most 2 allowances in HalfOpen, got %d", allowed)
	}
}

func TestStateString(t *testing.T) {
	if StateClosed.String() != "CLOSED" {
		t.Errorf("expected CLOSED, got %s", StateClosed)
	}
	if StateOpen.String() != "OPEN" {
		t.Errorf("expected OPEN, got %s", StateOpen)
	}
	if StateHalfOpen.String() != "HALF_OPEN" {
		t.Errorf("expected HALF_OPEN, got %s", StateHalfOpen)
	}
}
