package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sanhaji182/lintasan-go/internal/config"
	"github.com/sanhaji182/lintasan-go/internal/db"
)

type ProxyHandler struct {
	cfg    *config.Config
	db     *db.DB
	client *http.Client
}

func NewProxyHandler(cfg *config.Config, database *db.DB) *ProxyHandler {
	return &ProxyHandler{
		cfg: cfg,
		db:  database,
		client: &http.Client{
			Timeout: 300 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
			},
		},
	}
}

type Connection struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	Format     string `json:"format"`
	ChatPath   string `json:"chat_path"`
	AuthHeader string `json:"auth_header"`
	AuthPrefix string `json:"auth_prefix"`
	IsActive   int    `json:"is_active"`
	Priority   int    `json:"priority"`
}

func (p *ProxyHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	model, _ := req["model"].(string)
	if model == "" {
		http.Error(w, `{"error":"model is required"}`, http.StatusBadRequest)
		return
	}

	conn, err := p.findConnectionForModel(model)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"no connection found for model %s"}`, model), http.StatusNotFound)
		return
	}

	stream, _ := req["stream"].(bool)

	// Build upstream URL using connection's chat_path
	chatPath := conn.ChatPath
	if chatPath == "" {
		chatPath = "/v1/chat/completions"
	}
	upstreamURL := strings.TrimRight(conn.BaseURL, "/") + chatPath

	upReq, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, `{"error":"failed to create upstream request"}`, http.StatusInternalServerError)
		return
	}

	upReq.Header.Set("Content-Type", "application/json")

	// Use connection's auth config
	authHeader := conn.AuthHeader
	if authHeader == "" {
		authHeader = "Authorization"
	}
	authPrefix := conn.AuthPrefix
	if authPrefix == "" {
		authPrefix = "Bearer "
	}
	if conn.APIKey != "" {
		upReq.Header.Set(authHeader, authPrefix+conn.APIKey)
	}

	// Forward extra headers from original request
	if xcc := r.Header.Get("X-Command-Code-Version"); xcc != "" {
		upReq.Header.Set("X-Command-Code-Version", xcc)
	}

	resp, err := p.client.Do(upReq)
	if err != nil {
		latency := time.Since(start).Milliseconds()
		p.logRequest(model, conn.ID, conn.Name, 502, latency, 0, 0, false, err.Error())
		http.Error(w, fmt.Sprintf(`{"error":"upstream error: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(resp.StatusCode)

		flusher, ok := w.(http.Flusher)
		if !ok {
			io.Copy(w, resp.Body)
			return
		}

		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	} else {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}

	latency := time.Since(start).Milliseconds()
	p.logRequest(model, conn.ID, conn.Name, resp.StatusCode, latency, 0, 0, false, "")
}

func (p *ProxyHandler) HandleEmbeddings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	model, _ := req["model"].(string)
	if model == "" {
		model = "text-embedding-3-small"
	}

	conn, err := p.findConnectionForModel(model)
	if err != nil {
		// Fallback: use first active connection
		conn, err = p.getFirstConnection()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"no connection found for model %s"}`, model), http.StatusNotFound)
			return
		}
	}

	upstreamURL := strings.TrimRight(conn.BaseURL, "/") + "/v1/embeddings"
	upReq, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, `{"error":"failed to create upstream request"}`, http.StatusInternalServerError)
		return
	}

	upReq.Header.Set("Content-Type", "application/json")
	authHeader := conn.AuthHeader
	if authHeader == "" {
		authHeader = "Authorization"
	}
	authPrefix := conn.AuthPrefix
	if authPrefix == "" {
		authPrefix = "Bearer "
	}
	if conn.APIKey != "" {
		upReq.Header.Set(authHeader, authPrefix+conn.APIKey)
	}

	resp, err := p.client.Do(upReq)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"upstream error: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ProxyHandler) findConnectionForModel(model string) (*Connection, error) {
	row := p.db.Conn().QueryRow(`
		SELECT c.id, c.name, c.base_url, c.api_key, c.format, c.chat_path, c.auth_header, c.auth_prefix, c.is_active, c.priority
		FROM discovered_models m
		JOIN connections c ON m.connection_id = c.id
		WHERE m.model_id = ? AND m.is_active = 1 AND c.is_active = 1
		ORDER BY c.priority DESC
		LIMIT 1
	`, model)

	var conn Connection
	err := row.Scan(&conn.ID, &conn.Name, &conn.BaseURL, &conn.APIKey, &conn.Format, &conn.ChatPath, &conn.AuthHeader, &conn.AuthPrefix, &conn.IsActive, &conn.Priority)
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", model)
	}
	return &conn, nil
}

func (p *ProxyHandler) getFirstConnection() (*Connection, error) {
	row := p.db.Conn().QueryRow(`
		SELECT id, name, base_url, api_key, format, chat_path, auth_header, auth_prefix, is_active, priority
		FROM connections
		WHERE is_active = 1
		ORDER BY priority DESC
		LIMIT 1
	`)

	var conn Connection
	err := row.Scan(&conn.ID, &conn.Name, &conn.BaseURL, &conn.APIKey, &conn.Format, &conn.ChatPath, &conn.AuthHeader, &conn.AuthPrefix, &conn.IsActive, &conn.Priority)
	if err != nil {
		return nil, fmt.Errorf("no active connections")
	}
	return &conn, nil
}

func (p *ProxyHandler) logRequest(model, connID, provider string, status int, latencyMs int64, tokensIn, tokensOut int, cached bool, errMsg string) {
	cachedInt := 0
	if cached {
		cachedInt = 1
	}

	id := uuid.New().String()
	p.db.Conn().Exec(`
		INSERT INTO request_logs (id, connection_id, provider, model, status, input_tokens, output_tokens, latency_ms, cached, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, id, connID, provider, model, status, tokensIn, tokensOut, latencyMs, cachedInt, errMsg)
}
