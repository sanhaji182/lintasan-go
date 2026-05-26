// Package fallback provides a two-tier fallback chain engine (model + connection)
// with in-memory metrics tracking. Based on Lintasan Node.js lib/fallback-chain.js.
package fallback

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/db"
)

// TriggerReason describes why a fallback was initiated.
type TriggerReason string

const (
	ReasonTimeout TriggerReason = "timeout"
	Reason5xx     TriggerReason = "5xx"
	Reason429     TriggerReason = "429"
	Reason401     TriggerReason = "401"
	ReasonCircuit TriggerReason = "circuit_open"
)

// ValidTriggerReason returns true if the given reason is a recognized trigger.
func ValidTriggerReason(r TriggerReason) bool {
	switch r {
	case ReasonTimeout, Reason5xx, Reason429, Reason401, ReasonCircuit:
		return true
	default:
		return false
	}
}

// Event records a single fallback transition.
type Event struct {
	FromModel  string        `json:"from_model"`
	ToModel    string        `json:"to_model"`
	FromConn   string        `json:"from_conn"`
	ToConn     string        `json:"to_conn"`
	Reason     TriggerReason `json:"reason"`
	StatusCode int           `json:"status_code"`
	Timestamp  int64         `json:"timestamp"`
}

// Settings keys matching the Node.js implementation.
const (
	settingModelChains      = "fallback_model_chains"
	settingConnectionChains = "fallback_connection_chains"
)

// maxEvents is the ring buffer capacity.
const maxEvents = 100

// Engine manages model and connection fallback chains with an in-memory
// event log for metrics.
type Engine struct {
	db          *db.DB
	modelChains map[string][]string // model -> [fallback models]
	connChains  map[string][]string // conn ID -> [fallback conn IDs]
	mu          sync.RWMutex
	events      []Event // ring buffer, up to maxEvents
	head        int     // write position in ring buffer
	count       int     // total events written (for sizing the window)
}

// New creates a fallback Engine backed by the given database handle.
// Call LoadChains() afterwards to populate chains from settings.
func New(database *db.DB) *Engine {
	return &Engine{
		db:          database,
		modelChains: make(map[string][]string),
		connChains:  make(map[string][]string),
		events:      make([]Event, 0, maxEvents),
	}
}

// LoadChains reads fallback_model_chains and fallback_connection_chains from
// the settings table. Both are stored as JSON strings of map[string][]string.
// Missing keys are treated as empty maps (no-op).
func (e *Engine) LoadChains() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Model chains.
	raw, err := e.db.GetSetting(settingModelChains)
	if err != nil {
		return fmt.Errorf("fallback: read model chains: %w", err)
	}
	modelChains := make(map[string][]string)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &modelChains); err != nil {
			return fmt.Errorf("fallback: parse model chains: %w", err)
		}
	}

	// Connection chains.
	raw, err = e.db.GetSetting(settingConnectionChains)
	if err != nil {
		return fmt.Errorf("fallback: read connection chains: %w", err)
	}
	connChains := make(map[string][]string)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &connChains); err != nil {
			return fmt.Errorf("fallback: parse connection chains: %w", err)
		}
	}

	e.modelChains = modelChains
	e.connChains = connChains
	return nil
}

// GetModelFallback returns the ordered fallback model list for the given
// model. Returns nil/empty if no chain is configured.
func (e *Engine) GetModelFallback(model string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	chain, ok := e.modelChains[model]
	if !ok {
		return nil
	}
	// Return a copy to prevent mutation.
	out := make([]string, len(chain))
	copy(out, chain)
	return out
}

// GetConnFallback returns the ordered fallback connection list for the given
// connection ID. Returns nil/empty if no chain is configured.
func (e *Engine) GetConnFallback(connID string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	chain, ok := e.connChains[connID]
	if !ok {
		return nil
	}
	out := make([]string, len(chain))
	copy(out, chain)
	return out
}

// RecordEvent logs a fallback event into the in-memory ring buffer.
// The event is stored with a Unix timestamp (seconds).
func (e *Engine) RecordEvent(from, to string, reason TriggerReason, statusCode int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ev := Event{
		FromModel:  from,
		ToModel:    to,
		Reason:     reason,
		StatusCode: statusCode,
		Timestamp:  time.Now().Unix(),
	}

	if len(e.events) < maxEvents {
		e.events = append(e.events, ev)
	} else {
		e.events[e.head] = ev
	}
	e.head = (e.head + 1) % maxEvents
	e.count++
}

// GetRecentEvents returns the most recent N fallback events, in chronological
// order (oldest first). If N exceeds the stored events, all events are returned.
func (e *Engine) GetRecentEvents(n int) []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	available := len(e.events)
	if n > available {
		n = available
	}
	if n <= 0 {
		return nil
	}

	// The ring buffer wraps at maxEvents. Determine the contiguous
	// window of the last n events.
	var out []Event
	if len(e.events) < maxEvents {
		// Buffer hasn't wrapped yet — simple slice from the end.
		out = make([]Event, n)
		copy(out, e.events[available-n:])
	} else {
		// Buffer has wrapped. head points to the next write slot,
		// so the oldest event is at head (if wrapped), and the newest
		// is at (head-1+maxEvents)%maxEvents.
		out = make([]Event, n)
		start := (e.head - n + maxEvents) % maxEvents
		for i := 0; i < n; i++ {
			out[i] = e.events[(start+i)%maxEvents]
		}
	}
	return out
}

// Stats returns a summary map suitable for serialisation:
//
//	{
//	  "total": N,
//	  "by_reason": {"timeout": 3, "5xx": 1, ...},
//	  "top_pairs": [{"from": ..., "to": ..., "count": ...}, ...],
//	}
func (e *Engine) Stats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	events := e.allEventsLocked()

	result := map[string]interface{}{
		"total":        len(events),
		"by_reason":    make(map[TriggerReason]int),
		"top_pairs":    []map[string]interface{}{},
		"top_conn_pairs": []map[string]interface{}{},
	}

	byReason := result["by_reason"].(map[TriggerReason]int)
	modelPairCounts := map[pairKey]int{}
	connPairCounts := map[pairKey]int{}

	for _, ev := range events {
		byReason[ev.Reason]++

		if ev.FromModel != "" || ev.ToModel != "" {
			modelPairCounts[pairKey{ev.FromModel, ev.ToModel}]++
		}
		if ev.FromConn != "" || ev.ToConn != "" {
			connPairCounts[pairKey{ev.FromConn, ev.ToConn}]++
		}
	}

	// Top model pairs (up to 10).
	topModel := topPairs(modelPairCounts, 10)
	result["top_pairs"] = topModel

	// Top connection pairs (up to 10).
	topConn := topPairs(connPairCounts, 10)
	result["top_conn_pairs"] = topConn

	return result
}

// pairKey is a composite key for aggregating (from, to) pairs.
type pairKey struct {
	from, to string
}

// topPairs returns the top-N (from, to) pairs by count, descending.
func topPairs(counts map[pairKey]int, n int) []map[string]interface{} {
	type entry struct {
		k pairKey
		v int
	}
	list := make([]entry, 0, len(counts))
	for k, v := range counts {
		list = append(list, entry{k, v})
	}
	// Sort descending by count.
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].v > list[i].v {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	if n > len(list) {
		n = len(list)
	}
	out := make([]map[string]interface{}, n)
	for i := 0; i < n; i++ {
		out[i] = map[string]interface{}{
			"from":  list[i].k.from,
			"to":    list[i].k.to,
			"count": list[i].v,
		}
	}
	return out
}

// allEventsLocked returns all events in chronological order.
// Caller must hold e.mu (read lock is sufficient).
func (e *Engine) allEventsLocked() []Event {
	if len(e.events) == 0 {
		return nil
	}
	if len(e.events) < maxEvents {
		cp := make([]Event, len(e.events))
		copy(cp, e.events)
		return cp
	}
	// Wrapped buffer.
	out := make([]Event, maxEvents)
	for i := 0; i < maxEvents; i++ {
		out[i] = e.events[(e.head+i)%maxEvents]
	}
	return out
}

// ShouldTriggerFallback determines whether a given error/status should
// trigger a fallback. Mirrors the Node.js shouldTriggerFallback().
//
//   - circuitOpen: circuit breaker is open → ReasonCircuit
//   - status 429 → Reason429
//   - status >= 500 → Reason5xx
//   - isTimeout → ReasonTimeout
//
// Returns the reason and whether a fallback should be attempted.
func ShouldTriggerFallback(statusCode int, isTimeout bool, circuitOpen bool) (bool, TriggerReason) {
	if circuitOpen {
		return true, ReasonCircuit
	}
	if isTimeout {
		return true, ReasonTimeout
	}
	if statusCode == 429 {
		return true, Reason429
	}
	if statusCode == 401 {
		return true, Reason401
	}
	if statusCode >= 500 && statusCode < 600 {
		return true, Reason5xx
	}
	return false, ""
}
