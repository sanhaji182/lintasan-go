package expprovider

// G5 — Admission Harness skeleton (Pillar D).
//
// The harness turns admission into a MECHANICAL gate, not a judgment call. A
// candidate either passes every applicable gate or it does not become `active`.
// This is the SKELETON: the gate framework, the result matrix, and the gate
// definitions with their pass/fail contracts. The per-provider probe bodies
// (launch the real CLI, drive a real tool loop) are filled in at onboarding —
// the skeleton defines WHAT must pass and produces the Go/No-Go verdict.
//
// Four gate suites (run in order; any RED blocks admission):
//   1. Isolation conformance  (all tiers) — foreign-secret / egress / process-escape probes
//   2. Protocol conformance   (ACP)       — handshake / session / tool round-trip / terminal honesty
//   3. Acceptance run         (ACP)       — real CLI in staging; the tool loop MUST close (M5 principle)
//   4. Membrane guard         (all, always-on) — no Experimental symbol reachable from Official routing
//
// Acceptance principle (locked, carried from Codex M5): valid ONLY if the tool
// loop completes. Stream-text-only is NOT acceptance. Identifier mismatch in the
// loop = NO-GO.

import (
	"context"
	"fmt"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/provider"
)

// GateName identifies a harness gate.
type GateName string

const (
	GateIsolation  GateName = "isolation_conformance"
	GateProtocol   GateName = "protocol_conformance"
	GateAcceptance GateName = "acceptance_run"
	GateMembrane   GateName = "membrane_guard"
)

// GateOutcome is the result of one gate.
type GateOutcome string

const (
	GatePass GateOutcome = "PASS"
	GateFail GateOutcome = "FAIL"
	GateSkip GateOutcome = "SKIP" // not applicable to this provider's tier
)

// GateResult is one gate's outcome with a human reason and timing.
type GateResult struct {
	Gate    GateName
	Outcome GateOutcome
	Reason  string
	Elapsed time.Duration
}

// Gate is a single admission check. Probe returns (PASS/FAIL/SKIP, reason). A
// Probe MUST NOT panic out to the caller — the harness recovers, but a probe
// should surface failures as GateFail with a reason, not as a Go panic.
type Gate struct {
	Name  GateName
	Probe func(ctx context.Context, c Candidate) (GateOutcome, string)
}

// Candidate is what the harness evaluates: the adapter under test plus the
// metadata the gates need. The adapter is the real ACPProvider; the gates probe
// its behavior (isolation) and drive it (protocol/acceptance).
type Candidate struct {
	Provider string
	Adapter  *ACPProvider
	Spec     LaunchSpec
	// MembraneCheck is the always-on invariant probe: it must confirm the
	// candidate's Track is Experimental and that it is NOT present in the
	// production-routable pool. Supplied by the host (which holds the registry).
	MembraneCheck func() (ok bool, reason string)
}

// Report is the full Go/No-Go matrix for a candidate.
type Report struct {
	Provider string
	Results  []GateResult
	Verdict  GateOutcome // PASS only if every applicable gate is PASS
}

// Go reports whether the candidate may be admitted (every applicable gate PASS;
// no FAIL). A report with only PASS/SKIP results is a GO.
func (r Report) Go() bool { return r.Verdict == GatePass }

// Harness runs an ordered set of gates against a candidate and produces a
// Report. The default harness wires the four standard gates; the per-provider
// probe bodies are injected by the host at onboarding (the skeleton ships the
// membrane gate live and the other three as explicit NOT-IMPLEMENTED fails so a
// candidate can never be admitted before its probes exist).
type Harness struct {
	gates []Gate
}

// NewHarness builds a harness with the standard gate ORDER. By default the
// isolation/protocol/acceptance gates are "not implemented" (they FAIL closed):
// a provider cannot be admitted until onboarding supplies real probes via
// WithGate. The membrane gate is live from the skeleton because it depends only
// on the registry, not on the concrete provider.
func NewHarness() *Harness {
	return &Harness{gates: []Gate{
		{Name: GateIsolation, Probe: notImplemented(GateIsolation)},
		{Name: GateProtocol, Probe: notImplemented(GateProtocol)},
		{Name: GateAcceptance, Probe: notImplemented(GateAcceptance)},
		{Name: GateMembrane, Probe: membraneProbe},
	}}
}

// WithGate replaces the probe for a named gate (onboarding injects real probes).
// Unknown gate names are ignored. Returns the harness for chaining.
func (h *Harness) WithGate(name GateName, probe func(ctx context.Context, c Candidate) (GateOutcome, string)) *Harness {
	for i := range h.gates {
		if h.gates[i].Name == name {
			h.gates[i].Probe = probe
		}
	}
	return h
}

// Run executes every gate in order and returns the Report. It recovers from a
// panicking probe (contains it as a GateFail) so the harness itself can never
// crash the caller — the same containment posture as E1.
func (h *Harness) Run(ctx context.Context, c Candidate) Report {
	rep := Report{Provider: c.Provider, Verdict: GatePass}
	for _, g := range h.gates {
		res := runGate(ctx, g, c)
		rep.Results = append(rep.Results, res)
		if res.Outcome == GateFail {
			rep.Verdict = GateFail
		}
	}
	return rep
}

func runGate(ctx context.Context, g Gate, c Candidate) (res GateResult) {
	start := time.Now()
	res = GateResult{Gate: g.Name, Outcome: GateFail, Reason: "probe did not complete"}
	defer func() {
		if r := recover(); r != nil {
			res.Outcome = GateFail
			res.Reason = fmt.Sprintf("probe panicked (contained): %v", r)
		}
		res.Elapsed = time.Since(start)
	}()
	if g.Probe == nil {
		res.Reason = "no probe configured"
		return res
	}
	out, reason := g.Probe(ctx, c)
	res.Outcome = out
	res.Reason = reason
	return res
}

// notImplemented returns a probe that FAILS closed with a clear reason. Used for
// the gates whose real probes are supplied at onboarding — a candidate can never
// reach `active` while any required probe is still the skeleton stub.
func notImplemented(name GateName) func(ctx context.Context, c Candidate) (GateOutcome, string) {
	return func(ctx context.Context, c Candidate) (GateOutcome, string) {
		return GateFail, string(name) + " probe not implemented (supply via WithGate at onboarding)"
	}
}

// membraneProbe is the always-on invariant-1 gate: the candidate must be
// Experimental-track and absent from the production-routable pool. It runs from
// the skeleton because it needs only the host-supplied MembraneCheck closure.
func membraneProbe(ctx context.Context, c Candidate) (GateOutcome, string) {
	if c.Adapter == nil {
		return GateFail, "no adapter on candidate"
	}
	if c.Adapter.Track() != provider.TrackExperimental {
		return GateFail, "candidate is not Experimental-track (membrane violation)"
	}
	if c.MembraneCheck == nil {
		return GateFail, "no membrane check supplied (cannot prove candidate is out of the routable pool)"
	}
	if ok, reason := c.MembraneCheck(); !ok {
		return GateFail, "membrane check failed: " + reason
	}
	return GatePass, "Experimental-track and absent from production-routable pool"
}
