package auth

import (
	"context"
	"net/http"
	"strings"
)

// ContextKey is a type-safe key for context values.
type ContextKey string

const (
	// UserContextKey is the key for the authenticated user in context.
	UserContextKey ContextKey = "user"
)

// Middleware validates JWT tokens and injects the user into the request context.
// Protected routes should use this middleware.
func (m *UserManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip health check
		if r.URL.Path == "/health" || r.URL.Path == "/api/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Also check for cookie-based auth (dashboard)
			cookie, err := r.Cookie("lintasan_token")
			if err == nil && cookie.Value != "" {
				authHeader = "Bearer " + cookie.Value
			}
		}

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		user, err := m.ValidateToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser extracts the authenticated user from the request context.
func GetUser(r *http.Request) *User {
	user, ok := r.Context().Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// RequireAdmin is middleware that checks if the user is an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CORSHeaders adds permissive CORS headers for dashboard access.
func CORSHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
