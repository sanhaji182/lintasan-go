package provider

import (
	"testing"
)

// F2.3 Option A — shadow evidence aggregator tests. Verify the 6 evidence
// categories accumulate correctly and that the false-positive audit honors the
// fail-open invariant (default-tier candidates never audited as exclusions).

func TestShadowAggregator_SixCategories(t *testing.T) {
	a := NewShadowAggregator()

	// Request 1: vision required. gpt-4o (model, has vision) satisfies; a groq
	// host (provider, no vision) is a data-backed would-exclude; a mystery host
	// (default tier) must NOT be excluded (fail-open).
	r1 := ShadowEvaluateIdentity(
		RequestSignals{HasVision: true, Stream: true},
		[]CandidateIdentity{
			{Format: "openai", Model: "gpt-4o", BaseURL: "https://api.openai.com"},
			{Format: "openai", Model: "unlisted", BaseURL: "https://api.groq.com"},
			{Format: "openai", Model: "unknown-x", BaseURL: "https://mystery.example.com"},
		})
	a.Record(r1)

	// Request 2: stream only, all satisfiable.
	r2 := ShadowEvaluateIdentity(
		RequestSignals{Stream: true},
		[]CandidateIdentity{
			{Format: "openai", Model: "gpt-4o-mini", BaseURL: "https://api.openai.com"},
		})
	a.Record(r2)

	s := a.Snapshot()

	// (overall counts)
	if s.Requests != 2 {
		t.Fatalf("requests=%d want 2", s.Requests)
	}
	if s.CandidateDecisions != 4 {
		t.Fatalf("candidate_decisions=%d want 4", s.CandidateDecisions)
	}

	// 1. Tier distribution: r1 = model(gpt-4o) + provider(groq) + default(mystery);
	//    r2 = model(gpt-4o-mini). So model=2, provider=1, default=1.
	if s.TierDistribution["model"] != 2 || s.TierDistribution["provider"] != 1 || s.TierDistribution["default"] != 1 {
		t.Fatalf("tier_distribution wrong: %+v", s.TierDistribution)
	}

	// 5. Confidence mirrors tiers: high=2, medium=1, low=1.
	if s.ConfidenceDistribution["high"] != 2 || s.ConfidenceDistribution["medium"] != 1 || s.ConfidenceDistribution["low"] != 1 {
		t.Fatalf("resolver_confidence wrong: %+v", s.ConfidenceDistribution)
	}

	// 4. Unknown capability resolutions = default-tier count = 1.
	if s.UnknownCapabilityResolutions != 1 {
		t.Fatalf("unknown_capability_resolutions=%d want 1", s.UnknownCapabilityResolutions)
	}

	// 3. Would-exclude: only groq (data-backed, no vision) in r1. mystery is
	//    fail-open (default) and must NOT count.
	if s.WouldExcludeRequests != 1 {
		t.Fatalf("would_exclude_requests=%d want 1", s.WouldExcludeRequests)
	}
	if s.WouldExcludeEvents != 1 {
		t.Fatalf("would_exclude_events=%d want 1 (groq only, mystery fail-open)", s.WouldExcludeEvents)
	}

	// 6. False-positive audit: exactly one entry (groq), NOT the default-tier one.
	if len(s.FalsePositiveAudit) != 1 {
		t.Fatalf("false_positive_candidates=%d want 1: %+v", len(s.FalsePositiveAudit), s.FalsePositiveAudit)
	}
	if s.FalsePositiveAudit[0].Label != "provider:groq" {
		t.Errorf("audit label = %q want provider:groq", s.FalsePositiveAudit[0].Label)
	}
	if s.FalsePositiveAudit[0].Tier == "default" {
		t.Error("FAIL-OPEN VIOLATION: default-tier candidate in false-positive audit")
	}

	// 2. Capability coverage: vision required once (r1), satisfied by 1 of 3
	//    candidates (gpt-4o only). streaming required twice (r1+r2).
	vis, ok := s.CapabilityCoverage["vision"]
	if !ok || vis.Required != 1 {
		t.Fatalf("vision coverage wrong: %+v", vis)
	}
	if vis.Satisfied != 1 {
		t.Errorf("vision satisfied=%d want 1 (gpt-4o only)", vis.Satisfied)
	}
}

// TestShadowAggregator_NilSafe verifies a nil aggregator is a safe no-op (so the
// server never panics if it forgot to construct one).
func TestShadowAggregator_NilSafe(t *testing.T) {
	var a *ShadowAggregator
	a.Record(ShadowEvaluateIdentity(RequestSignals{Stream: true}, []CandidateIdentity{{Model: "gpt-4o"}}))
	s := a.Snapshot()
	if s.Requests != 0 {
		t.Fatalf("nil aggregator should yield empty snapshot, got %+v", s)
	}
}

// TestShadowAggregator_DedupAudit verifies recurring exclusion tuples are folded
// (count increments) rather than duplicated — keeps the audit bounded.
func TestShadowAggregator_DedupAudit(t *testing.T) {
	a := NewShadowAggregator()
	mk := func() ShadowResult {
		return ShadowEvaluateIdentity(
			RequestSignals{HasVision: true},
			[]CandidateIdentity{{Format: "openai", Model: "unlisted", BaseURL: "https://api.groq.com"}})
	}
	a.Record(mk())
	a.Record(mk())
	a.Record(mk())
	s := a.Snapshot()
	if len(s.FalsePositiveAudit) != 1 {
		t.Fatalf("expected 1 deduped audit entry, got %d", len(s.FalsePositiveAudit))
	}
	if s.FalsePositiveAudit[0].Count != 3 {
		t.Errorf("audit count=%d want 3 (deduped)", s.FalsePositiveAudit[0].Count)
	}
}
