package provider

import "sort"

// Capability shadow evaluation (F2.3 — F2 Design Baseline, shadow/observe-only).
//
// SCOPE LOCK (F2.3): this is the FIRST capability-AWARE routing surface, but it
// is OBSERVE-ONLY. It computes, for a set of candidate provider identities,
// whether each would satisfy a request's required capabilities — and returns
// that as data. It NEVER excludes, reorders, or selects anything. Enforcement
// (dropping non-satisfying providers from the pool) is F2.4, gated separately.
//
// DESIGN: the whole evaluation (RequiredCaps derivation + Satisfies) lives HERE,
// inside the provider package. The server passes only primitive request signals
// (RequestSignals) and candidate identities, and receives a ShadowResult. This
// is deliberate: it keeps `Satisfies`, `CapabilitiesFor`, and the capability
// vocabulary constants OUT of the server package, so the F2.0/F2.1/F2.2
// non-consumption guards stay green UNCHANGED. The server's only new capability
// symbol is the ShadowEvaluate facade — exactly the F2.2 facade discipline.
//
// COARSENESS NOTE: capabilities are looked up by provider identity (Format) via
// the F2.1 table. Many live connections share Format="openai" (deepseek, groq,
// etc.), so this lookup is COARSE. That is acceptable for SHADOW because it
// gates nothing — it is observability. F2.4 enforcement will need a precise
// per-connection identity before it can safely exclude on this data.

// RequestSignals are the raw, capability-vocabulary-free facts the server
// extracts from an inbound chat request. Keeping these primitive means the
// server never needs to import capability constants — the mapping to canonical
// capabilities happens here.
type RequestSignals struct {
	Stream      bool // request asked for SSE streaming
	HasTools    bool // request carried tools/functions
	HasVision   bool // request carried image/multimodal content
	HasJSONMode bool // request asked for json_object/json_schema response_format
}

// requiredCaps maps the primitive request signals to the canonical capability
// set the request requires. Pure; the single place signals become capabilities.
func (s RequestSignals) requiredCaps() CapabilitySet {
	req := CapabilitySet{}
	if s.Stream {
		req[CapStreaming] = true
	}
	if s.HasTools {
		req[CapToolCalling] = true
	}
	if s.HasVision {
		req[CapVision] = true
	}
	if s.HasJSONMode {
		req[CapJSONMode] = true
	}
	return req
}

// ShadowDecision is one candidate provider's shadow eligibility result.
type ShadowDecision struct {
	// Provider is the candidate provider identity (Format) evaluated.
	Provider string `json:"provider"`
	// Declared are the caps the F2.1 lookup declares for this identity.
	Declared []Capability `json:"declared"`
	// Satisfies reports whether Declared covers every RequiredCap. In SHADOW
	// mode this is RECORDED ONLY — a false here does NOT exclude the candidate.
	Satisfies bool `json:"satisfies"`
	// Tier records HOW the caps were resolved (model / provider / default).
	// Only data-backed tiers (model, provider) may drive would-exclude; a
	// default-tier decision is FAIL-OPEN and never contributes to exclusion.
	// Empty for the legacy coarse path (ShadowEvaluate).
	Tier IdentityTier `json:"tier,omitempty"`
	// Label is a human-readable identity label for logs (e.g. "model:gpt-4o").
	Label string `json:"label,omitempty"`
	// FailOpen is true when this candidate was NOT data-backed (default tier):
	// it is kept regardless of the cap math, recording that the decision was
	// made under missing data per the enforcement invariant.
	FailOpen bool `json:"fail_open,omitempty"`
}

// ShadowResult is the full observe-only evaluation for one request against its
// candidate pool. It is pure data; the caller logs/metrics it and otherwise
// ignores it (selection is unchanged).
type ShadowResult struct {
	// Required is the canonical capability set the request needs.
	Required []Capability `json:"required"`
	// Decisions is one row per candidate, in input order.
	Decisions []ShadowDecision `json:"decisions"`
	// WouldExclude lists candidate identities that do NOT satisfy Required.
	// This is the load-bearing observability signal: under F2.4 enforcement
	// these WOULD be dropped. In F2.3 they are kept; this set must be observed
	// (ideally empty or expected) under real traffic before F2.4 is greenlit.
	//
	// FAIL-OPEN INVARIANT: a candidate appears here ONLY if its caps were
	// resolved from real data (Tier model/provider) AND positively fail to
	// satisfy Required. Candidates resolved at the default tier (missing data)
	// are NEVER listed — absence of data is not disqualification.
	WouldExclude []string `json:"would_exclude"`
	// TierCounts tallies how candidates were resolved, for re-bake evidence:
	// how often the precise per-model path fired vs the provider derivation vs
	// the conservative default. Empty for the legacy coarse path.
	TierCounts map[IdentityTier]int `json:"tier_counts,omitempty"`
}

// ShadowEvaluate performs the observe-only capability evaluation. For each
// candidate provider identity it looks up the declared caps (F2.1 table) and
// records whether they satisfy the request's required caps. It NEVER mutates,
// reorders, filters, or selects — it only returns what it observed.
//
// This is the ONLY capability symbol the server references for F2.3, and it is
// read-only by construction: it takes value inputs and returns a fresh value.
func ShadowEvaluate(signals RequestSignals, providerIDs []string) ShadowResult {
	required := signals.requiredCaps()
	res := ShadowResult{
		Required:     sortedCaps(required),
		Decisions:    make([]ShadowDecision, 0, len(providerIDs)),
		WouldExclude: []string{},
	}
	for _, id := range providerIDs {
		declared, _ := CapabilitiesFor(id) // fallback-safe; never errors
		ok := declared.Satisfies(required)
		res.Decisions = append(res.Decisions, ShadowDecision{
			Provider:  id,
			Declared:  sortedCaps(declared),
			Satisfies: ok,
		})
		if !ok {
			res.WouldExclude = append(res.WouldExclude, id)
		}
	}
	sort.Strings(res.WouldExclude)
	return res
}

// ShadowEvaluateIdentity is the F2.3 RE-BAKE evaluator: it resolves each
// candidate's capabilities through the APPROVED 3-tier identity resolver
// (per-model catalog → canonical provider-id → conservative default) instead of
// keying on the coarse wire Format. It remains STRICTLY observe-only: it never
// mutates, reorders, filters, or selects. It returns richer observability — the
// tier each candidate resolved at, plus a fail-open-correct WouldExclude.
//
// FAIL-OPEN INVARIANT (Sans, 2026-05-31): a candidate is added to WouldExclude
// ONLY when its caps were resolved from real data (Tier model or provider) AND
// they positively fail to satisfy the request. A candidate that fell through to
// the conservative default (missing model entry, underivable provider) is marked
// FailOpen and is NEVER excluded — absence of data must not eliminate a provider.
//
// This is the SAME resolver F2.4 enforcement will use; baking shadow on it first
// is the precondition that lets the F2.3 evidence predict F2.4 reality.
func ShadowEvaluateIdentity(signals RequestSignals, identities []CandidateIdentity) ShadowResult {
	required := signals.requiredCaps()
	res := ShadowResult{
		Required:     sortedCaps(required),
		Decisions:    make([]ShadowDecision, 0, len(identities)),
		WouldExclude: []string{},
		TierCounts:   map[IdentityTier]int{},
	}
	for _, ident := range identities {
		resolved := resolveIdentityCaps(ident)
		ok := resolved.Caps.Satisfies(required)
		res.TierCounts[resolved.Tier]++

		// FAIL-OPEN: only data-backed tiers may drive exclusion. A default-tier
		// candidate is kept no matter what the cap math says.
		failOpen := !resolved.Tier.DataBacked()

		res.Decisions = append(res.Decisions, ShadowDecision{
			Provider:  ident.Format,
			Declared:  sortedCaps(resolved.Caps),
			Satisfies: ok,
			Tier:      resolved.Tier,
			Label:     resolved.Label,
			FailOpen:  failOpen,
		})

		if !ok && !failOpen {
			res.WouldExclude = append(res.WouldExclude, resolved.Label)
		}
	}
	sort.Strings(res.WouldExclude)
	return res
}
