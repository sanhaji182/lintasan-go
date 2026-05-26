package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_SuccessFirstAttempt(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	var callCount int32
	err := Do(ctx, cfg, func() (bool, error) {
		atomic.AddInt32(&callCount, 1)
		return false, nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestDo_RetryThenSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseDelay = 10 * time.Millisecond
	cfg.MaxDelay = 50 * time.Millisecond
	ctx := context.Background()

	var callCount int32
	err := Do(ctx, cfg, func() (bool, error) {
		count := int(atomic.AddInt32(&callCount, 1))
		if count < 3 {
			return true, errors.New("transient error")
		}
		return false, nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

func TestDo_ExhaustRetries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseDelay = 1 * time.Millisecond
	cfg.MaxDelay = 5 * time.Millisecond
	ctx := context.Background()

	var callCount int32
	err := Do(ctx, cfg, func() (bool, error) {
		atomic.AddInt32(&callCount, 1)
		return true, errors.New("always fail")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 1 initial + 3 retries = 4 calls.
	if atomic.LoadInt32(&callCount) != 4 {
		t.Fatalf("expected 4 calls, got %d", callCount)
	}
}

func TestDo_NonRetryableError(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	var callCount int32
	err := Do(ctx, cfg, func() (bool, error) {
		atomic.AddInt32(&callCount, 1)
		return false, errors.New("fatal error")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseDelay = 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	var callCount int32
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, cfg, func() (bool, error) {
		atomic.AddInt32(&callCount, 1)
		return true, errors.New("transient")
	})

	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestDo_ContextDeadlineExceeded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseDelay = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	var callCount int32
	err := Do(ctx, cfg, func() (bool, error) {
		atomic.AddInt32(&callCount, 1)
		return true, errors.New("transient")
	})

	if err == nil {
		t.Fatal("expected error from deadline exceeded")
	}
}

func TestShouldRetryOnStatus(t *testing.T) {
	retryable := []int{429, 502, 503, 504}

	if !ShouldRetryOnStatus(429, retryable) {
		t.Error("429 should be retryable")
	}
	if !ShouldRetryOnStatus(503, retryable) {
		t.Error("503 should be retryable")
	}
	if ShouldRetryOnStatus(200, retryable) {
		t.Error("200 should not be retryable")
	}
	if ShouldRetryOnStatus(500, retryable) {
		t.Error("500 should not be retryable (not in list)")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.BaseDelay != 1*time.Second {
		t.Errorf("expected BaseDelay=1s, got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay=30s, got %v", cfg.MaxDelay)
	}
	if len(cfg.RetryOnStatus) != 4 {
		t.Errorf("expected 4 retryable statuses, got %d", len(cfg.RetryOnStatus))
	}
}
