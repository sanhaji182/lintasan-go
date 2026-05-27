package cost

import (
	"database/sql"
	"testing"
	"time"
	_ "github.com/mattn/go-sqlite3"
)

func setupDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestNew(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	if tr == nil {
		t.Fatal("New returned nil")
	}
}

func TestRecord(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	tr.Record("gpt-4o", "conn-1", 1000, 500)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM cost_entries").Scan(&count)
	if count != 1 {
		t.Errorf("entries = %d, want 1", count)
	}
}

func TestRegisterPricing(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	tr.RegisterPricing("custom-model", ModelPrice{5.00, 20.00})

	price, ok := tr.pricing["custom-model"]
	if !ok {
		t.Fatal("custom-model not found in pricing")
	}
	if price.InputPrice != 5.00 {
		t.Errorf("inputPrice = %f, want 5.00", price.InputPrice)
	}
}

func TestSummary(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	tr.RegisterPricing("gpt-4o", ModelPrice{2.50, 10.00})
	tr.Record("gpt-4o", "conn-1", 10000, 5000)
	tr.Record("gpt-4o", "conn-1", 5000, 3000)

	s := tr.Summary()
	if s.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", s.TotalRequests)
	}
	if s.TotalInputTokens != 15000 {
		t.Errorf("TotalInputTokens = %d, want 15000", s.TotalInputTokens)
	}
	if s.TotalOutputTokens != 8000 {
		t.Errorf("TotalOutputTokens = %d, want 8000", s.TotalOutputTokens)
	}
	if len(s.ByModel) == 0 {
		t.Error("ByModel should have entries")
	}
}

func TestSummarySince(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	tr.RegisterPricing("gpt-4o", ModelPrice{2.50, 10.00})
	tr.Record("gpt-4o", "conn-1", 1000, 500)

	s := tr.SummarySince(1 * time.Hour)
	if s.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", s.TotalRequests)
	}

	sOld := tr.SummarySince(1 * time.Microsecond)
	if sOld.TotalRequests != 1 {
		t.Fatal("SummarySince with microsecond cutoff should include everything")
	}
}

func TestDefaultPricing(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	models := []string{"gpt-4o", "gpt-4o-mini", "deepseek-v4-pro", "claude-sonnet-4-20250514", "gemini-2.5-pro"}
	for _, m := range models {
		if _, ok := tr.pricing[m]; !ok {
			t.Errorf("default pricing missing for %s", m)
		}
	}
}

func TestEmptySummary(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	tr := New(db)
	s := tr.Summary()
	if s.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", s.TotalRequests)
	}
	if len(s.ByModel) != 0 {
		t.Error("ByModel should be empty")
	}
}
