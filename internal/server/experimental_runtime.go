package server

// experimental_runtime.go — R2: Runtime Reachability for Experimental Providers.
//
// This file adds the integration point that allows an active Experimental
// provider to be resolved and executed from the proxy hot path, ONLY when an
// explicit opt-in signal is present (model prefix "experimental/<name>" or
// X-Lintasan-Track: experimental header).
//
// MEMBRANE PRESERVATION:
//   - This code is reached ONLY via DetectExperimental (explicit signal).
//   - The normal resolveRoute path is NEVER modified.
//   - ResolveRoutable / RoutableProviders are NEVER called here.
//   - Official routing is byte-for-byte identical.
//
// FLOW:
//   1. HandleChatCompletions parses model from request body.
//   2. handleExperimentalRoute checks for explicit experimental signal.
//   3. If signal present: resolve provider from registry via ResolveExperimental,
//      drive it via the Agent interface (Run), and write the response.
//   4. If no signal: fall through to normal resolveRoute (unchanged).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/expprovider"
	"github.com/sanhaji182/lintasan-go/internal/experimental"
)

// handleExperimentalRoute checks for an explicit experimental opt-in signal and,
// if present, resolves + executes the experimental provider. Returns true if the
// request was handled (caller should return), false if the normal path should
// continue.
//
// This is the ONLY entry point from the proxy hot path into the Experimental
// container. It is deliberately a separate function (not inlined into
// resolveRoute) so the two pools can never be confused at a call site.
func (p *ProxyHandler) handleExperimentalRoute(w http.ResponseWriter, r *http.Request, model string, req map[string]any, body []byte) bool {
	// Detect explicit experimental signal (model prefix or header).
	signal, ok := expprovider.DetectExperimental(model, r.Header)
	if !ok {
		return false // No signal — fall through to normal routing.
	}

	start := time.Now()

	// Resolve the experimental provider from the registry via the explicit door.
	prov, found := p.providerReg.ResolveExperimental(signal.Provider)
	if !found {
		http.Error(w, fmt.Sprintf(`{"error":"experimental provider %q not found or not active"}`, signal.Provider), http.StatusNotFound)
		return true
	}

	// Type-assert to Agent interface — ACPProvider satisfies both Provider and Agent.
	agent, ok := prov.(expprovider.Agent)
	if !ok {
		http.Error(w, fmt.Sprintf(`{"error":"experimental provider %q does not support the Agent interface"}`, signal.Provider), http.StatusInternalServerError)
		return true
	}

	// Extract the prompt from the messages array (last user message).
	prompt := extractPromptFromMessages(req)

	// Build the agent turn.
	turn := expprovider.AgentTurn{
		Prompt: prompt,
		// No permission handler — deny by default (safe: agent tools run in
		// the agent's own sandbox, not the host).
		OnPermission: nil,
	}

	// Execute with a bounded timeout.
	timeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	result, err := agent.Run(ctx, turn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[experimental] runtime: provider %q Run error: %v\n", signal.Provider, err)
		http.Error(w, fmt.Sprintf(`{"error":"experimental provider %q execution failed: %v"}`, signal.Provider, err), http.StatusBadGateway)
		return true
	}

	// Build an OpenAI-compatible chat completion response from the agent result.
	resp := buildExperimentalResponse(signal.Provider, model, result, start)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Lintasan-Track", "experimental")
	w.Header().Set("X-Lintasan-Provider", signal.Provider)
	w.Header().Set("X-Lintasan-Signal-Via", signal.Via)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	return true
}

// extractPromptFromMessages extracts the prompt text from the OpenAI messages
// array. It takes the content of the last user message, which is the standard
// pattern for a single-turn agent invocation.
func extractPromptFromMessages(req map[string]any) string {
	messages, _ := req["messages"].([]any)
	if len(messages) == 0 {
		return ""
	}
	// Walk backwards to find the last user message.
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role == "user" {
			content, _ := msg["content"].(string)
			return content
		}
	}
	// Fallback: use the last message regardless of role.
	if msg, ok := messages[len(messages)-1].(map[string]any); ok {
		content, _ := msg["content"].(string)
		return content
	}
	return ""
}

// buildExperimentalResponse constructs an OpenAI-compatible chat completion
// response from an experimental agent's PromptResult.
func buildExperimentalResponse(providerName, model string, result *experimental.PromptResult, start time.Time) map[string]any {
	content := ""
	if result != nil {
		// Extract text from the result. PromptResult contains the terminal
		// response from the agent's prompt turn.
		content = formatPromptResult(result)
	}

	return map[string]any{
		"id":      fmt.Sprintf("exp-%s-%d", providerName, time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
		"x_lintasan": map[string]any{
			"track":    "experimental",
			"provider": providerName,
			"latency":  time.Since(start).Milliseconds(),
		},
	}
}

// formatPromptResult extracts readable text from an experimental.PromptResult.
// The broker assembles Text from the session/update stream; Content is the raw
// JSON content blocks from the agent's terminal response.
func formatPromptResult(result *experimental.PromptResult) string {
	if result == nil {
		return ""
	}
	// Prefer the pre-assembled Text field (broker convenience).
	if result.Text != "" {
		return result.Text
	}
	// Fallback: parse Content (json.RawMessage) as an array of content blocks.
	if len(result.Content) == 0 {
		return ""
	}
	var blocks []map[string]any
	if err := json.Unmarshal(result.Content, &blocks); err == nil {
		var text string
		for _, block := range blocks {
			if t, ok := block["text"].(string); ok {
				if text != "" {
					text += "\n"
				}
				text += t
			}
		}
		return text
	}
	// Last resort: return Content as raw string.
	return string(result.Content)
}
