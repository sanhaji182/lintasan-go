package combo

import (
	"sync"
	"sync/atomic"
)

// TierType represents a subscription/payment tier.
type TierType string

const (
	TierSubscription TierType = "subscription"
	TierAPIKey       TierType = "api_key"
	TierCheap        TierType = "cheap"
	TierFree         TierType = "free"
)

// TierConfig defines the configuration for a multi-tier combo.
type TierConfig struct {
	Tiers []TierDef `json:"tiers"`
}

// TierDef defines a single tier with its providers and exhaustion threshold.
type TierDef struct {
	Type       TierType   `json:"type"`
	Providers  []Provider `json:"providers"`
	MaxTokens  int64      `json:"max_tokens,omitempty"`  // 0 = unlimited
	MaxCost    float64    `json:"max_cost,omitempty"`    // 0 = unlimited
}

// TierUsage tracks per-tier usage statistics.
type TierUsage struct {
	TokensUsed int64
	CostUsed   float64
	Requests   int64
}

// TieredCombo implements multi-tier fallback with auto-downgrade.
type TieredCombo struct {
	mu       sync.RWMutex
	tiers    []TierDef
	usage    []TierUsage
	current  int // current tier index
	config   TierConfig
}

// NewTieredCombo creates a new tiered combo from the given config.
// The combo starts at tier 0 (highest priority) and auto-downgrades when exhausted.
func NewTieredCombo(config TierConfig) *TieredCombo {
	tc := &TieredCombo{
		tiers:   config.Tiers,
		usage:   make([]TierUsage, len(config.Tiers)),
		current: 0,
		config:  config,
	}
	return tc
}

// Resolve returns the provider list from the current active tier.
// If the current tier is exhausted, it auto-downgrades to the next tier.
func (tc *TieredCombo) Resolve() []Provider {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	for i := tc.current; i < len(tc.tiers); i++ {
		if tc.tierAvailable(i) {
			tc.current = i
			return tc.tiers[i].Providers
		}
	}
	// All tiers exhausted — return last tier anyway (best effort)
	if len(tc.tiers) > 0 {
		return tc.tiers[len(tc.tiers)-1].Providers
	}
	return nil
}

// RecordUsage records token consumption for the current tier.
func (tc *TieredCombo) RecordUsage(tokens int64, cost float64) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.current >= 0 && tc.current < len(tc.usage) {
		tc.usage[tc.current].TokensUsed += tokens
		tc.usage[tc.current].CostUsed += cost
		tc.usage[tc.current].Requests++

		// Check if tier is now exhausted and auto-downgrade
		if !tc.tierAvailable(tc.current) && tc.current < len(tc.tiers)-1 {
			tc.current++
		}
	}
}

// CurrentTier returns the index and type of the currently active tier.
func (tc *TieredCombo) CurrentTier() (int, TierType) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.current >= 0 && tc.current < len(tc.tiers) {
		return tc.current, tc.tiers[tc.current].Type
	}
	return -1, ""
}

// TierStats returns usage stats for all tiers.
func (tc *TieredCombo) TierStats() []TierStatsEntry {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	stats := make([]TierStatsEntry, len(tc.tiers))
	for i := range tc.tiers {
		stats[i] = TierStatsEntry{
			Type:       tc.tiers[i].Type,
			Available:  tc.tierAvailable(i),
			TokensUsed: tc.usage[i].TokensUsed,
			CostUsed:   tc.usage[i].CostUsed,
			Requests:   tc.usage[i].Requests,
			MaxTokens:  tc.tiers[i].MaxTokens,
			MaxCost:    tc.tiers[i].MaxCost,
		}
	}
	return stats
}

// Reset resets the tier index to 0 and clears usage stats.
func (tc *TieredCombo) Reset() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.current = 0
	tc.usage = make([]TierUsage, len(tc.tiers))
}

// TierStatsEntry is the public view of per-tier usage stats.
type TierStatsEntry struct {
	Type       TierType
	Available  bool
	TokensUsed int64
	CostUsed   float64
	Requests   int64
	MaxTokens  int64
	MaxCost    float64
}

// tierAvailable checks if a tier still has capacity. Must be called with lock held.
func (tc *TieredCombo) tierAvailable(idx int) bool {
	if idx < 0 || idx >= len(tc.tiers) {
		return false
	}
	tier := tc.tiers[idx]
	usage := tc.usage[idx]

	if tier.MaxTokens > 0 && atomic.LoadInt64(&usage.TokensUsed) >= tier.MaxTokens {
		return false
	}
	if tier.MaxCost > 0 && usage.CostUsed >= tier.MaxCost {
		return false
	}
	return true
}

// AutoBuildCombo creates a TieredCombo automatically from a flat provider list,
// grouping providers into tiers based on their cost characteristics.
func AutoBuildCombo(providers []Provider) *TieredCombo {
	var sub, api, cheap, free []Provider
	for _, p := range providers {
		switch {
		case p.CostPerToken <= 0:
			free = append(free, p)
		case p.CostPerToken < 0.5:
			cheap = append(cheap, p)
		case p.CostPerToken < 5.0:
			api = append(api, p)
		default:
			sub = append(sub, p)
		}
	}

	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: sub},
			{Type: TierAPIKey, Providers: api},
			{Type: TierCheap, Providers: cheap},
			{Type: TierFree, Providers: free},
		},
	}
	return NewTieredCombo(config)
}
