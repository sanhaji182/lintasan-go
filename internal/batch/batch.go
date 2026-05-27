package batch

import (
	"sync"
	"time"
)

// Request bundles a chat completion request with a reply channel.
type Request struct {
	Model    string
	Messages []map[string]any
	Body     []byte
	Reply    chan *Response
}

// Response is the upstream response for a batched request.
type Response struct {
	Body []byte
	Err  error
}

// BatchRouter collects concurrent requests to the same model within a
// time window and merges them into a single upstream call.
type BatchRouter struct {
	window        time.Duration
	maxBatchSize  int
	maxTokens     int
	mu            sync.Mutex
	buckets       map[string]*bucket
	fire          func(model string, bodies [][]byte) ([]*Response, error)
}

type bucket struct {
	requests []*Request
	timer    *time.Timer
}

// Config configures the batch router.
type Config struct {
	Window       time.Duration // collection window (default 200ms)
	MaxBatchSize int           // max requests per batch (default 10)
	MaxTokens    int           // max total tokens per batch (default 100000)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Window:       200 * time.Millisecond,
		MaxBatchSize: 10,
		MaxTokens:    100_000,
	}
}

// New creates a new BatchRouter.
func New(cfg Config, fire func(model string, bodies [][]byte) ([]*Response, error)) *BatchRouter {
	if cfg.Window == 0 {
		cfg.Window = 200 * time.Millisecond
	}
	if cfg.MaxBatchSize == 0 {
		cfg.MaxBatchSize = 10
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 100_000
	}
	return &BatchRouter{
		window:       cfg.Window,
		maxBatchSize: cfg.MaxBatchSize,
		maxTokens:    cfg.MaxTokens,
		buckets:      make(map[string]*bucket),
		fire:         fire,
	}
}

// Submit adds a request to the batch queue. When the window closes or
// maxBatchSize is reached, all queued requests fire together.
func (b *BatchRouter) Submit(req *Request) {
	b.mu.Lock()
	bk, exists := b.buckets[req.Model]
	if !exists {
		bk = &bucket{}
		b.buckets[req.Model] = bk
	}
	bk.requests = append(bk.requests, req)

	shouldFire := len(bk.requests) >= b.maxBatchSize
	if !shouldFire && bk.timer == nil {
		bk.timer = time.AfterFunc(b.window, func() {
			b.mu.Lock()
			b.fireBucket(req.Model)
			b.mu.Unlock()
		})
	}
	b.mu.Unlock()

	if shouldFire {
		b.mu.Lock()
		b.fireBucket(req.Model)
		b.mu.Unlock()
	}
}

func (b *BatchRouter) fireBucket(model string) {
	bk, exists := b.buckets[model]
	if !exists || len(bk.requests) == 0 {
		return
	}
	if bk.timer != nil {
		bk.timer.Stop()
	}

	reqs := bk.requests
	delete(b.buckets, model)

	bodies := make([][]byte, len(reqs))
	for i, r := range reqs {
		bodies[i] = r.Body
	}

	responses, err := b.fire(model, bodies)
	if err != nil {
		for _, r := range reqs {
			r.Reply <- &Response{Err: err}
		}
		return
	}

	for i, r := range reqs {
		if i < len(responses) {
			r.Reply <- responses[i]
		} else {
			r.Reply <- &Response{Err: err}
		}
	}
}

// Stats returns batching statistics.
func (b *BatchRouter) Stats() map[string]int {
	b.mu.Lock()
	defer b.mu.Unlock()
	stats := make(map[string]int)
	stats["active_buckets"] = len(b.buckets)
	total := 0
	for _, bk := range b.buckets {
		total += len(bk.requests)
	}
	stats["queued_requests"] = total
	return stats
}
