package server

import (
	"encoding/json"
	"net/http"
)

// handleProviderDiscover scans localhost for free LLM providers and returns
// discovery results. Auto-registers any found providers as connections.
func (s *Server) handleProviderDiscover(w http.ResponseWriter, r *http.Request) {
	if s.fpScanner == nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": "scanner not initialized"})
		return
	}

	resp := s.fpScanner.Discover()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleWebSearch performs a web search via DuckDuckGo (and optionally SerpAPI)
// and returns formatted results. Accepts JSON body: {query, max_results}.
func (s *Server) handleWebSearch(w http.ResponseWriter, r *http.Request) {
	if s.webSearch == nil {
		writeJSONStatus(w, http.StatusInternalServerError, map[string]string{"error": "web search not initialized"})
		return
	}

	var req struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if req.Query == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": "query is required"})
		return
	}

	if req.MaxResults <= 0 {
		req.MaxResults = 5
	}

	resp := s.webSearch.Search(req.Query, req.MaxResults)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeJSONStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
