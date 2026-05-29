package logging

import (
	"fmt"
	"testing"
	"time"
)

func makeLog(method, path, provider, model, ip string, status, tokensIn, tokensOut int, latency float64) AccessLog {
	return AccessLog{
		Timestamp: time.Now(),
		Method:    method,
		Path:      path,
		Status:    status,
		Latency:   latency,
		Provider:  provider,
		Model:     model,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		IP:        ip,
	}
}

func TestRecordAndSearch(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 150.0))

	results := store.Search("", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Provider != "openai" {
		t.Fatalf("expected openai provider, got %s", results[0].Provider)
	}
}

func TestSearchFilter(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 100.0))
	store.Record(makeLog("POST", "/v1/chat/completions", "anthropic", "claude-3", "10.0.0.1", 200, 200, 80, 200.0))
	store.Record(makeLog("GET", "/v1/models", "openai", "gpt-4", "127.0.0.1", 200, 0, 0, 10.0))

	// Search by provider
	results := store.Search("anthropic", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for anthropic, got %d", len(results))
	}

	// Search by model
	results = store.Search("gpt-4", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results for gpt-4, got %d", len(results))
	}

	// Search by IP
	results = store.Search("10.0.0.1", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 10.0.0.1, got %d", len(results))
	}

	// Search by method
	results = store.Search("GET", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for GET, got %d", len(results))
	}
}

func TestLogStats(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 100.0))
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 500, 0, 0, 200.0))
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 300.0))

	stats := store.Stats()
	if stats.Total != 3 {
		t.Fatalf("expected total 3, got %d", stats.Total)
	}
	if stats.AvgLatency != 200.0 {
		t.Fatalf("expected avg latency 200, got %f", stats.AvgLatency)
	}
	expectedErrorRate := 1.0 / 3.0 * 100
	if stats.ErrorRate < expectedErrorRate-0.1 || stats.ErrorRate > expectedErrorRate+0.1 {
		t.Fatalf("expected error rate ~33.3, got %f", stats.ErrorRate)
	}
}

func TestLogStatsEmpty(t *testing.T) {
	store := NewLogStore()
	stats := store.Stats()
	if stats.Total != 0 {
		t.Fatalf("expected total 0, got %d", stats.Total)
	}
}

func TestSearchLimit(t *testing.T) {
	store := NewLogStore()
	for i := 0; i < 100; i++ {
		store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 100.0))
	}

	results := store.Search("", 10)
	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

	results = store.Search("", 200)
	if len(results) != 100 {
		t.Fatalf("expected 100 results, got %d", len(results))
	}
}

func TestRingBufferOverflow(t *testing.T) {
	store := NewLogStore()
	// Write more than maxLogs entries
	for i := 0; i < maxLogs+500; i++ {
		store.Record(makeLog("POST", "/v1/chat/completions", "provider", fmt.Sprintf("model-%d", i), "127.0.0.1", 200, 100, 50, 100.0))
	}

	if store.Count() != maxLogs {
		t.Fatalf("expected count %d, got %d", maxLogs, store.Count())
	}

	// The newest entries should be present
	results := store.Search(fmt.Sprintf("model-%d", maxLogs+499), 10)
	if len(results) != 1 {
		t.Fatalf("expected to find newest entry, got %d", len(results))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 100.0))

	results := store.Search("", 50)
	if len(results) != 1 {
		t.Fatalf("expected 1 result with empty query, got %d", len(results))
	}
}

func TestSearchNoMatch(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 200, 100, 50, 100.0))

	results := store.Search("nonexistent", 50)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for nonexistent query, got %d", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "OpenAI", "GPT-4", "127.0.0.1", 200, 100, 50, 100.0))

	results := store.Search("openai", 50)
	if len(results) != 1 {
		t.Fatalf("expected case-insensitive match, got %d", len(results))
	}
}

func TestSearchNewestFirst(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/first", "p1", "m1", "127.0.0.1", 200, 100, 50, 100.0))
	store.Record(makeLog("POST", "/second", "p2", "m2", "127.0.0.1", 200, 100, 50, 200.0))
	store.Record(makeLog("POST", "/third", "p3", "m3", "127.0.0.1", 200, 100, 50, 300.0))

	results := store.Search("", 10)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// Newest should be first
	if results[0].Path != "/third" {
		t.Fatalf("expected /third first, got %s", results[0].Path)
	}
	if results[2].Path != "/first" {
		t.Fatalf("expected /first last, got %s", results[2].Path)
	}
}

func TestStatsErrorRateAllErrors(t *testing.T) {
	store := NewLogStore()
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 500, 0, 0, 100.0))
	store.Record(makeLog("POST", "/v1/chat/completions", "openai", "gpt-4", "127.0.0.1", 403, 0, 0, 100.0))

	stats := store.Stats()
	if stats.ErrorRate != 100.0 {
		t.Fatalf("expected 100%% error rate, got %f", stats.ErrorRate)
	}
}

func TestCountEmpty(t *testing.T) {
	store := NewLogStore()
	if store.Count() != 0 {
		t.Fatalf("expected count 0, got %d", store.Count())
	}
}
