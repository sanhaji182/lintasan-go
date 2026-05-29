package quota

import (
	"testing"
	"time"
)

func TestTrackUsageBasic(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsage("openai", 100, 200)

	stats := pt.GetStats("openai")
	if stats.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", stats.InputTokens)
	}
	if stats.OutputTokens != 200 {
		t.Errorf("expected 200 output tokens, got %d", stats.OutputTokens)
	}
	if stats.TotalTokens != 300 {
		t.Errorf("expected 300 total tokens, got %d", stats.TotalTokens)
	}
	if stats.Requests != 1 {
		t.Errorf("expected 1 request, got %d", stats.Requests)
	}
}

func TestTrackUsageAccumulates(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsage("openai", 100, 200)
	pt.TrackUsage("openai", 50, 100)

	stats := pt.GetStats("openai")
	if stats.TotalTokens != 450 {
		t.Errorf("expected 450 total tokens, got %d", stats.TotalTokens)
	}
	if stats.Requests != 2 {
		t.Errorf("expected 2 requests, got %d", stats.Requests)
	}
}

func TestTrackUsageMultipleProviders(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsage("openai", 100, 200)
	pt.TrackUsage("anthropic", 50, 75)

	openai := pt.GetStats("openai")
	anthropic := pt.GetStats("anthropic")

	if openai.TotalTokens != 300 {
		t.Errorf("openai: expected 300 total, got %d", openai.TotalTokens)
	}
	if anthropic.TotalTokens != 125 {
		t.Errorf("anthropic: expected 125 total, got %d", anthropic.TotalTokens)
	}
}

func TestTrackUsageWithAccount(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsageWithAccount("openai", "account-1", 100, 200)
	pt.TrackUsageWithAccount("openai", "account-2", 50, 75)

	stats1 := pt.GetStatsWithAccount("openai", "account-1")
	stats2 := pt.GetStatsWithAccount("openai", "account-2")

	if stats1.TotalTokens != 300 {
		t.Errorf("account-1: expected 300, got %d", stats1.TotalTokens)
	}
	if stats2.TotalTokens != 125 {
		t.Errorf("account-2: expected 125, got %d", stats2.TotalTokens)
	}
}

func TestBudgetLimitTokens(t *testing.T) {
	pt := NewProviderTracker()
	pt.SetBudget("openai", BudgetLimit{MaxTokens: 1000})

	pt.TrackUsage("openai", 400, 300) // 700 total

	stats := pt.GetStats("openai")
	if stats.Remaining.TokensRemaining != 300 {
		t.Errorf("expected 300 tokens remaining, got %d", stats.Remaining.TokensRemaining)
	}
	if stats.Remaining.PercentRemaining < 29 || stats.Remaining.PercentRemaining > 31 {
		t.Errorf("expected ~30%% remaining, got %.1f%%", stats.Remaining.PercentRemaining)
	}
}

func TestBudgetLimitCost(t *testing.T) {
	pt := NewProviderTracker()
	pt.SetBudget("openai", BudgetLimit{MaxCost: 10.0})

	pt.TrackUsage("openai", 1000, 1000) // 2000 tokens → ~$0.006

	stats := pt.GetStats("openai")
	if stats.Remaining.CostRemaining <= 0 {
		t.Errorf("expected positive cost remaining, got %f", stats.Remaining.CostRemaining)
	}
}

func TestResetCountdown(t *testing.T) {
	pt := NewProviderTracker()
	pt.SetResetPeriod("openai", ResetDaily)

	pt.TrackUsage("openai", 100, 100)

	countdown := pt.ResetCountdown("openai")
	if countdown <= 0 || countdown > 25*time.Hour {
		t.Errorf("expected reasonable countdown, got %v", countdown)
	}
}

func TestResetProvider(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsage("openai", 100, 200)
	pt.ResetProvider("openai")

	stats := pt.GetStats("openai")
	if stats.TotalTokens != 0 {
		t.Errorf("expected 0 after reset, got %d", stats.TotalTokens)
	}
	if stats.Requests != 0 {
		t.Errorf("expected 0 requests after reset, got %d", stats.Requests)
	}
}

func TestAllStats(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsage("openai", 100, 100)
	pt.TrackUsage("anthropic", 50, 50)
	pt.TrackUsageWithAccount("openai", "account-1", 200, 200)

	all := pt.AllStats()
	if len(all) != 3 {
		t.Errorf("expected 3 entries, got %d", len(all))
	}
}

func TestCostEstimation(t *testing.T) {
	pt := NewProviderTracker()

	pt.TrackUsage("openai", 1000000, 1000000) // 2M tokens

	stats := pt.GetStats("openai")
	// Default cost: $3/1M tokens → 2M * $3/1M = $6
	if stats.CostEstimate < 5.0 || stats.CostEstimate > 7.0 {
		t.Errorf("expected ~$6 cost estimate, got $%.2f", stats.CostEstimate)
	}
}

func TestGetStatsUnknownProvider(t *testing.T) {
	pt := NewProviderTracker()

	stats := pt.GetStats("nonexistent")
	if stats.TotalTokens != 0 {
		t.Errorf("expected 0 for unknown provider, got %d", stats.TotalTokens)
	}
	if stats.Remaining.PercentRemaining != 100 {
		t.Errorf("expected 100%% remaining for unknown, got %.1f%%", stats.Remaining.PercentRemaining)
	}
}

func TestSetResetPeriod(t *testing.T) {
	pt := NewProviderTracker()
	pt.TrackUsage("openai", 100, 100)
	pt.SetResetPeriod("openai", ResetWeekly)

	countdown := pt.ResetCountdown("openai")
	if countdown < 6*24*time.Hour {
		t.Errorf("expected weekly countdown (~7d), got %v", countdown)
	}
}
