package provider

import (
	"net/url"
	"strings"

	"github.com/sanhaji182/lintasan-go/internal/models"
)

// capability_identity.go — F2.3 re-bake: the canonical 3-tier capability
// IDENTITY RESOLVER (APPROVED identity model, 2026-05-31).
//
// This replaces the coarse "key capabilities by conn.Format" approach that the
// first F2.3 shadow cut used. Format is a WIRE PROTOCOL, not an identity:
// DeepSeek, Groq, Together, OpenRouter and local servers all carry
// Format="openai", so keying caps on Format collapses them into one bucket and
// fabricates capabilities a provider may not have. The resolver below restores
// precision while staying STRICTLY observe-only (no selection, no exclusion).
//
// Resolution order (most precise → safest):
//
//	Tier F  PER-MODEL CATALOG    models.FindModel(model).Capabilities → canonical
//	                             caps. Capability is fundamentally a property of
//	                             the (provider × model) pair; the model is the
//	                             finest identity and is already in hand at the hook.
//	Tier E  CANONICAL PROVIDER   derive a canonical provider-id from owned_by /
//	                             base_url host (NOT from Format) → F2.1 declared
//	                             caps. Restores deepseek≠groq≠openai.
//	Default CONSERVATIVE         streaming+tool_calling baseline, the same safety
//	                             net F2.1 pins. Reached ONLY when data is missing.
//
// FAIL-OPEN INVARIANT (Sans, 2026-05-31): a provider may be flagged would-exclude
// ONLY if it is POSITIVELY proven to lack a required capability. Missing data,
// missing mapping, missing catalog entry, or resolver fallthrough MUST NOT flag
// exclusion. That is why the resolver returns the TIER alongside the caps: a
// Default-tier resolution means "data was missing", and the shadow evaluator
// treats it as fail-open (never excluded) regardless of the cap math.

// IdentityTier records HOW a candidate's capabilities were resolved. It is the
// load-bearing signal for the fail-open invariant: only data-backed tiers
// (Model, Provider) are eligible to contribute to would-exclude.
type IdentityTier string

const (
	// TierModel: caps came from the per-model catalog entry (most precise,
	// data-backed). The model exists in the catalog and we trust its tags.
	TierModel IdentityTier = "model"
	// TierProvider: caps came from a canonical provider-id derived from
	// owned_by / base_url host, then the F2.1 declared table (data-backed).
	TierProvider IdentityTier = "provider"
	// TierDefault: NO confident data was found; the conservative baseline was
	// used. This tier is FAIL-OPEN — a candidate resolved here is NEVER
	// flagged would-exclude, because absence of data is not disqualification.
	TierDefault IdentityTier = "default"
)

// DataBacked reports whether the tier reflects real capability data (and is thus
// eligible to drive a would-exclude observation). TierDefault is never data-backed.
func (t IdentityTier) DataBacked() bool { return t == TierModel || t == TierProvider }

// CandidateIdentity is the primitive, capability-vocabulary-free identity info
// the server extracts for one candidate connection. The server never imports
// capability constants; it only fills these plain strings and lets the resolver
// map them to capabilities inside the provider package (facade discipline).
type CandidateIdentity struct {
	// Format is the wire protocol (conn.Format). Retained ONLY as a label and a
	// last-resort hint — it is deliberately NOT used as the capability key.
	Format string
	// Model is the requested model id (req["model"]). Drives Tier F.
	Model string
	// OwnedBy is the catalog owned_by for the model/connection when known
	// (e.g. "openai", "deepseek", "google"). Primary Tier E signal. May be "".
	OwnedBy string
	// BaseURL is the connection base_url. Secondary Tier E signal (host → id).
	BaseURL string
}

// providerIDFromOwnedBy maps a model's owned_by string to a canonical
// provider-id (an F2.1 officialCapabilities key). Returns false when it cannot
// confidently classify — the caller must then try the next signal, never guess.
func providerIDFromOwnedBy(ownedBy string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(ownedBy))
	if s == "" {
		return "", false
	}
	switch {
	case strings.Contains(s, "openai"):
		return "openai", true
	case strings.Contains(s, "anthropic"):
		return "anthropic", true
	case strings.Contains(s, "deepseek"):
		return "deepseek", true
	case strings.Contains(s, "groq"):
		return "groq", true
	case strings.Contains(s, "google"), strings.Contains(s, "gemini"):
		return "gemini", true
	}
	return "", false
}

// providerIDFromHost maps a connection base_url host to a canonical provider-id.
// Returns false for unknown hosts — the caller falls through to the conservative
// default (fail-open), it does NOT guess a provider from an unrecognized host.
func providerIDFromHost(baseURL string) (string, bool) {
	raw := strings.TrimSpace(baseURL)
	if raw == "" {
		return "", false
	}
	// url.Parse needs a scheme to populate Host; tolerate bare hosts.
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", false
	}
	switch {
	case strings.Contains(host, "api.openai.com"):
		return "openai", true
	case strings.Contains(host, "anthropic.com"):
		return "anthropic", true
	case strings.Contains(host, "deepseek.com"):
		return "deepseek", true
	case strings.Contains(host, "groq.com"):
		return "groq", true
	case strings.Contains(host, "generativelanguage.googleapis.com"),
		strings.Contains(host, "googleapis.com"):
		return "gemini", true
	}
	return "", false
}

// canonicalProviderID derives a canonical provider identity from primitive
// connection signals WITHOUT using conn.Format. owned_by is preferred (it is the
// model's declared owner); base_url host is the fallback. Returns false when no
// confident derivation exists — the resolver then uses the conservative default.
func canonicalProviderID(id CandidateIdentity) (string, bool) {
	if pid, ok := providerIDFromOwnedBy(id.OwnedBy); ok {
		return pid, true
	}
	if pid, ok := providerIDFromHost(id.BaseURL); ok {
		return pid, true
	}
	return "", false
}

// ResolvedCaps is the resolver output: the capability set plus the tier that
// produced it and a human-readable identity label for observability logs.
type ResolvedCaps struct {
	Caps  CapabilitySet
	Tier  IdentityTier
	Label string // e.g. "model:gpt-4o", "provider:groq", "default:openai"
}

// resolveIdentityCaps implements the APPROVED 3-tier resolver: per-model catalog
// (F) → canonical provider-id (E) → conservative default. It is pure and
// read-only: it consults the static catalog and the F2.1 declared table and
// returns a fresh set. It makes NO selection or eligibility decision — it only
// answers "what does the system know this candidate can do, and how sure is it?".
func resolveIdentityCaps(id CandidateIdentity) ResolvedCaps {
	// Tier F — per-model catalog identity (most precise, data-backed).
	if m := strings.TrimSpace(id.Model); m != "" {
		if mi := models.FindModel(m); mi != nil {
			return ResolvedCaps{
				Caps:  CatalogTagsToSet(mi.Capabilities),
				Tier:  TierModel,
				Label: "model:" + m,
			}
		}
	}

	// Tier E — canonical provider-id (data-backed when derivable + in F2.1 table).
	if pid, ok := canonicalProviderID(id); ok {
		if caps, found := CapabilitiesFor(pid); found {
			return ResolvedCaps{
				Caps:  caps,
				Tier:  TierProvider,
				Label: "provider:" + pid,
			}
		}
	}

	// Default — DATA MISSING. Conservative baseline, FAIL-OPEN (never excluded).
	label := "default"
	if f := strings.TrimSpace(id.Format); f != "" {
		label = "default:" + f
	}
	return ResolvedCaps{
		Caps:  defaultDeclaredCaps(),
		Tier:  TierDefault,
		Label: label,
	}
}
