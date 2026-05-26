package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sanhaji182/lintasan-go/internal/config"
	"github.com/sanhaji182/lintasan-go/internal/db"
	"github.com/sanhaji182/lintasan-go/internal/cache"
	"github.com/sanhaji182/lintasan-go/internal/quota"
	"github.com/sanhaji182/lintasan-go/internal/optimizer"
	"github.com/sanhaji182/lintasan-go/internal/plugin"
	"github.com/sanhaji182/lintasan-go/internal/webhook"
	"github.com/sanhaji182/lintasan-go/internal/reasoning"
)

type ProxyHandler struct {
	cfg    *config.Config
	db     *db.DB
	pm     *plugin.Manager
	wm     *webhook.Manager
	client *http.Client
}

func NewProxyHandler(cfg *config.Config, database *db.DB) *ProxyHandler {
	return &ProxyHandler{
		cfg: cfg,
		db:  database,
		pm:  plugin.NewManager(database.Conn()),
		wm:  webhook.NewManager(database.Conn()),
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

func (p *ProxyHandler) getSetting(key, def string) string {
	val, err := p.db.GetSetting(key)
	if err != nil || val == "" { return def }
	return val
}

func (p *ProxyHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintf(os.Stderr, "PANIC in HandleChatCompletions: %v\n", rec)
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", buf[:n])
			http.Error(w, fmt.Sprintf(`{"error":"internal server error: %v"}`, rec), http.StatusInternalServerError)
		}
	}()
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
	
	stream, _ := req["stream"].(bool)
	messages, _ := req["messages"].([]any)
	
	// Optimizer
	if p.getSetting("prompt_optimizer_enabled", "false") == "true" {
		msgs, saved := optimizer.OptimizeMessages(messages)
		req["messages"] = msgs
		_ = saved
	}
	
	// Semantic Cache
	semanticEnabled := p.getSetting("semantic_cache_enabled", "true") == "true"
	if semanticEnabled {
		if respBody, ok := cache.GetSemanticMatch(p.db.Conn(), model, messages, 0.92); ok {
			p.logRequest(model, "cache", "cache", 200, time.Since(start).Milliseconds(), 0, 0, true, "")
			
			if stream {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.WriteHeader(200)
				
				// SSE cache replay format
				// In Node.js version we replay full SSE events, here we send as one big event
				// For real UI parsing we need to structure this similar to OpenAI chunks
				
				w.Write([]byte("data: " + respBody + "\n\n"))
				w.Write([]byte("data: [DONE]\n\n"))
				
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(respBody))
			}
			return
		}
	}

	candidates, resolvedModel, err := p.resolveRoute(model)
	if err != nil || len(candidates) == 0 {
		http.Error(w, fmt.Sprintf(`{"error":"no route found for model %s"}`, model), http.StatusNotFound)
		return
	}
	req["model"] = resolvedModel
	body, _ = json.Marshal(req)

	var lastErr string
	for i, conn := range candidates {
		resp, err := p.doUpstream(r, conn, body)
		if err != nil {
			lastErr = err.Error()
			p.logRequest(resolvedModel, conn.ID, conn.Name, 502, time.Since(start).Milliseconds(), 0, 0, false, lastErr)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 && i < len(candidates)-1 {
			b, _ := io.ReadAll(resp.Body)
			lastErr = string(b)
			p.logRequest(resolvedModel, conn.ID, conn.Name, resp.StatusCode, time.Since(start).Milliseconds(), 0, 0, false, lastErr)
			continue
		}

		// Quota success
		if resp.StatusCode == 200 { quota.RecordQuota(p.db.Conn(), conn.ID, 0) } // Token parsing later

		for k, v := range resp.Header {
			for _, vv := range v { w.Header().Add(k, vv) }
		}

		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(resp.StatusCode)
			flusher, ok := w.(http.Flusher)
			
			var streamBuffer []byte
			if !ok { 
				b, _ := io.ReadAll(resp.Body)
				w.Write(b)
				streamBuffer = b
			} else {
				buf := make([]byte, 4096)
				for {
					n, er := resp.Body.Read(buf)
					if n > 0 { 
						w.Write(buf[:n])
						flusher.Flush() 
						streamBuffer = append(streamBuffer, buf[:n]...)
					}
					if er != nil { break }
				}
			}
			
			p.logRequest(resolvedModel, conn.ID, conn.Name, resp.StatusCode, time.Since(start).Milliseconds(), 0, 0, false, "")
			
			if semanticEnabled && resp.StatusCode == 200 {
				cache.SaveSemanticMatch(p.db.Conn(), model, messages, string(streamBuffer), 3600)
			}
			return
		}

		b, _ := io.ReadAll(resp.Body)

		// Translate CC Alpha SSE → OpenAI JSON
		if conn.Format == "commandcode" {
			b = translateCCAlphaToOpenAI(b)
		}

		// Reasoning extraction: DeepSeek V4 Pro puts answer in reasoning_content not content
		b = reasoning.ExtractReasoningContent(b)

		w.WriteHeader(resp.StatusCode)
		w.Write(b)
		p.logRequest(resolvedModel, conn.ID, conn.Name, resp.StatusCode, time.Since(start).Milliseconds(), 0, 0, false, "")
		
		if p.wm != nil {
			p.wm.Fire("request.success", map[string]interface{}{
				"model": resolvedModel,
				"status": resp.StatusCode,
			})
		}
		
		if semanticEnabled && resp.StatusCode == 200 {
			cache.SaveSemanticMatch(p.db.Conn(), model, messages, string(b), 3600)
		}
		return
	}
	
		if p.wm != nil {
			p.wm.Fire("request.error", map[string]interface{}{
				"model": model,
				"error": lastErr, // lastErr is already a string in Go implementation
			})
		}
	
	http.Error(w, fmt.Sprintf(`{"error":{"message":"all routes failed","details":%q}}`, lastErr), http.StatusBadGateway)
}

func (p *ProxyHandler) doUpstream(r *http.Request, conn *Connection, body []byte) (*http.Response, error) {
	chatPath := conn.ChatPath
	if chatPath == "" { chatPath = "/v1/chat/completions" }
	upstreamURL := strings.TrimRight(conn.BaseURL, "/") + chatPath

	// Format translation: OpenAI body → commandcode body
	requestBody := body
	if conn.Format == "commandcode" {
		requestBody = transformForCommandCode(body)
	}

	upReq, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, strings.NewReader(string(requestBody)))
	if err != nil { return nil, err }
	upReq.Header.Set("Content-Type", "application/json")
	authHeader := conn.AuthHeader; if authHeader == "" { authHeader = "Authorization" }
	authPrefix := conn.AuthPrefix; if authPrefix == "" { authPrefix = "Bearer " }
	if conn.APIKey != "" { upReq.Header.Set(authHeader, authPrefix+conn.APIKey) }
	if xcc := r.Header.Get("X-Command-Code-Version"); xcc != "" {
		upReq.Header.Set("X-Command-Code-Version", xcc)
	}
	// CommandCode Alpha requires version header
	if conn.Format == "commandcode" {
		upReq.Header.Set("x-command-code-version", "0.26.25")
	}
	return p.client.Do(upReq)
}

// transformForCommandCode converts an OpenAI-format chat completions body
// into the CommandCode Alpha format (threadId/config/params).
func transformForCommandCode(body []byte) []byte {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	// Extract model — CC Alpha needs provider prefix for DeepSeek models
	model, _ := req["model"].(string)
	// CC Alpha recognizes "deepseek/deepseek-v4-pro" but not bare "deepseek-v4-pro"
	if strings.HasPrefix(model, "deepseek-v4") && !strings.Contains(model, "/") {
		model = "deepseek/" + model
	}

	// Extract system prompt from messages
	var systemPrompt string
	var userMessages []map[string]any
	if msgs, ok := req["messages"].([]any); ok {
		for _, m := range msgs {
			msg, _ := m.(map[string]any)
			role, _ := msg["role"].(string)
			if role == "system" || role == "developer" {
				if content, ok := msg["content"].(string); ok {
					systemPrompt += content + "\n\n"
				}
			} else {
				userMessages = append(userMessages, msg)
			}
		}
	}
	systemPrompt = strings.TrimSpace(systemPrompt)

	// Max tokens — default high for reasoning models (DeepSeek V4 Pro uses
	// reasoning_content which eats into the output budget). Floor at 8192
	// so reasoning + content both fit.
	maxTokens := 16384.0
	if mt, ok := req["max_tokens"].(float64); ok && mt > 0 {
		if mt > 200000 {
			maxTokens = 200000
		} else if mt < 8192 {
			maxTokens = 8192 // floor: reasoning models need headroom
		} else {
			maxTokens = mt
		}
	}

	// Stream: CC Alpha only supports streaming. Always send stream=true;
	// the handler will convert SSE → JSON for non-streaming clients.
	stream := true

	params := map[string]any{
		"model":      model,
		"messages":   userMessages,
		"system":     systemPrompt,
		"max_tokens": int(maxTokens),
		"stream":     stream,
	}

	out := map[string]any{
		"threadId": uuid.New().String(),
		"memory":   "",
		"config": map[string]any{
			"workingDir":    "/tmp",
			"date":          time.Now().Format("2006-01-02"),
			"environment":   "linux",
			"structure":     []any{},
			"isGitRepo":     false,
			"currentBranch": "",
			"mainBranch":    "",
			"gitStatus":     "",
			"recentCommits": []any{},
		},
		"params": params,
	}

	result, _ := json.Marshal(out)
	return result
}

// translateCCAlphaToOpenAI converts CC Alpha SSE events into a standard OpenAI
// JSON chat completion response. CC Alpha streams line-delimited JSON events with
// types like "text-delta", "reasoning-delta", "finish", "provider-metadata", etc.
func translateCCAlphaToOpenAI(raw []byte) []byte {
	lines := strings.Split(string(raw), "\n")
	var contentText, reasoningText strings.Builder
	var finishReason string
	var usage map[string]any
	var modelUsed string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "null" || !strings.HasPrefix(line, "{") {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		typ, _ := event["type"].(string)
		switch typ {
		case "text-delta":
			if t, ok := event["text"].(string); ok {
				contentText.WriteString(t)
			}
		case "reasoning-start":
			// Just an event marker — no text
		case "reasoning-delta":
			if t, ok := event["text"].(string); ok {
				reasoningText.WriteString(t)
			}
		case "reasoning-end":
			// Marker event
		case "finish", "finish-step":
			if fr, ok := event["finishReason"].(string); ok && finishReason == "" {
				finishReason = fr
			}
			if u, ok := event["totalUsage"].(map[string]any); ok && usage == nil {
				usage = u
			}
			if u, ok := event["usage"].(map[string]any); ok && usage == nil {
				usage = u
			}
		case "provider-metadata":
			// Extract model from provider metadata
			if pm, ok := event["providerMetadata"].(map[string]any); ok {
				if g, ok := pm["gateway"].(map[string]any); ok {
					if r, ok := g["routing"].(map[string]any); ok {
						if slug, ok := r["canonicalSlug"].(string); ok && modelUsed == "" {
							modelUsed = slug
						}
					}
				}
			}
		case "start-step":
			// Extract model from request body
			if req, ok := event["request"].(map[string]any); ok {
				if body, ok := req["body"].(map[string]any); ok {
					if opts, ok := body["providerOptions"].(map[string]any); ok {
						if gw, ok := opts["gateway"].(map[string]any); ok {
							if only, ok := gw["only"].([]any); ok && len(only) > 0 && modelUsed == "" {
								if s, ok := only[0].(string); ok {
									modelUsed = s
								}
							}
						}
					}
				}
			}
		}
	}

	content := strings.TrimSpace(contentText.String())
	reasoning := strings.TrimSpace(reasoningText.String())

	if modelUsed == "" {
		modelUsed = "deepseek-v4-pro"
	}

	// Build OpenAI-compatible response
	message := map[string]any{
		"role": "assistant",
	}
	if content != "" {
		message["content"] = content
	}
	if reasoning != "" {
		message["reasoning_content"] = reasoning
	}

	resp := map[string]any{
		"id":      "cc-alpha-" + uuid.New().String()[:8],
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   modelUsed,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
	}
	if usage != nil {
		resp["usage"] = usage
	}

	out, _ := json.Marshal(resp)
	return out
}

func (p *ProxyHandler) resolveRoute(model string) ([]*Connection, string, error) {
	model = p.resolveAlias(model)
	if conns, resolved, ok := p.resolveCombo(model); ok && len(conns) > 0 { return conns, resolved, nil }
	conn, err := p.findConnectionForModel(model)
	if err != nil { return nil, model, err }
	return []*Connection{conn}, model, nil
}

func (p *ProxyHandler) resolveAlias(model string) string {
	v, _ := p.db.GetSetting("aliases")
	if v == "" { return model }
	var m map[string]any
	if json.Unmarshal([]byte(v), &m) != nil { return model }
	if raw, ok := m[model]; ok {
		switch t := raw.(type) {
		case string: return t
		case map[string]any: if s, _ := t["model"].(string); s != "" { return s }
		}
	}
	return model
}

func (p *ProxyHandler) resolveCombo(name string) ([]*Connection, string, bool) {
	v, _ := p.db.GetSetting("combos")
	if v == "" { return nil, name, false }
	var combos []map[string]any
	if json.Unmarshal([]byte(v), &combos) != nil { return nil, name, false }
	for _, c := range combos {
		if c["name"] != name { continue }
		entries, _ := c["entries"].([]any)
		strategy, _ := c["strategy"].(string)
		if strategy == "round-robin" && len(entries) > 1 {
			idx := p.nextRoundRobinIndex("combo_rr_"+name, len(entries))
			entries = append(entries[idx:], entries[:idx]...)
		}
		var out []*Connection
		resolvedModel := name
		for _, e := range entries {
			em, _ := e.(map[string]any); if em == nil { continue }
			m, _ := em["model"].(string); if m == "" { continue }
			if resolvedModel == name { resolvedModel = m }
			ids := stringSlice(em["connection_ids"])
			conns := p.connectionsForModelAndIDs(m, ids)
			out = append(out, conns...)
		}
		return out, resolvedModel, true
	}
	return nil, name, false
}

func stringSlice(v any) []string { arr, _ := v.([]any); out := []string{}; for _, x := range arr { if s, ok := x.(string); ok { out = append(out, s) } }; return out }

func (p *ProxyHandler) nextRoundRobinIndex(key string, n int) int {
	if n <= 1 { return 0 }
	v, _ := p.db.GetSetting(key); var i int; fmt.Sscanf(v, "%d", &i); next := (i+1)%n; p.db.SetSetting(key, fmt.Sprintf("%d", next)); return i % n
}

func (p *ProxyHandler) connectionsForModelAndIDs(model string, ids []string) []*Connection {
	query := `SELECT c.id, c.name, c.base_url, c.api_key, c.format, c.chat_path, c.auth_header, c.auth_prefix, c.is_active, c.priority FROM discovered_models m JOIN connections c ON m.connection_id=c.id WHERE m.model_id=? AND m.is_active=1 AND c.is_active=1`
	args := []any{model}
	if len(ids) > 0 { ph:=make([]string,len(ids)); for i,id:=range ids{ph[i]="?"; args=append(args,id)}; query += " AND c.id IN ("+strings.Join(ph,",")+")" }
	query += " ORDER BY c.priority DESC"
	rows, err := p.db.Conn().Query(query, args...); if err != nil { return nil }; defer rows.Close()
	var out []*Connection
	for rows.Next(){ var c Connection; if rows.Scan(&c.ID,&c.Name,&c.BaseURL,&c.APIKey,&c.Format,&c.ChatPath,&c.AuthHeader,&c.AuthPrefix,&c.IsActive,&c.Priority)==nil{ out=append(out,&c) } }
	return out
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
	cachedInt := 0; if cached { cachedInt = 1 }
	id := uuid.New().String()
	p.db.Conn().Exec(`
		INSERT INTO request_logs (id, connection_id, provider, model, status, input_tokens, output_tokens, latency_ms, cached, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, id, connID, provider, model, status, tokensIn, tokensOut, latencyMs, cachedInt, errMsg)
}

func (p *ProxyHandler) HandleEmbeddings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil { http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest); return }
	defer r.Body.Close()

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil { http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest); return }

	model, _ := req["model"].(string)
	if model == "" { model = "text-embedding-3-small" }

	conn, err := p.findConnectionForModel(model)
	if err != nil {
		conn, err = p.getFirstConnection()
		if err != nil { http.Error(w, fmt.Sprintf(`{"error":"no connection found for model %s"}`, model), http.StatusNotFound); return }
	}

	upstreamURL := strings.TrimRight(conn.BaseURL, "/") + "/v1/embeddings"
	upReq, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, strings.NewReader(string(body)))
	if err != nil { http.Error(w, `{"error":"failed to create upstream request"}`, http.StatusInternalServerError); return }

	upReq.Header.Set("Content-Type", "application/json")
	authHeader := conn.AuthHeader; if authHeader == "" { authHeader = "Authorization" }
	authPrefix := conn.AuthPrefix; if authPrefix == "" { authPrefix = "Bearer " }
	if conn.APIKey != "" { upReq.Header.Set(authHeader, authPrefix+conn.APIKey) }

	resp, err := p.client.Do(upReq)
	if err != nil { http.Error(w, fmt.Sprintf(`{"error":"upstream error: %s"}`, err.Error()), http.StatusBadGateway); return }
	defer resp.Body.Close()

	for k, v := range resp.Header { for _, vv := range v { w.Header().Add(k, vv) } }
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ProxyHandler) proxyPath(w http.ResponseWriter, r *http.Request, upstreamPath string){
    body,_:=io.ReadAll(r.Body); var reqBody io.Reader=bytes.NewReader(body)
    conn,err:=p.getFirstConnection(); if err!=nil{http.Error(w,`{"error":"no active connections"}`,404);return}
    upReq,err:=http.NewRequestWithContext(r.Context(),"POST",strings.TrimRight(conn.BaseURL,"/")+upstreamPath,reqBody); if err!=nil{http.Error(w,err.Error(),500);return}
    upReq.Header.Set("Content-Type", r.Header.Get("Content-Type")); if upReq.Header.Get("Content-Type")==""{upReq.Header.Set("Content-Type","application/json")}
    if conn.APIKey!=""{h:=conn.AuthHeader; if h==""{h="Authorization"}; pfx:=conn.AuthPrefix; if pfx==""{pfx="Bearer "}; upReq.Header.Set(h,pfx+conn.APIKey)}
    resp,err:=p.client.Do(upReq); if err!=nil{http.Error(w,fmt.Sprintf(`{"error":"upstream error: %s"}`,err.Error()),502);return}; defer resp.Body.Close()
    for k,v:=range resp.Header{for _,vv:=range v{w.Header().Add(k,vv)}}; w.WriteHeader(resp.StatusCode); io.Copy(w,resp.Body)
}
func (p *ProxyHandler) HandleImages(w http.ResponseWriter,r *http.Request){ p.proxyPath(w,r,"/v1/images/generations") }
func (p *ProxyHandler) HandleAudioSpeech(w http.ResponseWriter,r *http.Request){ p.proxyPath(w,r,"/v1/audio/speech") }
func (p *ProxyHandler) HandleAudioTranscriptions(w http.ResponseWriter,r *http.Request){ p.proxyPath(w,r,"/v1/audio/transcriptions") }
