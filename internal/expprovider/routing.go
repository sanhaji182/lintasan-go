package expprovider

// G3 — Experimental routing entry (the explicit opt-in signal).
//
// INVARIANT 1 corollary: an Experimental provider is reachable ONLY by an
// explicit, opt-in signal — never by default/auto/smart routing. This file is
// the single parser that detects that signal and extracts the target provider
// name. It does NOT route, resolve, or execute; it only answers "did the caller
// explicitly ask for an Experimental provider, and which one?"
//
// Three accepted signals (any one suffices), in priority order:
//  1. Model prefix  "experimental/<name>[/<model>]"   (e.g. experimental/codex)
//  2. Header        "X-Lintasan-Track: experimental" + "X-Lintasan-Provider: <name>"
//  3. (reserved)    an explicit per-connection flag — represented by the caller
//     passing an explicit name to DetectExperimentalName; kept as a separate
//     entry point so a connection-flag path never shares code with the default
//     model-routing path.
//
// The default production path calls NONE of this — so absence of an explicit
// signal means the request can never land on an Experimental provider.

import (
	"net/http"
	"strings"
)

// TrackHeader is the header that opts a request into the Experimental track.
const TrackHeader = "X-Lintasan-Track"

// ProviderHeader names the Experimental provider when using the header signal.
const ProviderHeader = "X-Lintasan-Provider"

// ExperimentalModelPrefix is the model-name prefix that opts in.
const ExperimentalModelPrefix = "experimental/"

// trackValueExperimental is the accepted value of TrackHeader.
const trackValueExperimental = "experimental"

// ExperimentalSignal is the parsed result of an opt-in detection.
type ExperimentalSignal struct {
	// Provider is the Experimental provider name requested (e.g. "codex").
	Provider string
	// Model is the residual model name after the provider prefix, if any
	// (e.g. "experimental/codex/gpt-5" -> Model "gpt-5"). Empty if not present.
	Model string
	// Via records which signal matched, for diagnostics/logging.
	Via string
}

// DetectExperimental inspects a model name and request headers for an explicit
// Experimental opt-in. It returns (signal, true) ONLY when an explicit signal
// is present. With no signal it returns (zero, false) — the caller MUST then
// stay on the production (Official-only) path.
//
// It performs NO registry lookup and NO membrane decision; it is pure signal
// parsing. The caller pairs the returned Provider name with
// provider.ResolveExperimental (the explicit door) to actually obtain the
// provider — keeping detection and resolution as two deliberate steps.
func DetectExperimental(model string, headers http.Header) (ExperimentalSignal, bool) {
	// Signal 1: model prefix.
	if strings.HasPrefix(model, ExperimentalModelPrefix) {
		rest := strings.TrimPrefix(model, ExperimentalModelPrefix)
		name, sub, _ := strings.Cut(rest, "/")
		name = strings.TrimSpace(name)
		if name != "" {
			return ExperimentalSignal{Provider: name, Model: strings.TrimSpace(sub), Via: "model_prefix"}, true
		}
	}

	// Signal 2: header pair.
	if headers != nil {
		if strings.EqualFold(strings.TrimSpace(headers.Get(TrackHeader)), trackValueExperimental) {
			name := strings.TrimSpace(headers.Get(ProviderHeader))
			if name != "" {
				// A model may still be supplied verbatim in the body; we don't
				// parse it here. Model stays empty for the header signal.
				return ExperimentalSignal{Provider: name, Via: "header"}, true
			}
		}
	}

	return ExperimentalSignal{}, false
}

// IsExperimentalModel is a cheap predicate for the model-prefix signal alone,
// useful where headers are unavailable. It does not validate the provider name
// beyond non-emptiness.
func IsExperimentalModel(model string) bool {
	if !strings.HasPrefix(model, ExperimentalModelPrefix) {
		return false
	}
	name, _, _ := strings.Cut(strings.TrimPrefix(model, ExperimentalModelPrefix), "/")
	return strings.TrimSpace(name) != ""
}
