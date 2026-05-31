package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestRebakeEvidence is NOT a pass/fail unit test — it is a deterministic
// re-bake harness. It drives the 3-tier resolver + ShadowEvaluateIdentity over a
// representative matrix of (request × candidate-pool) scenarios that mirror real
// Lintasan traffic shapes, and emits a JSON evidence report to stderr + a file.
// Run with:  go test ./internal/provider/ -run TestRebakeEvidence -v
//
// It exists to answer the F2.4 precondition questions WITHOUT touching prod:
//  1. Does the resolver pick the most precise tier when data exists?
//  2. Is the fail-open invariant actually honored (default tier never excluded)?
//  3. What does would_exclude look like across realistic requests?
func TestRebakeEvidence(t *testing.T) {
	type scenario struct {
		Name       string
		Signals    RequestSignals
		Candidates []CandidateIdentity
	}

	scenarios := []scenario{
		{
			Name:    "text-stream / mixed real providers",
			Signals: RequestSignals{Stream: true},
			Candidates: []CandidateIdentity{
				{Format: "openai", Model: "gpt-4o", BaseURL: "https://api.openai.com"},
				{Format: "openai", Model: "deepseek-chat", OwnedBy: "deepseek", BaseURL: "https://api.deepseek.com"},
				{Format: "openai", Model: "llama-3.3-70b", BaseURL: "https://api.groq.com"},
			},
		},
		{
			Name:    "vision request / groq cannot serve (Format-collapse trap)",
			Signals: RequestSignals{HasVision: true, Stream: true},
			Candidates: []CandidateIdentity{
				{Format: "openai", Model: "gpt-4o", BaseURL: "https://api.openai.com"},      // has vision (model)
				{Format: "openai", Model: "unlisted-groq", BaseURL: "https://api.groq.com"}, // groq, no vision (provider)
				{Format: "anthropic", Model: "claude-3-5-sonnet", BaseURL: "https://api.anthropic.com"},
			},
		},
		{
			Name:    "json_mode request / unknown self-hosted provider (fail-open)",
			Signals: RequestSignals{HasJSONMode: true, Stream: true},
			Candidates: []CandidateIdentity{
				{Format: "openai", Model: "gpt-4o-mini", BaseURL: "https://api.openai.com"},      // model
				{Format: "openai", Model: "my-local-model", BaseURL: "https://llm.internal.lan"}, // DEFAULT → fail-open
			},
		},
		{
			Name:    "tools request / all unresolvable (total fail-open, no exclusions)",
			Signals: RequestSignals{HasTools: true},
			Candidates: []CandidateIdentity{
				{Format: "openai", Model: "mystery-a", BaseURL: "https://a.unknown.test"},
				{Format: "openai", Model: "mystery-b", BaseURL: "https://b.unknown.test"},
			},
		},
	}

	type report struct {
		Scenario     string           `json:"scenario"`
		Required     []Capability     `json:"required"`
		Decisions    []ShadowDecision `json:"decisions"`
		WouldExclude []string         `json:"would_exclude"`
		TierCounts   map[string]int   `json:"tier_counts"`
	}

	var out []report
	failOpenViolations := 0
	for _, sc := range scenarios {
		res := ShadowEvaluateIdentity(sc.Signals, sc.Candidates)
		tc := map[string]int{
			"model":    res.TierCounts[TierModel],
			"provider": res.TierCounts[TierProvider],
			"default":  res.TierCounts[TierDefault],
		}
		// Invariant audit: no default-tier label may appear in would_exclude.
		for _, d := range res.Decisions {
			if d.Tier == TierDefault {
				for _, we := range res.WouldExclude {
					if we == d.Label {
						failOpenViolations++
						t.Errorf("FAIL-OPEN VIOLATION in %q: %q excluded", sc.Name, d.Label)
					}
				}
			}
		}
		out = append(out, report{
			Scenario: sc.Name, Required: res.Required, Decisions: res.Decisions,
			WouldExclude: res.WouldExclude, TierCounts: tc,
		})
	}

	blob, _ := json.MarshalIndent(out, "", "  ")
	_ = os.WriteFile("/tmp/f2.3-rebake/rebake-evidence.json", blob, 0644)
	fmt.Fprintf(os.Stderr, "\n===== F2.3 RE-BAKE EVIDENCE =====\n%s\n", blob)
	fmt.Fprintf(os.Stderr, "fail_open_violations=%d (MUST be 0)\n", failOpenViolations)
}
