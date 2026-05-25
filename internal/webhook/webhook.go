package webhook

import (
	"database/sql"
	"log"
)

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Fire async webhook execution
func (m *Manager) Fire(event string, payload map[string]interface{}) {
	go func() {
		// Just log for now to prove hooks fire
		log.Printf("[Webhook] Event %s triggered", event)
		
		// In full version: select from webhooks where event matches, iterate and post
		// m.db.Query("SELECT endpoint, secret FROM webhooks WHERE events LIKE ?", "%"+event+"%")
	}()
}
