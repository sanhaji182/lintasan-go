package combo

import (
	"math"
	"sort"
	"sync"
	"time"
)

// AutoMode defines the auto-routing optimization target.
type AutoMode string

const (
	AutoModeDefault AutoMode = "auto"
	AutoModeCoding  AutoMode = "auto/coding"
	AutoModeFast    AutoMode = "auto/fast"
	AutoModeCheap   AutoMode = "auto/cheap"
)

// Provider represents a connected provider with runtime metrics for auto-routing.
type Provider struct {
	ID             string
	Name           string
	Model          string
	ConnectionID   string
	APIKey         string
	Health         int       // 0-100 health score
	QuotaRemaining float64   // 0.0-1.0 fraction of quota remaining
	CostPerToken   float64   // cost per 1K tokens (input+output average)
	Latency        float64   // EWMA latency in ms
	SuccessRate    float64   // 0.0-1.0 success rate
	LastUsed       time.Time // last time this provider was used
	Capabilities   []string  // "chat", "vision", "tools", "coding", etc.
}

// autoEngine manages auto-combo resolution with LGP sticky state.
type autoEngine struct {
	mu       sync.RWMutex
	lastGood map[string]string // mode → provider ID of last successful provider
}

var defaultAutoEngine = &autoEngine{
	lastGood: make(map[string]string),
}

// AutoAliasExists returns true if the given model string is a recognized auto alias.
func AutoAliasExists(model string) bool {
	switch AutoMode(model) {
	case AutoModeDefault, AutoModeCoding, AutoModeFast, AutoModeCheap:
		return true
	}
	return false
}

// ResolveAuto resolves an auto model alias to a sorted list of providers.
// Providers are scored on 6 factors and sorted by composite score (highest first).
// The Last-Good-Provider (LGP) gets a bonus to prefer it.
func ResolveAuto(mode string, providers []Provider) []Provider {
	return defaultAutoEngine.resolve(AutoMode(mode), providers)
}

func (ae *autoEngine) resolve(mode AutoMode, providers []Provider) []Provider {
	if len(providers) == 0 {
		return nil
	}

	weights := modeWeights(mode)

	// Get LGP for this mode
	ae.mu.RLock()
	lgpID := ae.lastGood[string(mode)]
	ae.mu.RUnlock()

	type scored struct {
		provider Provider
		score    float64
	}

	scoredList := make([]scored, 0, len(providers))
	for _, p := range providers {
		s := scoreProvider(p, weights)
		// LGP bonus: +15% score for last-good-provider
		if lgpID != "" && p.ID == lgpID {
			s *= 1.15
		}
		scoredList = append(scoredList, scored{provider: p, score: s})
	}

	// Sort by score descending
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	result := make([]Provider, len(scoredList))
	for i, s := range scoredList {
		result[i] = s.provider
	}
	return result
}

// RecordAutoSuccess records that a provider succeeded for auto/LGP tracking.
func RecordAutoSuccess(mode string, providerID string) {
	defaultAutoEngine.mu.Lock()
	defaultAutoEngine.lastGood[mode] = providerID
	defaultAutoEngine.mu.Unlock()
}

// modeWeights returns factor weights for each auto mode.
// Weights: [health, quota, cost, latency, successRate, freshness]
func modeWeights(mode AutoMode) [6]float64 {
	switch mode {
	case AutoModeCoding:
		return [6]float64{0.15, 0.10, 0.10, 0.15, 0.35, 0.15} // prioritize success rate
	case AutoModeFast:
		return [6]float64{0.10, 0.05, 0.05, 0.50, 0.20, 0.10} // prioritize latency
	case AutoModeCheap:
		return [6]float64{0.10, 0.15, 0.45, 0.05, 0.15, 0.10} // prioritize cost
	default: // auto
		return [6]float64{0.20, 0.15, 0.15, 0.15, 0.25, 0.10} // balanced
	}
}

// scoreProvider computes a composite score for a provider given factor weights.
func scoreProvider(p Provider, w [6]float64) float64 {
	// Normalize each factor to 0.0-1.0
	healthNorm := float64(p.Health) / 100.0
	quotaNorm := clamp01(p.QuotaRemaining)
	costNorm := normalizeCost(p.CostPerToken)
	latencyNorm := normalizeLatency(p.Latency)
	successNorm := clamp01(p.SuccessRate)
	freshnessNorm := normalizeFreshness(p.LastUsed)

	return w[0]*healthNorm +
		w[1]*quotaNorm +
		w[2]*costNorm +
		w[3]*latencyNorm +
		w[4]*successNorm +
		w[5]*freshnessNorm
}

// normalizeCost maps cost to a 0-1 score where lower cost = higher score.
// Range: 0-50 per 1K tokens → 1.0-0.0
func normalizeCost(cost float64) float64 {
	if cost <= 0 {
		return 1.0
	}
	// Use inverse log scale; $0.01→~1.0, $1.0→~0.5, $50→~0.0
	return clamp01(1.0 - math.Log10(math.Max(cost, 0.001))/math.Log10(50))
}

// normalizeLatency maps latency (ms) to a 0-1 score where lower latency = higher score.
// Range: 0-5000ms
func normalizeLatency(latency float64) float64 {
	if latency <= 0 {
		return 1.0 // unknown latency → treat as fast
	}
	return clamp01(1.0 - latency/5000.0)
}

// normalizeFreshness maps time-since-last-use to a 0-1 score.
// Recent use = high score. >1h ago = 0.5, >24h = 0.1, never = 0.5 (neutral).
func normalizeFreshness(lastUsed time.Time) float64 {
	if lastUsed.IsZero() {
		return 0.5 // never used — neutral
	}
	elapsed := time.Since(lastUsed).Seconds()
	if elapsed < 60 {
		return 1.0
	}
	return clamp01(1.0 - math.Log10(elapsed/60)/4.0)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
