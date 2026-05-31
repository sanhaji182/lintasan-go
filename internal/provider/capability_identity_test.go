package provider

import (
	"testing"
)

// F2.3 re-bake tests — the APPROVED 3-tier identity resolver and the fail-open
// invariant. These are the load-bearing checks the F2.4 enforcement greenlight
// depends on: prove that (1) the resolver picks the most precise available
// identity, and (2) missing data NEVER drives an exclusion.

// TestResolver_TierF_PerModel: a known catalog model resolves at TierModel with
// caps derived from its catalog tags — not from the coarse wire Format.
func TestResolver_TierF_PerModel(t *testing.T) {
	// gpt-4o is in the catalog with vision+tools+streaming+json_mode tags.
	got := resolveIdentityCaps(CandidateIdentity{Format: "openai", Model: "gpt-4o"})
	if got.Tier != TierModel {
		t.Fatalf("expected TierModel for catalog model gpt-4o, got %q", got.Tier)
	}
	if !got.Caps.Has(CapVision) {
		t.Errorf("gpt-4o should resolve vision from catalog tags; caps=%v", got.Caps.List())
	}
	if got.Label != "model:gpt-4o" {
		t.Errorf("label = %q, want model:gpt-4o", got.Label)
	}
}

// TestResolver_TierE_ProviderFromHost: an unknown model but a recognizable
// base_url host resolves at TierProvider via canonical provider-id derivation —
// proving Format is NOT the key (a groq host stays groq even if Format=openai).
func TestResolver_TierE_ProviderFromHost(t *testing.T) {
	got := resolveIdentityCaps(CandidateIdentity{
		Format:  "openai", // wire protocol — deliberately misleading
		Model:   "some-unlisted-model",
		BaseURL: "https://api.groq.com/openai/v1",
	})
	if got.Tier != TierProvider {
		t.Fatalf("expected TierProvider from groq host, got %q (label=%s)", got.Tier, got.Label)
	}
	if got.Label != "provider:groq" {
		t.Errorf("label = %q, want provider:groq — Format must NOT win over host", got.Label)
	}
	// Groq declares streaming+tool_calling only (no vision) — proving we did NOT
	// inherit OpenAI's rich caps despite Format="openai".
	if got.Caps.Has(CapVision) {
		t.Errorf("groq must NOT have vision; Format-collapse defect present: %v", got.Caps.List())
	}
}

// TestResolver_TierE_ProviderFromOwnedBy: owned_by is preferred over host.
func TestResolver_TierE_ProviderFromOwnedBy(t *testing.T) {
	got := resolveIdentityCaps(CandidateIdentity{
		Format:  "openai",
		Model:   "unlisted",
		OwnedBy: "deepseek",
		BaseURL: "https://api.openai.com", // host says openai, owned_by says deepseek
	})
	if got.Tier != TierProvider || got.Label != "provider:deepseek" {
		t.Fatalf("owned_by should win: got tier=%q label=%q", got.Tier, got.Label)
	}
}

// TestResolver_Default_FailOpen: no model match and no derivable provider must
// fall through to the conservative default tier (NOT data-backed).
func TestResolver_Default_FailOpen(t *testing.T) {
	got := resolveIdentityCaps(CandidateIdentity{
		Format:  "openai",
		Model:   "totally-unknown-model",
		BaseURL: "https://some-random-host.example.com",
	})
	if got.Tier != TierDefault {
		t.Fatalf("expected TierDefault for unresolvable identity, got %q", got.Tier)
	}
	if got.Tier.DataBacked() {
		t.Error("TierDefault must NOT be data-backed (fail-open requires this)")
	}
	// Conservative baseline = streaming + tool_calling.
	if !got.Caps.Has(CapStreaming) || !got.Caps.Has(CapToolCalling) {
		t.Errorf("default caps should be conservative baseline, got %v", got.Caps.List())
	}
}

// TestShadowIdentity_FailOpenInvariant is THE precondition check: a candidate
// whose caps are UNRESOLVABLE (default tier) must NEVER appear in WouldExclude,
// even for a request it cannot be shown to satisfy. Absence of data ≠ exclusion.
func TestShadowIdentity_FailOpenInvariant(t *testing.T) {
	signals := RequestSignals{HasVision: true} // requires vision

	identities := []CandidateIdentity{
		// (a) Data-backed + positively insufficient: groq host, no vision →
		//     MUST be in would_exclude.
		{Format: "openai", Model: "unlisted", BaseURL: "https://api.groq.com"},
		// (b) Unresolvable (default tier) → MUST NOT be in would_exclude even
		//     though we can't prove it satisfies vision.
		{Format: "openai", Model: "unknown-model", BaseURL: "https://mystery.example.com"},
		// (c) Data-backed + sufficient: gpt-4o has vision → not excluded.
		{Format: "openai", Model: "gpt-4o", BaseURL: "https://api.openai.com"},
	}

	res := ShadowEvaluateIdentity(signals, identities)

	// (b) must be fail-open and absent from would_exclude.
	var sawDefault, sawGroqExcluded bool
	for _, d := range res.Decisions {
		if d.Tier == TierDefault {
			sawDefault = true
			if !d.FailOpen {
				t.Errorf("default-tier decision must be FailOpen: %+v", d)
			}
			for _, we := range res.WouldExclude {
				if we == d.Label {
					t.Errorf("FAIL-OPEN VIOLATION: default-tier %q in would_exclude", d.Label)
				}
			}
		}
		if d.Label == "provider:groq" && !d.Satisfies {
			sawGroqExcluded = true
		}
	}
	if !sawDefault {
		t.Fatal("test setup: expected a default-tier candidate")
	}
	if !sawGroqExcluded {
		t.Fatal("test setup: expected groq to positively fail vision")
	}
	// groq (data-backed, insufficient) SHOULD be excluded; mystery (default) NOT.
	if !contains(res.WouldExclude, "provider:groq") {
		t.Errorf("data-backed insufficient provider:groq should be in would_exclude: %v", res.WouldExclude)
	}
	if contains(res.WouldExclude, "default") {
		t.Errorf("default-tier candidate must never be excluded: %v", res.WouldExclude)
	}
}

// TestShadowIdentity_TierCounts verifies the re-bake evidence counters.
func TestShadowIdentity_TierCounts(t *testing.T) {
	signals := RequestSignals{Stream: true}
	identities := []CandidateIdentity{
		{Model: "gpt-4o"}, // model
		{Model: "x", BaseURL: "https://api.anthropic.com"},   // provider
		{Model: "y", BaseURL: "https://nowhere.example.com"}, // default
	}
	res := ShadowEvaluateIdentity(signals, identities)
	if res.TierCounts[TierModel] != 1 || res.TierCounts[TierProvider] != 1 || res.TierCounts[TierDefault] != 1 {
		t.Fatalf("tier counts wrong: %+v", res.TierCounts)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
