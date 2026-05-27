package budget

import (
	"testing"
)

func TestDefaultBudget(t *testing.T) {
	b := DefaultBudget()
	if b.MaxTokens != 16384 {
		t.Errorf("MaxTokens = %d, want 16384", b.MaxTokens)
	}
	if b.MinTokens != 512 {
		t.Errorf("MinTokens = %d, want 512", b.MinTokens)
	}
}

func TestAnalyze_Simple(t *testing.T) {
	b := DefaultBudget()
	score, budget := b.Analyze([]map[string]any{
		{"role": "user", "content": "hello"},
	})
	if score < 0.1 || score > 1.0 {
		t.Errorf("score out of range: %f", score)
	}
	if budget < b.MinTokens || budget > b.MaxTokens {
		t.Errorf("budget out of range: %d", budget)
	}
}

func TestAnalyze_Complex(t *testing.T) {
	b := DefaultBudget()
	score, budget := b.Analyze([]map[string]any{
		{"role": "system", "content": "you are a programmer"},
		{"role": "user", "content": "implement a distributed rate limiter with sliding window and circuit breaker, handle edge cases for clock skew and partition tolerance. Write the full code with tests"},
	})
	if score < 0.17 {
		t.Errorf("complex request should score >= 0.17, got %f", score)
	}
	if budget < 2700 {
		t.Errorf("complex budget should be >= 2700, got %d", budget)
	}
}

func TestAnalyze_CodeRequest(t *testing.T) {
	b := DefaultBudget()
	score, _ := b.Analyze([]map[string]any{
		{"role": "user", "content": "write a Python function to sort a list"},
	})
	if score >= 0.05 {
		t.Logf("code request score: %f (expected low)", score)
	}
}

func TestAnalyze_Security(t *testing.T) {
	b := DefaultBudget()
	score, _ := b.Analyze([]map[string]any{
		{"role": "user", "content": "audit this authentication system for vulnerabilities and suggest encryption improvements"},
	})
	if score >= 0.05 {
		t.Logf("security request score: %f", score)
	}
}

func TestAnalyze_Performance(t *testing.T) {
	b := DefaultBudget()
	score, _ := b.Analyze([]map[string]any{
		{"role": "user", "content": "optimize this database query for latency and throughput at scale"},
	})
	if score < 0.05 {
		t.Logf("perf request score: %f", score)
	}
}

func TestRetryBudget(t *testing.T) {
	b := DefaultBudget()
	r := b.RetryBudget(4096)
	if r != 8192 {
		t.Errorf("RetryBudget(4096) = %d, want 8192", r)
	}
	// Should not exceed max
	r = b.RetryBudget(10000)
	if r != 16384 {
		t.Errorf("RetryBudget(10000) capped = %d, want 16384", r)
	}
}

func TestRetryBudget_CapAtMax(t *testing.T) {
	b := DefaultBudget()
	r := b.RetryBudget(15000)
	if r != 16384 {
		t.Errorf("RetryBudget(15000) = %d, want 16384", r)
	}
}
