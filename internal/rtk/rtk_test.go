package rtk

import (
	"strings"
	"testing"
)

// TestDefaultConfig verifies defaults are sensible.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if cfg.MaxInputSize != 500_000 {
		t.Errorf("MaxInputSize = %d, want 500000", cfg.MaxInputSize)
	}
}

// TestNewCompressor creates a compressor and checks availability.
func TestNewCompressor(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		MaxInputSize: 100_000,
		RTKPath:      "/nonexistent/rtk",
	}
	c := New(cfg)
	if c == nil {
		t.Fatal("expected non-nil compressor")
	}
	if c.IsAvailable() {
		t.Log("RTK unexpectedly available at /nonexistent/rtk")
	}
}

// TestRewriteShortContent tests that short content passes through unchanged.
func TestRewriteShortContent(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		MaxInputSize: 100_000,
		RTKPath:      "/nonexistent/rtk", // force builtin fallback
	}
	c := New(cfg)

	short := "hello"
	result := c.Rewrite(short)
	if result != short {
		t.Errorf("short content should pass through unchanged, got %q", result)
	}
}

// TestRewriteEmptyContent tests empty string handling.
func TestRewriteEmptyContent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RTKPath = "/nonexistent/rtk"
	c := New(cfg)

	result := c.Rewrite("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}

// TestBuiltinCompressGitDiff tests git diff compression.
func TestBuiltinCompressGitDiff(t *testing.T) {
	cfg := Config{Enabled: true, MaxInputSize: 100_000, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	// Create a large diff with many context lines
	var sb strings.Builder
	sb.WriteString("diff --git a/file.go b/file.go\n")
	sb.WriteString("--- a/file.go\n")
	sb.WriteString("+++ b/file.go\n")
	sb.WriteString("@@ -1,10 +1,12 @@\n")
	sb.WriteString(" unchanged line 1\n")
	sb.WriteString(" unchanged line 2\n")
	sb.WriteString(" unchanged line 3\n")
	sb.WriteString(" unchanged line 4\n")
	sb.WriteString(" unchanged line 5\n")
	sb.WriteString(" unchanged line 6\n")
	sb.WriteString(" unchanged line 7\n")
	sb.WriteString("-old line\n")
	sb.WriteString("+new line\n")
	sb.WriteString(" unchanged line 9\n")
	sb.WriteString(" unchanged line 10\n")

	input := sb.String()
	result := c.builtinCompress(input)

	// Should contain "..." for collapsed context
	if !strings.Contains(result, "...") {
		t.Error("expected '...' in compressed diff output")
	}

	// Should preserve the actual changes
	if !strings.Contains(result, "-old line") {
		t.Error("expected '-old line' preserved in compressed diff")
	}
	if !strings.Contains(result, "+new line") {
		t.Error("expected '+new line' preserved in compressed diff")
	}

	// Verify compression actually reduced size
	if len(result) >= len(input) {
		t.Errorf("expected compressed output to be smaller: in=%d out=%d", len(input), len(result))
	}
}

// TestBuiltinCompressDirectoryListing tests ls -la compression.
func TestBuiltinCompressDirectoryListing(t *testing.T) {
	cfg := Config{Enabled: true, MaxInputSize: 100_000, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	input := "drwxr-xr-x  5 user  staff    160 May 27 10:00 src\n"
	input += "-rw-r--r--  1 user  staff   1024 May 20 09:30 main.go\n"
	input += "-rw-r--r--  1 user  staff    512 Apr 15 08:00 README.md\n"

	result := c.builtinCompress(input)

	// Permission columns should be stripped
	if strings.Contains(result, "drwxr-xr-x") {
		t.Error("expected permission columns to be stripped")
	}
	// Filenames should be preserved
	if !strings.Contains(result, "main.go") {
		t.Error("expected 'main.go' to be preserved")
	}
	if !strings.Contains(result, "README.md") {
		t.Error("expected 'README.md' to be preserved")
	}
}

// TestBuiltinCompressGrep tests grep output compression.
func TestBuiltinCompressGrep(t *testing.T) {
	cfg := Config{Enabled: true, MaxInputSize: 100_000, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	input := "src/main.go:10:func main() {\n"
	input += "src/main.go:15:    fmt.Println(\"hello\")\n"
	input += "src/lib.go:3:package lib\n"
	input += "src/lib.go:7:func Helper() {}\n"

	result := c.builtinCompress(input)

	// File headers should be present
	if !strings.Contains(result, "src/main.go:") {
		t.Error("expected file header for main.go")
	}
	if !strings.Contains(result, "src/lib.go:") {
		t.Error("expected file header for lib.go")
	}
	// Line numbers should be preserved
	if !strings.Contains(result, "10:") {
		t.Error("expected line number 10")
	}
}

// TestBuiltinCompressJSON tests JSON minification.
func TestBuiltinCompressJSON(t *testing.T) {
	cfg := Config{Enabled: true, MaxInputSize: 100_000, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	// Pretty-printed JSON
	input := "{\n  \"name\": \"test\",\n  \"values\": [\n    1,\n    2,\n    3\n  ]\n}\n"
	result := c.builtinCompress(input)

	// Minified JSON should be smaller
	if len(result) >= len(input) {
		t.Errorf("expected minified JSON to be smaller: in=%d out=%d", len(input), len(result))
	}
	// Should still be valid JSON
	if !strings.Contains(result, "\"name\"") {
		t.Error("expected 'name' key in minified output")
	}
}

// TestBuiltinCompressLogOutput tests log deduplication.
func TestBuiltinCompressLogOutput(t *testing.T) {
	cfg := Config{Enabled: true, MaxInputSize: 100_000, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	// Generate 30 nearly-identical log lines
	input := ""
	for i := 0; i < 25; i++ {
		input += "2024-05-27 10:00:00 INFO Connection accepted from 127.0.0.1\n"
	}
	// Add a few distinct lines
	input += "2024-05-27 10:01:00 ERROR Connection refused\n"
	input += "2024-05-27 10:02:00 WARN Rate limit approaching\n"

	result := c.builtinCompress(input)

	// Check that compression actually happened (result is shorter)
	if len(result) >= len(input) {
		t.Errorf("expected compression: in=%d out=%d", len(input), len(result))
	}
}

// TestCompressMessages verifies the pre-tool-call hook behavior.
func TestCompressMessages(t *testing.T) {
	cfg := Config{Enabled: true, MaxInputSize: 100_000, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	// Build a large tool result that should be compressed
	largeContent := ""
	for i := 0; i < 50; i++ {
		largeContent += "drwxr-xr-x  5 user  staff    160 May 27 10:00 file_" + string(rune('a'+i%26)) + "\n"
	}

	messages := []map[string]any{
		{"role": "user", "content": "List files"},
		{"role": "tool", "content": largeContent},
		{"role": "system", "content": "short system prompt"},
	}

	compressed, changed := c.CompressMessages(messages)

	if !changed {
		t.Error("expected compression to change messages")
	}

	// Tool content should be compressed
	toolContent, ok := compressed[1]["content"].(string)
	if !ok {
		t.Fatal("tool content missing")
	}
	if toolContent == largeContent {
		t.Error("tool content should have been compressed")
	}
	if len(toolContent) >= len(largeContent) {
		t.Errorf("tool content not compressed: in=%d out=%d", len(largeContent), len(toolContent))
	}

	// System message (short) should NOT be compressed
	sysContent, ok := compressed[2]["content"].(string)
	if !ok {
		t.Fatal("system content missing")
	}
	if sysContent != "short system prompt" {
		t.Errorf("short system prompt should be unchanged, got %q", sysContent)
	}
}

// TestCompressMessagesDisabled verifies that disabled compressor is a no-op.
func TestCompressMessagesDisabled(t *testing.T) {
	cfg := Config{Enabled: false, RTKPath: "/nonexistent/rtk"}
	c := New(cfg)

	messages := []map[string]any{
		{"role": "tool", "content": "some tool output that is long enough to compress but it won't be"},
	}

	compressed, changed := c.CompressMessages(messages)

	if changed {
		t.Error("disabled compressor should not change messages")
	}
	if len(compressed) != len(messages) {
		t.Error("disabled compressor should not modify message count")
	}
}
