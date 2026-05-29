package server

import (
	"net/http"
)

func (s *Server) registerDashboardRoutes() {
	// Dashboard API — used by SvelteKit frontend
	mux := s.mux
	mux.HandleFunc("GET /api/dashboard/stats", s.handleDashboardStats)
	mux.HandleFunc("GET /api/dashboard/connections", s.handleDashboardConnections)
	mux.HandleFunc("GET /api/dashboard/cache", s.handleDashboardCache)
	mux.HandleFunc("GET /api/dashboard/combo", s.handleDashboardCombo)
	mux.HandleFunc("GET /api/dashboard/logs", s.handleDashboardLogs)
	mux.HandleFunc("PUT /api/dashboard/settings", s.handleDashboardSettings)

	// Root redirect
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
}
