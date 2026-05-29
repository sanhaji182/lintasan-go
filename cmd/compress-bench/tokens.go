package main

import (
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
	tiktokenloader "github.com/pkoukk/tiktoken-go-loader"
)

// Token counting uses the REAL cl100k_base BPE tokenizer (the encoding used by
// GPT-3.5/4 and a close proxy for most modern chat models), NOT a char/4
// estimate. The offline loader embeds the vocab so the harness needs no network
// and produces byte-identical, deterministic counts on every run.

var (
	encOnce sync.Once
	enc     *tiktoken.Tiktoken
	encErr  error
)

func tokenizer() (*tiktoken.Tiktoken, error) {
	encOnce.Do(func() {
		// Force the offline (embedded) BPE loader so we never hit the network.
		tiktoken.SetBpeLoader(tiktokenloader.NewOfflineLoader())
		enc, encErr = tiktoken.GetEncoding("cl100k_base")
	})
	return enc, encErr
}

// countTokens returns the exact cl100k_base token count for a string.
func countTokens(s string) int {
	e, err := tokenizer()
	if err != nil {
		// Should never happen with the embedded loader; fail loud in the runner.
		panic("tiktoken init failed: " + err.Error())
	}
	return len(e.Encode(s, nil, nil))
}

// renderMessage flattens one chat message into the text that actually gets
// tokenized on the wire: the role label plus its content. Counting the rendered
// form captures per-message role/formatting overhead the same way for both the
// before and after snapshots, so the % reduction stays apples-to-apples.
func renderMessage(msg map[string]any) string {
	role, _ := msg["role"].(string)
	var sb strings.Builder
	sb.WriteString(role)
	sb.WriteString("\n")
	sb.WriteString(extractContent(msg))
	return sb.String()
}

// extractContent pulls the text content from a message, handling both the
// plain-string and structured-array ([]{"type":"text","text":...}) shapes.
func extractContent(msg map[string]any) string {
	switch v := msg["content"].(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// countMessageTokens sums the rendered-token count across all messages.
func countMessageTokens(messages []map[string]any) int {
	total := 0
	for _, msg := range messages {
		total += countTokens(renderMessage(msg))
	}
	return total
}

// joinMessageContent concatenates every message's content into one blob, used
// by the quality check to scan for surviving critical markers.
func joinMessageContent(messages []map[string]any) string {
	var parts []string
	for _, msg := range messages {
		parts = append(parts, extractContent(msg))
	}
	return strings.Join(parts, "\n")
}
