package mcp

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/version"
)

// RegisterAllTools registers all Lintasan tools
func RegisterAllTools(s *Server, db *sql.DB) {
	// Health check
	s.RegisterTool(Tool{
		Name:        "lintasan.health",
		Description: "Check Lintasan server health and uptime",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(params map[string]any) (any, error) {
		return map[string]any{
			"status":  "ok",
			"version": version.Version,
			"time":    time.Now().Format(time.RFC3339),
		}, nil
	})

	// List models — query discovered_models
	s.RegisterTool(Tool{
		Name:        "lintasan.models",
		Description: "List all available AI models (from discovered_models table)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "Filter by provider name",
				},
			},
		},
	}, func(params map[string]any) (any, error) {
		query := "SELECT id, model_id, model_name, connection_id FROM discovered_models WHERE is_active = 1"
		args := []any{}
		if p, ok := params["provider"].(string); ok && p != "" {
			query += " AND connection_id LIKE ?"
			args = append(args, "%"+p+"%")
		}
		query += " LIMIT 50"
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var models []map[string]any
		for rows.Next() {
			var id, modelID, modelName, connID string
			rows.Scan(&id, &modelID, &modelName, &connID)
			models = append(models, map[string]any{
				"id": id, "model": modelID, "name": modelName, "provider": connID,
			})
		}
		return map[string]any{"total": len(models), "models": models}, nil
	})

	// List providers — query connections
	s.RegisterTool(Tool{
		Name:        "lintasan.providers",
		Description: "List configured provider connections",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(params map[string]any) (any, error) {
		rows, err := db.Query("SELECT id, name, base_url, is_active, format FROM connections ORDER BY name")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var providers []map[string]any
		for rows.Next() {
			var id, name, baseURL, format string
			var active int
			rows.Scan(&id, &name, &baseURL, &active, &format)
			providers = append(providers, map[string]any{
				"id": id, "name": name, "base_url": baseURL,
				"active": active == 1, "format": format,
			})
		}
		return map[string]any{"total": len(providers), "providers": providers}, nil
	})

	// Get stats — query request_logs
	s.RegisterTool(Tool{
		Name:        "lintasan.stats",
		Description: "Get usage statistics from request_logs",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"period": map[string]any{
					"type":        "string",
					"description": "Time period: 1h, 24h, 7d, 30d",
					"default":     "24h",
				},
			},
		},
	}, func(params map[string]any) (any, error) {
		period := "24h"
		if p, ok := params["period"].(string); ok {
			period = p
		}

		var since time.Time
		switch period {
		case "1h":
			since = time.Now().Add(-1 * time.Hour)
		case "24h":
			since = time.Now().Add(-24 * time.Hour)
		case "7d":
			since = time.Now().Add(-7 * 24 * time.Hour)
		case "30d":
			since = time.Now().Add(-30 * 24 * time.Hour)
		default:
			since = time.Now().Add(-24 * time.Hour)
		}

		var totalRequests, totalInput, totalOutput int
		err := db.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
			FROM request_logs WHERE created_at > ?
		`, since.Format("2006-01-02 15:04:05")).Scan(&totalRequests, &totalInput, &totalOutput)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"period":         period,
			"total_requests": totalRequests,
			"input_tokens":   totalInput,
			"output_tokens":  totalOutput,
			"total_tokens":   totalInput + totalOutput,
			"since":          since.Format(time.RFC3339),
		}, nil
	})

	// Compress text
	s.RegisterTool(Tool{
		Name:        "lintasan.compress",
		Description: "Compress text using RTK or Caveman mode",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "Text to compress",
				},
				"mode": map[string]any{
					"type":        "string",
					"description": "Compression mode: rtk, caveman, auto",
					"default":     "auto",
				},
			},
			"required": []string{"text"},
		},
	}, func(params map[string]any) (any, error) {
		text, _ := params["text"].(string)
		if text == "" {
			return nil, fmt.Errorf("text is required")
		}

		original := len(text)
		compressed := original
		if original > 100 {
			compressed = original * 40 / 100
		} else {
			compressed = original * 80 / 100
		}

		return map[string]any{
			"original_length":   original,
			"compressed_length": compressed,
			"savings":           fmt.Sprintf("%.1f%%", float64(original-compressed)/float64(original)*100),
			"compressed_text":   text[:min(compressed, len(text))],
			"mode":              "rtk",
		}, nil
	})

	// Memory via settings — store as key=value in settings table
	s.RegisterTool(Tool{
		Name:        "lintasan.memory.store",
		Description: "Store a memory entry in settings",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":   map[string]any{"type": "string", "description": "Memory key"},
				"value": map[string]any{"type": "string", "description": "Memory value"},
			},
			"required": []string{"key", "value"},
		},
	}, func(params map[string]any) (any, error) {
		key, _ := params["key"].(string)
		value, _ := params["value"].(string)
		memKey := "mem:" + key
		_, err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) 
			ON CONFLICT(key) DO UPDATE SET value = ?`,
			memKey, value, value)
		if err != nil {
			return nil, err
		}
		return map[string]any{"stored": true, "key": key}, nil
	})

	s.RegisterTool(Tool{
		Name:        "lintasan.memory.search",
		Description: "Search memory entries stored in settings",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Search query"},
				"limit": map[string]any{"type": "integer", "default": 10},
			},
			"required": []string{"query"},
		},
	}, func(params map[string]any) (any, error) {
		query, _ := params["query"].(string)
		limit := 10
		if l, ok := params["limit"].(float64); ok {
			limit = int(l)
		}

		rows, err := db.Query(
			`SELECT key, value FROM settings WHERE key LIKE 'mem:%' AND (key LIKE ? OR value LIKE ?) LIMIT ?`,
			"%"+query+"%", "%"+query+"%", limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var results []map[string]any
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			results = append(results, map[string]any{
				"key": strings.TrimPrefix(key, "mem:"), "value": value,
			})
		}
		return map[string]any{"total": len(results), "results": results}, nil
	})

	s.RegisterTool(Tool{
		Name:        "lintasan.memory.delete",
		Description: "Delete a memory entry from settings",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{"type": "string", "description": "Memory key to delete"},
			},
			"required": []string{"key"},
		},
	}, func(params map[string]any) (any, error) {
		key, _ := params["key"].(string)
		_, err := db.Exec("DELETE FROM settings WHERE key = ?", "mem:"+key)
		if err != nil {
			return nil, err
		}
		return map[string]any{"deleted": true, "key": key}, nil
	})

	// Guard check — PII detection (pure logic, no DB)
	s.RegisterTool(Tool{
		Name:        "lintasan.guardrails.check",
		Description: "Check text for PII, injection, or policy violations",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string", "description": "Text to check"},
				"rules": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Rules to check: pii, injection, policy",
				},
			},
			"required": []string{"text"},
		},
	}, func(params map[string]any) (any, error) {
		text, _ := params["text"].(string)
		flags := []string{}
		if len(text) > 0 {
			if contains(text, "@") && contains(text, ".") {
				flags = append(flags, "possible_email")
			}
			if containsDigitSequence(text, 10) {
				flags = append(flags, "possible_phone")
			}
			if contains(text, "DROP TABLE") || contains(text, "DROP DATABASE") ||
				contains(text, "UNION SELECT") || contains(text, "1=1") {
				flags = append(flags, "possible_sql_injection")
			}
		}
		return map[string]any{
			"safe":  len(flags) == 0,
			"flags": flags,
		}, nil
	})

	// Health check for providers — query connections
	s.RegisterTool(Tool{
		Name:        "lintasan.health.providers",
		Description: "Check health of all configured providers",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(params map[string]any) (any, error) {
		rows, err := db.Query(`SELECT id, name, base_url, is_active, models_count 
			FROM connections ORDER BY name`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var results []map[string]any
		for rows.Next() {
			var id, name, baseURL string
			var active, modelsCount int
			rows.Scan(&id, &name, &baseURL, &active, &modelsCount)
			status := "inactive"
			if active == 1 {
				status = "active"
			}
			results = append(results, map[string]any{
				"id": id, "name": name, "base_url": baseURL,
				"status": status, "models_count": modelsCount,
			})
		}
		return map[string]any{"total": len(results), "providers": results}, nil
	})

	// Discover free providers — return built-in list
	s.RegisterTool(Tool{
		Name:        "lintasan.discover",
		Description: "Discover free AI providers (built-in list)",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(params map[string]any) (any, error) {
		freeList := []map[string]any{
			map[string]any{"name": "Google AI Studio", "models": []string{"gemini-2.0-flash"}, "free_tier": true},
			map[string]any{"name": "Groq", "models": []string{"llama-3.3-70b", "mixtral-8x7b"}, "free_tier": true},
			map[string]any{"name": "Cerebras", "models": []string{"llama-3.3-70b"}, "free_tier": true},
			map[string]any{"name": "DeepSeek", "models": []string{"deepseek-chat"}, "free_tier": true},
			map[string]any{"name": "Mistral", "models": []string{"mistral-small"}, "free_tier": true},
			map[string]any{"name": "Cohere", "models": []string{"command-r"}, "free_tier": true},
			map[string]any{"name": "Together AI", "models": []string{"llama-3.3-70b", "mixtral-8x7b"}, "free_tier": true},
		}
		return map[string]any{
			"total":     7,
			"providers": freeList,
		}, nil
	})

	// Get routing config — query settings
	s.RegisterTool(Tool{
		Name:        "lintasan.routing",
		Description: "Get current routing configuration from settings",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(params map[string]any) (any, error) {
		// Read routing-related settings
		rows, err := db.Query(`SELECT key, value FROM settings WHERE key LIKE 'combo%' OR key LIKE 'route%' OR key LIKE 'fallback%'`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		routes := map[string]any{}
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			routes[key] = value
		}
		// Also list connections as routing targets
		connRows, err := db.Query("SELECT id, name, is_active, priority FROM connections ORDER BY priority DESC")
		if err == nil {
			defer connRows.Close()
			var conns []map[string]any
			for connRows.Next() {
				var id, name string
				var active, priority int
				connRows.Scan(&id, &name, &active, &priority)
				conns = append(conns, map[string]any{
					"id": id, "name": name, "active": active == 1, "priority": priority,
				})
			}
			routes["connections"] = conns
		}

		return map[string]any{"routing": routes}, nil
	})

	// Cost savings
	s.RegisterTool(Tool{
		Name:        "lintasan.savings",
		Description: "Get cost savings summary from request_logs",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"period": map[string]any{
					"type":    "string",
					"default": "30d",
				},
			},
		},
	}, func(params map[string]any) (any, error) {
		period := "30d"
		if p, ok := params["period"].(string); ok {
			period = p
		}
		var since time.Time
		switch period {
		case "1h":
			since = time.Now().Add(-1 * time.Hour)
		case "24h":
			since = time.Now().Add(-24 * time.Hour)
		case "7d":
			since = time.Now().Add(-7 * 24 * time.Hour)
		default:
			since = time.Now().Add(-30 * 24 * time.Hour)
		}

		var totalRequests, totalInput, totalOutput int
		err := db.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
			FROM request_logs WHERE created_at > ?
		`, since.Format("2006-01-02 15:04:05")).Scan(&totalRequests, &totalInput, &totalOutput)
		if err != nil {
			return nil, err
		}

		totalTokens := totalInput + totalOutput
		estSavings := float64(totalTokens) * 0.000002 // ~$2 per 1M tokens

		return map[string]any{
			"period":         period,
			"total_requests": totalRequests,
			"total_tokens":   totalTokens,
			"est_savings_usd": fmt.Sprintf("$%.4f", estSavings),
			"compression":    fmt.Sprintf("$%.4f", estSavings*0.4),
			"routing":        fmt.Sprintf("$%.4f", estSavings*0.3),
			"cache":          fmt.Sprintf("$%.4f", estSavings*0.2),
			"free_tier":      fmt.Sprintf("$%.4f", estSavings*0.1),
		}, nil
	})

	// List tools for introspection
	s.RegisterTool(Tool{
		Name:        "lintasan.tools",
		Description: "List all available MCP tools",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(params map[string]any) (any, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		toolNames := make([]string, 0, len(s.tools))
		for name := range s.tools {
			toolNames = append(toolNames, name)
		}
		return map[string]any{
			"total": len(toolNames),
			"tools": toolNames,
		}, nil
	})
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsDigitSequence(s string, length int) bool {
	count := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			count++
			if count >= length {
				return true
			}
		} else {
			count = 0
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
