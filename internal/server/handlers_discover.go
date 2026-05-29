package server

import (
	"encoding/json"
	"net/http"

	"github.com/sanhaji182/lintasan-go/internal/discover"
)

// handleDiscoverFreeProviders handles GET /api/discover/free-providers
func (s *Server) handleDiscoverFreeProviders(w http.ResponseWriter, r *http.Request) {
	providers := discover.GetFreeProviders()

	type providerInfo struct {
		discover.FreeProvider
		Models []string `json:"models"`
	}

	var result []providerInfo
	for _, p := range providers {
		result = append(result, providerInfo{
			FreeProvider: p,
			Models:       p.Models,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"providers": result,
		"total":     len(result),
	})
}
