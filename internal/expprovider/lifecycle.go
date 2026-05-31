package expprovider

// G6 — Lifecycle state machine + risk badge (Pillar C + Invariant 5).
//
// A provider is always in exactly one Lifecycle state with explicit, validated
// transitions. There is deliberately NO active->Official edge: promotion is
// forbidden forever (Invariant 1 / vision §6.3). Retirement has zero Official
// blast radius because Experimental is never in the Official pool + credentials
// are isolated — flag-off + remove adapter, done.
//
// The RiskBadge is the Invariant-5 surface: Experimental capabilities are
// DECLARED but never trusted for Official routing; the badge is what the
// dashboard shows so a human always sees "Experimental — may break".

import (
	"errors"
	"fmt"
)

// State is a lifecycle state.
type State string

const (
	// StateProposed — named in the locked taxonomy, not yet pursued.
	StateProposed State = "proposed"
	// StateAdmitted — approved to build; adapter exists; not yet certified.
	StateAdmitted State = "admitted"
	// StateActive — live behind the Experimental flag, opt-in only. Entry
	// requires an admission-harness PASS (+ acceptance run for ACP providers).
	StateActive State = "active"
	// StateDeprecated — present but flagged for removal (breakage/ToS/low use).
	StateDeprecated State = "deprecated"
	// StateRetired — flag-off + adapter removed. Re-proposal starts over.
	StateRetired State = "retired"
)

// Lifecycle errors.
var (
	ErrBadTransition      = errors.New("expprovider: invalid lifecycle transition")
	ErrUnknownState       = errors.New("expprovider: unknown lifecycle state")
	ErrPromotionForbidden = errors.New("expprovider: promotion Experimental->Official is forbidden")
)

// allowedTransitions encodes the state machine. Note the absence of any edge to
// an "official" state — promotion is structurally impossible here.
var allowedTransitions = map[State]map[State]bool{
	StateProposed:   {StateAdmitted: true, StateRetired: true},
	StateAdmitted:   {StateActive: true, StateDeprecated: true, StateRetired: true},
	StateActive:     {StateDeprecated: true},
	StateDeprecated: {StateActive: true, StateRetired: true}, // can recover or retire
	StateRetired:    {StateProposed: true},                   // re-proposal restarts
}

// ValidState reports whether s is a known state.
func ValidState(s State) bool {
	_, ok := allowedTransitions[s]
	return ok
}

// CanTransition reports whether from->to is a permitted transition.
func CanTransition(from, to State) bool {
	if !ValidState(from) || !ValidState(to) {
		return false
	}
	return allowedTransitions[from][to]
}

// RiskBadge is the dashboard-facing risk descriptor for an Experimental
// provider (Invariant 5). It is metadata only — never consulted by routing.
type RiskBadge struct {
	// Label is the short risk tag (always non-empty for Experimental).
	Label string
	// Detail is a human sentence shown alongside the provider.
	Detail string
}

// ExperimentalBadge is the standard badge every Experimental provider carries.
func ExperimentalBadge() RiskBadge {
	return RiskBadge{
		Label:  "Experimental — may break",
		Detail: "Opt-in only. Not eligible for default/auto routing; capabilities are declared but not trusted for production selection.",
	}
}

// Record is the lifecycle bookkeeping for one provider. It is host-owned state
// (persisted by the host later); this struct + its Transition method are the
// validated mutation surface so an invalid state move is impossible.
type Record struct {
	Provider string
	State    State
	Badge    RiskBadge
}

// NewRecord creates a proposed-state record for a provider with the standard
// Experimental badge.
func NewRecord(providerName string) *Record {
	return &Record{Provider: providerName, State: StateProposed, Badge: ExperimentalBadge()}
}

// Transition moves the record to next, validating the edge. It returns
// ErrBadTransition (with context) for a disallowed move and leaves the state
// unchanged on error.
func (r *Record) Transition(next State) error {
	if !ValidState(next) {
		return fmt.Errorf("%w: %q", ErrUnknownState, next)
	}
	if !CanTransition(r.State, next) {
		return fmt.Errorf("%w: %s -> %s", ErrBadTransition, r.State, next)
	}
	r.State = next
	return nil
}
