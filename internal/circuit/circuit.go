// Package circuit provides a 3-state circuit breaker for protecting downstream services.
//
// States:
//   - Closed: normal operation, requests pass through.
//   - Open: circuit is tripped, requests are rejected immediately.
//   - HalfOpen: testing recovery, limited requests allowed through.
package circuit

import (
	"sync"
	"time"
)

// State represents the current state of the circuit breaker.
type State int

const (
	// StateClosed is the normal operating state — requests flow through.
	StateClosed State = iota

	// StateOpen means the circuit is tripped — requests are rejected.
	StateOpen

	// StateHalfOpen is the testing state — a limited number of probe
	// requests are allowed through to check if the downstream has recovered.
	StateHalfOpen
)

// String returns a human-readable representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Breaker implements a 3-state circuit breaker with configurable
// failure threshold, cooldown period, and half-open probe limit.
type Breaker struct {
	mu sync.RWMutex

	state       State
	failures    int
	lastFailure time.Time

	threshold     int           // consecutive failures to open (default 3)
	cooldown      time.Duration // time before transitioning to half-open (default 30s)
	halfOpenMax   int           // max test requests allowed in half-open (default 2)
	halfOpenCount int           // number of test requests sent in current half-open window
}

// New creates a new Breaker with the given failure threshold and cooldown duration.
// halfOpenMax defaults to 2.
func New(threshold int, cooldown time.Duration) *Breaker {
	if threshold <= 0 {
		threshold = 3
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &Breaker{
		state:       StateClosed,
		threshold:   threshold,
		cooldown:    cooldown,
		halfOpenMax: 2,
	}
}

// Allow returns true if the request should be allowed to proceed.
//
// Behavior by state:
//   - Closed: always returns true.
//   - Open: rejects immediately unless cooldown has expired, in which case
//     transitions to HalfOpen and allows the request.
//   - HalfOpen: allows up to halfOpenMax requests; rejects the rest.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true

	case StateOpen:
		if time.Since(b.lastFailure) >= b.cooldown {
			b.state = StateHalfOpen
			b.halfOpenCount = 1
			return true
		}
		return false

	case StateHalfOpen:
		if b.halfOpenCount < b.halfOpenMax {
			b.halfOpenCount++
			return true
		}
		return false

	default:
		return false
	}
}

// Success records a successful request.
//
// In HalfOpen: resets failure count and transitions to Closed (recovered).
// In Closed: resets failure count (no-op on zero).
// In Open: no effect.
func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateHalfOpen:
		b.failures = 0
		b.halfOpenCount = 0
		b.state = StateClosed

	case StateClosed:
		b.failures = 0

	case StateOpen:
		// Success while open is unexpected but harmless; ignore.
	}
}

// Failure records a failed request.
//
// In Closed: increments failure count; if threshold reached, transitions to Open.
// In HalfOpen: immediately transitions back to Open (recovery failed).
// In Open: records the failure time (resets cooldown timer).
func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.failures++
		b.lastFailure = time.Now()
		if b.failures >= b.threshold {
			b.state = StateOpen
		}

	case StateHalfOpen:
		b.state = StateOpen
		b.failures = b.threshold
		b.halfOpenCount = 0
		b.lastFailure = time.Now()

	case StateOpen:
		b.lastFailure = time.Now()
	}
}

// State returns the current state of the circuit breaker.
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Failures returns the current consecutive failure count.
func (b *Breaker) Failures() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.failures
}

// Reset returns the breaker to its initial Closed state, clearing all counters.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateClosed
	b.failures = 0
	b.halfOpenCount = 0
}

// HalfOpenMax returns the max number of test requests allowed in half-open state.
func (b *Breaker) HalfOpenMax() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.halfOpenMax
}
