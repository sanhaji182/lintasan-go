package batch

import (
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Window == 0 {
		t.Error("Window should not be zero")
	}
}

func TestNewBatchRouter(t *testing.T) {
	fire := func(model string, bodies [][]byte) ([]*Response, error) {
		return nil, nil
	}
	b := New(DefaultConfig(), fire)
	if b == nil {
		t.Fatal("New returned nil")
	}
}

func TestSubmitBatch(t *testing.T) {
	done := make(chan bool, 1)
	fire := func(model string, bodies [][]byte) ([]*Response, error) {
		resps := make([]*Response, len(bodies))
		for i := range bodies {
			resps[i] = &Response{Body: []byte("ok")}
		}
		return resps, nil
	}
	b := New(Config{Window: 100 * time.Millisecond, MaxBatchSize: 2}, fire)
	reply := make(chan *Response, 2)

	b.Submit(&Request{Model: "test", Body: []byte("a"), Reply: reply})
	b.Submit(&Request{Model: "test", Body: []byte("b"), Reply: reply})

	done <- true

	resp1 := <-reply
	resp2 := <-reply
	if resp1.Err != nil || resp2.Err != nil {
		t.Error("unexpected errors")
	}
}

func TestStatsEmpty(t *testing.T) {
	fire := func(model string, bodies [][]byte) ([]*Response, error) {
		return nil, nil
	}
	b := New(DefaultConfig(), fire)
	s := b.Stats()
	if s["active_buckets"] != 0 {
		t.Errorf("active_buckets = %d, want 0", s["active_buckets"])
	}
}
