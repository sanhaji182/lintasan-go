// Package ratelimit provides a sliding window rate limiter with burst mode.
//
// Features:
//   - Sliding window per API key and per IP (60s window)
//   - Burst mode: 2× normal limit for 5s when usage exceeds 80%
//   - Per-key RPM quota override
//   - Periodic cleanup goroutine to prevent memory leaks
//
// Defaults: 60 requests/min per key, 30 requests/min per IP.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a sliding window rate limiter with burst mode.
// It is safe for concurrent use.
type Limiter struct {
	mu         sync.RWMutex
	keyWindows map[string]*slidingWindow // per API key
	ipWindows  map[string]*ipWindow      // per IP
	burstState map[string]*burstEntry    // burst tracking per key

	// DefaultRPM is the default requests-per-minute limit per API key.
	DefaultRPM int

	// DefaultIPRPM is the default requests-per-minute limit per IP.
	DefaultIPRPM int

	// BurstMul is the multiplier applied during burst mode (default 2.0).
	BurstMul float64

	// BurstSecs is how long burst mode lasts in seconds (default 5).
	BurstSecs int

	// BurstThresh is the usage fraction that triggers burst mode (default 0.8).
	BurstThresh float64

	stopCleanup chan struct{}
}

// slidingWindow tracks request timestamps for a single API key.
type slidingWindow struct {
	timestamps []int64 // Unix nanoseconds
	mu         sync.Mutex
}

// ipWindow tracks request timestamps for a single IP address.
type ipWindow struct {
	timestamps []int64
	mu         sync.Mutex
}

// burstEntry tracks burst mode state for a single API key.
type burstEntry struct {
	burstStart time.Time
	active     bool
}

// Default values.
const (
	defaultRPM         = 60
	defaultIPRPM       = 30
	defaultBurstMul    = 2.0
	defaultBurstSecs   = 5
	defaultBurstThresh = 0.8
	windowSecs         = 60
	cleanupInterval    = 60 * time.Second
	// cleanupAge is how long a window must be idle before removal (2× window for safety).
	cleanupAge = 120 * time.Second
)

// New creates a new Limiter and starts the background cleanup goroutine.
// Pass zero or negative values to use defaults (60 RPM key, 30 RPM IP).
func New(keyRPM, ipRPM int) *Limiter {
	if keyRPM <= 0 {
		keyRPM = defaultRPM
	}
	if ipRPM <= 0 {
		ipRPM = defaultIPRPM
	}

	l := &Limiter{
		keyWindows:   make(map[string]*slidingWindow),
		ipWindows:    make(map[string]*ipWindow),
		burstState:   make(map[string]*burstEntry),
		DefaultRPM:   keyRPM,
		DefaultIPRPM: ipRPM,
		BurstMul:     defaultBurstMul,
		BurstSecs:    defaultBurstSecs,
		BurstThresh:  defaultBurstThresh,
		stopCleanup:  make(chan struct{}),
	}

	go l.cleanupLoop()
	return l
}

// AllowKey checks whether an API key is allowed to make a request.
//
// quotaOverride of 0 or less means use the configured DefaultRPM.
// A positive quotaOverride sets a per-key RPM limit.
//
// Returns (allowed, remaining). remaining is the number of requests still
// available in the current window (may be 0 when allowed is false).
func (l *Limiter) AllowKey(key string, quotaOverride int) (bool, int) {
	limit := l.DefaultRPM
	if quotaOverride > 0 {
		limit = quotaOverride
	}

	sw := l.getOrCreateKeyWindow(key)

	now := time.Now()
	nowNano := now.UnixNano()
	cutoff := nowNano - int64(windowSecs)*int64(time.Second)

	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Purge expired timestamps out of the sliding window.
	l.purgeExpired(&sw.timestamps, cutoff)

	count := len(sw.timestamps)
	effectiveLimit := limit

	// Burst mode logic.
	be := l.getOrCreateBurstEntry(key)
	if be.active {
		if now.Sub(be.burstStart) < time.Duration(l.BurstSecs)*time.Second {
			effectiveLimit = int(float64(limit) * l.BurstMul)
		} else {
			be.active = false
		}
	} else if limit > 0 && float64(count)/float64(limit) >= l.BurstThresh {
		be.active = true
		be.burstStart = now
		effectiveLimit = int(float64(limit) * l.BurstMul)
	}

	if count >= effectiveLimit {
		return false, 0
	}

	sw.timestamps = append(sw.timestamps, nowNano)
	remaining := effectiveLimit - (count + 1)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining
}

// AllowIP checks whether an IP address is allowed to make a request.
// IP rate limiting does not have burst mode.
func (l *Limiter) AllowIP(ip string) bool {
	limit := l.DefaultIPRPM

	iw := l.getOrCreateIPWindow(ip)

	nowNano := time.Now().UnixNano()
	cutoff := nowNano - int64(windowSecs)*int64(time.Second)

	iw.mu.Lock()
	defer iw.mu.Unlock()

	l.purgeExpired(&iw.timestamps, cutoff)

	if len(iw.timestamps) >= limit {
		return false
	}

	iw.timestamps = append(iw.timestamps, nowNano)
	return true
}

// Reset clears all counters and burst state for a given API key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.keyWindows, key)
	delete(l.burstState, key)
	l.mu.Unlock()
}

// ResetIP clears counters for a given IP.
func (l *Limiter) ResetIP(ip string) {
	l.mu.Lock()
	delete(l.ipWindows, ip)
	l.mu.Unlock()
}

// Shutdown stops the background cleanup goroutine.
// After Shutdown, the Limiter should not be used.
func (l *Limiter) Shutdown() {
	close(l.stopCleanup)
}

// KeyCount returns the number of API keys currently tracked.
func (l *Limiter) KeyCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.keyWindows)
}

// IPCount returns the number of IPs currently tracked.
func (l *Limiter) IPCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.ipWindows)
}

// --- Internal helpers ---

func (l *Limiter) getOrCreateKeyWindow(key string) *slidingWindow {
	l.mu.RLock()
	sw, ok := l.keyWindows[key]
	l.mu.RUnlock()
	if ok {
		return sw
	}

	sw = &slidingWindow{}
	l.mu.Lock()
	if existing, ok2 := l.keyWindows[key]; ok2 {
		l.mu.Unlock()
		return existing
	}
	l.keyWindows[key] = sw
	l.mu.Unlock()
	return sw
}

func (l *Limiter) getOrCreateIPWindow(ip string) *ipWindow {
	l.mu.RLock()
	iw, ok := l.ipWindows[ip]
	l.mu.RUnlock()
	if ok {
		return iw
	}

	iw = &ipWindow{}
	l.mu.Lock()
	if existing, ok2 := l.ipWindows[ip]; ok2 {
		l.mu.Unlock()
		return existing
	}
	l.ipWindows[ip] = iw
	l.mu.Unlock()
	return iw
}

func (l *Limiter) getOrCreateBurstEntry(key string) *burstEntry {
	l.mu.RLock()
	be, ok := l.burstState[key]
	l.mu.RUnlock()
	if ok {
		return be
	}

	be = &burstEntry{}
	l.mu.Lock()
	if existing, ok2 := l.burstState[key]; ok2 {
		l.mu.Unlock()
		return existing
	}
	l.burstState[key] = be
	l.mu.Unlock()
	return be
}

// purgeExpired removes timestamps older than cutoff from the slice in place.
func (l *Limiter) purgeExpired(ts *[]int64, cutoff int64) {
	valid := (*ts)[:0]
	for _, t := range *ts {
		if t > cutoff {
			valid = append(valid, t)
		}
	}
	*ts = valid
}

func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopCleanup:
			return
		case <-ticker.C:
			l.cleanup()
		}
	}
}

func (l *Limiter) cleanup() {
	cutoff := time.Now().UnixNano() - int64(cleanupAge)

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, sw := range l.keyWindows {
		sw.mu.Lock()
		hasRecent := false
		for _, ts := range sw.timestamps {
			if ts > cutoff {
				hasRecent = true
				break
			}
		}
		sw.mu.Unlock()
		if !hasRecent {
			delete(l.keyWindows, key)
			delete(l.burstState, key)
		}
	}

	for ip, iw := range l.ipWindows {
		iw.mu.Lock()
		hasRecent := false
		for _, ts := range iw.timestamps {
			if ts > cutoff {
				hasRecent = true
				break
			}
		}
		iw.mu.Unlock()
		if !hasRecent {
			delete(l.ipWindows, ip)
		}
	}
}
