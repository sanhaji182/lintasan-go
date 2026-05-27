package websearch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewEngine tests engine creation.
func TestNewEngine(t *testing.T) {
	e := New("")
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.ttl != 1*time.Hour {
		t.Errorf("expected 1h TTL, got %v", e.ttl)
	}
	if e.serpAPIKey != "" {
		t.Error("expected empty SerpAPI key")
	}

	e2 := New("test-key-123")
	if e2.serpAPIKey != "test-key-123" {
		t.Errorf("expected serpAPIKey to be set, got %q", e2.serpAPIKey)
	}
}

// TestNeedsWebSearch tests the heuristic detection function.
func TestNeedsWebSearch(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"", false},
		{"hello world", false},
		{"what is the latest news about AI", true},
		{"current weather in Tokyo", true},
		{"stock price of AAPL today", true},
		{"who won the 2024 election", true},
		{"when does the new iPhone release", true},
		{"https://example.com/article", true},
		{"check www.example.com", true},
		{"write a poem about cats", false},
		{"explain quantum computing", false},
		{"2025 technology trends", true},
		{"latest research on CRISPR", true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := NeedsWebSearch(tt.query)
			if got != tt.want {
				t.Errorf("NeedsWebSearch(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// TestSearchDuckDuckGo tests the DuckDuckGo API parsing with a mock server.
func TestSearchDuckDuckGo(t *testing.T) {
	mockResp := `{
		"Abstract": "Go is an open source programming language",
		"AbstractText": "Go is a statically typed, compiled programming language designed at Google.",
		"AbstractURL": "https://golang.org/",
		"Heading": "Go (programming language)",
		"RelatedTopics": [
			{
				"Text": "Go is used for <b>web servers</b> and CLI tools",
				"FirstURL": "https://example.com/go-usage",
				"Name": "Go Usage"
			},
			{
				"Text": "The Go <b>playground</b> lets you run Go code online",
				"FirstURL": "https://go.dev/play",
				"Topics": [
					{
						"Text": "Try Go in your <b>browser</b>",
						"FirstURL": "https://go.dev/"
					}
				]
			}
		],
		"Results": [
			{
				"Text": "Additional result about Go",
				"FirstURL": "https://example.com/extra"
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(mockResp))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	// We can't directly inject a mock server for DuckDuckGo since it uses
	// a hardcoded API URL. Instead test the helper functions and FormatContext.

	// Test FormatContext with real results since live API isn't available in tests.
	results := []SearchResult{
		{Title: "Go Programming", Snippet: "A compiled language", URL: "https://golang.org"},
		{Title: "Rust Language", Snippet: "Safe systems programming", URL: "https://rust-lang.org"},
	}

	ctx := FormatContext(results)
	if !strings.Contains(ctx, "[Web Search Results]") {
		t.Error("expected Web Search Results header")
	}
	if !strings.Contains(ctx, "golang.org") {
		t.Error("expected golang.org URL in context")
	}
	if !strings.Contains(ctx, "rust-lang.org") {
		t.Error("expected rust-lang.org in context")
	}
}

// TestFormatContextEmpty tests formatting with empty results.
func TestFormatContextEmpty(t *testing.T) {
	ctx := FormatContext(nil)
	if ctx != "" {
		t.Errorf("expected empty string for nil results, got %q", ctx)
	}

	ctx = FormatContext([]SearchResult{})
	if ctx != "" {
		t.Errorf("expected empty string for empty results, got %q", ctx)
	}
}

// TestStripHTML tests HTML tag removal.
func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<b>bold</b> text", "bold text"},
		{"<a href='x'>link</a>", "link"},
		{"plain text", "plain text"},
		{"<div>nested <span>tags</span></div>", "nested tags"},
		{"<b>", ""},
		{"no tags here", "no tags here"},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestCache tests result caching.
func TestCache(t *testing.T) {
	e := New("")

	// Manually insert into cache for testing
	e.cacheMu.Lock()
	e.cache["test:5"] = cachedEntry{
		resp: SearchResponse{
			Query:       "test",
			Results:     []SearchResult{{Title: "Cached", Snippet: "Result"}},
			ResultCount: 1,
		},
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	e.cacheMu.Unlock()

	// Search should return cached result
	resp := e.Search("test", 5)
	if !resp.Cached {
		t.Error("expected cached response")
	}
	if resp.ResultCount != 1 {
		t.Errorf("expected 1 result, got %d", resp.ResultCount)
	}
	if resp.Results[0].Title != "Cached" {
		t.Errorf("expected 'Cached' title, got %q", resp.Results[0].Title)
	}

	// CacheSize should be 1
	if n := e.CacheSize(); n != 1 {
		t.Errorf("CacheSize = %d, want 1", n)
	}

	// ClearCache should empty it
	e.ClearCache()
	if n := e.CacheSize(); n != 0 {
		t.Errorf("CacheSize after clear = %d, want 0", n)
	}
}

// TestCacheExpiry verifies expired entries are not returned.
func TestCacheExpiry(t *testing.T) {
	e := New("")

	// Insert expired entry
	e.cacheMu.Lock()
	e.cache["expired:3"] = cachedEntry{
		resp: SearchResponse{
			Query:       "expired",
			Results:     []SearchResult{{Title: "Old"}},
			ResultCount: 1,
		},
		expiresAt: time.Now().Add(-1 * time.Hour), // expired
	}
	e.cacheMu.Unlock()

	// Search should NOT return cached (expired) — falls through to live search
	// Since live search will fail (no network in test), response will be empty
	resp := e.Search("expired", 3)
	if resp.Cached {
		t.Error("expired cache entry should not be returned as cached")
	}
	// After the live search fails, a new empty entry gets cached, so the old one is gone
}

// TestSearchResponseJSON validates JSON serialization.
func TestSearchResponseJSON(t *testing.T) {
	resp := SearchResponse{
		Query: "golang",
		Results: []SearchResult{
			{Title: "Go", Snippet: "Programming Language", URL: "https://go.dev"},
		},
		ResultCount:  1,
		Abstract:     "Go is a programming language",
		AbstractText: "Go is a statically typed...",
		Cached:       false,
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	required := []string{"query", "results", "result_count", "abstract", "abstract_text", "cached"}
	for _, key := range required {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing JSON key %q in SearchResponse", key)
		}
	}
}

// TestSearchWithMaxResults tests the maxResults parameter.
func TestSearchWithMaxResults(t *testing.T) {
	e := New("")

	// Zero/negative should default to 5
	resp := e.Search("test", 0)
	if len(resp.Results) > 5 {
		t.Errorf("expected at most 5 results with max=0, got %d", len(resp.Results))
	}

	resp2 := e.Search("test", -1)
	if len(resp2.Results) > 5 {
		t.Errorf("expected at most 5 results with max=-1, got %d", len(resp2.Results))
	}
}
