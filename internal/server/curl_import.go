package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// curlImportRequest is the JSON body for POST /api/connections/import-curl.
type curlImportRequest struct {
	Curl string `json:"curl"`
}

// parseCurlCommand extracts URL, headers, and body from a curl command string.
// It handles:
//   - curl <url> [options]
//   - -H / --header "Key: Value"
//   - -d / --data / --data-raw '{"json":"body"}'
//   - Authorization: Bearer <key> → extracts api_key
func parseCurlCommand(raw string) (parsedCurl, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return parsedCurl{}, fmt.Errorf("empty curl command")
	}

	// Strip leading "curl" or "$ curl"
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(raw, "$"), "curl"))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "curl"))

	p := parsedCurl{
		headers: make(map[string]string),
	}

	tokens := tokenizeCurl(raw)
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok {
		case "-H", "--header":
			i++
			if i < len(tokens) {
				val := stripQuotes(tokens[i])
				parts := strings.SplitN(val, ":", 2)
				if len(parts) == 2 {
					k := strings.TrimSpace(parts[0])
					v := strings.TrimSpace(parts[1])
					p.headers[strings.ToLower(k)] = v
					p.headers[k] = v // also keep original casing
				}
			}
		case "-d", "--data", "--data-raw", "--data-binary":
			i++
			if i < len(tokens) {
				p.body = stripQuotes(tokens[i])
			}
		case "-X", "--request":
			i++
			if i < len(tokens) {
				p.method = stripQuotes(tokens[i])
			}
		default:
			// Maybe a URL
			if p.url == "" && (strings.HasPrefix(tok, "http://") || strings.HasPrefix(tok, "https://")) {
				p.url = stripQuotes(tok)
			}
		}
	}

	if p.url == "" {
		return parsedCurl{}, fmt.Errorf("no URL found in curl command")
	}
	if p.method == "" {
		if p.body != "" {
			p.method = "POST"
		} else {
			p.method = "GET"
		}
	}

	// Extract API key from Authorization header
	for k, v := range p.headers {
		kl := strings.ToLower(k)
		if kl == "authorization" {
			// "Bearer sk-xxx" → "sk-xxx"
			parts := strings.SplitN(v, " ", 2)
			if len(parts) == 2 {
				p.apiKey = parts[1]
			} else {
				p.apiKey = v
			}
			p.authHeader = k // keep original casing
			p.authPrefix = parts[0] + " "
		}
	}

	// If no explicit auth, check x-api-key header
	if p.apiKey == "" {
		for k, v := range p.headers {
			kl := strings.ToLower(k)
			if kl == "x-api-key" || kl == "api-key" {
				p.apiKey = v
				p.authHeader = k
				p.authPrefix = ""
			}
		}
	}

	// Parse URL for base_url, chat_path, and name
	u, err := url.Parse(p.url)
	if err != nil {
		return parsedCurl{}, fmt.Errorf("invalid URL: %w", err)
	}
	p.scheme = u.Scheme
	p.host = u.Host
	p.path = u.Path
	p.baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	p.chatPath = u.Path

	// Infer name from host
	p.name = inferNameFromHost(u.Host)

	return p, nil
}

type parsedCurl struct {
	url        string
	scheme     string
	host       string
	path       string
	baseURL    string
	chatPath   string
	method     string
	body       string
	headers    map[string]string
	apiKey     string
	authHeader string
	authPrefix string
	name       string
}

// tokenizeCurl splits a curl command into shell-like tokens.
func tokenizeCurl(raw string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && (inDouble || !inSingle) {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if !inSingle && !inDouble && (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\\') {
			if current.Len() > 0 {
				tok := current.String()
				if ch == '\\' && i+1 < len(raw) && raw[i+1] == '\n' {
					// line continuation — skip
					i++ // skip the \n
					current.Reset()
					continue
				}
				tokens = append(tokens, tok)
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// stripQuotes removes surrounding single or double quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// inferNameFromHost creates a human-readable provider name from a hostname.
// Examples:
//
//	api.openai.com        → "OpenAI"
//	api.deepseek.com      → "DeepSeek"
//	opengateway.gitlawb.com → "OpenGateway"
//	localhost:8080        → "Local"
var tldPattern = regexp.MustCompile(`\.(com|org|net|io|ai|dev|app|co|xyz|info|biz|gg|sh|run|cloud|fun|online|site)$`)

func inferNameFromHost(host string) string {
	// Strip port
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Split by dots
	parts := strings.Split(host, ".")
	if len(parts) == 1 {
		return strings.Title(strings.ToLower(parts[0]))
	}

	// Find the "main" domain part (before TLD)
	// e.g. api.openai.com → openai
	// e.g. opengateway.gitlawb.com → opengateway
	// e.g. api.example.co.uk → (best effort: example)

	// Try to find the part before known TLDs
	for i := len(parts) - 1; i >= 1; i-- {
		suffix := strings.Join(parts[i:], ".")
		if tldPattern.MatchString(suffix) || i == len(parts)-1 {
			// parts[i-1] is the main domain name
			return toTitle(parts[i-1])
		}
	}

	// Fallback: use the second-to-last part
	if len(parts) >= 2 {
		return toTitle(parts[len(parts)-2])
	}
	return toTitle(parts[0])
}

func toTitle(s string) string {
	if len(s) == 0 {
		return s
	}
	// Simple title casing: capitalize first letter of each segment
	segments := strings.Split(s, "-")
	for i, seg := range segments {
		if len(seg) > 0 {
			segments[i] = strings.ToUpper(seg[:1]) + strings.ToLower(seg[1:])
		}
	}
	return strings.Join(segments, "")
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

func (s *Server) handleCurlImport(w http.ResponseWriter, r *http.Request) {
	var input curlImportRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if input.Curl == "" {
		http.Error(w, `{"error":"curl field is required"}`, http.StatusBadRequest)
		return
	}

	parsed, err := parseCurlCommand(input.Curl)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Apply defaults
	name := parsed.name
	if name == "" {
		name = "Imported-" + uuid.New().String()[:8]
	}
	format := "openai"
	chatPath := parsed.chatPath
	if chatPath == "" {
		chatPath = "/v1/chat/completions"
	}
	modelsPath := "/v1/models"
	authHeader := parsed.authHeader
	if authHeader == "" {
		authHeader = "Authorization"
	}
	authPrefix := parsed.authPrefix
	if authPrefix == "" {
		authPrefix = "Bearer "
	}

	id := uuid.New().String()
	_, err = s.db.Conn().Exec(
		`INSERT INTO connections (id, name, base_url, api_key, format, priority, chat_path, models_path, auth_header, auth_prefix) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, parsed.baseURL, parsed.apiKey, format, 50, chatPath, modelsPath, authHeader, authPrefix,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, "failed to create connection: "+err.Error()), http.StatusInternalServerError)
		return
	}

	// Auto-discover models
	var syncResult *struct {
		ModelsCount int    `json:"models_count"`
		Status      string `json:"status"`
		Error       string `json:"error,omitempty"`
	}
	if s.discoverer != nil {
		res, err := s.discoverer.SyncConnection(id)
		if err != nil {
			syncResult = &struct {
				ModelsCount int    `json:"models_count"`
				Status      string `json:"status"`
				Error       string `json:"error,omitempty"`
			}{Status: "error", Error: err.Error()}
		} else {
			syncResult = &struct {
				ModelsCount int    `json:"models_count"`
				Status      string `json:"status"`
				Error       string `json:"error,omitempty"`
			}{ModelsCount: res.ModelsCount, Status: res.Status, Error: res.Error}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp := map[string]any{
		"id":      id,
		"name":    name,
		"base_url": parsed.baseURL,
		"format":  format,
		"status":  "created",
	}
	if syncResult != nil {
		resp["discovery"] = syncResult
	}
	json.NewEncoder(w).Encode(map[string]any{"data": resp})
}
