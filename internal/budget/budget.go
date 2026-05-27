package budget

import (
	"regexp"
	"strings"
)

// Budget tracks dynamic token allocation per request.
type Budget struct {
	MaxTokens    int // ceiling
	MinTokens    int // floor
	DefaultTokens int
}

// DefaultBudget returns sensible defaults.
func DefaultBudget() *Budget {
	return &Budget{
		MaxTokens:     16384,
		MinTokens:     512,
		DefaultTokens: 4096,
	}
}

// Analyze scores request complexity 0.0-1.0 and returns recommended token budget.
func (b *Budget) Analyze(messages []map[string]any) (score float64, budget int) {
	features := extractFeatures(messages)
	score = scoreFeatures(features)

	// Map score to budget range
	budget = b.MinTokens + int(score*float64(b.MaxTokens-b.MinTokens))
	if budget < b.MinTokens {
		budget = b.MinTokens
	}
	if budget > b.MaxTokens {
		budget = b.MaxTokens
	}
	return score, budget
}

// RetryBudget increases budget when finish_reason=length.
func (b *Budget) RetryBudget(currentBudget int) int {
	next := currentBudget * 2
	if next > b.MaxTokens {
		next = b.MaxTokens
	}
	return next
}

var (
	codeRe       = regexp.MustCompile("(?i)\b(implement|build|create|develop|write|code|function|class|api|endpoint)\b")
	debugRe      = regexp.MustCompile("(?i)\b(debug|fix|troubleshoot|error|bug|issue)\b")
	explainRe    = regexp.MustCompile("(?i)\b(explain|describe|how|why|what is)\b")
	compareRe    = regexp.MustCompile("(?i)\b(compare|vs|versus|difference|trade.?off)\b")
	securityRe   = regexp.MustCompile("(?i)\b(security|auth|encrypt|vulnerab|hack)\b")
	perfRe       = regexp.MustCompile("(?i)\b(performance|optimi[sz]e|scale|latency|throughput)\b")
	archRe       = regexp.MustCompile("(?i)\b(architect|design|system|infrastructure|pattern)\b")
)

type features struct {
	wordCount     float64
	charCount     float64
	hasCode       float64
	hasDebug      float64
	hasExplain    float64
	hasCompare    float64
	hasSecurity   float64
	hasPerf       float64
	hasArch       float64
	toolCount     float64
}

func extractFeatures(messages []map[string]any) features {
	var f features
	var totalText string
	for _, m := range messages {
		role, _ := m["role"].(string)
		if role == "system" {
			continue
		}
		if content, ok := m["content"].(string); ok {
			totalText += " " + content
		}
		if _, ok := m["tool_calls"]; ok {
			f.toolCount++
		}
	}

	words := strings.Fields(totalText)
	f.wordCount = float64(len(words))
	f.charCount = float64(len(totalText))

	if codeRe.MatchString(totalText) { f.hasCode = 1 }
	if debugRe.MatchString(totalText) { f.hasDebug = 1 }
	if explainRe.MatchString(totalText) { f.hasExplain = 1 }
	if compareRe.MatchString(totalText) { f.hasCompare = 1 }
	if securityRe.MatchString(totalText) { f.hasSecurity = 1 }
	if perfRe.MatchString(totalText) { f.hasPerf = 1 }
	if archRe.MatchString(totalText) { f.hasArch = 1 }

	return f
}

func scoreFeatures(f features) float64 {
	score := 0.0
	// Length-based
	score += minFloat(f.wordCount/200, 0.25)
	score += minFloat(f.charCount/4000, 0.15)
	// Complexity flags
	score += f.hasCode * 0.15
	score += f.hasDebug * 0.10
	score += f.hasSecurity * 0.08
	score += f.hasArch * 0.10
	score += f.hasPerf * 0.07
	// Simpler tasks get lower score
	score += f.hasExplain * 0.03
	score += f.hasCompare * 0.03
	// Tools
	score += f.toolCount * 0.04

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.1 {
		score = 0.1
	}
	return score
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
