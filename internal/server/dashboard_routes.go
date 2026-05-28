package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dashboard/static/*
var staticFS embed.FS

func (s *Server) registerDashboardRoutes() {
	// Dashboard API
	mux := s.mux
	mux.HandleFunc("GET /api/dashboard/stats", s.handleDashboardStats)
	mux.HandleFunc("GET /api/dashboard/connections", s.handleDashboardConnections)
	mux.HandleFunc("GET /api/dashboard/cache", s.handleDashboardCache)
	mux.HandleFunc("GET /api/dashboard/combo", s.handleDashboardCombo)
	mux.HandleFunc("GET /api/dashboard/logs", s.handleDashboardLogs)
	mux.HandleFunc("PUT /api/dashboard/settings", s.handleDashboardSettings)

	// Page routes — Go native HTMX dashboard
	pages := []string{
		"overview", "connections", "routing", "fallback",
		"logs", "usage", "analytics", "metrics", "keys", "teams", "users",
		"webhooks", "backup", "settings", "plugins", "playground",
		"docs",
	}
	for _, p := range pages {
		page := p
		mux.HandleFunc("GET /dashboard/"+page, func(w http.ResponseWriter, r *http.Request) {
			if s.dashboardEngine != nil {
				// Check if HTMX request — render only partial (no base layout)
				htmxOnly := r.Header.Get("HX-Request") == "true"
				s.dashboardEngine.renderPage(w, page, nil, htmxOnly)
				return
			}
			if s.nodeProxy != nil {
				s.nodeProxy.ServeHTTP(w, r)
			}
		})
	}

	// Dashboard index → overview
	mux.HandleFunc("GET /dashboard", func(w http.ResponseWriter, r *http.Request) {
		if s.dashboardEngine != nil {
			htmxOnly := r.Header.Get("HX-Request") == "true"
			s.dashboardEngine.renderPage(w, "overview", nil, htmxOnly)
			return
		}
		if s.nodeProxy != nil {
			s.nodeProxy.ServeHTTP(w, r)
		}
	})

	// Login
	mux.HandleFunc("GET /dashboard/login", func(w http.ResponseWriter, r *http.Request) {
		if s.dashboardEngine != nil {
			s.dashboardEngine.RenderPage(w, "login", nil)
			return
		}
		if s.nodeProxy != nil {
			s.nodeProxy.ServeHTTP(w, r)
		}
	})

	// Static files (CSS, JS)
	staticSub, _ := fs.Sub(staticFS, "dashboard/static")
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticSub)))
	mux.HandleFunc("GET /static/{path...}", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".html") || strings.HasSuffix(r.URL.Path, ".gohtml") {
			http.NotFound(w, r)
			return
		}
		staticHandler.ServeHTTP(w, r)
	})

	// Root redirect
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if s.nodeProxy != nil {
				s.nodeProxy.ServeHTTP(w, r)
			}
			return
		}
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
}
