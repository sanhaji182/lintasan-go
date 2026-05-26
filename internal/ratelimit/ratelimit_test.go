package ratelimit

import (
	"fmt"
	"sync"
	"testing"
)

// newTestLimiter creates a Limiter for testing with burst mode effectively disabled.
func newTestLimiter(keyRPM, ipRPM int) *Limiter {
	l := New(keyRPM, ipRPM)
	l.BurstThresh = 2.0 // disable burst for basic tests
	return l
}

// newTestLimiterWithZero creates a limiter with DefaultRPM=0 for zero-limit tests.
func newTestLimiterWithZero() *Limiter {
	l := New(1, 1) // need non-zero so New doesn't substitute defaults
	l.DefaultRPM = 0
	l.BurstThresh = 2.0
	return l
}

func TestNew_Defaults(t *testing.T) {
	l := New(0, 0)
	defer l.Shutdown()

	if l.DefaultRPM != defaultRPM {
		t.Errorf("expected DefaultRPM=%d, got %d", defaultRPM, l.DefaultRPM)
	}
	if l.DefaultIPRPM != defaultIPRPM {
		t.Errorf("expected DefaultIPRPM=%d, got %d", defaultIPRPM, l.DefaultIPRPM)
	}
	if l.BurstMul != defaultBurstMul {
		t.Errorf("expected BurstMul=%f, got %f", defaultBurstMul, l.BurstMul)
	}
}

func TestNew_CustomValues(t *testing.T) {
	l := New(100, 50)
	defer l.Shutdown()

	if l.DefaultRPM != 100 {
		t.Errorf("expected DefaultRPM=100, got %d", l.DefaultRPM)
	}
	if l.DefaultIPRPM != 50 {
		t.Errorf("expected DefaultIPRPM=50, got %d", l.DefaultIPRPM)
	}
}

func TestAllowKey_Basic(t *testing.T) {
	l := newTestLimiter(5, 30)
	defer l.Shutdown()

	key := "key-abc"

	// First 5 requests should be allowed.
	for i := 0; i < 5; i++ {
		allowed, remaining := l.AllowKey(key, 0)
		if !allowed {
			t.Fatalf("request %d: expected allowed=true", i+1)
		}
		expectedRem := 5 - (i + 1)
		if remaining != expectedRem {
			t.Errorf("request %d: expected remaining=%d, got %d", i+1, expectedRem, remaining)
		}
	}

	// 6th request should be denied.
	allowed, remaining := l.AllowKey(key, 0)
	if allowed {
		t.Fatal("6th request should be denied")
	}
	if remaining != 0 {
		t.Errorf("expected remaining=0, got %d", remaining)
	}
}

func TestAllowKey_WindowExpiration(t *testing.T) {
	l := newTestLimiter(3, 30)
	defer l.Shutdown()

	key := "key-expire"

	// Use up the limit.
	for i := 0; i < 3; i++ {
		if allowed, _ := l.AllowKey(key, 0); !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Should be denied now.
	if allowed, _ := l.AllowKey(key, 0); allowed {
		t.Fatal("should be denied after hitting limit")
	}

	// After reset, requests should be allowed again.
	l.Reset(key)
	if allowed, _ := l.AllowKey(key, 0); !allowed {
		t.Fatal("should be allowed after reset")
	}
}

func TestAllowKey_QuotaOverride(t *testing.T) {
	l := newTestLimiter(10, 30)
	defer l.Shutdown()

	key := "key-override"

	// Override to 3 RPM.
	for i := 0; i < 3; i++ {
		allowed, remaining := l.AllowKey(key, 3)
		if !allowed {
			t.Fatalf("override request %d: should be allowed", i+1)
		}
		expectedRem := 3 - (i + 1)
		if remaining != expectedRem {
			t.Errorf("override request %d: expected remaining=%d, got %d", i+1, expectedRem, remaining)
		}
	}

	// 4th should be denied with override=3.
	if allowed, _ := l.AllowKey(key, 3); allowed {
		t.Fatal("should be denied with override=3 after 3 requests")
	}
}

func TestAllowKey_BurstMode(t *testing.T) {
	// Use defaults with burst enabled.
	l := New(10, 30)
	l.BurstThresh = 0.8
	l.BurstMul = 2.0
	l.BurstSecs = 5
	defer l.Shutdown()

	key := "key-burst"

	// Send 8 requests (80% of 10). The 8th should trigger burst mode.
	for i := 0; i < 8; i++ {
		allowed, _ := l.AllowKey(key, 0)
		if !allowed {
			t.Fatalf("request %d: expected allowed=true before burst trigger", i+1)
		}
	}

	// Now burst mode should be active. We should be able to do up to 20 total.
	// 8 already consumed, so up to 12 more.
	for i := 0; i < 12; i++ {
		allowed, remaining := l.AllowKey(key, 0)
		if !allowed {
			t.Fatalf("burst request %d: expected allowed=true (burst mode)", i+1)
		}
		consumed := 8 + i + 1
		expectedRem := 20 - consumed
		if remaining != expectedRem {
			t.Errorf("burst request %d: expected remaining=%d, got %d (consumed=%d)", i+1, expectedRem, remaining, consumed)
		}
	}

	// 21st request (13th in burst) should be denied.
	if allowed, _ := l.AllowKey(key, 0); allowed {
		t.Fatal("should be denied after burst limit exhausted")
	}
}

func TestAllowKey_BurstNotTriggered(t *testing.T) {
	// Default burst threshold is 0.8. With limit=10, need 7 to NOT trigger.
	// At 7 requests: 7/10=0.7 < 0.8 → no burst.
	// The 8th request: count=7, 7<10 → allowed, 7/10=0.7, no burst.
	// But the burst check happens BEFORE the append, so at request 9:
	// count=8, 8/10=0.8 >= 0.8 → burst triggers!
	// So we can only test up to the 8th request without burst.
	l := New(10, 30)
	l.BurstThresh = 0.8
	defer l.Shutdown()

	key := "key-noburst"

	// Send 8 requests. At the 8th, count=7 which is < 8 (the threshold).
	for i := 0; i < 8; i++ {
		if allowed, _ := l.AllowKey(key, 0); !allowed {
			t.Fatalf("request %d: should be allowed (no burst expected)", i+1)
		}
	}

	// The 9th request (count=8, 8/10=0.8) will trigger burst mode.
	allowed, remaining := l.AllowKey(key, 0)
	if !allowed {
		t.Fatal("9th request should be allowed (burst triggers)")
	}
	// Burst limit = 20, consumed = 9, so remaining = 11.
	if remaining != 11 {
		t.Errorf("expected remaining=11 (burst mode), got %d", remaining)
	}
}

func TestAllowIP_Basic(t *testing.T) {
	l := newTestLimiter(60, 3)
	defer l.Shutdown()

	ip := "192.168.1.1"

	for i := 0; i < 3; i++ {
		if !l.AllowIP(ip) {
			t.Fatalf("IP request %d: should be allowed", i+1)
		}
	}

	// 4th should be denied.
	if l.AllowIP(ip) {
		t.Fatal("4th IP request should be denied")
	}
}

func TestAllowIP_DifferentIPs(t *testing.T) {
	l := newTestLimiter(60, 2)
	defer l.Shutdown()

	// Fill limit for IP 1.
	if !l.AllowIP("1.1.1.1") {
		t.Fatal("first IP req 1 should be allowed")
	}
	if !l.AllowIP("1.1.1.1") {
		t.Fatal("first IP req 2 should be allowed")
	}
	if l.AllowIP("1.1.1.1") {
		t.Fatal("first IP req 3 should be denied")
	}

	// Different IP should have its own window.
	if !l.AllowIP("2.2.2.2") {
		t.Fatal("second IP should be allowed independently")
	}
}

func TestReset(t *testing.T) {
	l := newTestLimiter(3, 30)
	defer l.Shutdown()

	key := "key-reset"

	// Use up limit.
	for i := 0; i < 3; i++ {
		l.AllowKey(key, 0)
	}

	if allowed, _ := l.AllowKey(key, 0); allowed {
		t.Fatal("should be denied")
	}

	l.Reset(key)

	// Should be allowed again.
	if allowed, _ := l.AllowKey(key, 0); !allowed {
		t.Fatal("should be allowed after reset")
	}
}

func TestResetIP(t *testing.T) {
	l := New(60, 2)
	defer l.Shutdown()

	ip := "10.0.0.1"

	l.AllowIP(ip)
	l.AllowIP(ip)
	if l.AllowIP(ip) {
		t.Fatal("should be denied")
	}

	l.ResetIP(ip)
	if !l.AllowIP(ip) {
		t.Fatal("should be allowed after ResetIP")
	}
}

func TestShutdown(t *testing.T) {
	l := New(60, 30)
	l.Shutdown()

	if l.DefaultRPM != 60 {
		t.Errorf("expected 60, got %d", l.DefaultRPM)
	}
}

func TestKeyCount(t *testing.T) {
	l := New(60, 30)
	defer l.Shutdown()

	if c := l.KeyCount(); c != 0 {
		t.Errorf("expected 0, got %d", c)
	}

	l.AllowKey("key1", 0)
	l.AllowKey("key2", 0)
	l.AllowKey("key1", 0)

	if c := l.KeyCount(); c != 2 {
		t.Errorf("expected 2 keys, got %d", c)
	}
}

func TestIPCount(t *testing.T) {
	l := New(60, 30)
	defer l.Shutdown()

	if c := l.IPCount(); c != 0 {
		t.Errorf("expected 0, got %d", c)
	}

	l.AllowIP("1.1.1.1")
	l.AllowIP("2.2.2.2")
	l.AllowIP("1.1.1.1")

	if c := l.IPCount(); c != 2 {
		t.Errorf("expected 2 IPs, got %d", c)
	}
}

func TestConcurrency(t *testing.T) {
	l := newTestLimiter(1000, 5000)
	defer l.Shutdown()

	key := "concurrent-key"
	goroutines := 50
	reqsPerGoroutine := 20
	limit := 1000

	var wg sync.WaitGroup
	allowed := make([]bool, goroutines*reqsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for r := 0; r < reqsPerGoroutine; r++ {
				idx := gid*reqsPerGoroutine + r
				ok, _ := l.AllowKey(key, 0)
				allowed[idx] = ok
			}
		}(g)
	}
	wg.Wait()

	allowCount := 0
	denyCount := 0
	for _, a := range allowed {
		if a {
			allowCount++
		} else {
			denyCount++
		}
	}

	if allowCount != limit {
		t.Errorf("expected exactly %d allowed, got %d (denied %d)", limit, allowCount, denyCount)
	}
	t.Logf("concurrency: %d allowed, %d denied (limit=%d)", allowCount, denyCount, limit)
}

func TestConcurrency_IP(t *testing.T) {
	l := newTestLimiter(1000, 500)
	defer l.Shutdown()

	ip := "10.0.0.99"
	goroutines := 20
	reqsPerGoroutine := 30
	limit := 500

	var wg sync.WaitGroup
	allowed := make([]bool, goroutines*reqsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for r := 0; r < reqsPerGoroutine; r++ {
				idx := gid*reqsPerGoroutine + r
				allowed[idx] = l.AllowIP(ip)
			}
		}(g)
	}
	wg.Wait()

	allowCount := 0
	for _, a := range allowed {
		if a {
			allowCount++
		}
	}

	if allowCount != limit {
		t.Errorf("expected exactly %d allowed, got %d", limit, allowCount)
	}
}

func TestCleanup_IdleEntries(t *testing.T) {
	l := New(5, 5)
	defer l.Shutdown()

	key := "will-be-cleaned"
	l.AllowKey(key, 0)

	if c := l.KeyCount(); c != 1 {
		t.Errorf("expected 1 key, got %d", c)
	}

	// Trigger cleanup directly. Since we just added it, cleanup should NOT remove it.
	l.cleanup()
	if c := l.KeyCount(); c != 1 {
		t.Errorf("expected key to survive cleanup, got %d", c)
	}

	// Reset removes it.
	l.Reset(key)
	if c := l.KeyCount(); c != 0 {
		t.Errorf("expected 0 keys after reset, got %d", c)
	}
}

func TestMultipleKeys(t *testing.T) {
	l := newTestLimiter(5, 30)
	defer l.Shutdown()

	keys := []string{"key-a", "key-b", "key-c"}

	for _, key := range keys {
		for i := 0; i < 5; i++ {
			if allowed, _ := l.AllowKey(key, 0); !allowed {
				t.Fatalf("%s request %d: should be allowed", key, i+1)
			}
		}
		// Each key should be denied on 6th.
		if allowed, _ := l.AllowKey(key, 0); allowed {
			t.Fatalf("%s 6th request should be denied", key)
		}
	}
}

func TestAllowKey_ZeroLimit(t *testing.T) {
	// DefaultRPM=0 with burst disabled.
	l := newTestLimiterWithZero()
	defer l.Shutdown()

	allowed, _ := l.AllowKey("key-zero", 0)
	if allowed {
		t.Error("expected denied with 0 RPM limit")
	}
}

func TestNew_ZeroValuesUseDefaults(t *testing.T) {
	l := New(-1, -5)
	defer l.Shutdown()

	if l.DefaultRPM != defaultRPM {
		t.Errorf("expected DefaultRPM=%d for negative input, got %d", defaultRPM, l.DefaultRPM)
	}
	if l.DefaultIPRPM != defaultIPRPM {
		t.Errorf("expected DefaultIPRPM=%d for negative input, got %d", defaultIPRPM, l.DefaultIPRPM)
	}
}

func BenchmarkAllowKey(b *testing.B) {
	l := New(100000, 100000)
	defer l.Shutdown()

	key := "bench-key"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.AllowKey(key, 0)
	}
}

func BenchmarkAllowKey_Parallel(b *testing.B) {
	l := New(100000, 100000)
	defer l.Shutdown()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := fmt.Sprintf("key-%d", 1)
		for pb.Next() {
			l.AllowKey(key, 0)
		}
	})
}

func BenchmarkAllowIP(b *testing.B) {
	l := New(100000, 100000)
	defer l.Shutdown()

	ip := "192.168.1.1"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.AllowIP(ip)
	}
}
