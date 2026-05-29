package logging

import (
	"strings"
	"sync"
	"time"
)

const maxLogs = 10000

// AccessLog represents a single API request log entry.
type AccessLog struct {
	Timestamp  time.Time `json:"timestamp"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	Latency    float64   `json:"latency_ms"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	TokensIn   int       `json:"tokens_in"`
	TokensOut  int       `json:"tokens_out"`
	IP         string    `json:"ip"`
}

// LogStats contains aggregate statistics about logged requests.
type LogStats struct {
	Total      int     `json:"total"`
	AvgLatency float64 `json:"avg_latency_ms"`
	ErrorRate  float64 `json:"error_rate"`
}

// LogStore is a thread-safe in-memory ring buffer for access logs.
type LogStore struct {
	mu   sync.RWMutex
	logs []AccessLog
	pos  int
	full bool
}

// NewLogStore creates a new LogStore with the default capacity (10,000).
func NewLogStore() *LogStore {
	return &LogStore{
		logs: make([]AccessLog, maxLogs),
	}
}

// Record appends a log entry to the ring buffer.
func (s *LogStore) Record(log AccessLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs[s.pos] = log
	s.pos++
	if s.pos >= maxLogs {
		s.pos = 0
		s.full = true
	}
}

// Count returns the number of stored log entries.
func (s *LogStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.full {
		return maxLogs
	}
	return s.pos
}

// Search returns log entries whose path, provider, model, method, or IP
// contain the query string (case-insensitive). If query is empty, the
// most recent `limit` entries are returned.
func (s *LogStore) Search(query string, limit int) []AccessLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	count := s.count()
	if count == 0 {
		return []AccessLog{}
	}

	// Walk newest→oldest
	results := make([]AccessLog, 0, limit)
	q := strings.ToLower(query)

	for i := 0; i < count && len(results) < limit; i++ {
		idx := s.logicalIndex(count - 1 - i) // newest first
		entry := s.logs[idx]

		if query == "" || s.matchesQuery(entry, q) {
			results = append(results, entry)
		}
	}
	return results
}

// Stats computes aggregate statistics across all stored logs.
func (s *LogStore) Stats() LogStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := s.count()
	if count == 0 {
		return LogStats{}
	}

	var totalLatency float64
	var errors int

	for i := 0; i < count; i++ {
		idx := s.logicalIndex(i)
		entry := s.logs[idx]
		totalLatency += entry.Latency
		if entry.Status >= 400 {
			errors++
		}
	}

	return LogStats{
		Total:      count,
		AvgLatency: totalLatency / float64(count),
		ErrorRate:  float64(errors) / float64(count) * 100,
	}
}

// ── helpers ────────────────────────────────────

// count returns the logical number of entries (must be called with lock held).
func (s *LogStore) count() int {
	if s.full {
		return maxLogs
	}
	return s.pos
}

// logicalIndex maps a logical index (0 = oldest) to a physical array index.
func (s *LogStore) logicalIndex(logical int) int {
	if s.full {
		return (s.pos + logical) % maxLogs
	}
	return logical
}

// matchesQuery checks if any text field contains the query string.
func (s *LogStore) matchesQuery(entry AccessLog, q string) bool {
	return strings.Contains(strings.ToLower(entry.Path), q) ||
		strings.Contains(strings.ToLower(entry.Provider), q) ||
		strings.Contains(strings.ToLower(entry.Model), q) ||
		strings.Contains(strings.ToLower(entry.Method), q) ||
		strings.Contains(strings.ToLower(entry.IP), q)
}
