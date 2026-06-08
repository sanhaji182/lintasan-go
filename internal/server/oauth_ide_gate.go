package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/sanhaji182/lintasan-go/internal/auth"
)

// oauthIdeDisabledJSON is returned when LINTASAN_OAUTH_IDE_ENABLED is false.
func oauthIdeDisabledJSON(w http.ResponseWriter) {
	writeJSONStatus(w, http.StatusNotFound, map[string]any{
		"error":   "oauth_ide_disabled",
		"hint":    "Set LINTASAN_OAUTH_IDE_ENABLED=true to enable the experimental IDE OAuth lab (admin-only, personal use).",
		"enabled": false,
	})
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	user := auth.GetUser(r)
	if user == nil || user.Role != "admin" {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{"error": "admin access required"})
		return nil, false
	}
	return user, true
}

func (s *Server) oauthIdeEnabled() bool {
	return s.cfg != nil && s.cfg.OAuthIDEEnabled
}

func (s *Server) oauthPublicBaseURL() string {
	if s.cfg != nil && s.cfg.OAuthPublicBaseURL != "" {
		return s.cfg.OAuthPublicBaseURL
	}
	if v := strings.TrimRight(os.Getenv("LINTASAN_OAUTH_PUBLIC_BASE_URL"), "/"); v != "" {
		return v
	}
	return "http://localhost:20180"
}

// isOAuthIdeCallback reports public OAuth redirect handlers (no JWT).
func isOAuthIdeCallback(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	return strings.HasPrefix(path, "/api/oauth/callback/")
}