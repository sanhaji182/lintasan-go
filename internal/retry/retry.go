// Package retry provides exponential backoff with jitter for retrying operations.
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds the retry configuration.
type Config struct {
	// MaxRetries is the maximum number of retry attempts (default 3).
	MaxRetries int
	// BaseDelay is the initial delay before the first retry (default 1s).
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff (default 30s).
	MaxDelay time.Duration
	// RetryOnStatus is a list of HTTP status codes that trigger a retry.
	// When nil or empty, retry decision is left entirely to the fn callback.
	RetryOnStatus []int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:    3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		RetryOnStatus: []int{429, 502, 503, 504},
	}
}

// Do executes fn with exponential backoff and jitter.
//
// fn returns (shouldRetry bool, err error):
//   - (false, nil): success, Do returns nil immediately.
//   - (true, err): retry after backoff delay.
//   - (false, err): non-retryable failure, Do returns err immediately.
//
// If all retries are exhausted, Do returns the last error wrapped with context.
func Do(ctx context.Context, cfg Config, fn func() (shouldRetry bool, err error)) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check context before each attempt.
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return lastErr
			}
			return err
		}

		shouldRetry, err := fn()
		if err == nil && !shouldRetry {
			return nil
		}

		lastErr = err

		if !shouldRetry {
			// Non-retryable error.
			return err
		}

		// Last attempt? Don't sleep, just return the error.
		if attempt == cfg.MaxRetries {
			break
		}

		// Calculate backoff: baseDelay * 2^attempt, capped at maxDelay.
		delay := time.Duration(float64(cfg.BaseDelay) * math.Pow(2, float64(attempt)))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		// Add ±25% jitter.
		jitter := time.Duration(float64(delay) * (rand.Float64()*0.5 - 0.25))
		delay += jitter
		if delay < 0 {
			delay = 0
		}

		// Wait with context cancellation.
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("retry: exhausted %d retries: %w", cfg.MaxRetries, lastErr)
}

// ShouldRetryOnStatus returns a helper that checks an HTTP status code against
// the configured list of retryable status codes.
func ShouldRetryOnStatus(status int, retryOnStatus []int) bool {
	for _, s := range retryOnStatus {
		if s == status {
			return true
		}
	}
	return false
}
