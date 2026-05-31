package provider

import (
	"sort"
	"sync"
)

// shadow_stats.go — F2.3 re-bake OBSERVABILITY (Option A, approved 2026-05-31).
//
// PURPOSE: the resolver already COMPUTES every evidence signal in ShadowResult,
// but a single response header + a would_exclude-only stderr line cannot deliver
// the full 6-category report Sans requires across real traffic. This file adds a
// process-level, thread-safe AGGREGATOR that accumulates one ShadowResult per
// chat request into a serializable snapshot mapping 1:1 to the 6 categories:
//
//	1. Tier distribution        → TierDistribution
//	2. Capability coverage       → CapabilityCoverage (required vs satisfied per cap)
//	3. Would-exclude counts      → WouldExcludeEvents + WouldExcludeRequests
//	4. Unknown capability counts → DefaultTierResolutions (data-missing = unknown)
//	5. Resolver confidence       → ConfidenceDistribution (high/medium/low by tier)
//	6. False-positive candidates → ExclusionAudit (data-backed exclusions to inspect)
//
// STRICTLY OBSERVE-ONLY: this aggregator is written to AFTER the routing decision
// is already made and is NEVER read by the selection path. It holds counts only —
// no request bodies, no secrets, no PII. The server consumes it as an opaque
// handle (Record / Snapshot), so the F2.0/F2.1/F2.2 non-consumption guards stay
// green — no vocabulary constant or Satisfies call leaks into the server.

// confidence levels derived from the resolution tier. This is the "resolver
// confidence" evidence: model > provider > default(fail-open).
const (
	confidenceHigh   = "high"   // TierModel — per-model catalog, most precise
	confidenceMedium = "medium" // TierProvider — canonical provider-id derivation
	confidenceLow    = "low"    // TierDefault — data missing, fail-open
)

// tierConfidence maps a resolution tier to its confidence label.
func tierConfidence(t IdentityTier) string {
	switch t {
	case TierModel:
		return confidenceHigh
	case TierProvider:
		return confidenceMedium
	default:
		return confidenceLow
	}
}

// capCoverage is one capability's coverage tally across baked traffic.
type capCoverage struct {
	Required  int `json:"required"`  // times this cap was required by a request
	Satisfied int `json:"satisfied"` // times a candidate satisfied it (per decision)
	Evaluated int `json:"evaluated"` // total candidate-decisions touching this cap
}

// exclusionAuditEntry records ONE data-backed would-exclude decision for human
// false-positive inspection. It carries no request content — only the resolved
// identity and the capability math, which is exactly what an auditor needs to
// decide "is this a genuine miss or a false-positive the resolver got wrong?".
type exclusionAuditEntry struct {
	Model    string   `json:"model"`    // resolved model the request targeted
	Label    string   `json:"label"`    // resolver identity label (e.g. provider:groq)
	Tier     string   `json:"tier"`     // model | provider (data-backed only)
	Required []string `json:"required"` // caps the request needed
	Declared []string `json:"declared"` // caps the resolver attributed to the candidate
	Count    int      `json:"count"`    // how many times this exact tuple recurred
}

// ShadowAggregator accumulates ShadowResults into the 6-category evidence
// snapshot. Safe for concurrent use (the router calls Record from many
// goroutines). All state is counts/labels — bounded and serializable.
type ShadowAggregator struct {
	mu sync.Mutex

	requests        int                             // total chat requests evaluated under shadow
	requestsWithReq int                             // requests that required >=1 capability
	tierCounts      map[string]int                  // 1. tier distribution (model/provider/default)
	confidence      map[string]int                  // 5. confidence distribution (high/medium/low)
	coverage        map[string]*capCoverage         // 2. per-capability coverage
	wouldExcludeEv  int                             // 3. total would-exclude candidate events
	wouldExcludeReq int                             // 3. requests with >=1 would-exclude
	defaultTier     int                             // 4. data-missing resolutions (unknown caps)
	candidates      int                             // total candidate-decisions evaluated
	audit           map[string]*exclusionAuditEntry // 6. dedup'd exclusion tuples
}

// NewShadowAggregator returns a ready aggregator with initialized maps.
func NewShadowAggregator() *ShadowAggregator {
	return &ShadowAggregator{
		tierCounts: map[string]int{},
		confidence: map[string]int{},
		coverage:   map[string]*capCoverage{},
		audit:      map[string]*exclusionAuditEntry{},
	}
}

// Record folds one ShadowResult into the running totals. It is called once per
// chat request when the shadow flag is ON, AFTER selection is already decided.
// Pure accounting — it changes nothing about the request.
func (a *ShadowAggregator) Record(res ShadowResult) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	a.requests++
	if len(res.Required) > 0 {
		a.requestsWithReq++
	}

	// 2. capability coverage — count each required cap once per request.
	reqSet := map[Capability]bool{}
	for _, c := range res.Required {
		reqSet[c] = true
		cc := a.coverage[string(c)]
		if cc == nil {
			cc = &capCoverage{}
			a.coverage[string(c)] = cc
		}
		cc.Required++
	}

	// 1 + 5: tier + confidence distribution from TierCounts.
	for tier, n := range res.TierCounts {
		a.tierCounts[string(tier)] += n
		a.confidence[tierConfidence(tier)] += n
		if tier == TierDefault {
			a.defaultTier += n // 4. unknown/data-missing
		}
	}

	// per-decision coverage (how often each required cap was satisfied).
	for _, d := range res.Decisions {
		a.candidates++
		declaredSet := map[string]bool{}
		for _, c := range d.Declared {
			declaredSet[string(c)] = true
		}
		for reqCap := range reqSet {
			cc := a.coverage[string(reqCap)]
			if cc == nil {
				cc = &capCoverage{}
				a.coverage[string(reqCap)] = cc
			}
			cc.Evaluated++
			if declaredSet[string(reqCap)] {
				cc.Satisfied++
			}
		}
	}

	// 3 + 6: would-exclude counts + false-positive audit candidates.
	if len(res.WouldExclude) > 0 {
		a.wouldExcludeReq++
		a.wouldExcludeEv += len(res.WouldExclude)
		// Build an audit entry per data-backed excluded decision.
		for _, d := range res.Decisions {
			if d.Satisfies || d.FailOpen {
				continue // only data-backed exclusions are audit-worthy
			}
			key := d.Label + "|" + reqKey(res.Required)
			e := a.audit[key]
			if e == nil {
				e = &exclusionAuditEntry{
					Model:    res.Model,
					Label:    d.Label,
					Tier:     string(d.Tier),
					Required: capStrings(res.Required),
					Declared: capStrings(d.Declared),
				}
				a.audit[key] = e
			}
			e.Count++
		}
	}
}

// ShadowStats is the serializable 6-category evidence snapshot. JSON tags map
// directly to Sans's success-criteria list.
type ShadowStats struct {
	Requests             int `json:"requests"`
	RequestsWithRequired int `json:"requests_with_required_caps"`
	CandidateDecisions   int `json:"candidate_decisions"`
	// 1.
	TierDistribution map[string]int `json:"tier_distribution"`
	// 2.
	CapabilityCoverage map[string]capCoverage `json:"capability_coverage"`
	// 3.
	WouldExcludeEvents   int `json:"would_exclude_events"`
	WouldExcludeRequests int `json:"would_exclude_requests"`
	// 4.
	UnknownCapabilityResolutions int `json:"unknown_capability_resolutions"`
	// 5.
	ConfidenceDistribution map[string]int `json:"resolver_confidence"`
	// 6.
	FalsePositiveAudit []exclusionAuditEntry `json:"false_positive_candidates"`
}

// Snapshot returns a deterministic, copied view of the current totals. Safe to
// serialize for the diagnostic endpoint; never shares internal maps.
func (a *ShadowAggregator) Snapshot() ShadowStats {
	if a == nil {
		return ShadowStats{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	cov := make(map[string]capCoverage, len(a.coverage))
	for k, v := range a.coverage {
		cov[k] = *v
	}
	audit := make([]exclusionAuditEntry, 0, len(a.audit))
	for _, e := range a.audit {
		audit = append(audit, *e)
	}
	sort.Slice(audit, func(i, j int) bool {
		if audit[i].Count != audit[j].Count {
			return audit[i].Count > audit[j].Count
		}
		return audit[i].Label < audit[j].Label
	})

	return ShadowStats{
		Requests:                     a.requests,
		RequestsWithRequired:         a.requestsWithReq,
		CandidateDecisions:           a.candidates,
		TierDistribution:             copyIntMap(a.tierCounts),
		CapabilityCoverage:           cov,
		WouldExcludeEvents:           a.wouldExcludeEv,
		WouldExcludeRequests:         a.wouldExcludeReq,
		UnknownCapabilityResolutions: a.defaultTier,
		ConfidenceDistribution:       copyIntMap(a.confidence),
		FalsePositiveAudit:           audit,
	}
}

// --- small pure helpers -----------------------------------------------------

func copyIntMap(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func capStrings(cs []Capability) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = string(c)
	}
	return out
}

// reqKey builds a stable dedup key from a required-caps slice (already sorted by
// sortedCaps upstream, but sort defensively).
func reqKey(cs []Capability) string {
	ss := capStrings(cs)
	sort.Strings(ss)
	key := ""
	for _, s := range ss {
		key += s + ","
	}
	return key
}
