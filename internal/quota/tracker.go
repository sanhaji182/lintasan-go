package quota

import (
	"sync"
	"time"
)

// ResetPeriod defines how often quota resets.
type ResetPeriod string

const (
	Reset5Hour  ResetPeriod = "5h"
	ResetDaily  ResetPeriod = "daily"
	ResetWeekly ResetPeriod = "weekly"
	ResetMonthly ResetPeriod = "monthly"
)

// ProviderTracker tracks detailed token consumption per provider/account with
// budget limits, reset countdowns, and cost estimation.
type ProviderTracker struct {
	mu       sync.RWMutex
	accounts map[string]*AccountQuota // key: "provider" or "provider:accountID"
	budgets  map[string]BudgetLimit   // provider → budget limit
}

// AccountQuota holds usage data for a single provider/account.
type AccountQuota struct {
	Provider     string
	AccountID    string
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	Requests     int64
	CostEstimate float64
	ResetPeriod  ResetPeriod
	LastReset    time.Time
	CreatedAt    time.Time
}

// BudgetLimit defines spending limits for a provider.
type BudgetLimit struct {
	MaxTokens  int64   // 0 = unlimited
	MaxCost    float64 // 0 = unlimited
	MaxRequests int64  // 0 = unlimited
}

// QuotaStats is the public view of quota statistics.
type QuotaStats struct {
	Provider     string       `json:"provider"`
	AccountID    string       `json:"account_id,omitempty"`
	InputTokens  int64        `json:"input_tokens"`
	OutputTokens int64        `json:"output_tokens"`
	TotalTokens  int64        `json:"total_tokens"`
	Requests     int64        `json:"requests"`
	CostEstimate float64      `json:"cost_estimate"`
	Remaining    QuotaRemaining `json:"remaining"`
	NextReset    time.Time    `json:"next_reset"`
}

// QuotaRemaining shows remaining quota within budget limits.
type QuotaRemaining struct {
	TokensRemaining  int64   `json:"tokens_remaining"`
	CostRemaining    float64 `json:"cost_remaining"`
	RequestsRemaining int64  `json:"requests_remaining"`
	PercentRemaining float64 `json:"percent_remaining"`
}

// NewProviderTracker creates a new provider quota tracker.
func NewProviderTracker() *ProviderTracker {
	return &ProviderTracker{
		accounts: make(map[string]*AccountQuota),
		budgets:  make(map[string]BudgetLimit),
	}
}

// TrackUsage records token consumption for a provider (and optionally an account).
// This is the primary public API for tracking usage.
func (pt *ProviderTracker) TrackUsage(provider string, inputTokens, outputTokens int) {
	pt.TrackUsageWithAccount(provider, "", inputTokens, outputTokens)
}

// TrackUsageWithAccount records token consumption for a specific provider:account.
func (pt *ProviderTracker) TrackUsageWithAccount(provider, accountID string, inputTokens, outputTokens int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := providerKey(provider, accountID)
	aq, ok := pt.accounts[key]
	if !ok {
		aq = &AccountQuota{
			Provider:    provider,
			AccountID:   accountID,
			ResetPeriod: ResetDaily,
			LastReset:   time.Now(),
			CreatedAt:   time.Now(),
		}
		pt.accounts[key] = aq
	}

	// Check if reset is needed
	pt.checkResetLocked(aq)

	total := int64(inputTokens + outputTokens)
	aq.InputTokens += int64(inputTokens)
	aq.OutputTokens += int64(outputTokens)
	aq.TotalTokens += total
	aq.Requests++

	// Cost estimation: use average $3/1M tokens as baseline
	costPerMillion := 3.0
	if budget, hasBudget := pt.budgets[provider]; hasBudget && budget.MaxCost > 0 {
		costPerMillion = budget.MaxCost * 1000000 / float64(budget.MaxTokens)
		if costPerMillion <= 0 || costPerMillion > 100 {
			costPerMillion = 3.0
		}
	}
	aq.CostEstimate += float64(total) * costPerMillion / 1000000.0
}

// SetBudget sets a budget limit for a provider.
func (pt *ProviderTracker) SetBudget(provider string, limit BudgetLimit) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.budgets[provider] = limit
}

// GetStats returns quota stats for a provider/account.
func (pt *ProviderTracker) GetStats(provider string) *QuotaStats {
	return pt.GetStatsWithAccount(provider, "")
}

// GetStatsWithAccount returns quota stats for a specific provider:account.
func (pt *ProviderTracker) GetStatsWithAccount(provider, accountID string) *QuotaStats {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	key := providerKey(provider, accountID)
	aq, ok := pt.accounts[key]
	if !ok {
		return &QuotaStats{
			Provider: provider,
			Remaining: QuotaRemaining{
				TokensRemaining:   -1, // unlimited
				CostRemaining:     -1,
				PercentRemaining:  100,
			},
		}
	}

	stats := &QuotaStats{
		Provider:     aq.Provider,
		AccountID:    aq.AccountID,
		InputTokens:  aq.InputTokens,
		OutputTokens: aq.OutputTokens,
		TotalTokens:  aq.TotalTokens,
		Requests:     aq.Requests,
		CostEstimate: aq.CostEstimate,
		NextReset:    pt.nextResetLocked(aq),
	}

	if budget, hasBudget := pt.budgets[provider]; hasBudget {
		remaining := QuotaRemaining{}
		if budget.MaxTokens > 0 {
			remaining.TokensRemaining = budget.MaxTokens - aq.TotalTokens
			if remaining.TokensRemaining < 0 {
				remaining.TokensRemaining = 0
			}
			remaining.PercentRemaining = float64(remaining.TokensRemaining) / float64(budget.MaxTokens) * 100
		} else {
			remaining.TokensRemaining = -1
			remaining.PercentRemaining = 100
		}
		if budget.MaxCost > 0 {
			remaining.CostRemaining = budget.MaxCost - aq.CostEstimate
			if remaining.CostRemaining < 0 {
				remaining.CostRemaining = 0
			}
		} else {
			remaining.CostRemaining = -1
		}
		if budget.MaxRequests > 0 {
			remaining.RequestsRemaining = budget.MaxRequests - aq.Requests
			if remaining.RequestsRemaining < 0 {
				remaining.RequestsRemaining = 0
			}
		} else {
			remaining.RequestsRemaining = -1
		}
		stats.Remaining = remaining
	} else {
		stats.Remaining = QuotaRemaining{
			TokensRemaining:   -1,
			CostRemaining:     -1,
			RequestsRemaining: -1,
			PercentRemaining:  100,
		}
	}

	return stats
}

// AllStats returns stats for all tracked providers/accounts.
func (pt *ProviderTracker) AllStats() []QuotaStats {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	stats := make([]QuotaStats, 0, len(pt.accounts))
	for _, aq := range pt.accounts {
		s := pt.getStatsLocked(aq.Provider, aq.AccountID)
		if s != nil {
			stats = append(stats, *s)
		}
	}
	return stats
}

// ResetCountdown returns the duration until the next quota reset.
func (pt *ProviderTracker) ResetCountdown(provider string) time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	key := providerKey(provider, "")
	aq, ok := pt.accounts[key]
	if !ok {
		return 0
	}
	next := pt.nextResetLocked(aq)
	d := time.Until(next)
	if d < 0 {
		return 0
	}
	return d
}

// SetResetPeriod sets the reset period for a provider.
func (pt *ProviderTracker) SetResetPeriod(provider string, period ResetPeriod) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := providerKey(provider, "")
	if aq, ok := pt.accounts[key]; ok {
		aq.ResetPeriod = period
	}
}

// ResetProvider resets usage counters for a provider.
func (pt *ProviderTracker) ResetProvider(provider string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := providerKey(provider, "")
	if aq, ok := pt.accounts[key]; ok {
		aq.InputTokens = 0
		aq.OutputTokens = 0
		aq.TotalTokens = 0
		aq.Requests = 0
		aq.CostEstimate = 0
		aq.LastReset = time.Now()
	}
}

// checkResetLocked checks if the quota period has elapsed and resets if needed.
// Must be called with mu held for writing.
func (pt *ProviderTracker) checkResetLocked(aq *AccountQuota) {
	now := time.Now()
	resetInterval := pt.resetInterval(aq.ResetPeriod)
	if now.Sub(aq.LastReset) >= resetInterval {
		aq.InputTokens = 0
		aq.OutputTokens = 0
		aq.TotalTokens = 0
		aq.Requests = 0
		aq.CostEstimate = 0
		aq.LastReset = now
	}
}

// nextResetLocked calculates the next reset time.
func (pt *ProviderTracker) nextResetLocked(aq *AccountQuota) time.Time {
	return aq.LastReset.Add(pt.resetInterval(aq.ResetPeriod))
}

// resetInterval returns the duration for a reset period.
func (pt *ProviderTracker) resetInterval(period ResetPeriod) time.Duration {
	switch period {
	case Reset5Hour:
		return 5 * time.Hour
	case ResetDaily:
		return 24 * time.Hour
	case ResetWeekly:
		return 7 * 24 * time.Hour
	case ResetMonthly:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// getStatsLocked builds stats for a provider:account. Must be called with RLock.
func (pt *ProviderTracker) getStatsLocked(provider, accountID string) *QuotaStats {
	key := providerKey(provider, accountID)
	aq, ok := pt.accounts[key]
	if !ok {
		return nil
	}
	// Reuse GetStats logic but with lock already held
	return &QuotaStats{
		Provider:     aq.Provider,
		AccountID:    aq.AccountID,
		InputTokens:  aq.InputTokens,
		OutputTokens: aq.OutputTokens,
		TotalTokens:  aq.TotalTokens,
		Requests:     aq.Requests,
		CostEstimate: aq.CostEstimate,
		NextReset:    pt.nextResetLocked(aq),
		Remaining:    QuotaRemaining{TokensRemaining: -1, CostRemaining: -1, RequestsRemaining: -1, PercentRemaining: 100},
	}
}

func providerKey(provider, accountID string) string {
	if accountID == "" {
		return provider
	}
	return provider + ":" + accountID
}
