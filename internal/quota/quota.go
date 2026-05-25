package quota

import (
	"database/sql"
	"time"
)

func InitQuotaSchema(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS quota_usage (
		connection_id TEXT PRIMARY KEY,
		tokens_today INTEGER DEFAULT 0,
		tokens_month INTEGER DEFAULT 0,
		requests_today INTEGER DEFAULT 0,
		requests_month INTEGER DEFAULT 0,
		last_reset_day TEXT,
		last_reset_month TEXT,
		updated_at TEXT DEFAULT (datetime('now'))
	)`)
}

func RecordQuota(db *sql.DB, connID string, tokens int) {
	day := time.Now().Format("2006-01-02")
	month := time.Now().Format("2006-01")
	
	res, _ := db.Exec(`
		UPDATE quota_usage 
		SET tokens_today=tokens_today+?, tokens_month=tokens_month+?, requests_today=requests_today+1, requests_month=requests_month+1, updated_at=datetime('now')
		WHERE connection_id=? AND last_reset_day=? AND last_reset_month=?
	`, tokens, tokens, connID, day, month)
	
	if n, _ := res.RowsAffected(); n == 0 {
		db.Exec(`
			INSERT OR REPLACE INTO quota_usage(connection_id, tokens_today, tokens_month, requests_today, requests_month, last_reset_day, last_reset_month)
			VALUES(?, ?, ?, 1, 1, ?, ?)
		`, connID, tokens, tokens, day, month)
	}
}

func GetQuota(db *sql.DB, connID string) map[string]any {
	day := time.Now().Format("2006-01-02")
	month := time.Now().Format("2006-01")
	var tToday, tMonth, rToday, rMonth int
	var lDay, lMonth string
	err := db.QueryRow("SELECT tokens_today, tokens_month, requests_today, requests_month, last_reset_day, last_reset_month FROM quota_usage WHERE connection_id=?", connID).Scan(&tToday, &tMonth, &rToday, &rMonth, &lDay, &lMonth)
	if err != nil { return map[string]any{"requests_today":0, "tokens_today":0} }
	
	if lDay != day { tToday = 0; rToday = 0 }
	if lMonth != month { tMonth = 0; rMonth = 0 }
	return map[string]any{
		"tokens_today": tToday, "tokens_month": tMonth,
		"requests_today": rToday, "requests_month": rMonth,
	}
}
