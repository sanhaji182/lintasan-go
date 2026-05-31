package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sanhaji182/lintasan-go/internal/provider"
)

// capability_shadow.go — F2.3 server-side shadow routing hook (observe-only).
//
// This is the server's ONLY F2.3 surface. It extracts primitive, capability-
// vocabulary-free signals from the request and delegates the entire capability
// evaluation to provider.ShadowEvaluate (the facade). It deliberately holds NO
// capability constants and performs no eligibility check itself — the lookup
// and matching all happen inside the provider package — so the F2.0/F2.1/F2.2
// server non-consumption guards stay green unchanged.
//
// CONTRACT: this function MUST NOT mutate, reorder, filter, or select from
// `candidates`. It reads them (by Format) and records an observation. Selection
// happens entirely after this call and is identical whether the flag is on/off.

// extractRequestSignals reads the primitive capability signals from a parsed
// chat request body. Pure; no capability vocabulary leaks into the server.
func extractRequestSignals(req map[string]any, stream bool) provider.RequestSignals {
	s := provider.RequestSignals{Stream: stream}

	if _, ok := req["tools"]; ok {
		s.HasTools = true
	}
	if _, ok := req["functions"]; ok {
		s.HasTools = true
	}

	// response_format: {"type":"json_object"} or {"type":"json_schema",...}
	if rf, ok := req["response_format"].(map[string]any); ok {
		if t, _ := rf["type"].(string); t == "json_object" || t == "json_schema" {
			s.HasJSONMode = true
		}
	}

	// Vision: any message whose content is a multipart array containing an
	// image / image_url part. Text-only requests (content is a string) are not
	// vision. Conservative: only flags on an explicit image part.
	if msgs, ok := req["messages"].([]any); ok {
		for _, m := range msgs {
			mm, ok := m.(map[string]any)
			if !ok {
				continue
			}
			parts, ok := mm["content"].([]any)
			if !ok {
				continue
			}
			for _, p := range parts {
				pp, ok := p.(map[string]any)
				if !ok {
					continue
				}
				switch t, _ := pp["type"].(string); t {
				case "image", "image_url", "input_image":
					s.HasVision = true
				}
			}
		}
	}
	return s
}

// runCapabilityShadow performs the F2.3 observe-only capability evaluation. It
// is a complete no-op when the flag is off (a single bool check, zero added
// latency). When on, it resolves each candidate's capabilities through the
// APPROVED 3-tier identity resolver (per-model catalog → canonical provider-id →
// conservative default) and RECORDS the result (response header + stderr log)
// WITHOUT changing `candidates` in any way.
//
// FAIL-OPEN: would_exclude only ever lists candidates whose caps were resolved
// from real data AND positively fail to satisfy the request. Candidates that
// fell through to the conservative default (missing data) are kept, recorded
// with fail_open=true, and NEVER listed — per the enforcement invariant.
func (p *ProxyHandler) runCapabilityShadow(w http.ResponseWriter, req map[string]any, resolvedModel string, stream bool, candidates []*Connection) {
	if !p.capabilityShadow {
		return // default OFF: zero behavior change, zero added work
	}
	if len(candidates) == 0 {
		return
	}

	signals := extractRequestSignals(req, stream)

	identities := make([]provider.CandidateIdentity, 0, len(candidates))
	for _, c := range candidates {
		identities = append(identities, provider.CandidateIdentity{
			Format:  c.Format,
			Model:   resolvedModel,
			BaseURL: c.BaseURL,
			// OwnedBy is left empty here: deriving it per-candidate would need a
			// catalog/DB lookup on the hot path. The resolver degrades to host
			// derivation (Tier E) and then the conservative default — both
			// fail-open. Per-model (Tier F) still fires from resolvedModel.
		})
	}

	result := provider.ShadowEvaluateIdentity(signals, identities)

	// Option A: fold this request into the evidence aggregator (always, not just
	// on would_exclude) so the 6-category report covers ALL baked traffic.
	p.shadowStats.Record(result)

	// Structured per-request evidence line (always emitted under the flag) so the
	// bake is reconstructable from logs alone. Single line, counts only, no PII.
	fmt.Fprintf(os.Stderr,
		"[capability-shadow] model=%q required=%v candidates=%d would_exclude=%d tiers=m%d/p%d/d%d wexc=%v\n",
		resolvedModel, result.Required, len(result.Decisions), len(result.WouldExclude),
		result.TierCounts[provider.TierModel],
		result.TierCounts[provider.TierProvider],
		result.TierCounts[provider.TierDefault],
		result.WouldExclude)

	// Observability ONLY. The selection logic below this hook is unaffected.
	w.Header().Set("X-Lintasan-Capability-Shadow",
		fmt.Sprintf("required=%d candidates=%d would_exclude=%d tiers=m%d/p%d/d%d",
			len(result.Required), len(result.Decisions), len(result.WouldExclude),
			result.TierCounts[provider.TierModel],
			result.TierCounts[provider.TierProvider],
			result.TierCounts[provider.TierDefault]))
}
