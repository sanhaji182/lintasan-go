package expprovider

import (
	"context"
	"strings"
	"testing"

	"github.com/sanhaji182/lintasan-go/internal/provider"
)

// --- G2 launcher registry ----------------------------------------------------

func validSpec() LaunchSpec {
	return LaunchSpec{
		Name:       "codex",
		Protocol:   ProtocolACP,
		Path:       "codex",
		Args:       []string{"--acp", "--stdio"},
		AuthMode:   AuthAPIKey,
		AuthEnvVar: "OPENAI_API_KEY",
		BaseEnv:    []string{"PATH=/usr/bin", "HOME=/home/agent"},
	}
}

func TestLaunchSpec_Validate(t *testing.T) {
	if err := validSpec().Validate(); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	cases := []struct {
		name string
		mut  func(*LaunchSpec)
		want error
	}{
		{"no name", func(s *LaunchSpec) { s.Name = "" }, ErrSpecNoName},
		{"no path", func(s *LaunchSpec) { s.Path = "" }, ErrSpecNoPath},
		{"bad protocol", func(s *LaunchSpec) { s.Protocol = "grpc" }, ErrSpecBadProtocol},
		{"bad authmode", func(s *LaunchSpec) { s.AuthMode = "magic" }, ErrSpecBadAuthMode},
		{"auth no envvar", func(s *LaunchSpec) { s.AuthEnvVar = "" }, ErrSpecNoAuthEnvVar},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validSpec()
			tc.mut(&s)
			if err := s.Validate(); err != tc.want {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestLauncherRegistry_RegisterGetDuplicate(t *testing.T) {
	r := NewLauncherRegistry()
	if err := r.Register(validSpec()); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := r.Register(validSpec()); err != ErrSpecDuplicate {
		t.Fatalf("duplicate register: got %v, want ErrSpecDuplicate", err)
	}
	if _, ok := r.Get("codex"); !ok {
		t.Fatal("codex not found after register")
	}
	if got := r.Names(); len(got) != 1 || got[0] != "codex" {
		t.Fatalf("Names = %v, want [codex]", got)
	}
}

// --- G4 credential injection (Invariant 3) -----------------------------------

func TestInjector_BuildEnv_InjectsExactlyOneSecret(t *testing.T) {
	src := CredentialSourceFunc(func(p string) (string, bool) {
		if p == "codex" {
			return "sk-secret-123", true
		}
		return "", false
	})
	in := NewInjector(src)
	env, err := in.BuildEnv(validSpec())
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}
	// BaseEnv (2) + exactly one injected secret.
	if len(env) != 3 {
		t.Fatalf("env len = %d, want 3 (2 base + 1 secret)", len(env))
	}
	var secretCount int
	for _, kv := range env {
		if strings.HasPrefix(kv, "OPENAI_API_KEY=") {
			secretCount++
			if kv != "OPENAI_API_KEY=sk-secret-123" {
				t.Fatalf("wrong secret value: %s", kv)
			}
		}
	}
	if secretCount != 1 {
		t.Fatalf("expected exactly 1 secret var, got %d", secretCount)
	}
}

func TestInjector_BuildEnv_MissingCredentialIsFatal(t *testing.T) {
	in := NewInjector(CredentialSourceFunc(func(string) (string, bool) { return "", false }))
	if _, err := in.BuildEnv(validSpec()); err != ErrCredentialMissing {
		t.Fatalf("got %v, want ErrCredentialMissing", err)
	}
}

func TestInjector_BuildEnv_RejectsSecretInBaseEnv(t *testing.T) {
	s := validSpec()
	s.BaseEnv = append(s.BaseEnv, "OPENAI_API_KEY=leaked-in-spec")
	in := NewInjector(CredentialSourceFunc(func(string) (string, bool) { return "x", true }))
	if _, err := in.BuildEnv(s); err != ErrBaseEnvHasSecret {
		t.Fatalf("got %v, want ErrBaseEnvHasSecret", err)
	}
}

func TestInjector_BuildEnv_AuthNoneAddsNoSecret(t *testing.T) {
	s := validSpec()
	s.AuthMode = AuthNone
	s.AuthEnvVar = ""
	in := NewInjector(nil) // no source needed
	env, err := in.BuildEnv(s)
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}
	if len(env) != len(s.BaseEnv) {
		t.Fatalf("AuthNone added env vars: %v", env)
	}
}

// The spec must never carry the secret value — only the var NAME.
func TestLaunchSpec_NeverCarriesSecretValue(t *testing.T) {
	s := validSpec()
	// AuthEnvVar names the var; there is no field that could hold a value.
	if strings.Contains(s.AuthEnvVar, "=") {
		t.Fatal("AuthEnvVar must be a name, not a key=value")
	}
}

// --- G1 adapter seam + membrane gating ---------------------------------------

func TestACPProvider_IsExperimentalTrack(t *testing.T) {
	p := NewACPProvider(validSpec(), provider.NewCapabilitySet(provider.CapCoding), NewInjector(nil))
	if p.Track() != provider.TrackExperimental {
		t.Fatalf("Track = %s, want experimental", p.Track())
	}
	if p.Name() != "codex" {
		t.Fatalf("Name = %s", p.Name())
	}
}

// The load-bearing membrane test: an ACPProvider registered in a registry is
// NOT reachable by the production resolver, only via the explicit door.
func TestACPProvider_MembraneGated(t *testing.T) {
	reg := provider.NewRegistry()
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	if err := reg.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Production door: must NOT return the Experimental provider.
	fallback := provider.NewDefaultProvider("fallback")
	if got := reg.ResolveRoutable("codex", fallback); got.Name() != "fallback" {
		t.Fatalf("ResolveRoutable returned Experimental provider %q — MEMBRANE BREACH", got.Name())
	}
	// Routable pool must be empty (no Official providers registered).
	if pool := reg.RoutableProviders(); len(pool) != 0 {
		t.Fatalf("routable pool leaked Experimental: %v", pool)
	}
	// Explicit door: MUST return it.
	got, ok := reg.ResolveExperimental("codex")
	if !ok || got.Name() != "codex" {
		t.Fatalf("ResolveExperimental failed to return codex: %v %v", got, ok)
	}
}

// HTTP path must be a loud, contained failure (agents run via Agent.Run).
func TestACPProvider_HTTPPathRefused(t *testing.T) {
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	if _, err := p.Prepare(context.Background(), &provider.Request{}, &provider.ConnConfig{}); err != ErrUseAgentInterface {
		t.Fatalf("Prepare: got %v, want ErrUseAgentInterface", err)
	}
	if _, err := p.Translate(context.Background(), nil, &provider.Request{}); err != ErrUseAgentInterface {
		t.Fatalf("Translate: got %v, want ErrUseAgentInterface", err)
	}
}

func TestACPProvider_StopAgentIdempotentBeforeStart(t *testing.T) {
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	if err := p.StopAgent(); err != nil {
		t.Fatalf("StopAgent before start should be nil, got %v", err)
	}
}

// --- G3 routing entry (explicit opt-in) --------------------------------------

func TestDetectExperimental_ModelPrefix(t *testing.T) {
	sig, ok := DetectExperimental("experimental/codex/gpt-5", nil)
	if !ok || sig.Provider != "codex" || sig.Model != "gpt-5" || sig.Via != "model_prefix" {
		t.Fatalf("bad signal: %+v ok=%v", sig, ok)
	}
	sig2, ok2 := DetectExperimental("experimental/claude-code", nil)
	if !ok2 || sig2.Provider != "claude-code" || sig2.Model != "" {
		t.Fatalf("bad signal2: %+v ok=%v", sig2, ok2)
	}
}

func TestDetectExperimental_Header(t *testing.T) {
	h := map[string][]string{
		TrackHeader:    {"experimental"},
		ProviderHeader: {"gemini-cli"},
	}
	sig, ok := DetectExperimental("gpt-4o", h)
	if !ok || sig.Provider != "gemini-cli" || sig.Via != "header" {
		t.Fatalf("bad header signal: %+v ok=%v", sig, ok)
	}
}

func TestDetectExperimental_NoSignal_DefaultPathSafe(t *testing.T) {
	if _, ok := DetectExperimental("gpt-4o", nil); ok {
		t.Fatal("plain model produced an experimental signal — DEFAULT PATH BREACH")
	}
	// header track without provider name → no signal
	h := map[string][]string{TrackHeader: {"experimental"}}
	if _, ok := DetectExperimental("gpt-4o", h); ok {
		t.Fatal("track header without provider name should not signal")
	}
	// empty provider after prefix → no signal
	if _, ok := DetectExperimental("experimental/", nil); ok {
		t.Fatal("empty experimental prefix should not signal")
	}
}

// --- G6 lifecycle ------------------------------------------------------------

func TestLifecycle_HappyPath(t *testing.T) {
	r := NewRecord("codex")
	if r.State != StateProposed {
		t.Fatalf("initial state = %s, want proposed", r.State)
	}
	for _, next := range []State{StateAdmitted, StateActive, StateDeprecated, StateRetired} {
		if err := r.Transition(next); err != nil {
			t.Fatalf("transition to %s: %v", next, err)
		}
	}
}

func TestLifecycle_NoPromotionEdge(t *testing.T) {
	// There is no "official" state and no edge to it; proposed->active is also
	// forbidden (must go through admitted). Assert the invariant structurally.
	if CanTransition(StateActive, "official") {
		t.Fatal("active->official transition exists — PROMOTION BREACH")
	}
	if CanTransition(StateProposed, StateActive) {
		t.Fatal("proposed->active skips admission")
	}
}

func TestLifecycle_BadTransitionRejected(t *testing.T) {
	r := NewRecord("codex")
	if err := r.Transition(StateRetired); err != nil {
		// proposed->retired is allowed; use a truly bad one instead.
		t.Fatalf("proposed->retired should be allowed: %v", err)
	}
	r2 := NewRecord("codex")
	_ = r2.Transition(StateAdmitted)
	_ = r2.Transition(StateActive)
	if err := r2.Transition(StateProposed); err == nil {
		t.Fatal("active->proposed should be rejected")
	}
}

func TestExperimentalBadge_NonEmpty(t *testing.T) {
	b := ExperimentalBadge()
	if b.Label == "" || b.Detail == "" {
		t.Fatal("experimental badge must be non-empty (Invariant 5)")
	}
}

// --- G5 admission harness ----------------------------------------------------

func TestHarness_FailsClosedBeforeProbesSupplied(t *testing.T) {
	reg := provider.NewRegistry()
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	_ = reg.Register(p)
	h := NewHarness()
	rep := h.Run(context.Background(), Candidate{
		Provider: "codex",
		Adapter:  p,
		Spec:     validSpec(),
		MembraneCheck: func() (bool, string) {
			// membrane holds: not in routable pool
			return len(reg.RoutableProviders()) == 0, "routable pool empty"
		},
	})
	// Membrane PASSes, but isolation/protocol/acceptance are not-implemented →
	// the candidate must NOT be admitted.
	if rep.Go() {
		t.Fatal("harness admitted a candidate with stub probes — must fail closed")
	}
	// Verify the membrane gate itself passed.
	var membraneOK bool
	for _, r := range rep.Results {
		if r.Gate == GateMembrane && r.Outcome == GatePass {
			membraneOK = true
		}
	}
	if !membraneOK {
		t.Fatal("membrane gate should PASS for a correctly-gated Experimental provider")
	}
}

func TestHarness_AllGatesPassIsGo(t *testing.T) {
	reg := provider.NewRegistry()
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	_ = reg.Register(p)
	pass := func(ctx context.Context, c Candidate) (GateOutcome, string) { return GatePass, "stub pass" }
	h := NewHarness().
		WithGate(GateIsolation, pass).
		WithGate(GateProtocol, pass).
		WithGate(GateAcceptance, pass)
	rep := h.Run(context.Background(), Candidate{
		Provider:      "codex",
		Adapter:       p,
		Spec:          validSpec(),
		MembraneCheck: func() (bool, string) { return true, "ok" },
	})
	if !rep.Go() {
		t.Fatalf("all-pass harness should be GO: %+v", rep.Results)
	}
}

func TestHarness_MembraneFailBlocksAdmission(t *testing.T) {
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	pass := func(ctx context.Context, c Candidate) (GateOutcome, string) { return GatePass, "ok" }
	h := NewHarness().
		WithGate(GateIsolation, pass).
		WithGate(GateProtocol, pass).
		WithGate(GateAcceptance, pass)
	// MembraneCheck reports a breach → must block even with everything else green.
	rep := h.Run(context.Background(), Candidate{
		Provider:      "codex",
		Adapter:       p,
		MembraneCheck: func() (bool, string) { return false, "leaked into routable pool" },
	})
	if rep.Go() {
		t.Fatal("membrane breach must block admission regardless of other gates")
	}
}

func TestHarness_RecoversPanickingProbe(t *testing.T) {
	p := NewACPProvider(validSpec(), nil, NewInjector(nil))
	boom := func(ctx context.Context, c Candidate) (GateOutcome, string) { panic("probe blew up") }
	h := NewHarness().WithGate(GateIsolation, boom)
	rep := h.Run(context.Background(), Candidate{
		Provider:      "codex",
		Adapter:       p,
		MembraneCheck: func() (bool, string) { return true, "ok" },
	})
	if rep.Go() {
		t.Fatal("panicking probe must not yield GO")
	}
	// And the harness itself must not have panicked (we got here).
}
