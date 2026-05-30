// Package web embeds the built SvelteKit dashboard (static SPA) into the Go
// binary so `lintasan start` serves the full UI from a single executable — no
// separate Node process required.
//
// The dashboard is a pure client-rendered SPA (ssr=false on every route, auth
// guard runs in the browser via /api/auth/me). We serve real files when they
// exist and fall back to index.html for any unknown non-asset path so the
// client-side router can take over (standard SPA hosting behaviour).
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// dist is the embedded build output rooted at the dist/ directory.
var dist fs.FS

// Available reports whether a real dashboard build was embedded. When the
// dist/ tree only contains the placeholder (no index.html), the binary still
// builds and runs as an API-only server.
var available bool

func init() {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return
	}
	dist = sub
	if _, err := fs.Stat(dist, "index.html"); err == nil {
		available = true
	}
}

// Available reports whether the embedded SPA is present and serveable.
func Available() bool { return available }

// indexHTML returns the SPA shell used as the fallback for client-side routes.
func indexHTML() ([]byte, error) {
	return fs.ReadFile(dist, "index.html")
}

// Handler returns an http.Handler that serves the embedded SPA with index.html
// fallback. It is mounted at "GET /" — Go 1.22's ServeMux gives more specific
// patterns (/api/, /v1/, /health, ...) priority, so this never shadows the API.
//
// Routing rules:
//   - exact file hit (e.g. /favicon.png, /_app/...) → serve the file
//   - "/" or any path without a file extension       → serve index.html (SPA)
//   - asset-looking path (has extension) that is missing → 404
func Handler() http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !available {
			http.Error(w, "dashboard UI not embedded in this build", http.StatusNotFound)
			return
		}

		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if reqPath == "" {
			serveIndex(w, r)
			return
		}

		// If the exact file exists in the embedded FS, serve it (hashed assets,
		// favicon, robots.txt, etc) with proper content-type + caching.
		if f, err := dist.Open(reqPath); err == nil {
			if st, serr := f.Stat(); serr == nil && !st.IsDir() {
				f.Close()
				// Immutable, content-hashed assets can be cached aggressively.
				if strings.HasPrefix(reqPath, "_app/immutable/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
			f.Close()
		}

		// A missing path that *looks* like a static asset (has a file
		// extension) is a genuine 404; anything else is a client-side route, so
		// serve the SPA shell and let the browser router handle it.
		if ext := path.Ext(reqPath); ext != "" {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	html, err := indexHTML()
	if err != nil {
		http.Error(w, "dashboard index unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The shell must never be cached — it references hashed asset filenames that
	// change every build.
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(html)
}
