package cache

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// newTestDB opens an in-memory SQLite database and initializes the exact cache table.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := InitExactCache(db); err != nil {
		t.Fatalf("InitExactCache failed: %v", err)
	}
	return db
}

func TestExact_GetExactMatch_EmptyWhenNoCache(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	resp, found := GetExactMatch(db, "gpt-4", []any{}, map[string]any{})
	if found {
		t.Error("expected found=false for empty cache")
	}
	if resp != "" {
		t.Errorf("expected empty response, got %q", resp)
	}
}

func TestExact_SaveAndRetrieve(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	messages := []any{
		map[string]any{"role": "user", "content": "Hello, world!"},
	}
	params := map[string]any{
		"temperature": 0.7,
		"max_tokens":  100,
	}
	responseBody := "Hi there! How can I help?"

	err := SaveExactMatch(db, model, messages, params, responseBody, 10, 20, 3600)
	if err != nil {
		t.Fatalf("SaveExactMatch failed: %v", err)
	}

	resp, found := GetExactMatch(db, model, messages, params)
	if !found {
		t.Fatal("expected found=true after save")
	}
	if resp != responseBody {
		t.Errorf("expected response %q, got %q", responseBody, resp)
	}
}

func TestExact_ExpiredEntryNotReturned(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	messages := []any{
		map[string]any{"role": "user", "content": "Expired test"},
	}
	params := map[string]any{"temperature": 0.5}

	// Save with TTL=1 second.
	err := SaveExactMatch(db, model, messages, params, "expired response", 10, 20, 1)
	if err != nil {
		t.Fatalf("SaveExactMatch failed: %v", err)
	}

	// Wait for expiry.
	time.Sleep(2 * time.Second)

	resp, found := GetExactMatch(db, model, messages, params)
	if found {
		t.Error("expected found=false for expired entry")
	}
	if resp != "" {
		t.Errorf("expected empty response for expired entry, got %q", resp)
	}
}

func TestExact_DifferentMessagesProduceDifferentHashes(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	params := map[string]any{"temperature": 0.7}

	messages1 := []any{
		map[string]any{"role": "user", "content": "Hello"},
	}
	messages2 := []any{
		map[string]any{"role": "user", "content": "Goodbye"},
	}

	err := SaveExactMatch(db, model, messages1, params, "response for hello", 10, 20, 3600)
	if err != nil {
		t.Fatalf("SaveExactMatch for messages1 failed: %v", err)
	}

	// messages2 should NOT match messages1.
	resp, found := GetExactMatch(db, model, messages2, params)
	if found {
		t.Errorf("expected found=false for different messages, got response=%q", resp)
	}
}

func TestExact_SameMessagesProduceSameHash(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "claude-3"
	messages := []any{
		map[string]any{"role": "system", "content": "You are helpful."},
		map[string]any{"role": "user", "content": "Hi"},
	}
	params := map[string]any{
		"temperature": 0.9,
		"max_tokens":  500,
		"top_p":       0.95,
	}

	// Save first.
	err := SaveExactMatch(db, model, messages, params, "first save", 5, 15, 3600)
	if err != nil {
		t.Fatalf("first SaveExactMatch failed: %v", err)
	}

	// Same request should retrieve the first save.
	resp, found := GetExactMatch(db, model, messages, params)
	if !found {
		t.Fatal("expected found=true for identical request")
	}
	if resp != "first save" {
		t.Errorf("expected 'first save', got %q", resp)
	}

	// Overwrite with same hash (INSERT OR REPLACE).
	err = SaveExactMatch(db, model, messages, params, "second save", 5, 15, 3600)
	if err != nil {
		t.Fatalf("second SaveExactMatch failed: %v", err)
	}

	resp2, found2 := GetExactMatch(db, model, messages, params)
	if !found2 {
		t.Fatal("expected found=true after overwrite")
	}
	if resp2 != "second save" {
		t.Errorf("expected 'second save' after overwrite, got %q", resp2)
	}
}

func TestExact_HitCountIncrements(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	messages := []any{
		map[string]any{"role": "user", "content": "Count my hits"},
	}
	params := map[string]any{}

	err := SaveExactMatch(db, model, messages, params, "cached response", 10, 20, 3600)
	if err != nil {
		t.Fatalf("SaveExactMatch failed: %v", err)
	}

	// Retrieve 3 times, ensure hit_count increments.
	for i := 0; i < 3; i++ {
		_, found := GetExactMatch(db, model, messages, params)
		if !found {
			t.Fatalf("retrieval %d: expected found=true", i+1)
		}
	}

	var hitCount int
	err = db.QueryRow("SELECT hit_count FROM response_cache WHERE hash=?",
		buildExactHash(model, messages, params)).Scan(&hitCount)
	if err != nil {
		t.Fatalf("failed to query hit_count: %v", err)
	}

	if hitCount != 3 {
		t.Errorf("expected hit_count=3, got %d", hitCount)
	}
}

func TestExact_DifferentModelsProduceDifferentHashes(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	messages := []any{
		map[string]any{"role": "user", "content": "Same message, different model"},
	}
	params := map[string]any{"temperature": 0.5}

	err := SaveExactMatch(db, "gpt-4", messages, params, "gpt-4 response", 10, 20, 3600)
	if err != nil {
		t.Fatalf("SaveExactMatch for gpt-4 failed: %v", err)
	}

	resp, found := GetExactMatch(db, "claude-3", messages, params)
	if found {
		t.Errorf("expected found=false for different model, got response=%q", resp)
	}
}

func TestExact_DifferentParamsProduceDifferentHashes(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	messages := []any{
		map[string]any{"role": "user", "content": "Test params"},
	}

	err := SaveExactMatch(db, model, messages, map[string]any{"temperature": 0.1}, "low temp", 10, 20, 3600)
	if err != nil {
		t.Fatalf("SaveExactMatch for temp=0.1 failed: %v", err)
	}

	resp, found := GetExactMatch(db, model, messages, map[string]any{"temperature": 0.9})
	if found {
		t.Errorf("expected found=false for different temperature, got response=%q", resp)
	}
}

func TestExact_ClearExpiredExact(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	messages := []any{
		map[string]any{"role": "user", "content": "Clear test"},
	}
	params := map[string]any{}

	// Save with short TTL.
	err := SaveExactMatch(db, model, messages, params, "expired soon", 1, 1, 1)
	if err != nil {
		t.Fatalf("SaveExactMatch failed: %v", err)
	}

	// Also save a long-lived entry.
	longMessages := []any{
		map[string]any{"role": "user", "content": "Long lived"},
	}
	err = SaveExactMatch(db, model, longMessages, params, "long lived", 1, 1, 86400)
	if err != nil {
		t.Fatalf("SaveExactMatch for long-lived failed: %v", err)
	}

	// Wait for the short one to expire.
	time.Sleep(2 * time.Second)

	deleted, err := ClearExpiredExact(db)
	if err != nil {
		t.Fatalf("ClearExpiredExact failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted row, got %d", deleted)
	}

	// Long-lived entry should still be retrievable.
	resp, found := GetExactMatch(db, model, longMessages, params)
	if !found {
		t.Error("expected long-lived entry to survive ClearExpiredExact")
	}
	if resp != "long lived" {
		t.Errorf("expected 'long lived', got %q", resp)
	}
}

func TestExact_SaveExactMatch_DefaultTTL(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	model := "gpt-4"
	messages := []any{
		map[string]any{"role": "user", "content": "Default TTL"},
	}
	params := map[string]any{}

	// Pass 0 for ttlSeconds — should default to 3600.
	err := SaveExactMatch(db, model, messages, params, "default ttl", 10, 20, 0)
	if err != nil {
		t.Fatalf("SaveExactMatch with ttl=0 failed: %v", err)
	}

	// Should still be retrievable.
	resp, found := GetExactMatch(db, model, messages, params)
	if !found {
		t.Fatal("expected entry with default TTL to be found")
	}
	if resp != "default ttl" {
		t.Errorf("expected 'default ttl', got %q", resp)
	}
}
