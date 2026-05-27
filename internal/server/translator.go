package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// OpenAI → Anthropic (request conversion)
// ─────────────────────────────────────────────────────────────────────────────

// OpenAIToAnthropic converts an OpenAI-compatible chat completion request
// into an Anthropic Messages API request.
func OpenAIToAnthropic(req map[string]any) map[string]any {
	anthropicReq := make(map[string]any)

	// Model
	if model, ok := req["model"].(string); ok {
		anthropicReq["model"] = model
	}

	// Stream
	anthropicReq["stream"] = req["stream"]

	// Extract system prompt(s) from messages and move to top-level "system" field
	var systemPrompt string
	var messages []map[string]any

	rawMessages, ok := req["messages"].([]any)
	if ok {
		for _, m := range rawMessages {
			msg, ok := m.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role == "system" || role == "developer" {
				// Concatenate multiple system messages
				if content, ok := msg["content"].(string); ok {
					systemPrompt = appendSystemPrompt(systemPrompt, content)
				} else if contentAny, ok := msg["content"].([]any); ok {
					// Handle content arrays (e.g., text blocks)
					for _, part := range contentAny {
						if partMap, ok := part.(map[string]any); ok {
							if text, ok := partMap["text"].(string); ok {
								systemPrompt = appendSystemPrompt(systemPrompt, text)
							}
						}
					}
				}
			} else {
				messages = append(messages, msg)
			}
		}
	}

	if systemPrompt != "" {
		anthropicReq["system"] = systemPrompt
	}

	if len(messages) > 0 {
		anthropicReq["messages"] = convertMessagesToAnthropic(messages)
	}

	// Top-level params
	copyParam(req, anthropicReq, "max_tokens")
	copyParam(req, anthropicReq, "temperature")
	copyParam(req, anthropicReq, "top_p")
	copyParam(req, anthropicReq, "stop_sequences", "stop")
	copyParam(req, anthropicReq, "metadata")
	copyParam(req, anthropicReq, "thinking")
	copyParam(req, anthropicReq, "thinking_types")
	copyParam(req, anthropicReq, "tools")
	copyParam(req, anthropicReq, "tool_choice")
	copyParam(req, anthropicReq, "betas")

	return anthropicReq
}

// convertMessagesToAnthropic converts an OpenAI-style messages array to
// Anthropic's messages format (role/content blocks).
func convertMessagesToAnthropic(messages []map[string]any) []map[string]any {
	var result []map[string]any

	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content := msg["content"]

		// Normalize role: "assistant" stays, "user" stays
		// OpenAI "developer" → Anthropic "developer"
		normalizedRole := role
		if role == "developer" {
			normalizedRole = "developer"
		}

		// Handle content: string or array of content blocks
		var anthropicContent any

		switch c := content.(type) {
		case string:
			if c != "" {
				anthropicContent = []map[string]any{
					{"type": "text", "text": c},
				}
			} else {
				anthropicContent = []map[string]any{}
			}
		case []any:
			var parts []map[string]any
			for _, item := range c {
				if itemMap, ok := item.(map[string]any); ok {
					itemType, _ := itemMap["type"].(string)
					switch itemType {
					case "text":
						if text, ok := itemMap["text"].(string); ok {
							parts = append(parts, map[string]any{"type": "text", "text": text})
						}
					case "image_url":
						// OpenAI image_url → Anthropic image content block
						if source, ok := itemMap["source"].(map[string]any); ok {
							mediaType, _ := source["type"].(string) // "url" or "base64"
							// Anthropic supports base64 images directly
							if mediaType == "base64" {
								if data, ok := source["data"].(string); ok {
									mimeType, _ := source["mime_type"].(string)
									if mimeType == "" {
										mimeType = "image/jpeg"
									}
									parts = append(parts, map[string]any{
										"type": "image",
										"source": map[string]any{
											"type":      "base64",
											"media_type": mimeType,
											"data":      data,
										},
									})
								}
							} else if mediaType == "url" || mediaType == "" {
								// URL → base64 isn't possible directly; embed as text URL
								if url, ok := source["url"].(string); ok {
									parts = append(parts, map[string]any{
										"type": "image",
										"source": map[string]any{
											"type":      "url",
											"media_type": "image/jpeg",
											"data":      url,
										},
									})
								}
							}
						}
					case "tool_use":
						if id, ok := itemMap["id"].(string); ok {
							name, _ := itemMap["name"].(string)
							args := itemMap["arguments"] // already JSON string from test/API
							parts = append(parts, map[string]any{
								"type":  "tool_use",
								"id":    id,
								"name":  name,
								"input": args,
							})
						}
					}
				}
			}
			if len(parts) > 0 {
				// Convert to []any for JSON compatibility
				anyParts := make([]any, len(parts))
				for i, p := range parts {
					anyParts[i] = p
				}
				anthropicContent = anyParts
			} else {
				anthropicContent = []map[string]any{}
			}
		default:
			anthropicContent = []map[string]any{}
		}

		result = append(result, map[string]any{
			"role":    normalizedRole,
			"content": anthropicContent,
		})
	}

	return result
}

// appendSystemPrompt appends a content string to the existing system prompt.
func appendSystemPrompt(existing, new string) string {
	if existing == "" {
		return new
	}
	return existing + "\n\n" + new
}

// ─────────────────────────────────────────────────────────────────────────────
// Anthropic → OpenAI (response conversion)
// ─────────────────────────────────────────────────────────────────────────────

// AnthropicToOpenAI converts an Anthropic non-streaming response (JSON bytes)
// into an OpenAI-compatible chat completion response.
func AnthropicToOpenAI(respBody []byte) ([]byte, error) {
	var anthropicResp map[string]any
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	// Extract content
	var textContent string
	if content, ok := anthropicResp["content"].([]any); ok {
		for _, block := range content {
			if blockMap, ok := block.(map[string]any); ok {
				if blockType, _ := blockMap["type"].(string); blockType == "text" {
					if text, ok := blockMap["text"].(string); ok {
						textContent += text
					}
				}
			}
		}
	}

	// Extract usage
	usage := extractAnthropicUsage(anthropicResp)

	// Build OpenAI response
	model, _ := anthropicResp["model"].(string)
	id, _ := anthropicResp["id"].(string)
	if id == "" {
		id = "anthropic-" + model
	}

	openAIResp := map[string]any{
		"id":      id,
		"object":  "chat.completion",
		"created": anthropicResp["created_at"],
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": textContent,
				},
				"finish_reason": anthropicResp["stop_reason"],
			},
		},
		"usage": usage,
	}

	return json.Marshal(openAIResp)
}

// extractAnthropicUsage extracts token usage from an Anthropic response.
func extractAnthropicUsage(resp map[string]any) map[string]any {
	usage := map[string]any{
		"prompt_tokens":     0,
		"completion_tokens": 0,
		"total_tokens":      0,
	}

	if usageData, ok := resp["usage"].(map[string]any); ok {
		if v, ok := usageData["input_tokens"].(float64); ok {
			usage["prompt_tokens"] = v
		}
		if v, ok := usageData["output_tokens"].(float64); ok {
			usage["completion_tokens"] = v
		}
		usage["total_tokens"] = usage["prompt_tokens"].(float64) + usage["completion_tokens"].(float64)
	}

	return usage
}

// TranslateAnthropicStreamChunk translates a single SSE line from Anthropic
// streaming into an OpenAI chat.completion.chunk SSE event.
// Returns the OpenAI-format SSE line (e.g., "data: {...}\n\n") or empty string
// if the line should be dropped.
func TranslateAnthropicStreamChunk(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:") {
		// strip prefix
		if idx := strings.Index(line, ":"); idx >= 0 && (strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:")) {
			line = strings.TrimSpace(line[idx+1:])
		}
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return ""
	}

	eventType, _ := event["type"].(string)

	switch eventType {
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]any)
		deltaType, _ := delta["type"].(string)
		if deltaType == "thinking_delta" {
			// Skip thinking deltas (not supported in OpenAI format)
			return ""
		}
		if deltaType == "text_delta" {
			text, _ := delta["text"].(string)
			chunk := map[string]any{
				"id":                event["message_id"],
				"object":            "chat.completion.chunk",
				"created":           event["created_at"],
				"model":             event["model"],
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"content": text,
						},
						"finish_reason": nil,
					},
				},
			}
			return "data: " + mustMarshal(chunk) + "\n\n"
		}
		if deltaType == "input_json_delta" {
			partial, _ := delta["partial"].(string)
			chunk := map[string]any{
				"id":                event["message_id"],
				"object":            "chat.completion.chunk",
				"created":           event["created_at"],
				"model":             event["model"],
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"content": partial,
						},
						"finish_reason": nil,
					},
				},
			}
			return "data: " + mustMarshal(chunk) + "\n\n"
		}
	}

	return ""
}

// TranslateAnthropicStream translates an entire Anthropic SSE stream body
// into an OpenAI SSE stream.
func TranslateAnthropicStream(body io.Reader) ([]byte, error) {
	var buf bytes.Buffer

	// Read all body
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream body: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		translated := TranslateAnthropicStreamChunk(line)
		if translated != "" {
			buf.WriteString(translated)
		}
	}

	return buf.Bytes(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// OpenAI → Gemini (request conversion)
// ─────────────────────────────────────────────────────────────────────────────

// OpenAIToGemini converts an OpenAI-compatible chat completion request into
// a Gemini REST API request.
func OpenAIToGemini(req map[string]any) map[string]any {
	geminiReq := make(map[string]any)

	model, _ := req["model"].(string)
	// Strip prefix if present (e.g., "models/gemini-2.5-pro")
	modelName := strings.TrimPrefix(model, "models/")
	geminiReq["model"] = modelName

	// Convert messages to contents/parts
	rawMessages, _ := req["messages"].([]any)
	contents := convertMessagesToGemini(rawMessages)
	if len(contents) > 0 {
		geminiReq["contents"] = contents
	}

	// System instruction
	var systemParts []map[string]any
	for _, m := range rawMessages {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if role, _ := msg["role"].(string); role == "system" {
			if text := extractTextContent(msg["content"]); text != "" {
				systemParts = append(systemParts, map[string]any{"text": text})
			}
		}
	}
	if len(systemParts) > 0 {
		geminiReq["systemInstruction"] = map[string]any{
			"parts": systemParts,
		}
	}

	// Generation config
	genConfig := make(map[string]any)
	if mt, ok := req["max_tokens"].(float64); ok {
		genConfig["maxOutputTokens"] = int(mt)
	} else if mt, ok := req["max_tokens"].(int); ok {
		genConfig["maxOutputTokens"] = mt
	}
	if temp, ok := req["temperature"].(float64); ok {
		genConfig["temperature"] = temp
	}
	if tp, ok := req["top_p"].(float64); ok {
		genConfig["topP"] = tp
	}
	if tp, ok := req["top_k"].(float64); ok {
		genConfig["topK"] = int(tp)
	}
	if stop, ok := req["stop"].(string); ok && stop != "" {
		genConfig["stopSequences"] = []string{stop}
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	// Safety settings (keep default)
	geminiReq["safetySettings"] = []map[string]string{
		{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"},
		{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"},
		{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"},
		{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"},
	}

	// Tools
	if tools, ok := req["tools"].([]any); ok && len(tools) > 0 {
		geminiReq["tools"] = convertToolsToGemini(tools)
	}

	return geminiReq
}

// convertMessagesToGemini converts OpenAI-style messages to Gemini contents format.
func convertMessagesToGemini(messages []any) []map[string]any {
	var contents []map[string]any

	for _, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		// Skip system/developer — they go to systemInstruction, not contents
		if role == "system" || role == "developer" {
			continue
		}
		// Map OpenAI role to Gemini role
		// user → "user", assistant → "model"
		geminiRole := "user"
		if role == "assistant" {
			geminiRole = "model"
		}

		parts := extractGeminiParts(msg["content"])
		if len(parts) == 0 {
			continue
		}

		contents = append(contents, map[string]any{
			"role":  geminiRole,
			"parts": parts,
		})
	}

	return contents
}

// extractGeminiParts extracts Gemini-format parts from OpenAI content.
func extractGeminiParts(content any) []map[string]any {
	switch c := content.(type) {
	case string:
		if c == "" {
			return nil
		}
		return []map[string]any{{"text": c}}
	case []any:
		var parts []map[string]any
		for _, item := range c {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			itemType, _ := itemMap["type"].(string)
			switch itemType {
			case "text":
				if text, ok := itemMap["text"].(string); ok {
					parts = append(parts, map[string]any{"text": text})
				}
			case "image_url":
				if source, ok := itemMap["source"].(map[string]any); ok {
					mediaType, _ := source["type"].(string)
					if mediaType == "base64" {
						if data, ok := source["data"].(string); ok {
							mimeType := "image/jpeg"
							if mt, ok := source["mime_type"].(string); ok {
								mimeType = mt
							}
							parts = append(parts, map[string]any{
								"inlineData": map[string]any{
									"mimeType": mimeType,
									"data":     data,
								},
							})
						}
					} else if mediaType == "url" || mediaType == "" {
						if url, ok := source["url"].(string); ok {
							// Gemini doesn't support URL images directly — convert URL to text part
							parts = append(parts, map[string]any{
								"text": "[Image: " + url + "]",
							})
						}
					}
				}
			case "tool_use":
				if id, ok := itemMap["id"].(string); ok {
					name, _ := itemMap["name"].(string)
					args, _ := json.Marshal(itemMap["arguments"])
					parts = append(parts, map[string]any{
						"functionCall": map[string]any{
							"name": name,
							"args": string(args),
						},
						"id": id,
					})
				}
			}
		}
		return parts
	}
	return nil
}

// convertToolsToGemini converts OpenAI tools to Gemini tool declarations.
func convertToolsToGemini(tools []any) []map[string]any {
	var declarations []map[string]any
	for _, t := range tools {
		toolMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		funcDecl, ok := toolMap["function"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := funcDecl["name"].(string)
		description, _ := funcDecl["description"].(string)
		params, _ := funcDecl["parameters"].(map[string]any)
		declarations = append(declarations, map[string]any{
			"functionDeclarations": []map[string]any{
				{
					"name":        name,
					"description": description,
					"parameters": params,
				},
			},
		})
	}
	return declarations
}

// extractTextContent extracts a plain text string from OpenAI content (string or array).
func extractTextContent(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var sb strings.Builder
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
		return sb.String()
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Gemini → OpenAI (response conversion)
// ─────────────────────────────────────────────────────────────────────────────

// GeminiToOpenAI converts a Gemini REST API response (JSON bytes) into an
// OpenAI-compatible chat completion response.
func GeminiToOpenAI(respBody []byte) ([]byte, error) {
	var geminiResp map[string]any
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	// Extract text from candidates
	var textContent string
	var finishReason string

	if candidates, ok := geminiResp["candidates"].([]any); ok && len(candidates) > 0 {
		if cand, ok := candidates[0].(map[string]any); ok {
			if content, ok := cand["content"].(map[string]any); ok {
				if parts, ok := content["parts"].([]any); ok {
					for _, part := range parts {
						if partMap, ok := part.(map[string]any); ok {
							if text, ok := partMap["text"].(string); ok {
								textContent += text
							}
							if fc, ok := partMap["functionCall"].(map[string]any); ok {
								name, _ := fc["name"].(string)
								args, _ := fc["args"].(string)
								// Convert function call to tool role message
								id, _ := fc["id"].(string)
								if id != "" {
									textContent += fmt.Sprintf("[tool_call id=%s name=%s args=%s]", id, name, args)
								}
							}
						}
					}
				}
			}
			if fr, ok := cand["finishReason"].(string); ok {
				finishReason = geminiFinishReasonToOpenAI(fr)
			}
		}
	}

	// Extract usage
	usage := extractGeminiUsage(geminiResp)

	model, _ := geminiResp["model"].(string)
	openAIResp := map[string]any{
		"id":      "gemini-" + model,
		"object":  "chat.completion",
		"created": 0,
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": textContent,
				},
				"finish_reason": finishReason,
			},
		},
		"usage": usage,
	}

	return json.Marshal(openAIResp)
}

// geminiFinishReasonToOpenAI maps Gemini finish reasons to OpenAI finish reasons.
func geminiFinishReasonToOpenAI(fr string) string {
	switch fr {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION", "MALFORMED_FUNCTION_CALL":
		return "content_filter"
	default:
		return "stop"
	}
}

// extractGeminiUsage extracts token usage from a Gemini response.
func extractGeminiUsage(resp map[string]any) map[string]any {
	usage := map[string]any{
		"prompt_tokens":     0,
		"completion_tokens": 0,
		"total_tokens":      0,
	}

	if usageData, ok := resp["usageMetadata"].(map[string]any); ok {
		if v, ok := usageData["promptTokenCount"].(float64); ok {
			usage["prompt_tokens"] = v
		}
		if v, ok := usageData["candidatesTokenCount"].(float64); ok {
			usage["completion_tokens"] = v
		}
		if v, ok := usageData["totalTokenCount"].(float64); ok {
			usage["total_tokens"] = v
		}
	}

	return usage
}

// TranslateGeminiStreamChunk translates a single SSE line from Gemini streaming
// into an OpenAI chat.completion.chunk SSE event.
func TranslateGeminiStreamChunk(line string) string {
	line = strings.TrimSpace(line)
	// Strip "data:" prefix (5 chars)
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(line[len("data:"):])
	}
	if line == "" {
		return ""
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return ""
	}

	candidates, _ := event["candidates"].([]any)
	if len(candidates) == 0 {
		return ""
	}

	cand, ok := candidates[0].(map[string]any)
	if !ok {
		return ""
	}

	content, ok := cand["content"].(map[string]any)
	if !ok {
		return ""
	}

	parts, _ := content["parts"].([]any)
	var textContent string
	for _, part := range parts {
		if partMap, ok := part.(map[string]any); ok {
			if text, ok := partMap["text"].(string); ok {
				textContent += text
			}
		}
	}

	fr, _ := cand["finishReason"].(string)
	finish := ""
	if fr != "" && fr != "FINISH_REASON_UNSPECIFIED" {
		finish = geminiFinishReasonToOpenAI(fr)
	}

	model, _ := event["modelVersion"].(string)
	chunk := map[string]any{
		"id":      "gemini-stream-" + model,
		"object":  "chat.completion.chunk",
		"created": 0,
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"content": textContent,
				},
				"finish_reason": finish,
			},
		},
	}

	return "data: " + mustMarshal(chunk) + "\n\n"
}

// TranslateGeminiStream translates an entire Gemini SSE stream body into an
// OpenAI SSE stream.
func TranslateGeminiStream(body io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini stream body: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		translated := TranslateGeminiStreamChunk(line)
		if translated != "" {
			buf.WriteString(translated)
		}
	}

	return buf.Bytes(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Utility helpers
// ─────────────────────────────────────────────────────────────────────────────

// copyParam copies a field from src to dst, optionally renaming it.
func copyParam(src, dst map[string]any, srcKey string, dstKeys ...string) {
	dstKey := srcKey
	if len(dstKeys) > 0 {
		dstKey = dstKeys[0]
	}
	if v, ok := src[srcKey]; ok {
		dst[dstKey] = v
	}
}

// mustMarshal JSON-encodes a value; panics on error (used in SSE line builders).
func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
