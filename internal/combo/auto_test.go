package combo

import (
	"testing"
	"time"
)

func TestAutoAliasExists(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"auto", true},
		{"auto/coding", true},
		{"auto/fast", true},
		{"auto/cheap", true},
		{"gpt-4o", false},
		{"claude-sonnet-4", false},
		{"", false},
	}
	for _, tc := range cases {
		got := AutoAliasExists(tc.input)
		if got != tc.want {
			t.Errorf("AutoAliasExists(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestResolveAutoBalanced(t *testing.T) {
	providers := []Provider{
		{ID: "a", Name: "Provider A", Health: 90, QuotaRemaining: 0.8, CostPerToken: 1.0, Latency: 100, SuccessRate: 0.95, LastUsed: time.Now()},
		{ID: "b", Name: "Provider B", Health: 50, QuotaRemaining: 0.2, CostPerToken: 10.0, Latency: 500, SuccessRate: 0.60, LastUsed: time.Now().Add(-2 * time.Hour)},
		{ID: "c", Name: "Provider C", Health: 80, QuotaRemaining: 0.5, CostPerToken: 3.0, Latency: 200, SuccessRate: 0.85, LastUsed: time.Now().Add(-30 * time.Minute)},
	}

	result := ResolveAuto("auto", providers)
	if len(result) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(result))
	}
	// Provider A should rank first (best health, best success rate, best latency)
	if result[0].ID != "a" {
		t.Errorf("expected provider 'a' first, got '%s'", result[0].ID)
	}
}

func TestResolveAutoFastPrioritizesLatency(t *testing.T) {
	providers := []Provider{
		{ID: "slow", Name: "Slow", Health: 95, QuotaRemaining: 0.9, CostPerToken: 0.5, Latency: 2000, SuccessRate: 0.99, LastUsed: time.Now()},
		{ID: "fast", Name: "Fast", Health: 70, QuotaRemaining: 0.5, CostPerToken: 5.0, Latency: 50, SuccessRate: 0.80, LastUsed: time.Now()},
	}

	result := ResolveAuto("auto/fast", providers)
	if result[0].ID != "fast" {
		t.Errorf("auto/fast: expected 'fast' first, got '%s'", result[0].ID)
	}
}

func TestResolveAutoCheapPrioritizesCost(t *testing.T) {
	providers := []Provider{
		{ID: "expensive", Name: "Expensive", Health: 95, QuotaRemaining: 0.9, CostPerToken: 15.0, Latency: 100, SuccessRate: 0.99, LastUsed: time.Now()},
		{ID: "cheap", Name: "Cheap", Health: 70, QuotaRemaining: 0.5, CostPerToken: 0.1, Latency: 300, SuccessRate: 0.80, LastUsed: time.Now()},
	}

	result := ResolveAuto("auto/cheap", providers)
	if result[0].ID != "cheap" {
		t.Errorf("auto/cheap: expected 'cheap' first, got '%s'", result[0].ID)
	}
}

func TestResolveAutoCodingPrioritizesSuccessRate(t *testing.T) {
	providers := []Provider{
		{ID: "unreliable", Name: "Unreliable", Health: 95, QuotaRemaining: 0.9, CostPerToken: 1.0, Latency: 100, SuccessRate: 0.50, LastUsed: time.Now()},
		{ID: "reliable", Name: "Reliable", Health: 80, QuotaRemaining: 0.5, CostPerToken: 5.0, Latency: 300, SuccessRate: 0.99, LastUsed: time.Now()},
	}

	result := ResolveAuto("auto/coding", providers)
	if result[0].ID != "reliable" {
		t.Errorf("auto/coding: expected 'reliable' first, got '%s'", result[0].ID)
	}
}

func TestResolveAutoLGPSticky(t *testing.T) {
	// Record LGP for "auto" mode
	RecordAutoSuccess("auto", "b")

	providers := []Provider{
		{ID: "a", Name: "A", Health: 80, QuotaRemaining: 0.5, CostPerToken: 2.0, Latency: 200, SuccessRate: 0.9, LastUsed: time.Now()},
		{ID: "b", Name: "B", Health: 80, QuotaRemaining: 0.5, CostPerToken: 2.0, Latency: 200, SuccessRate: 0.9, LastUsed: time.Now()},
	}

	result := ResolveAuto("auto", providers)
	// With equal scores, LGP "b" gets a 15% bonus
	if result[0].ID != "b" {
		t.Errorf("LGP sticky: expected 'b' first, got '%s'", result[0].ID)
	}
}

func TestResolveAutoEmpty(t *testing.T) {
	result := ResolveAuto("auto", nil)
	if result != nil {
		t.Errorf("expected nil for empty providers, got %v", result)
	}
}

func TestNormalizeCost(t *testing.T) {
	// Free → 1.0
	if v := normalizeCost(0); v != 1.0 {
		t.Errorf("normalizeCost(0) = %f, want 1.0", v)
	}
	// Cheap should score higher than expensive
	cheap := normalizeCost(0.1)
	expensive := normalizeCost(20.0)
	if cheap <= expensive {
		t.Errorf("cheap (%f) should score higher than expensive (%f)", cheap, expensive)
	}
}

func TestNormalizeLatency(t *testing.T) {
	// No latency → 1.0
	if v := normalizeLatency(0); v != 1.0 {
		t.Errorf("normalizeLatency(0) = %f, want 1.0", v)
	}
	// Low latency should score higher than high latency
	low := normalizeLatency(50)
	high := normalizeLatency(3000)
	if low <= high {
		t.Errorf("low latency (%f) should score higher than high (%f)", low, high)
	}
}

func TestNormalizeFreshness(t *testing.T) {
	// Zero time → neutral 0.5
	if v := normalizeFreshness(time.Time{}); v != 0.5 {
		t.Errorf("normalizeFreshness(zero) = %f, want 0.5", v)
	}
	// Recent → high
	recent := normalizeFreshness(time.Now().Add(-10 * time.Second))
	old := normalizeFreshness(time.Now().Add(-12 * time.Hour))
	if recent <= old {
		t.Errorf("recent (%f) should score higher than old (%f)", recent, old)
	}
}
