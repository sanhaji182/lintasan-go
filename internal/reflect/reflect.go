package reflect

import (
	"context"
	"fmt"
	"time"
)

// Verifier is a function that evaluates generated output and returns a score + errors.
type Verifier func(output string) (result VerifyResult)

// VerifyResult holds the result of a single verification run.
type VerifyResult struct {
	Score  float64  // 0.0-1.0, 1.0 = perfect
	Passed int
	Total  int
	Errors []string // human-readable error messages
	Output string   // raw verification output (pytest stdout)
}

// Generator is a function that calls the LLM with a prompt and returns the response.
type Generator func(prompt string, previousErrors []string) (response string, err error)

// Attempt records one iteration of the loop.
type Attempt struct {
	Iteration int
	Prompt    string // the prompt sent to LLM
	Response  string // raw LLM response
	Score     float64
	Passed    int
	Total     int
	Errors    []string
	Duration  time.Duration
}

// Result is the final output of the reflect loop.
type Result struct {
	BestResponse string
	BestScore    float64
	Attempts     []Attempt
	Iterations   int
	Duration     time.Duration
}

// Reflect runs the self-review loop.
// - maxIterations: maximum regeneration attempts (1-5, default 3)
// - generator: function that calls LLM
// - verifier: function that scores the output
// - initialPrompt: the original user prompt
// - earlyExit: if true, return immediately on perfect score (1.0)
func Reflect(ctx context.Context, maxIterations int, generator Generator, verifier Verifier, initialPrompt string, earlyExit bool) (*Result, error) {
	if maxIterations < 1 {
		maxIterations = 1
	}
	if maxIterations > 5 {
		maxIterations = 5
	}

	result := &Result{}
	start := time.Now()

	prompt := initialPrompt
	var prevErrors []string

	for i := 0; i < maxIterations; i++ {
		attemptStart := time.Now()

		// Generate
		response, err := generator(prompt, prevErrors)
		if err != nil {
			return result, fmt.Errorf("iteration %d: generate: %w", i+1, err)
		}

		// Verify
		vr := verifier(response)

		attempt := Attempt{
			Iteration: i + 1,
			Prompt:    prompt,
			Response:  response,
			Score:     vr.Score,
			Passed:    vr.Passed,
			Total:     vr.Total,
			Errors:    vr.Errors,
			Duration:  time.Since(attemptStart),
		}
		result.Attempts = append(result.Attempts, attempt)

		// Track best
		if vr.Score >= result.BestScore {
			result.BestScore = vr.Score
			result.BestResponse = response
		}

		// Perfect? Early exit
		if earlyExit && vr.Score >= 1.0 {
			result.Iterations = i + 1
			result.Duration = time.Since(start)
			return result, nil
		}

		// Build error context for next iteration
		if vr.Score < 1.0 && i < maxIterations-1 {
			prevErrors = vr.Errors
			prompt = buildFixPrompt(initialPrompt, response, vr)
		}
	}

	result.Iterations = maxIterations
	result.Duration = time.Since(start)
	return result, nil
}

// buildFixPrompt creates the follow-up prompt with error feedback.
func buildFixPrompt(original string, lastResponse string, vr VerifyResult) string {
	errorSummary := ""
	limit := len(vr.Errors)
	if limit > 5 {
		limit = 5
	}
	for _, e := range vr.Errors[:limit] {
		errorSummary += "- " + e + "\n"
	}

	return fmt.Sprintf(`You previously generated this code which scored %.0f%% (%d/%d tests passed):

Failing tests:
%s
Your previous response:
%s

Fix ALL failing tests. Return ONLY the corrected code in a Python code block.`,
		vr.Score*100, vr.Passed, vr.Total, errorSummary, truncateStr(lastResponse, 3000))
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}
