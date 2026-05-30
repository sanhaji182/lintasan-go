package cache

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// H2: compare per-request DB costs head-to-head on a single shared connection,
// the way the proxy actually runs them. Answers "is sync logging more expensive
// than the semantic cache scan?"
func openSharedDB(b *testing.B) *sql.DB {
	if _, err := os.Stat("/tmp/h2test.db"); err != nil {
		b.Skip("benchmark fixture /tmp/h2test.db absent (local perf bench only)")
	}
	db, err := sql.Open("sqlite3", "/tmp/h2test.db?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxOpenConns(1) // match production
	return db
}

// Synchronous logging INSERT (the hot-path write).
func BenchmarkLogInsert(b *testing.B) {
	db := openSharedDB(b)
	defer db.Close()
	exp := time.Now().Format("2006-01-02 15:04:05")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Exec(`INSERT INTO request_logs (id, connection_id, provider, model, status, input_tokens, output_tokens, latency_ms, cached, error, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("bench-%d-%d", i, time.Now().UnixNano()), "c1", "mock", "mock-model", 200, 20, 12, 50, 0, "", exp)
	}
}

// The settings SELECT storm: ~6-10 GetSetting calls per request.
func BenchmarkSettingsStorm(b *testing.B) {
	db := openSharedDB(b)
	defer db.Close()
	keys := []string{"exact_cache_enabled", "stream_cache_enabled", "semantic_cache_enabled",
		"prompt_optimizer_enabled", "ml_router_enabled", "quota_enabled", "combos", "cost_quality_floor"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, k := range keys {
			var v string
			db.QueryRow("SELECT value FROM settings WHERE key = ?", k).Scan(&v)
		}
	}
}
