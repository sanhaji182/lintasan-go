package combo

import (
	"testing"
)

func TestNewTieredCombo(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1", CostPerToken: 10}}},
			{Type: TierAPIKey, Providers: []Provider{{ID: "api1", CostPerToken: 3}}},
			{Type: TierCheap, Providers: []Provider{{ID: "cheap1", CostPerToken: 0.5}}},
			{Type: TierFree, Providers: []Provider{{ID: "free1", CostPerToken: 0}}},
		},
	}
	tc := NewTieredCombo(config)

	// Should start at tier 0 (subscription)
	idx, tierType := tc.CurrentTier()
	if idx != 0 || tierType != TierSubscription {
		t.Errorf("expected tier 0 (subscription), got %d (%s)", idx, tierType)
	}
}

func TestTieredComboResolve(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1"}, {ID: "sub2"}}},
			{Type: TierFree, Providers: []Provider{{ID: "free1"}}},
		},
	}
	tc := NewTieredCombo(config)

	providers := tc.Resolve()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers from tier 0, got %d", len(providers))
	}
	if providers[0].ID != "sub1" {
		t.Errorf("expected sub1, got %s", providers[0].ID)
	}
}

func TestTieredComboAutoDowngrade(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1"}}, MaxTokens: 100},
			{Type: TierAPIKey, Providers: []Provider{{ID: "api1"}}, MaxTokens: 200},
			{Type: TierFree, Providers: []Provider{{ID: "free1"}}},
		},
	}
	tc := NewTieredCombo(config)

	// Exhaust tier 0
	tc.RecordUsage(100, 1.0)

	// Should auto-downgrade to tier 1
	idx, _ := tc.CurrentTier()
	if idx != 1 {
		t.Errorf("expected tier 1 after exhaustion, got %d", idx)
	}

	providers := tc.Resolve()
	if len(providers) != 1 || providers[0].ID != "api1" {
		t.Errorf("expected api1 from tier 1, got %v", providers)
	}
}

func TestTieredComboCostExhaustion(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1"}}, MaxCost: 5.0},
			{Type: TierFree, Providers: []Provider{{ID: "free1"}}},
		},
	}
	tc := NewTieredCombo(config)

	// Exhaust by cost
	tc.RecordUsage(50, 5.0)

	idx, _ := tc.CurrentTier()
	if idx != 1 {
		t.Errorf("expected tier 1 after cost exhaustion, got %d", idx)
	}
}

func TestTieredComboStats(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1"}}, MaxTokens: 1000},
			{Type: TierFree, Providers: []Provider{{ID: "free1"}}},
		},
	}
	tc := NewTieredCombo(config)

	tc.RecordUsage(100, 1.0)
	tc.RecordUsage(200, 2.0)

	stats := tc.TierStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 tier stats, got %d", len(stats))
	}
	if stats[0].TokensUsed != 300 {
		t.Errorf("expected 300 tokens used, got %d", stats[0].TokensUsed)
	}
	if stats[0].Requests != 2 {
		t.Errorf("expected 2 requests, got %d", stats[0].Requests)
	}
	if !stats[0].Available {
		t.Error("tier 0 should still be available")
	}
}

func TestTieredComboReset(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1"}}, MaxTokens: 100},
			{Type: TierFree, Providers: []Provider{{ID: "free1"}}},
		},
	}
	tc := NewTieredCombo(config)

	tc.RecordUsage(100, 1.0)
	idx, _ := tc.CurrentTier()
	if idx != 1 {
		t.Fatalf("expected tier 1, got %d", idx)
	}

	tc.Reset()
	idx, tierType := tc.CurrentTier()
	if idx != 0 || tierType != TierSubscription {
		t.Errorf("after reset expected tier 0, got %d (%s)", idx, tierType)
	}
}

func TestTieredComboAllTiersExhausted(t *testing.T) {
	config := TierConfig{
		Tiers: []TierDef{
			{Type: TierSubscription, Providers: []Provider{{ID: "sub1"}}, MaxTokens: 50},
			{Type: TierFree, Providers: []Provider{{ID: "free1"}}, MaxTokens: 50},
		},
	}
	tc := NewTieredCombo(config)

	tc.RecordUsage(50, 0) // exhaust tier 0
	tc.RecordUsage(50, 0) // exhaust tier 1

	// Should still return last tier's providers (best effort)
	providers := tc.Resolve()
	if len(providers) != 1 {
		t.Errorf("expected 1 provider from last tier, got %d", len(providers))
	}
}

func TestAutoBuildCombo(t *testing.T) {
	providers := []Provider{
		{ID: "expensive", CostPerToken: 15.0},  // subscription tier
		{ID: "mid", CostPerToken: 3.0},          // api_key tier
		{ID: "cheap", CostPerToken: 0.3},        // cheap tier
		{ID: "free", CostPerToken: 0},           // free tier
	}

	tc := AutoBuildCombo(providers)
	stats := tc.TierStats()
	if len(stats) != 4 {
		t.Fatalf("expected 4 tiers, got %d", len(stats))
	}
}
