package fallback

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/sanhaji182/lintasan-go/internal/db"
)

// newTestDB creates an in-memory SQLite database for tests.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// seedChains writes model and connection chain settings into the DB.
func seedChains(t *testing.T, database *db.DB, modelChains, connChains map[string][]string) {
	t.Helper()
	if modelChains != nil {
		b, err := json.Marshal(modelChains)
		if err != nil {
			t.Fatalf("marshal model chains: %v", err)
		}
		if err := database.SetSetting(settingModelChains, string(b)); err != nil {
			t.Fatalf("set model chains: %v", err)
		}
	}
	if connChains != nil {
		b, err := json.Marshal(connChains)
		if err != nil {
			t.Fatalf("marshal conn chains: %v", err)
		}
		if err := database.SetSetting(settingConnectionChains, string(b)); err != nil {
			t.Fatalf("set conn chains: %v", err)
		}
	}
}

func TestNew(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.db != database {
		t.Fatal("engine.db should be the passed database")
	}
}

// === LoadChains ===

func TestLoadChains_EmptySettings(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	if err := engine.LoadChains(); err != nil {
		t.Fatalf("LoadChains failed: %v", err)
	}

	if len(engine.GetModelFallback("anything")) != 0 {
		t.Error("expected empty fallback for unknown model")
	}
	if len(engine.GetConnFallback("any-conn")) != 0 {
		t.Error("expected empty fallback for unknown connection")
	}
}

func TestLoadChains_WithModelChains(t *testing.T) {
	database := newTestDB(t)
	seedChains(t, database, map[string][]string{
		"deepseek-v4-pro":   {"deepseek-v4-flash", "gpt-4o"},
		"claude-sonnet":     {"claude-haiku"},
	}, nil)

	engine := New(database)
	if err := engine.LoadChains(); err != nil {
		t.Fatalf("LoadChains failed: %v", err)
	}

	deep := engine.GetModelFallback("deepseek-v4-pro")
	if len(deep) != 2 || deep[0] != "deepseek-v4-flash" || deep[1] != "gpt-4o" {
		t.Errorf("unexpected deepseek fallback: %v", deep)
	}

	sonnet := engine.GetModelFallback("claude-sonnet")
	if len(sonnet) != 1 || sonnet[0] != "claude-haiku" {
		t.Errorf("unexpected claude fallback: %v", sonnet)
	}

	// Unknown model returns empty.
	if len(engine.GetModelFallback("unknown")) != 0 {
		t.Error("expected empty for unknown model")
	}
}

func TestLoadChains_WithConnChains(t *testing.T) {
	database := newTestDB(t)
	seedChains(t, database, nil, map[string][]string{
		"conn-primary":    {"conn-secondary", "conn-tertiary"},
		"conn-europe":     {"conn-us"},
	})

	engine := New(database)
	if err := engine.LoadChains(); err != nil {
		t.Fatalf("LoadChains failed: %v", err)
	}

	primary := engine.GetConnFallback("conn-primary")
	if len(primary) != 2 || primary[0] != "conn-secondary" || primary[1] != "conn-tertiary" {
		t.Errorf("unexpected conn-primary fallback: %v", primary)
	}

	europe := engine.GetConnFallback("conn-europe")
	if len(europe) != 1 || europe[0] != "conn-us" {
		t.Errorf("unexpected conn-europe fallback: %v", europe)
	}

	if len(engine.GetConnFallback("unknown")) != 0 {
		t.Error("expected empty for unknown connection")
	}
}

// === RecordEvent & GetRecentEvents ===

func TestRecordEvent_Single(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	engine.RecordEvent("deepseek-v4-pro", "gpt-4o", ReasonTimeout, 0)

	events := engine.GetRecentEvents(10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].FromModel != "deepseek-v4-pro" {
		t.Errorf("expected from deepseek-v4-pro, got %s", events[0].FromModel)
	}
	if events[0].ToModel != "gpt-4o" {
		t.Errorf("expected to gpt-4o, got %s", events[0].ToModel)
	}
	if events[0].Reason != ReasonTimeout {
		t.Errorf("expected reason timeout, got %s", events[0].Reason)
	}
	if events[0].Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
}

func TestGetRecentEvents_Limit(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	// Record 5 events.
	for i := 0; i < 5; i++ {
		engine.RecordEvent("from", "to", Reason5xx, 502)
	}

	// Ask for 3.
	events := engine.GetRecentEvents(3)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Ask for more than available.
	events = engine.GetRecentEvents(100)
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
}

func TestGetRecentEvents_Empty(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	events := engine.GetRecentEvents(10)
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestRecordEvent_RingBufferWrap(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	// Write more than maxEvents (100) to force the ring buffer to wrap.
	total := maxEvents + 50
	for i := 0; i < total; i++ {
		engine.RecordEvent("from", "to", Reason429, int(429))
	}

	events := engine.GetRecentEvents(maxEvents)
	if len(events) != maxEvents {
		t.Fatalf("expected %d events (capped), got %d", maxEvents, len(events))
	}
	// All should have the correct values.
	for _, ev := range events {
		if ev.Reason != Reason429 {
			t.Errorf("expected reason 429, got %s", ev.Reason)
		}
	}
}

// === Stats ===

func TestStats_Empty(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	stats := engine.Stats()
	if stats["total"].(int) != 0 {
		t.Errorf("expected total 0, got %v", stats["total"])
	}
	byReason := stats["by_reason"].(map[TriggerReason]int)
	if len(byReason) != 0 {
		t.Errorf("expected empty by_reason, got %v", byReason)
	}
}

func TestStats_WithEvents(t *testing.T) {
	database := newTestDB(t)
	engine := New(database)

	// Mix of reasons.
	engine.RecordEvent("a", "b", ReasonTimeout, 0)
	engine.RecordEvent("a", "b", Reason5xx, 502)
	engine.RecordEvent("a", "c", ReasonTimeout, 0)
	engine.RecordEvent("x", "y", Reason429, 429)
	engine.RecordEvent("x", "y", Reason429, 429)

	stats := engine.Stats()
	if stats["total"].(int) != 5 {
		t.Errorf("expected total 5, got %v", stats["total"])
	}

	byReason := stats["by_reason"].(map[TriggerReason]int)
	if byReason[ReasonTimeout] != 2 {
		t.Errorf("expected 2 timeouts, got %d", byReason[ReasonTimeout])
	}
	if byReason[Reason5xx] != 1 {
		t.Errorf("expected 1 5xx, got %d", byReason[Reason5xx])
	}
	if byReason[Reason429] != 2 {
		t.Errorf("expected 2 429s, got %d", byReason[Reason429])
	}

	// Top pairs: a->b (2), x->y (2), a->c (1)
	topPairs := stats["top_pairs"].([]map[string]interface{})
	if len(topPairs) != 3 {
		t.Errorf("expected 3 top pairs, got %d: %v", len(topPairs), topPairs)
	}
}

// === ShouldTriggerFallback ===

func TestShouldTriggerFallback_CircuitOpen(t *testing.T) {
	should, reason := ShouldTriggerFallback(0, false, true)
	if !should {
		t.Error("expected fallback when circuit open")
	}
	if reason != ReasonCircuit {
		t.Errorf("expected circuit_open, got %s", reason)
	}
}

func TestShouldTriggerFallback_Timeout(t *testing.T) {
	should, reason := ShouldTriggerFallback(0, true, false)
	if !should {
		t.Error("expected fallback on timeout")
	}
	if reason != ReasonTimeout {
		t.Errorf("expected timeout, got %s", reason)
	}
}

func TestShouldTriggerFallback_429(t *testing.T) {
	should, reason := ShouldTriggerFallback(429, false, false)
	if !should {
		t.Error("expected fallback on 429")
	}
	if reason != Reason429 {
		t.Errorf("expected 429, got %s", reason)
	}
}

func TestShouldTriggerFallback_401(t *testing.T) {
	should, reason := ShouldTriggerFallback(401, false, false)
	if !should {
		t.Error("expected fallback on 401")
	}
	if reason != Reason401 {
		t.Errorf("expected 401, got %s", reason)
	}
}

func TestShouldTriggerFallback_500(t *testing.T) {
	should, reason := ShouldTriggerFallback(500, false, false)
	if !should {
		t.Error("expected fallback on 500")
	}
	if reason != Reason5xx {
		t.Errorf("expected 5xx, got %s", reason)
	}
}

func TestShouldTriggerFallback_502(t *testing.T) {
	should, reason := ShouldTriggerFallback(502, false, false)
	if !should {
		t.Error("expected fallback on 502")
	}
	if reason != Reason5xx {
		t.Errorf("expected 5xx, got %s", reason)
	}
}

func TestShouldTriggerFallback_200_NoFallback(t *testing.T) {
	should, _ := ShouldTriggerFallback(200, false, false)
	if should {
		t.Error("expected no fallback on 200")
	}
}

func TestShouldTriggerFallback_404_NoFallback(t *testing.T) {
	should, _ := ShouldTriggerFallback(404, false, false)
	if should {
		t.Error("expected no fallback on 404")
	}
}

// === ValidTriggerReason ===

func TestValidTriggerReason(t *testing.T) {
	if !ValidTriggerReason(ReasonTimeout) {
		t.Error("timeout should be valid")
	}
	if !ValidTriggerReason(Reason5xx) {
		t.Error("5xx should be valid")
	}
	if !ValidTriggerReason(Reason429) {
		t.Error("429 should be valid")
	}
	if !ValidTriggerReason(ReasonCircuit) {
		t.Error("circuit_open should be valid")
	}
	if ValidTriggerReason("bogus") {
		t.Error("bogus should not be valid")
	}
}

// === Mutation safety ===

func TestGetModelFallback_ReturnsCopy(t *testing.T) {
	database := newTestDB(t)
	seedChains(t, database, map[string][]string{
		"m": {"fb1", "fb2"},
	}, nil)

	engine := New(database)
	if err := engine.LoadChains(); err != nil {
		t.Fatalf("LoadChains: %v", err)
	}

	fallbacks := engine.GetModelFallback("m")
	fallbacks[0] = "corrupted"

	// Original chain should be unaffected.
	fresh := engine.GetModelFallback("m")
	if fresh[0] != "fb1" {
		t.Errorf("expected fb1, got %s — chain was mutated", fresh[0])
	}
}

func TestGetConnFallback_ReturnsCopy(t *testing.T) {
	database := newTestDB(t)
	seedChains(t, database, nil, map[string][]string{
		"c": {"cb1", "cb2"},
	})

	engine := New(database)
	if err := engine.LoadChains(); err != nil {
		t.Fatalf("LoadChains: %v", err)
	}

	fallbacks := engine.GetConnFallback("c")
	fallbacks[0] = "corrupted"

	fresh := engine.GetConnFallback("c")
	if fresh[0] != "cb1" {
		t.Errorf("expected cb1, got %s — chain was mutated", fresh[0])
	}
}

// ensure the test db can be closed and cleaned up
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
