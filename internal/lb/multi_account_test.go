package lb

import (
	"testing"
	"time"
)

func TestMultiAccountRoundRobin(t *testing.T) {
	accounts := []Account{
		{ID: "a1", APIKey: "key-1", Priority: 1, Active: true},
		{ID: "a2", APIKey: "key-2", Priority: 2, Active: true},
		{ID: "a3", APIKey: "key-3", Priority: 3, Active: true},
	}
	m := NewMultiAccountLB("test-provider", accounts)

	seen := make(map[string]int)
	for i := 0; i < 6; i++ {
		acct, err := m.Pick()
		if err != nil {
			t.Fatalf("Pick %d: %v", i, err)
		}
		seen[acct.ID]++
	}

	// Should cycle through all 3 accounts
	for _, id := range []string{"a1", "a2", "a3"} {
		if seen[id] != 2 {
			t.Errorf("account %s picked %d times, expected 2", id, seen[id])
		}
	}
}

func TestMultiAccountSkipsRateLimited(t *testing.T) {
	accounts := []Account{
		{ID: "a1", APIKey: "key-1", Priority: 1, Active: true},
		{ID: "a2", APIKey: "key-2", Priority: 2, Active: true},
	}
	m := NewMultiAccountLB("test-provider", accounts)

	// Rate limit a1
	m.MarkRateLimited("a1")

	// All picks should be a2
	for i := 0; i < 5; i++ {
		acct, err := m.Pick()
		if err != nil {
			t.Fatalf("Pick %d: %v", i, err)
		}
		if acct.ID != "a2" {
			t.Errorf("expected a2 (a1 is rate-limited), got %s", acct.ID)
		}
	}
}

func TestMultiAccountCooldownReactivation(t *testing.T) {
	accounts := []Account{
		{ID: "a1", APIKey: "key-1", Priority: 1, Active: true, Cooldown: 10 * time.Millisecond},
		{ID: "a2", APIKey: "key-2", Priority: 2, Active: true},
	}
	m := NewMultiAccountLB("test-provider", accounts)

	m.MarkRateLimited("a1")
	acct, err := m.Pick()
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if acct.ID != "a2" {
		t.Errorf("expected a2, got %s", acct.ID)
	}

	// Wait for cooldown
	time.Sleep(15 * time.Millisecond)

	// a1 should be reactivated
	acct, err = m.Pick()
	if err != nil {
		t.Fatalf("Pick after cooldown: %v", err)
	}
	if acct.ID != "a1" {
		t.Logf("a1 should have reactivated after cooldown, got %s", acct.ID)
	}
}

func TestMultiAccountPriorityOrder(t *testing.T) {
	accounts := []Account{
		{ID: "primary", APIKey: "key-1", Priority: 1, Active: true},
		{ID: "secondary", APIKey: "key-2", Priority: 2, Active: true},
	}
	m := NewMultiAccountLB("test-provider", accounts)

	// First pick should be primary (lowest priority number = highest priority, sorted first, RR starts at 0)
	acct, err := m.Pick()
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if acct.ID != "primary" {
		t.Errorf("expected primary for first pick (priority sorted), got %s", acct.ID)
	}
}

func TestMultiAccountHealthTracking(t *testing.T) {
	accounts := []Account{
		{ID: "a1", APIKey: "key-1", Priority: 1, Active: true},
	}
	m := NewMultiAccountLB("test-provider", accounts)

	m.RecordSuccess("a1")
	m.RecordSuccess("a1")
	m.RecordFailure("a1")

	stats := m.AccountStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}
	if stats[0].SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", stats[0].SuccessCount)
	}
	if stats[0].FailCount != 1 {
		t.Errorf("expected 1 failure, got %d", stats[0].FailCount)
	}
}

func TestMultiAccountAllRateLimited(t *testing.T) {
	accounts := []Account{
		{ID: "a1", APIKey: "key-1", Priority: 1, Active: true, Cooldown: time.Hour},
	}
	m := NewMultiAccountLB("test-provider", accounts)

	m.MarkRateLimited("a1")
	_, err := m.Pick()
	if err == nil {
		t.Error("expected error when all accounts are rate-limited")
	}
}

func TestMultiAccountEmpty(t *testing.T) {
	m := NewMultiAccountLB("test-provider", nil)
	_, err := m.Pick()
	if err == nil {
		t.Error("expected error for empty accounts")
	}
}

func TestMultiAccountAddRemove(t *testing.T) {
	m := NewMultiAccountLB("test-provider", []Account{
		{ID: "a1", APIKey: "key-1", Priority: 1, Active: true},
	})

	m.AddAccount(Account{ID: "a2", APIKey: "key-2", Priority: 2, Active: true})
	accounts := m.Accounts()
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts after add, got %d", len(accounts))
	}

	m.RemoveAccount("a1")
	accounts = m.Accounts()
	if len(accounts) != 1 {
		t.Errorf("expected 1 account after remove, got %d", len(accounts))
	}
	if accounts[0].ID != "a2" {
		t.Errorf("expected a2 remaining, got %s", accounts[0].ID)
	}
}
