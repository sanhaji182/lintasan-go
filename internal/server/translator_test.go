package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// msgs is a helper to build []any message slices without triggering Go's
// "invalid composite literal type any" errors.
func msgs(msgs ...any) []any { return msgs }

// msg is a helper to build map[string]any for test message maps.
func msg(role, content any) map[string]any {
	return map[string]any{"role": role, "content": content}
}

// ─────────────────────────────────────────────────────────────────────────────
// OpenAI → Anthropic request conversion
// ─────────────────────────────────────────────────────────────────────────────

func TestOpenAIToAnthropic_Basic(t *testing.T) {
	req := map[string]any{
		"model":       "claude-opus-4-20250514",
		"max_tokens":  float64(4096),
		"temperature": 0.7,
		"messages": msgs(
			msg("system", "You are a helpful assistant."),
			msg("user", "Hello!"),
		),
	}

	result := OpenAIToAnthropic(req)

	if result["model"] != "claude-opus-4-20250514" {
		t.Errorf("model = %v, want claude-opus-4-20250514", result["model"])
	}
	if result["max_tokens"] != float64(4096) {
		t.Errorf("max_tokens = %v, want 4096", result["max_tokens"])
	}
	if result["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want 0.7", result["temperature"])
	}
}

func TestOpenAIToAnthropic_SystemPromptExtraction(t *testing.T) {
	req := map[string]any{
		"model": "claude-opus-4",
		"messages": msgs(
			msg("system", "System prompt"),
			msg("user", "Hello"),
		),
	}

	result := OpenAIToAnthropic(req)

	if result["system"] != "System prompt" {
		t.Errorf("system = %v, want 'System prompt'", result["system"])
	}

	messages, ok := result["messages"].([]map[string]any)
	if !ok {
		t.Fatalf("messages is not []map[string]any")
	}
	if len(messages) != 1 {
		t.Errorf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0]["role"] != "user" {
		t.Errorf("messages[0].role = %v, want 'user'", messages[0]["role"])
	}
}

func TestOpenAIToAnthropic_SystemPromptConcatenation(t *testing.T) {
	req := map[string]any{
		"model": "claude-sonnet-4-20250514",
		"messages": msgs(
			msg("system", "First system"),
			msg("system", "Second system"),
			msg("user", "Hello"),
		),
	}

	result := OpenAIToAnthropic(req)

	system, ok := result["system"].(string)
	if !ok {
		t.Fatalf("system is not string")
	}
	if !strings.Contains(system, "First system") {
		t.Errorf("system does not contain 'First system': %s", system)
	}
	if !strings.Contains(system, "Second system") {
		t.Errorf("system does not contain 'Second system': %s", system)
	}
}

func TestOpenAIToAnthropic_MultiModalContent(t *testing.T) {
	req := map[string]any{
		"model": "claude-opus-4-20250514",
		"messages": msgs(
			map[string]any{
				"role": "user",
				"content": msgs(
					map[string]any{"type": "text", "text": "Describe this image."},
					map[string]any{
						"type": "image_url",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/png",
							"data":       "abc123",
						},
					},
				),
			},
		),
	}

	result := OpenAIToAnthropic(req)

	messages, ok := result["messages"].([]map[string]any)
	if !ok {
		t.Fatalf("messages is not []map[string]any")
	}

	content, ok := messages[0]["content"].([]any)
	if !ok {
		t.Fatalf("content is not []any")
	}
	if len(content) != 2 {
		t.Errorf("len(content) = %d, want 2", len(content))
	}

	textBlock, ok := content[0].(map[string]any)
	if !ok || textBlock["type"] != "text" || textBlock["text"] != "Describe this image." {
		t.Errorf("text block = %v", content[0])
	}

	imgBlock, ok := content[1].(map[string]any)
	if !ok || imgBlock["type"] != "image" {
		t.Errorf("image block = %v", content[1])
	}
}

func TestOpenAIToAnthropic_ToolUse(t *testing.T) {
	req := map[string]any{
		"model": "claude-opus-4",
		"messages": msgs(
			map[string]any{
				"role": "assistant",
				"content": msgs(
					map[string]any{
						"type":      "tool_use",
						"id":        "tool_1",
						"name":      "get_weather",
						"arguments": "{\"city\":\"NYC\"}",
					},
				),
			},
		),
	}

	result := OpenAIToAnthropic(req)

	messages, ok := result["messages"].([]map[string]any)
	if !ok {
		t.Fatalf("messages is not []map[string]any")
	}

	content, ok := messages[0]["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content is empty")
	}

	toolUse, ok := content[0].(map[string]any)
	if !ok || toolUse["type"] != "tool_use" {
		t.Errorf("tool_use block = %v", content[0])
	}
	if toolUse["id"] != "tool_1" {
		t.Errorf("tool_use.id = %v, want tool_1", toolUse["id"])
	}
}

func TestOpenAIToAnthropic_StreamFlag(t *testing.T) {
	req := map[string]any{
		"model": "claude-opus-4",
		"stream": true,
		"messages": msgs(
			msg("user", "Hello"),
		),
	}

	result := OpenAIToAnthropic(req)
	if result["stream"] != true {
		t.Errorf("stream = %v, want true", result["stream"])
	}
}

func TestOpenAIToAnthropic_DeveloperRole(t *testing.T) {
	req := map[string]any{
		"model": "claude-opus-4",
		"messages": msgs(
			msg("developer", "You are a pirate."),
			msg("user", "Tell me a story."),
		),
	}

	result := OpenAIToAnthropic(req)
	// developer role → treated like system, extracted to top-level "system" field
	if result["system"] != "You are a pirate." {
		t.Errorf("system = %v, want 'You are a pirate.'", result["system"])
	}
	messages, _ := result["messages"].([]map[string]any)
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1 (only user message remains)", len(messages))
	}
	if messages[0]["role"] != "user" {
		t.Errorf("role = %v, want 'user'", messages[0]["role"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Anthropic → OpenAI response conversion
// ─────────────────────────────────────────────────────────────────────────────

func TestAnthropicToOpenAI_Basic(t *testing.T) {
	anthropicResp := map[string]any{
		"id":      "msg_123",
		"type":    "message",
		"model":   "claude-opus-4-20250514",
		"content": msgs(map[string]any{"type": "text", "text": "Hello, world!"}),
		"stop_reason": "end_turn",
		"usage": map[string]any{
			"input_tokens":  10.0,
			"output_tokens": 20.0,
		},
	}

	body, err := json.Marshal(anthropicResp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result, err := AnthropicToOpenAI(body)
	if err != nil {
		t.Fatalf("AnthropicToOpenAI error: %v", err)
	}

	var openAI map[string]any
	if err := json.Unmarshal(result, &openAI); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if openAI["id"] != "msg_123" {
		t.Errorf("id = %v, want msg_123", openAI["id"])
	}
	if openAI["object"] != "chat.completion" {
		t.Errorf("object = %v, want chat.completion", openAI["object"])
	}

	choices, ok := openAI["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatalf("choices is empty")
	}
	choice, _ := choices[0].(map[string]any)
	msg2, _ := choice["message"].(map[string]any)
	if msg2["content"] != "Hello, world!" {
		t.Errorf("content = %v, want 'Hello, world!'", msg2["content"])
	}

	usage, _ := openAI["usage"].(map[string]any)
	if v, ok := usage["prompt_tokens"].(float64); !ok || int(v) != 10 {
		t.Errorf("prompt_tokens = %v, want 10", usage["prompt_tokens"])
	}
}

func TestAnthropicToOpenAI_InvalidJSON(t *testing.T) {
	_, err := AnthropicToOpenAI([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExtractAnthropicUsage(t *testing.T) {
	resp := map[string]any{
		"usage": map[string]any{
			"input_tokens":  100.0,
			"output_tokens": 200.0,
		},
	}

	usage := extractAnthropicUsage(resp)
	if v, ok := usage["prompt_tokens"].(float64); !ok || int(v) != 100 {
		t.Errorf("prompt_tokens = %v, want 100", usage["prompt_tokens"])
	}
	if v, ok := usage["completion_tokens"].(float64); !ok || int(v) != 200 {
		t.Errorf("completion_tokens = %v, want 200", usage["completion_tokens"])
	}
	if v, ok := usage["total_tokens"].(float64); !ok || int(v) != 300 {
		t.Errorf("total_tokens = %v, want 300", usage["total_tokens"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// OpenAI → Gemini request conversion
// ─────────────────────────────────────────────────────────────────────────────

func TestOpenAIToGemini_Basic(t *testing.T) {
	req := map[string]any{
		"model":       "gemini-2.5-pro",
		"max_tokens":  float64(1024),
		"temperature": 0.5,
		"messages": msgs(
			msg("user", "Hello!"),
			msg("assistant", "Hi there!"),
			msg("user", "How are you?"),
		),
	}

	result := OpenAIToGemini(req)

	if result["model"] != "gemini-2.5-pro" {
		t.Errorf("model = %v, want gemini-2.5-pro", result["model"])
	}

	gc, ok := result["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig missing")
	}
	if gc["maxOutputTokens"] != 1024 {
		t.Errorf("maxOutputTokens = %v, want 1024", gc["maxOutputTokens"])
	}

	contents, ok := result["contents"].([]map[string]any)
	if !ok {
		t.Fatalf("contents is not []map[string]any")
	}
	if len(contents) != 3 {
		t.Errorf("len(contents) = %d, want 3", len(contents))
	}

	if contents[0]["role"] != "user" {
		t.Errorf("contents[0].role = %v, want 'user'", contents[0]["role"])
	}
	if contents[1]["role"] != "model" {
		t.Errorf("contents[1].role = %v, want 'model'", contents[1]["role"])
	}
}

func TestOpenAIToGemini_SystemInstruction(t *testing.T) {
	req := map[string]any{
		"model": "gemini-2.5-flash",
		"messages": msgs(
			msg("system", "You are a coding assistant."),
			msg("user", "Write a function."),
		),
	}

	result := OpenAIToGemini(req)

	si, ok := result["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatalf("systemInstruction missing")
	}
	parts, ok := si["parts"].([]map[string]any)
	if !ok || len(parts) == 0 {
		t.Fatalf("systemInstruction.parts missing")
	}
	if parts[0]["text"] != "You are a coding assistant." {
		t.Errorf("system text = %v", parts[0]["text"])
	}

	contents, _ := result["contents"].([]map[string]any)
	if len(contents) != 1 {
		t.Errorf("len(contents) = %d, want 1", len(contents))
	}
}

func TestOpenAIToGemini_MultiModalContent(t *testing.T) {
	req := map[string]any{
		"model": "gemini-2.5-pro",
		"messages": msgs(
			map[string]any{
				"role": "user",
				"content": msgs(
					map[string]any{"type": "text", "text": "Describe this image."},
					map[string]any{
						"type": "image_url",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/jpeg",
							"data":       "SGVsbG8=",
						},
					},
				),
			},
		),
	}

	result := OpenAIToGemini(req)

	contents, _ := result["contents"].([]map[string]any)
	parts, _ := contents[0]["parts"].([]map[string]any)

	if parts[0]["text"] != "Describe this image." {
		t.Errorf("text part = %v", parts[0]["text"])
	}

	inline, ok := parts[1]["inlineData"].(map[string]any)
	if !ok {
		t.Fatalf("inlineData missing")
	}
	if inline["data"] != "SGVsbG8=" {
		t.Errorf("inlineData.data = %v", inline["data"])
	}
	if inline["mimeType"] != "image/jpeg" {
		t.Errorf("inlineData.mimeType = %v", inline["mimeType"])
	}
}

func TestOpenAIToGemini_StopSequenceRename(t *testing.T) {
	req := map[string]any{
		"model": "gemini-2.5-pro",
		"messages": msgs(
			msg("user", "Hello"),
		),
		"stop": "STOP",
	}

	result := OpenAIToGemini(req)

	gc, ok := result["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig missing")
	}
	stopSeq, ok := gc["stopSequences"].([]string)
	if !ok || len(stopSeq) == 0 || stopSeq[0] != "STOP" {
		t.Errorf("stopSequences = %v, want ['STOP']", gc["stopSequences"])
	}
}

func TestOpenAIToGemini_TopK(t *testing.T) {
	req := map[string]any{
		"model": "gemini-2.5-pro",
		"messages": msgs(
			msg("user", "Hello"),
		),
		"top_k": 40.0,
	}

	result := OpenAIToGemini(req)
	gc := result["generationConfig"].(map[string]any)
	if gc["topK"] != 40 {
		t.Errorf("topK = %v, want 40", gc["topK"])
	}
}

func TestOpenAIToGemini_URLImageAsText(t *testing.T) {
	req := map[string]any{
		"model": "gemini-2.5-pro",
		"messages": msgs(
			map[string]any{
				"role": "user",
				"content": msgs(
					map[string]any{
						"type": "image_url",
						"source": map[string]any{
							"type": "url",
							"url":   "https://example.com/image.jpg",
						},
					},
				),
			},
		),
	}

	result := OpenAIToGemini(req)
	contents, _ := result["contents"].([]map[string]any)
	parts, _ := contents[0]["parts"].([]map[string]any)

	if len(parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(parts))
	}
	text, ok := parts[0]["text"].(string)
	if !ok || !strings.Contains(text, "https://example.com/image.jpg") {
		t.Errorf("URL image not converted to text part: %v", parts[0])
	}
}

func TestOpenAIToGemini_EmptyMessages(t *testing.T) {
	req := map[string]any{
		"model":    "gemini-2.5-pro",
		"messages": msgs(),
	}

	result := OpenAIToGemini(req)
	if _, ok := result["contents"]; ok {
		contents := result["contents"].([]map[string]any)
		if len(contents) != 0 {
			t.Errorf("empty messages should produce empty contents, got %d", len(contents))
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Gemini → OpenAI response conversion
// ─────────────────────────────────────────────────────────────────────────────

func TestGeminiToOpenAI_Basic(t *testing.T) {
	geminiResp := map[string]any{
		"model": "gemini-2.5-pro",
		"candidates": msgs(
			map[string]any{
				"content": map[string]any{
					"parts": msgs(map[string]any{"text": "Hello from Gemini!"}),
					"role":  "model",
				},
				"finishReason": "STOP",
			},
		),
		"usageMetadata": map[string]any{
			"promptTokenCount":      10.0,
			"candidatesTokenCount":  20.0,
			"totalTokenCount":       30.0,
		},
	}

	body, err := json.Marshal(geminiResp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result, err := GeminiToOpenAI(body)
	if err != nil {
		t.Fatalf("GeminiToOpenAI error: %v", err)
	}

	var openAI map[string]any
	if err := json.Unmarshal(result, &openAI); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if openAI["object"] != "chat.completion" {
		t.Errorf("object = %v, want chat.completion", openAI["object"])
	}

	choices, _ := openAI["choices"].([]any)
	choice, _ := choices[0].(map[string]any)
	msg2, _ := choice["message"].(map[string]any)
	if msg2["content"] != "Hello from Gemini!" {
		t.Errorf("content = %v, want 'Hello from Gemini!'", msg2["content"])
	}

	usage, _ := openAI["usage"].(map[string]any)
	if v, ok := usage["prompt_tokens"].(float64); !ok || int(v) != 10 {
		t.Errorf("prompt_tokens = %v, want 10", usage["prompt_tokens"])
	}
	if v, ok := usage["completion_tokens"].(float64); !ok || int(v) != 20 {
		t.Errorf("completion_tokens = %v, want 20", usage["completion_tokens"])
	}
}

func TestGeminiToOpenAI_InvalidJSON(t *testing.T) {
	_, err := GeminiToOpenAI([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGeminiFinishReasonToOpenAI(t *testing.T) {
	tests := []struct {
		gemini string
		openai string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"MALFORMED_FUNCTION_CALL", "content_filter"},
		{"OTHER", "stop"},
	}

	for _, tt := range tests {
		result := geminiFinishReasonToOpenAI(tt.gemini)
		if result != tt.openai {
			t.Errorf("geminiFinishReasonToOpenAI(%q) = %q, want %q", tt.gemini, result, tt.openai)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Stream translation
// ─────────────────────────────────────────────────────────────────────────────

func TestTranslateAnthropicStreamChunk(t *testing.T) {
	line := `{"type":"content_block_delta","message_id":"msg_1","model":"claude-opus-4-20250514","created_at":1234567890,"index":0,"delta":{"type":"text_delta","text":"Hello"}}`
	result := TranslateAnthropicStreamChunk(line)
	if result == "" {
		t.Fatal("expected non-empty result for text_delta")
	}
	if !strings.HasPrefix(result, "data: ") {
		t.Errorf("expected SSE data prefix, got: %s", result)
	}
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected 'Hello' in result: %s", result)
	}
	if !strings.Contains(result, "chat.completion.chunk") {
		t.Errorf("expected OpenAI chunk object type: %s", result)
	}
}

func TestTranslateAnthropicStreamChunk_SkipThinking(t *testing.T) {
	line := `{"type":"content_block_delta","message_id":"msg_1","delta":{"type":"thinking_delta","thinking":"..."}}`
	result := TranslateAnthropicStreamChunk(line)
	if result != "" {
		t.Errorf("expected empty result for thinking_delta, got: %s", result)
	}
}

func TestTranslateAnthropicStreamChunk_EmptyLine(t *testing.T) {
	result := TranslateAnthropicStreamChunk("")
	if result != "" {
		t.Errorf("expected empty result for empty line, got: %s", result)
	}
}

func TestTranslateGeminiStreamChunk(t *testing.T) {
	line := `data: {"candidates":[{"content":{"parts":[{"text":"Hi"}]},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash"}`
	result := TranslateGeminiStreamChunk(line)
	if result == "" {
		t.Fatal("expected non-empty result for Gemini stream chunk")
	}
	if !strings.HasPrefix(result, "data: ") {
		t.Errorf("expected SSE data prefix, got: %s", result)
	}
	if !strings.Contains(result, "Hi") {
		t.Errorf("expected 'Hi' in result: %s", result)
	}
}

func TestTranslateGeminiStreamChunk_EmptyLine(t *testing.T) {
	result := TranslateGeminiStreamChunk("")
	if result != "" {
		t.Errorf("expected empty result for empty line, got: %s", result)
	}
}

func TestTranslateAnthropicStream_Integration(t *testing.T) {
	body := "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"message_id\":\"msg_1\",\"model\":\"claude-sonnet-4-20250514\",\"created_at\":1234567890,\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello world\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\",\"message_id\":\"msg_1\"}\n"

	reader := strings.NewReader(body)
	result, err := TranslateAnthropicStream(reader)
	if err != nil {
		t.Fatalf("TranslateAnthropicStream error: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "Hello world") {
		t.Errorf("expected 'Hello world' in translated stream: %s", resultStr)
	}
}

func TestTranslateGeminiStream_Integration(t *testing.T) {
	body := "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]},\"finishReason\":\"STOP\"}],\"modelVersion\":\"gemini-2.5-flash\"}\n"

	reader := strings.NewReader(body)
	result, err := TranslateGeminiStream(reader)
	if err != nil {
		t.Fatalf("TranslateGeminiStream error: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "Hello") {
		t.Errorf("expected 'Hello' in translated stream: %s", resultStr)
	}
	if !strings.Contains(resultStr, "chat.completion.chunk") {
		t.Errorf("expected OpenAI chunk format: %s", resultStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper utilities
// ─────────────────────────────────────────────────────────────────────────────

func TestCopyParam(t *testing.T) {
	src := map[string]any{
		"temperature": 0.7,
		"max_tokens":  float64(1024),
		"top_p":       0.9,
	}
	dst := make(map[string]any)

	copyParam(src, dst, "temperature")
	if dst["temperature"] != 0.7 {
		t.Errorf("temperature not copied")
	}

	copyParam(src, dst, "max_tokens")
	if dst["max_tokens"] != float64(1024) {
		t.Errorf("max_tokens not copied")
	}

	copyParam(src, dst, "max_tokens", "maxOutputTokens")
	if dst["maxOutputTokens"] != float64(1024) {
		t.Errorf("maxOutputTokens not set via rename")
	}
}

func TestMustMarshal(t *testing.T) {
	result := mustMarshal(map[string]any{"key": "value"})
	if !strings.Contains(result, "key") {
		t.Errorf("expected 'key' in marshaled output: %s", result)
	}
}

func TestExtractTextContent(t *testing.T) {
	result := extractTextContent("Hello world")
	if result != "Hello world" {
		t.Errorf("string content = %q, want 'Hello world'", result)
	}

	result = extractTextContent(msgs(
		map[string]any{"type": "text", "text": "Part 1"},
		map[string]any{"type": "text", "text": " Part 2"},
	))
	if result != "Part 1 Part 2" {
		t.Errorf("array content = %q, want 'Part 1 Part 2'", result)
	}

	result = extractTextContent(123)
	if result != "" {
		t.Errorf("unknown type = %q, want ''", result)
	}
}

func TestConversionsCompile(t *testing.T) {
	anthropicReq := OpenAIToAnthropic(map[string]any{
		"model":      "claude-opus-4",
		"stream":     true,
		"max_tokens": float64(4096),
		"messages": msgs(
			msg("system", "You are helpful."),
			msg("user", "Hello"),
		),
	})
	if anthropicReq == nil {
		t.Error("OpenAIToAnthropic returned nil")
	}

	geminiReq := OpenAIToGemini(map[string]any{
		"model": "gemini-2.5-pro",
		"messages": msgs(
			msg("user", "Hello"),
		),
	})
	if geminiReq == nil {
		t.Error("OpenAIToGemini returned nil")
	}
}

// Verify translator functions accept io.Reader (stream interfaces)
func TestStreamInterfaces(t *testing.T) {
	var r1 io.Reader = bytes.NewReader([]byte("test"))
	var r2 io.Reader = bytes.NewReader([]byte("test"))

	if _, err := TranslateAnthropicStream(r1); err != nil {
		t.Errorf("TranslateAnthropicStream failed: %v", err)
	}
	if _, err := TranslateGeminiStream(r2); err != nil {
		t.Errorf("TranslateGeminiStream failed: %v", err)
	}
}

// Ensure the http import is actually used
var _ http.RoundTripper = nil
