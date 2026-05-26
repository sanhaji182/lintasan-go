package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Manager struct {
	db *sql.DB
}

type Webhook struct {
	ID        string
	Endpoint  string
	Secret    string
	Events    string // comma-separated: request.success,request.error
	Active    bool
	CreatedAt string
}

type Delivery struct {
	ID        string
	WebhookID string
	Event     string
	Status    int // 0=pending, 1=success, 2=failed
	Response  string
	CreatedAt string
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// EnsureTables creates the webhooks table if it doesn't exist
func (m *Manager) EnsureTables() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id TEXT PRIMARY KEY,
			endpoint TEXT NOT NULL,
			secret TEXT DEFAULT '',
			events TEXT NOT NULL,
			active INTEGER DEFAULT 1,
			created_at TEXT DEFAULT (datetime('now'))
		)
	`)
	return err
}

// Fire triggers async webhook delivery for matching subscriptions
func (m *Manager) Fire(event string, payload map[string]interface{}) {
	go m.deliver(event, payload)
}

func (m *Manager) deliver(event string, payload map[string]interface{}) {
	rows, err := m.db.Query(`
		SELECT id, endpoint, secret, events FROM webhooks
		WHERE active = 1 AND events LIKE ?
	`, "%"+event+"%")
	if err != nil {
		log.Printf("[Webhook] Failed to query webhooks: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var wh Webhook
		if err := rows.Scan(&wh.ID, &wh.Endpoint, &wh.Secret, &wh.Events); err != nil {
			continue
		}
		go m.deliverOne(wh, event, payload)
	}
}

func (m *Manager) deliverOne(wh Webhook, event string, payload map[string]interface{}) {
	body, _ := json.Marshal(map[string]interface{}{
		"event":   event,
		"payload": payload,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})

	req, _ := http.NewRequest("POST", wh.Endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-Time", fmt.Sprintf("%d", time.Now().Unix()))

	// Sign with HMAC-SHA256 if secret is set
	if wh.Secret != "" {
		h := hmac.New(sha256.New, []byte(wh.Secret))
		h.Write(body)
		req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(h.Sum(nil)))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.logDelivery(wh.ID, event, 2, err.Error())
		log.Printf("[Webhook] Delivery failed for %s: %v", wh.Endpoint, err)
		return
	}
	defer resp.Body.Close()

	// Read response body for logging (limited)
	respBody := make([]byte, 1024)
	n, _ := resp.Body.Read(respBody)
	respStr := string(respBody[:n])
	if n >= 1024 {
		respStr += "..."
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logDelivery(wh.ID, event, 1, respStr)
		log.Printf("[Webhook] Delivered %s to %s (status %d)", event, wh.Endpoint, resp.StatusCode)
	} else {
		m.logDelivery(wh.ID, event, 2, respStr)
		log.Printf("[Webhook] Delivery failed %s to %s (status %d): %s", event, wh.Endpoint, resp.StatusCode, respStr)
	}
}

func (m *Manager) logDelivery(webhookID, event string, status int, response string) {
	id := fmt.Sprintf("del_%d", time.Now().UnixNano())
	m.db.Exec(`
		INSERT INTO webhook_deliveries (id, webhook_id, event, status, response, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
	`, id, webhookID, event, status, response)
}
