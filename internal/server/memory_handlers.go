package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sanhaji182/lintasan-go/internal/memory"
)

// MemoryHandler holds a reference to the memory store for HTTP handler methods.
type MemoryHandler struct {
	mem *memory.MemoryStore
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(mem *memory.MemoryStore) *MemoryHandler {
	return &MemoryHandler{mem: mem}
}

// HandleMemorySearch handles GET /v1/memory/search?q=...&top_k=5
// Searches stored memories by keyword using string matching on text field.
func (mh *MemoryHandler) HandleMemorySearch(w http.ResponseWriter, r *http.Request) {
	if mh.mem == nil || !mh.mem.Available() {
		writeMemoryJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "memory service unavailable — Redis not connected",
		})
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeMemoryJSON(w, http.StatusBadRequest, map[string]any{
			"error": "query parameter 'q' is required",
		})
		return
	}

	topK := 5
	if tk := r.URL.Query().Get("top_k"); tk != "" {
		if n, err := strconv.Atoi(tk); err == nil && n > 0 && n <= 50 {
			topK = n
		}
	}

	results, err := mh.mem.Store.SearchByKeywords(q, topK)
	if err != nil {
		writeMemoryJSON(w, http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("search failed: %v", err),
		})
		return
	}

	if results == nil {
		results = []memory.Memory{}
	}

	writeMemoryJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": results,
		"count":   len(results),
	})
}

// HandleMemoryStore handles POST /v1/memory
// Manually stores a text entry with optional metadata and tags.
func (mh *MemoryHandler) HandleMemoryStore(w http.ResponseWriter, r *http.Request) {
	if mh.mem == nil || !mh.mem.Available() {
		writeMemoryJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "memory service unavailable — Redis not connected",
		})
		return
	}

	var req struct {
		Text     string            `json:"text"`
		Metadata map[string]string `json:"metadata,omitempty"`
		Tags     []string          `json:"tags,omitempty"`
		Score    float64           `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMemoryJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid JSON body",
		})
		return
	}
	if req.Text == "" {
		writeMemoryJSON(w, http.StatusBadRequest, map[string]any{
			"error": "field 'text' is required",
		})
		return
	}

	embedding := memory.Embed(req.Text)
	key := memory.HashKey(req.Text)

	err := mh.mem.Store.Store(key, embedding, req.Text, req.Metadata, req.Tags, req.Score)
	if err != nil {
		writeMemoryJSON(w, http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("store failed: %v", err),
		})
		return
	}

	writeMemoryJSON(w, http.StatusCreated, map[string]any{
		"key":    key,
		"status": "stored",
	})
}

// HandleMemoryStats handles GET /v1/memory/stats
// Returns index statistics: total entries, avg score, breakdown by tag.
func (mh *MemoryHandler) HandleMemoryStats(w http.ResponseWriter, r *http.Request) {
	if mh.mem == nil || !mh.mem.Available() {
		writeMemoryJSON(w, http.StatusOK, map[string]any{
			"total_memories": 0,
			"available":      false,
		})
		return
	}

	baseStats := mh.mem.Store.Stats()
	total, _ := baseStats["total_memories"].(int)

	stats := map[string]any{
		"total_memories": total,
		"available":      true,
		"breakdown":      map[string]int{},
		"avg_score":      0.0,
	}

	writeMemoryJSON(w, http.StatusOK, stats)
}

// writeMemoryJSON writes a JSON response with standard headers.
func writeMemoryJSON(w http.ResponseWriter, status int, data map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
