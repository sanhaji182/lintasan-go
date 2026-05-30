package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// BenchmarkSemanticScan measures GetSemanticMatch cost as the cache grows.
// This isolates the linear-scan cost (H3) with no HTTP / upstream noise.
func seedSemantic(b *testing.B, db *sql.DB, n int) {
	db.Exec("DELETE FROM semantic_cache")
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("INSERT INTO semantic_cache(model,fingerprint,messages_hash,response,expires_at) VALUES(?,?,?,?,?)")
	exp := time.Now().Add(time.Hour).Format("2006-01-02 15:04:05")
	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < n; i++ {
		tf := map[string]int{}
		for j := 0; j < 40; j++ {
			w := make([]byte, 6)
			for k := range w {
				w[k] = letters[rand.Intn(26)]
			}
			tf[string(w)] = rand.Intn(5) + 1
		}
		fp, _ := json.Marshal(tf)
		stmt.Exec("mock-model", string(fp), fmt.Sprintf("h%d", i), fmt.Sprintf("resp %d", i), exp)
	}
	tx.Commit()
}

func BenchmarkSemanticScan(b *testing.B) {
	if _, err := os.Stat("/tmp/bench-data2/lintasan.db"); err != nil {
		b.Skip("benchmark fixture /tmp/bench-data2 absent (local perf bench only)")
	}
	db, err := sql.Open("sqlite3", "/tmp/bench-data2/lintasan.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Realistic query message (won't match → forces full scan + cosine on every row)
	msgs := []any{
		map[string]any{"role": "user", "content": "explain the tradeoffs of single connection pooling in sqlite under concurrent read load"},
	}

	for _, n := range []int{100, 1000, 5000, 20000, 50000} {
		seedSemantic(b, db, n)
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				GetSemanticMatch(db, "mock-model", msgs, 0.75)
			}
		})
	}
}
