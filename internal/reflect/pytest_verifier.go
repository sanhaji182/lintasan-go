package reflect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// NewPytestVerifier creates a Verifier that runs Python code through pytest.
// testFile: path to the pytest test file (e.g., "/tmp/test_foo.py")
// codeModule: path where the generated code will be written (e.g., "/tmp/duel_foo_buggy.py")
func NewPytestVerifier(testFile, codeModule string) Verifier {
	return func(output string) VerifyResult {
		// Extract code from response (try ```python block first)
		code := ExtractCodeBlock(output, "python")
		if code == "" {
			code = ExtractCodeBlock(output, "")
		}
		if code == "" {
			return VerifyResult{Score: 0, Errors: []string{"no code block found in response"}}
		}

		// Write code to module file
		if err := os.WriteFile(codeModule, []byte(code), 0644); err != nil {
			return VerifyResult{Score: 0, Errors: []string{fmt.Sprintf("write error: %v", err)}}
		}

		// Clear __pycache__
		os.RemoveAll(strings.TrimSuffix(codeModule, ".py") + "/__pycache__")
		os.RemoveAll("/tmp/__pycache__")

		// Run pytest
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "python3", "-m", "pytest", testFile,
			"-v", "--tb=short", "--no-header", "-p", "no:cacheprovider")
		cmd.Dir = "/tmp"
		output2, _ := cmd.CombinedOutput()
		outStr := string(output2)

		// Parse results
		passed := len(regexp.MustCompile(`::test_\w+ PASSED`).FindAllString(outStr, -1))
		failed := len(regexp.MustCompile(`::test_\w+ FAILED`).FindAllString(outStr, -1))
		total := passed + failed

		if total == 0 {
			// Fallback to summary line
			if m := regexp.MustCompile(`(\d+) passed`).FindStringSubmatch(outStr); m != nil {
				fmt.Sscanf(m[1], "%d", &passed)
			}
			if m := regexp.MustCompile(`(\d+) failed`).FindStringSubmatch(outStr); m != nil {
				fmt.Sscanf(m[1], "%d", &failed)
			}
			total = passed + failed
		}

		if total == 0 {
			return VerifyResult{
				Score:  0,
				Errors: []string{"pytest could not parse: " + outStr[:minInt(200, len(outStr))]},
			}
		}

		score := float64(passed) / float64(total)

		// Extract failing test names
		var errors []string
		failMatches := regexp.MustCompile(`FAILED (\S+)`).FindAllStringSubmatch(outStr, -1)
		for _, m := range failMatches {
			if len(m) > 1 {
				errors = append(errors, m[1])
			}
		}

		return VerifyResult{
			Score:  score,
			Passed: passed,
			Total:  total,
			Errors: errors,
			Output: outStr,
		}
	}
}

// ExtractCodeBlock extracts a code block of given language from text.
func ExtractCodeBlock(text, lang string) string {
	pattern := "```" + lang + `\s*\n(.*?)` + "```"
	re := regexp.MustCompile(`(?s)` + pattern)
	matches := re.FindAllStringSubmatch(text, -1)
	var best string
	for _, m := range matches {
		if len(m) > 1 && len(m[1]) > len(best) {
			best = m[1]
		}
	}
	return best
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
