package lb

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Account represents an API key/account for a provider.
type Account struct {
	ID           string
	APIKey       string
	Priority     int       // lower = higher priority (drain primary first)
	Active       bool
	SuccessCount int64
	FailCount    int64
	RateLimited  bool
	RateLimitedAt time.Time
	Cooldown     time.Duration // how long to skip after rate limit
}

// MultiAccountLB implements round-robin across multiple API key accounts
// for a single provider, with rate-limit awareness and priority drain.
type MultiAccountLB struct {
	mu         sync.RWMutex
	accounts   []Account
	rrIndex    uint64
	providerID string
}

// NewMultiAccountLB creates a new multi-account round-robin load balancer.
func NewMultiAccountLB(providerID string, accounts []Account) *MultiAccountLB {
	for i := range accounts {
		if accounts[i].Cooldown == 0 {
			accounts[i].Cooldown = 60 * time.Second // default cooldown
		}
	}
	return &MultiAccountLB{
		providerID: providerID,
		accounts:   accounts,
	}
}

// Pick returns the next available account using round-robin.
// Skips rate-limited accounts (cooldown expired accounts are reactivated).
// Priority: drains primary (lowest priority number) first.
func (m *MultiAccountLB) Pick() (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.accounts) == 0 {
		return nil, fmt.Errorf("multi-account: no accounts for provider %s", m.providerID)
	}

	now := time.Now()
	available := m.availableAccountsLocked(now)
	if len(available) == 0 {
		return nil, fmt.Errorf("multi-account: all accounts for provider %s are rate-limited", m.providerID)
	}

	// Round-robin among available accounts (sorted by priority)
	rrIdx := atomic.AddUint64(&m.rrIndex, 1) - 1
	chosen := available[rrIdx%uint64(len(available))]
	idx := m.accountIndexLocked(chosen.ID)
	if idx >= 0 {
		return &m.accounts[idx], nil
	}
	return &chosen, nil
}

// RecordSuccess marks an account as having succeeded.
func (m *MultiAccountLB) RecordSuccess(accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.accounts {
		if m.accounts[i].ID == accountID {
			m.accounts[i].SuccessCount++
			m.accounts[i].Active = true
			break
		}
	}
}

// RecordFailure marks an account as having failed.
func (m *MultiAccountLB) RecordFailure(accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.accounts {
		if m.accounts[i].ID == accountID {
			m.accounts[i].FailCount++
			break
		}
	}
}

// MarkRateLimited marks an account as rate-limited with a cooldown period.
func (m *MultiAccountLB) MarkRateLimited(accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.accounts {
		if m.accounts[i].ID == accountID {
			m.accounts[i].RateLimited = true
			m.accounts[i].RateLimitedAt = time.Now()
			break
		}
	}
}

// Accounts returns a snapshot of all accounts.
func (m *MultiAccountLB) Accounts() []Account {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Account, len(m.accounts))
	copy(out, m.accounts)
	return out
}

// AccountStats returns health stats per account.
func (m *MultiAccountLB) AccountStats() []AccountStatsEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := make([]AccountStatsEntry, len(m.accounts))
	for i, a := range m.accounts {
		total := a.SuccessCount + a.FailCount
		successRate := 0.0
		if total > 0 {
			successRate = float64(a.SuccessCount) / float64(total)
		}
		stats[i] = AccountStatsEntry{
			ID:           a.ID,
			SuccessCount: a.SuccessCount,
			FailCount:    a.FailCount,
			SuccessRate:  successRate,
			RateLimited:  a.RateLimited,
			Active:       a.Active,
		}
	}
	return stats
}

// AccountStatsEntry is the public view of account health stats.
type AccountStatsEntry struct {
	ID           string
	SuccessCount int64
	FailCount    int64
	SuccessRate  float64
	RateLimited  bool
	Active       bool
}

// AddAccount adds a new account to the pool.
func (m *MultiAccountLB) AddAccount(acct Account) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acct.Cooldown == 0 {
		acct.Cooldown = 60 * time.Second
	}
	m.accounts = append(m.accounts, acct)
}

// RemoveAccount removes an account by ID.
func (m *MultiAccountLB) RemoveAccount(accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.accounts {
		if m.accounts[i].ID == accountID {
			m.accounts = append(m.accounts[:i], m.accounts[i+1:]...)
			break
		}
	}
}

// availableAccountsLocked returns accounts that are active and not rate-limited.
// Must be called with mu held.
func (m *MultiAccountLB) availableAccountsLocked(now time.Time) []Account {
	var available []Account
	for i := range m.accounts {
		a := &m.accounts[i]
		// Auto-reactivate if cooldown has passed
		if a.RateLimited && a.Cooldown > 0 && now.Sub(a.RateLimitedAt) >= a.Cooldown {
			a.RateLimited = false
		}
		if a.Active && !a.RateLimited {
			available = append(available, *a)
		}
	}
	// Sort by priority (lower number = higher priority)
	sortAccountsByPriority(available)
	return available
}

func (m *MultiAccountLB) accountIndexLocked(id string) int {
	for i := range m.accounts {
		if m.accounts[i].ID == id {
			return i
		}
	}
	return -1
}

func sortAccountsByPriority(accounts []Account) {
	for i := 1; i < len(accounts); i++ {
		for j := i; j > 0 && accounts[j].Priority < accounts[j-1].Priority; j-- {
			accounts[j], accounts[j-1] = accounts[j-1], accounts[j]
		}
	}
}
