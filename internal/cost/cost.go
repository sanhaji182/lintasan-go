package cost

import (
	"database/sql"
	"sync"
	"time"
)

// Entry represents a single cost-tracking record.
type Entry struct {
	Timestamp       time.Time `json:"timestamp"`
	Model           string    `json:"model"`
	ConnectionID    string    `json:"connection_id"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	InputCostUSD    float64   `json:"input_cost_usd"`
	OutputCostUSD   float64   `json:"output_cost_usd"`
}

// Summary aggregates costs over a time period.
type Summary struct {
	TotalRequests   int     `json:"total_requests"`
	TotalInputTokens  int   `json:"total_input_tokens"`
	TotalOutputTokens int   `json:"total_output_tokens"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	ByModel         map[string]*ModelSummary `json:"by_model"`
	ByConnection    map[string]*ConnSummary  `json:"by_connection"`
}

// ModelSummary aggregates per-model costs.
type ModelSummary struct {
	Requests      int     `json:"requests"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	CostUSD       float64 `json:"cost_usd"`
}

// ConnSummary aggregates per-connection costs.
type ConnSummary struct {
	Requests int     `json:"requests"`
	CostUSD  float64 `json:"cost_usd"`
}

// Tracker records and queries cost data.
type Tracker struct {
	db       *sql.DB
	mu       sync.RWMutex
	entries  []Entry
	pricing  map[string]ModelPrice // model_id -> pricing
}

// ModelPrice holds per-token pricing for a model.
type ModelPrice struct {
	InputPrice  float64 // USD per 1M input tokens
	OutputPrice float64 // USD per 1M output tokens
}

// New creates a new cost Tracker.
func New(database *sql.DB) *Tracker {
	t := &Tracker{
		db:      database,
		pricing: make(map[string]ModelPrice),
	}
	t.initSchema()
	t.loadPricing()
	return t
}

func (t *Tracker) initSchema() {
	t.db.Exec(`CREATE TABLE IF NOT EXISTS cost_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		model TEXT NOT NULL,
		connection_id TEXT NOT NULL,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		input_cost_usd REAL DEFAULT 0,
		output_cost_usd REAL DEFAULT 0
	)`)
}

func (t *Tracker) loadPricing() {
	// Default pricing from catalog
	defaults := map[string]ModelPrice{
		"gpt-4o":             {2.50, 10.00},
		"gpt-4o-mini":        {0.15, 0.60},
		"claude-sonnet-4-20250514": {3.00, 15.00},
		"claude-opus-4-20250514":   {15.00, 75.00},
		"claude-haiku-3-5":   {0.80, 4.00},
		"deepseek-v4-pro":    {0.55, 2.20},
		"deepseek-v3":        {0.27, 1.10},
		"gemini-2.5-pro":     {1.25, 10.00},
		"gemini-2.5-flash":   {0.15, 0.60},
	}
	for k, v := range defaults {
		if _, ok := t.pricing[k]; !ok {
			t.pricing[k] = v
		}
	}
}

// RegisterPricing adds custom model pricing.
func (t *Tracker) RegisterPricing(modelID string, price ModelPrice) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pricing[modelID] = price
}

// Record logs a cost entry.
func (t *Tracker) Record(model, connID string, inputTokens, outputTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	price, ok := t.pricing[model]
	if !ok {
		price = ModelPrice{0, 0}
	}
	inputCost := float64(inputTokens) / 1_000_000 * price.InputPrice
	outputCost := float64(outputTokens) / 1_000_000 * price.OutputPrice

	now := time.Now().UTC().Format(time.RFC3339)
	t.db.Exec(
		"INSERT INTO cost_entries (timestamp, model, connection_id, input_tokens, output_tokens, input_cost_usd, output_cost_usd) VALUES (?,?,?,?,?,?,?)",
		now, model, connID, inputTokens, outputTokens, inputCost, outputCost,
	)
}

// Summary returns aggregated costs for today.
func (t *Tracker) Summary() *Summary {
	rows, err := t.db.Query(`SELECT model, connection_id, COUNT(*) as requests,
		SUM(input_tokens), SUM(output_tokens),
		SUM(input_cost_usd), SUM(output_cost_usd)
		FROM cost_entries WHERE date(timestamp) = date('now')
		GROUP BY model, connection_id`)
	if err != nil {
		return &Summary{ByModel: map[string]*ModelSummary{}, ByConnection: map[string]*ConnSummary{}}
	}
	defer rows.Close()

	s := &Summary{
		ByModel:      make(map[string]*ModelSummary),
		ByConnection: make(map[string]*ConnSummary),
	}

	for rows.Next() {
		var model, connID string
		var reqs, inT, outT int
		var inC, outC float64
		rows.Scan(&model, &connID, &reqs, &inT, &outT, &inC, &outC)

		s.TotalRequests += reqs
		s.TotalInputTokens += inT
		s.TotalOutputTokens += outT
		s.TotalCostUSD += inC + outC

		ms, ok := s.ByModel[model]
		if !ok {
			ms = &ModelSummary{}
			s.ByModel[model] = ms
		}
		ms.Requests += reqs
		ms.InputTokens += inT
		ms.OutputTokens += outT
		ms.CostUSD += inC + outC

		cs, ok := s.ByConnection[connID]
		if !ok {
			cs = &ConnSummary{}
			s.ByConnection[connID] = cs
		}
		cs.Requests += reqs
		cs.CostUSD += inC + outC
	}
	return s
}

// SummarySince returns aggregated costs since a given time.
func (t *Tracker) SummarySince(since time.Duration) *Summary {
	cutoff := time.Now().Add(-since).UTC().Format(time.RFC3339)
	rows, err := t.db.Query(`SELECT model, connection_id, COUNT(*) as requests,
		SUM(input_tokens), SUM(output_tokens),
		SUM(input_cost_usd), SUM(output_cost_usd)
		FROM cost_entries WHERE timestamp >= ?
		GROUP BY model, connection_id`, cutoff)
	if err != nil {
		return &Summary{ByModel: map[string]*ModelSummary{}, ByConnection: map[string]*ConnSummary{}}
	}
	defer rows.Close()

	s := &Summary{
		ByModel:      make(map[string]*ModelSummary),
		ByConnection: make(map[string]*ConnSummary),
	}

	for rows.Next() {
		var model, connID string
		var reqs, inT, outT int
		var inC, outC float64
		rows.Scan(&model, &connID, &reqs, &inT, &outT, &inC, &outC)

		s.TotalRequests += reqs
		s.TotalInputTokens += inT
		s.TotalOutputTokens += outT
		s.TotalCostUSD += inC + outC

		ms, ok := s.ByModel[model]
		if !ok {
			ms = &ModelSummary{}
			s.ByModel[model] = ms
		}
		ms.Requests += reqs
		ms.InputTokens += inT
		ms.OutputTokens += outT
		ms.CostUSD += inC + outC

		cs, ok := s.ByConnection[connID]
		if !ok {
			cs = &ConnSummary{}
			s.ByConnection[connID] = cs
		}
		cs.Requests += reqs
		cs.CostUSD += inC + outC
	}
	return s
}
