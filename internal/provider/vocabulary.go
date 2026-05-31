package provider

// Canonical capability vocabulary (F2.0 — F2 Design Baseline, decision D3).
//
// SCOPE LOCK (F2.0): this file establishes the SINGLE source of truth for the
// capability vocabulary and provides READ-ONLY lookup + PURE mapping helpers.
// It is deliberately inert at runtime:
//   - It is NOT consumed by the proxy/router on any request path.
//   - It does NOT influence provider selection.
//   - It does NOT perform capability filtering.
//   - It introduces NO API, route, or schema change.
//
// Its only job is to turn the F2 design decisions into a stable code structure
// so later, separately-approved phases (F2.1 lookup wiring, F2.2 diagnostic
// registry, F2.4 routing) have a fixed vocabulary to build on. Changing the
// return values here cannot change system behavior because nothing reads them
// on a live path — that property is asserted by the F2.0 guard tests.

// CanonicalVocabulary is the complete, ordered set of capabilities the system
// recognizes. It is the authoritative list (D3): the models catalog tags and
// the combo auto-modes MAP INTO this vocabulary rather than defining their own.
//
// Order is stable (functional capabilities first, then orthogonal tier hints)
// for deterministic diagnostics and dashboard display.
var CanonicalVocabulary = []Capability{
	// Functional capabilities.
	CapCoding,
	CapReasoning,
	CapToolCalling,
	CapStreaming,
	CapVision,
	CapEmbeddings,
	CapLongContext,
	CapJSONMode,
	// Tier hints (orthogonal axis).
	CapCheap,
	CapFast,
	CapPremium,
}

// canonicalSet is the membership index for the vocabulary, built once.
var canonicalSet = func() map[Capability]bool {
	m := make(map[Capability]bool, len(CanonicalVocabulary))
	for _, c := range CanonicalVocabulary {
		m[c] = true
	}
	return m
}()

// IsCanonical reports whether c is a recognized capability in the canonical
// vocabulary. Read-only; safe for concurrent use.
func IsCanonical(c Capability) bool { return canonicalSet[c] }

// Vocabulary returns a copy of the canonical vocabulary in stable order. A copy
// is returned so callers (diagnostics, dashboard) cannot mutate the source of
// truth.
func Vocabulary() []Capability {
	out := make([]Capability, len(CanonicalVocabulary))
	copy(out, CanonicalVocabulary)
	return out
}

// --- Pure mapping helpers (declaration-only, no runtime caller in F2.0) -------
//
// These translate the two OTHER vocabularies that exist in the codebase into
// the canonical one, per D3. They are pure functions: deterministic, no I/O, no
// global mutation. F2.1 will be free to wire them into a lookup table; F2.0
// only fixes the mapping itself.

// CatalogTagToCapability maps a single models-catalog tag string to its
// canonical capability. The bool is false for tags that have no functional
// capability meaning ("chat" is baseline) or that are unknown — callers should
// skip those rather than inventing a capability. Pure; no runtime caller.
func CatalogTagToCapability(tag string) (Capability, bool) {
	switch tag {
	case "streaming":
		return CapStreaming, true
	case "tools":
		return CapToolCalling, true
	case "vision":
		return CapVision, true
	case "json_mode":
		return CapJSONMode, true
	case "chat":
		// Baseline: every model can chat. Not a distinguishing capability.
		return "", false
	default:
		return "", false
	}
}

// CatalogTagsToSet maps a slice of models-catalog tags (a ModelInfo.Capabilities
// value) to a canonical CapabilitySet, dropping baseline/unknown tags. Pure; not
// called from any runtime path in F2.0.
func CatalogTagsToSet(tags []string) CapabilitySet {
	s := make(CapabilitySet, len(tags))
	for _, t := range tags {
		if cap, ok := CatalogTagToCapability(t); ok {
			s[cap] = true
		}
	}
	return s
}

// AutoModeToCapability maps a combo auto-routing mode string (combo/auto.go,
// e.g. "auto/coding", "auto/fast", "auto/cheap") to the canonical tier/functional
// capability it expresses. The bare "auto" (balanced) mode expresses no specific
// capability and returns false. Pure; no runtime caller in F2.0.
func AutoModeToCapability(mode string) (Capability, bool) {
	switch mode {
	case "auto/coding":
		return CapCoding, true
	case "auto/fast":
		return CapFast, true
	case "auto/cheap":
		return CapCheap, true
	case "auto":
		return "", false
	default:
		return "", false
	}
}
