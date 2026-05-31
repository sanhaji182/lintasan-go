// Package expprovider is the Experimental-provider substrate (G1–G6): the seam
// that lets an Experimental agent (driven over ACP, isolated by E1) be modeled
// as a provider.Provider so it is registry-resident and gated by the Phase-2
// membrane (Track == Experimental can never be reached by production routing).
//
// SCOPE / SAFETY (locked):
//   - This package is provider-agnostic substrate. It implements NO concrete
//     provider (no Codex/Claude Code/Gemini CLI/Copilot). Those are onboarded
//     later, each behind its own checkpoint + admission run.
//   - It is additive and dormant: nothing here is wired into the live proxy hot
//     path. The current proxy routes connection-based (DB); these are the
//     forward primitives the Experimental container will route through once
//     onboarding begins. Importing this package changes no production behavior.
//   - It depends on internal/provider (the SDK contract + membrane) and
//     internal/experimental (E1 subprocess + ACP broker). Neither of those
//     depends on this package — the dependency is one-way, so this package can
//     never drag Experimental wiring into the core SDK.
//
// The six substrate pieces:
//
//	G1 adapter seam        — ACPProvider: provider.Provider + the Agent exec iface
//	G2 launcher registry   — LaunchSpec + LauncherRegistry (name -> how to launch)
//	G3 routing entry        — DetectExperimental: the explicit opt-in signal parser
//	G4 credential injection — Injector: per-provider secret, injected post-prepare
//	G5 admission harness     — Harness: isolation + protocol + acceptance gates
//	G6 lifecycle/badge       — Lifecycle state machine + RiskBadge metadata
package expprovider
