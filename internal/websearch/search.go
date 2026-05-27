// Package websearch provides web search integration via DuckDuckGo
// Instant Answer API (zero auth) and optionally SerpAPI. Results are
// cached in-memory for 1 hour.
package websearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SearchResult represents a single web search result.
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

// SearchResponse is the top-level response from a web search.
type SearchResponse struct {
	Query       string         `json:"query"`
	Results     []SearchResult `json:"results"`
	ResultCount int            `json:"result_count"`
	Abstract    string         `json:"abstract,omitempty"`     // DuckDuckGo Instant Answer
	AbstractText string        `json:"abstract_text,omitempty"`
	Cached      bool           `json:"cached"`
}

// cachedEntry holds a cached search response with expiry.
type cachedEntry struct {
	resp     SearchResponse
	expiresAt time.Time
}

// Engine performs web searches with caching and optional SerpAPI fallback.
type Engine struct {
	client    *http.Client
	cache     map[string]cachedEntry
	cacheMu   sync.RWMutex
	ttl       time.Duration
	serpAPIKey string
}

// New creates a new search Engine with optional SerpAPI key.
// If serpAPIKey is empty, only DuckDuckGo is used.
func New(serpAPIKey string) *Engine {
	return &Engine{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:      make(map[string]cachedEntry),
		ttl:        1 * time.Hour,
		serpAPIKey: serpAPIKey,
	}
}

// Search performs a web search for the given query, returning up to maxResults.
// Results are cached for 1 hour by query string.
func (e *Engine) Search(query string, maxResults int) SearchResponse {
	if maxResults <= 0 {
		maxResults = 5
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s:%d", query, maxResults)
	e.cacheMu.RLock()
	if entry, ok := e.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		resp := entry.resp
		resp.Cached = true
		e.cacheMu.RUnlock()
		return resp
	}
	e.cacheMu.RUnlock()

	var resp SearchResponse

	// Try SerpAPI if key is configured
	if e.serpAPIKey != "" {
		if sr, err := e.searchSerpAPI(query, maxResults); err == nil {
			resp = sr
		}
	}

	// Fall back to DuckDuckGo
	if len(resp.Results) == 0 {
		resp = e.searchDuckDuckGo(query, maxResults)
	}

	// Cache the result
	e.cacheMu.Lock()
	e.cache[cacheKey] = cachedEntry{
		resp:     resp,
		expiresAt: time.Now().Add(e.ttl),
	}
	e.cacheMu.Unlock()

	return resp
}

// searchDuckDuckGo queries the DuckDuckGo Instant Answer API.
func (e *Engine) searchDuckDuckGo(query string, maxResults int) SearchResponse {
	resp := SearchResponse{
		Query:   query,
		Results: make([]SearchResult, 0),
	}

	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return resp
	}
	req.Header.Set("User-Agent", "Lintasan/2.0")

	httpResp, err := e.client.Do(req)
	if err != nil {
		return resp
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return resp
	}

	var ddgResp struct {
		Abstract     string `json:"Abstract"`
		AbstractText string `json:"AbstractText"`
		AbstractURL  string `json:"AbstractURL"`
		Heading      string `json:"Heading"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
			Name     string `json:"Name"`
			Topics   []struct {
				Text     string `json:"Text"`
				FirstURL string `json:"FirstURL"`
			} `json:"Topics"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return resp
	}

	resp.Abstract = ddgResp.Abstract
	resp.AbstractText = ddgResp.AbstractText

	// Add instant answer as first result if present
	if ddgResp.AbstractText != "" && ddgResp.AbstractURL != "" {
		resp.Results = append(resp.Results, SearchResult{
			Title:   ddgResp.Heading,
			Snippet: ddgResp.AbstractText,
			URL:     ddgResp.AbstractURL,
		})
	}

	// Flatten RelatedTopics
	for _, topic := range ddgResp.RelatedTopics {
		if topic.Text != "" && topic.FirstURL != "" {
			resp.Results = append(resp.Results, SearchResult{
				Title:   topic.Name,
				Snippet: stripHTML(topic.Text),
				URL:     topic.FirstURL,
			})
		}
		// Nested topics
		for _, sub := range topic.Topics {
			if sub.Text != "" && sub.FirstURL != "" {
				resp.Results = append(resp.Results, SearchResult{
					Title:   "",
					Snippet: stripHTML(sub.Text),
					URL:     sub.FirstURL,
				})
			}
		}
	}

	// Add Results array entries
	for _, r := range ddgResp.Results {
		if r.Text != "" {
			resp.Results = append(resp.Results, SearchResult{
				Snippet: stripHTML(r.Text),
				URL:     r.FirstURL,
			})
		}
	}

	// Truncate to maxResults
	if len(resp.Results) > maxResults {
		resp.Results = resp.Results[:maxResults]
	}

	resp.ResultCount = len(resp.Results)
	return resp
}

// searchSerpAPI searches via SerpAPI (requires API key).
func (e *Engine) searchSerpAPI(query string, maxResults int) (SearchResponse, error) {
	resp := SearchResponse{
		Query:   query,
		Results: make([]SearchResult, 0),
	}

	apiURL := fmt.Sprintf("https://serpapi.com/search?q=%s&api_key=%s&num=%d&engine=google",
		url.QueryEscape(query), e.serpAPIKey, maxResults)

	httpResp, err := e.client.Get(apiURL)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		return resp, fmt.Errorf("serpapi returned status %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return resp, err
	}

	var sr struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Link    string `json:"link"`
		} `json:"organic_results"`
		AnswerBox struct {
			Answer string `json:"answer"`
			Title  string `json:"title"`
			Link   string `json:"link"`
		} `json:"answer_box"`
	}

	if err := json.Unmarshal(body, &sr); err != nil {
		return resp, err
	}

	// Add answer box if present
	if sr.AnswerBox.Answer != "" {
		resp.Results = append(resp.Results, SearchResult{
			Title:   sr.AnswerBox.Title,
			Snippet: sr.AnswerBox.Answer,
			URL:     sr.AnswerBox.Link,
		})
	}

	for _, r := range sr.OrganicResults {
		resp.Results = append(resp.Results, SearchResult{
			Title:   r.Title,
			Snippet: r.Snippet,
			URL:     r.Link,
		})
	}

	resp.ResultCount = len(resp.Results)
	return resp, nil
}

// FormatContext formats search results as context text for injection into LLM prompts.
func FormatContext(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[Web Search Results]\n")
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("   Source: %s\n", r.URL))
		}
	}
	sb.WriteString("\nUse the above search results as context to answer the user's question. Cite sources when relevant.\n")
	return sb.String()
}

// NeedsWebSearch heuristically detects whether a query likely benefits from
// web search results.
func NeedsWebSearch(query string) bool {
	if query == "" {
		return false
	}

	lower := strings.ToLower(query)

	// URL patterns
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "www.") {
		return true
	}

	keywords := []string{
		"latest", "current", "today", "yesterday", "recent",
		"2024", "2025", "2026", "news", "price", "stock",
		"weather", "update", "released", "announced", "launched",
		"trending", "now", "this week", "this month",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	// Question patterns needing current info
	patterns := []string{
		"what is the",
		"who won",
		"who is leading",
		"when does",
		"when did",
		"when will",
		"when is",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	return false
}

// stripHTML removes HTML tags from a string.
func stripHTML(s string) string {
	// Simple regex-based HTML tag removal
	result := strings.Builder{}
	inTag := false
	for _, ch := range s {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(ch)
		}
	}
	return strings.TrimSpace(result.String())
}

// ClearCache clears the search result cache.
func (e *Engine) ClearCache() {
	e.cacheMu.Lock()
	e.cache = make(map[string]cachedEntry)
	e.cacheMu.Unlock()
}

// CacheSize returns the number of cached entries.
func (e *Engine) CacheSize() int {
	e.cacheMu.RLock()
	defer e.cacheMu.RUnlock()
	return len(e.cache)
}
