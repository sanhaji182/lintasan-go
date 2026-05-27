// Package rtk provides RTK (Rust Token Killer) integration for
// compressing tool output before it's sent to the LLM. This reduces
// input tokens by collapsing verbose terminal output (git diffs,
// directory listings, grep output, JSON, logs) using a subprocess
// call to `rtk rewrite` when available, or falling back to built-in
// Go regex compressors.
package rtk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Config holds the optional configuration for the RTK compressor.
type Config struct {
	// RTKPath is the path to the `rtk` binary. If empty, "rtk" is used
	// (looked up via PATH).
	RTKPath string `json:"rtk_path"`
	// Enabled controls whether RTK compression is active.
	Enabled bool `json:"enabled"`
	// MaxInputSize is the maximum size in bytes of content to pass to RTK.
	// Content larger than this is compressed with built-in methods only.
	MaxInputSize int `json:"max_input_size"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		RTKPath:      "",
		Enabled:      true,
		MaxInputSize: 500_000,
	}
}

// Compressor applies RTK-style token saving transformations to tool output.
type Compressor struct {
	cfg      Config
	rtkFound bool // cached result of checking for rtk binary
}

// New creates a new Compressor with the given config.
func New(cfg Config) *Compressor {
	c := &Compressor{cfg: cfg}
	c.rtkFound = c.checkRTK()
	return c
}

// checkRTK verifies if the `rtk` binary is available and reports it.
func (c *Compressor) checkRTK() bool {
	path := c.cfg.RTKPath
	if path == "" {
		path = "rtk"
	}
	_, err := exec.LookPath(path)
	return err == nil
}

// IsAvailable returns true when RTK is installed and enabled.
func (c *Compressor) IsAvailable() bool {
	return c.cfg.Enabled && c.rtkFound
}

// Rewrite compresses tool output. Attempts `rtk rewrite` first,
// then falls back to built-in compression. Returns the original
// input unchanged if compression wouldn't save tokens.
func (c *Compressor) Rewrite(input string) string {
	if !c.cfg.Enabled || input == "" {
		return input
	}

	// Skip very short content (no savings possible)
	if len(input) < 100 {
		return input
	}

	var result string
	var usedRTK bool

	// Try rtk rewrite subprocess first (respects max_input_size)
	if c.rtkFound && len(input) <= c.cfg.MaxInputSize {
		if out, ok := c.callRTK(input); ok {
			result = out
			usedRTK = true
		}
	}

	// Fall back to built-in Go compressors
	if !usedRTK {
		result = c.builtinCompress(input)
	}

	// Safety: never return empty, never grow input
	if result == "" || len(result) >= len(input) {
		return input
	}
	return result
}

// callRTK invokes `rtk rewrite` as a subprocess.
func (c *Compressor) callRTK(input string) (string, bool) {
	path := c.cfg.RTKPath
	if path == "" {
		path = "rtk"
	}

	cmd := exec.Command(path, "rewrite")
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = nil // discard stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", false
	}

	result := out.String()
	if result == "" {
		return "", false
	}
	return result, true
}

// builtinCompress applies regex-based compression for common tool output formats.
func (c *Compressor) builtinCompress(input string) string {
	contentType := detectContentType(input)

	switch contentType {
	case "git-diff":
		return compressGitDiff(input)
	case "directory-listing":
		return compressDirectoryListing(input)
	case "grep-output":
		return compressGrepOutput(input)
	case "json":
		return compressJSON(input)
	case "log-output":
		return compressLogOutput(input)
	default:
		return compressGeneric(input)
	}
}

// detectContentType classifies the first 20 lines of content.
func detectContentType(text string) string {
	lines := strings.Split(text, "\n")
	checkLines := lines
	if len(checkLines) > 20 {
		checkLines = checkLines[:20]
	}

	// Git diff
	for _, l := range checkLines {
		if strings.HasPrefix(l, "diff --git") || strings.HasPrefix(l, "@@") {
			return "git-diff"
		}
	}
	hasPlus := false
	hasMinus := false
	for _, l := range checkLines {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			hasPlus = true
		}
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			hasMinus = true
		}
	}
	if hasPlus && hasMinus {
		return "git-diff"
	}

	// Directory listing (ls -la, tree)
	for _, l := range checkLines {
		if matched, _ := regexp.MatchString(`^[d\-rwx]{10}`, l); matched {
			return "directory-listing"
		}
		if strings.HasPrefix(l, "├") || strings.HasPrefix(l, "└") || strings.HasPrefix(l, "│") {
			return "directory-listing"
		}
	}

	// Grep output (file:line:content)
	grepCount := 0
	grepPattern := regexp.MustCompile(`^[^:]+:\d+:`)
	for _, l := range checkLines {
		if grepPattern.MatchString(l) {
			grepCount++
		}
	}
	if grepCount > 3 {
		return "grep-output"
	}

	// JSON
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var dummy any
		if json.Unmarshal([]byte(text), &dummy) == nil {
			return "json"
		}
	}

	// Log output (timestamps)
	logCount := 0
	logPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}|^\[\d{2}:\d{2}|^[A-Z]{3,5}\s`)
	for _, l := range checkLines {
		if logPattern.MatchString(l) {
			logCount++
		}
	}
	if logCount > 3 {
		return "log-output"
	}

	return "generic"
}

// compressGitDiff reduces context lines in unified diffs.
func compressGitDiff(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	contextCount := 0
	maxContext := 2

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "@@") {
			contextCount = 0
			result = append(result, line)
		} else if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			contextCount = 0
			result = append(result, line)
		} else {
			contextCount++
			if contextCount <= maxContext {
				result = append(result, line)
			} else if contextCount == maxContext+1 {
				result = append(result, "  ...")
			}
		}
	}
	return strings.Join(result, "\n")
}

// compressDirectoryListing cleans up ls/tree output.
func compressDirectoryListing(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	permRe := regexp.MustCompile(`^[d\-rwx]{10}\s+\d+\s+\S+\s+\S+\s+\d+\s+\S+\s+\d+\s+[\d:]+\s+`)

	for _, line := range lines {
		// Strip permission/size columns from ls -la
		cleaned := permRe.ReplaceAllString(line, "")
		// Count tree depth
		depth := strings.Count(line, "│") + strings.Count(line, "├") + strings.Count(line, "└")
		// Also count leading spaces for non-tree listings
		leadingSpaces := 0
		for _, ch := range line {
			if ch == ' ' {
				leadingSpaces++
			} else {
				break
			}
		}
		treeDepth := depth + leadingSpaces/2

		if treeDepth <= 8 || strings.Contains(cleaned, "/") {
			result = append(result, cleaned)
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// compressGrepOutput deduplicates file paths in grep output.
func compressGrepOutput(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	lastFile := ""
	grepPattern := regexp.MustCompile(`^([^:]+):(\d+):(.*)`)

	for _, line := range lines {
		match := grepPattern.FindStringSubmatch(line)
		if match != nil {
			file := match[1]
			lineNum := match[2]
			content := match[3]
			if file != lastFile {
				result = append(result, "")
				result = append(result, file+":")
				lastFile = file
			}
			result = append(result, fmt.Sprintf("  %s: %s", lineNum, strings.TrimSpace(content)))
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// compressJSON minifies pretty-printed JSON when savings exceed 20%.
func compressJSON(text string) string {
	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return text
	}
	minified, err := json.Marshal(parsed)
	if err != nil {
		return text
	}
	if float64(len(minified)) < float64(len(text))*0.8 {
		return string(minified)
	}
	return text
}

// compressLogOutput deduplicates repeated log patterns.
func compressLogOutput(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 20 {
		return text
	}

	var result []string
	var lastPattern string
	repeatCount := 0
	tsRe := regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[.\d]*Z?`)
	tsBracketRe := regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\]`)

	for _, line := range lines {
		pattern := tsRe.ReplaceAllString(line, "[TS]")
		pattern = tsBracketRe.ReplaceAllString(pattern, "[TS]")

		if pattern == lastPattern {
			repeatCount++
		} else {
			if repeatCount > 1 {
				result = append(result, fmt.Sprintf("  ... (×%d similar)", repeatCount))
			}
			result = append(result, line)
			lastPattern = pattern
			repeatCount = 1
		}
	}
	if repeatCount > 1 {
		result = append(result, fmt.Sprintf("  ... (×%d similar)", repeatCount))
	}
	return strings.Join(result, "\n")
}

// compressGeneric removes excessive whitespace.
func compressGeneric(text string) string {
	// Max 2 consecutive newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	// Trailing whitespace
	text = regexp.MustCompile(`(?m)[ \t]+$`).ReplaceAllString(text, "")
	return text
}

// CompressMessages compresses tool_result content in OpenAI-format messages.
// This is the pre-tool-call hook: compress terminal output before sending to LLM.
// It mutates messages in place and returns whether any compression occurred.
func (c *Compressor) CompressMessages(messages []map[string]any) ([]map[string]any, bool) {
	if !c.cfg.Enabled {
		return messages, false
	}

	changed := false
	for i, msg := range messages {
		role, _ := msg["role"].(string)

		// OpenAI tool messages
		if role == "tool" {
			if content, ok := msg["content"].(string); ok && len(content) > 100 {
				compressed := c.Rewrite(content)
				if compressed != content {
					msg["content"] = compressed
					messages[i] = msg
					changed = true
				}
			}
		}

		// System messages with very long content
		if role == "system" {
			if content, ok := msg["content"].(string); ok && len(content) > 2000 {
				compressed := c.Rewrite(content)
				if compressed != content {
					msg["content"] = compressed
					messages[i] = msg
					changed = true
				}
			}
		}
	}

	return messages, changed
}
