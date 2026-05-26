package memory

import (
	"math"
	"testing"
)

func TestEmbedBasic(t *testing.T) {
	vec := Embed("hello world this is a test of the embedding system")
	if len(vec) != EmbeddingDim {
		t.Fatalf("expected %d dimensions, got %d", EmbeddingDim, len(vec))
	}
	var sqSum float64
	for _, v := range vec {
		sqSum += v * v
	}
	if sqSum == 0 {
		t.Fatal("expected non-zero embedding for non-empty text")
	}
}

func TestEmbedEmpty(t *testing.T) {
	vec := Embed("")
	if len(vec) != EmbeddingDim {
		t.Fatalf("expected %d dimensions, got %d", EmbeddingDim, len(vec))
	}
	for _, v := range vec {
		if v != 0 {
			t.Fatal("expected all zeros for empty text")
		}
	}
}

func TestEmbedNormalization(t *testing.T) {
	vec := Embed("the quick brown fox jumps over the lazy dog")
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm < 0.99 || norm > 1.01 {
		t.Fatalf("expected normalized vector (norm≈1), got %f", norm)
	}
}

func TestEmbedReproducibility(t *testing.T) {
	text := "test reproducibility of TF-IDF embeddings"
	v1 := Embed(text)
	v2 := Embed(text)
	if len(v1) != len(v2) {
		t.Fatal("embedding lengths differ for same input")
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("dimension %d differs: %f vs %f", i, v1[i], v2[i])
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	if CosineSimilarity(a, b) != 0 {
		t.Fatal("orthogonal vectors should have 0 similarity")
	}
	if CosineSimilarity(a, a) < 0.999 {
		t.Fatal("identical vectors should have similarity 1")
	}
	if CosineSimilarity(nil, nil) != 0 {
		t.Fatal("nil vectors should return 0")
	}
	if CosineSimilarity([]float64{1, 2}, []float64{1}) != 0 {
		t.Fatal("different-length vectors should return 0")
	}
}

func TestHashKey(t *testing.T) {
	h1 := HashKey("hello world")
	h2 := HashKey("hello world")
	h3 := HashKey("different text")

	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	if h1 == h3 {
		t.Fatal("different inputs should produce different hashes")
	}
	if len(h1) != 64 {
		t.Fatalf("SHA-256 hash should be 64 hex chars, got %d", len(h1))
	}
}

func TestIndexScore(t *testing.T) {
	if s := IndexScore(5, 10); s != 50.0 {
		t.Fatalf("expected 50.0, got %f", s)
	}
	if s := IndexScore(0, 10); s != 0 {
		t.Fatalf("expected 0, got %f", s)
	}
	if s := IndexScore(0, 0); s != 0 {
		t.Fatalf("expected 0 for zero total, got %f", s)
	}
	if s := IndexScore(10, 10); s != 100.0 {
		t.Fatalf("expected 100.0, got %f", s)
	}
}

func TestRespPing(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	result, err := c.Do("PING")
	if err != nil {
		t.Fatalf("PING failed: %v", err)
	}
	s := respToString(result)
	if s != "PONG" {
		t.Fatalf("expected PONG, got %q", s)
	}
}

func TestRespSetGet(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	const testKey = "test:memory:resp_setget"
	const testVal = "hello-from-resp-test"

	_, err = c.Do("SET", testKey, testVal)
	if err != nil {
		t.Fatalf("SET failed: %v", err)
	}
	defer c.Do("DEL", testKey)

	result, err := c.Do("GET", testKey)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	got := respToString(result)
	if got != testVal {
		t.Fatalf("expected %q, got %q", testVal, got)
	}
}

func TestStoreAndSearch(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	sm := NewStoreManager(c)
	text := "machine learning vector embedding search test"
	key := HashKey(text + "_store_search")
	emb := Embed(text)

	err = sm.Store(key, emb, text, map[string]string{"source": "test"}, []string{"ai", "ml"}, 85.0)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	defer sm.Delete(key)

	results, err := sm.Search(emb, 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		// Fallback: VSET may not be available, but localSimilaritySearch should work
		t.Fatal("expected at least 1 result from search")
	}

	found := false
	for _, r := range results {
		if r.Key == key {
			found = true
			if r.Text != text {
				t.Errorf("text mismatch: got %q, want %q", r.Text, text)
			}
			if r.Score != 85.0 {
				t.Errorf("score mismatch: got %f, want 85.0", r.Score)
			}
			break
		}
	}
	if !found {
		t.Errorf("stored key %q not found in search results (got %d results)", key, len(results))
	}
}

func TestSearchByKeywords(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	sm := NewStoreManager(c)
	text := "quantum computing breakthroughs in 2026"
	key := HashKey(text + "_kw_search")
	emb := Embed(text)

	err = sm.Store(key, emb, text, map[string]string{"field": "physics"}, []string{"science"}, 90.0)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	defer sm.Delete(key)

	results, err := sm.SearchByKeywords("quantum", 5)
	if err != nil {
		t.Fatalf("SearchByKeywords failed: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Key == key {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("stored key %q not found by keyword 'quantum' (got %d results)", key, len(results))
	}
}

func TestIndexCompletion(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	sm := NewStoreManager(c)
	prompt := Prompt{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "What is the Go programming language?"},
		},
	}
	response := "Go is a statically typed, compiled programming language designed at Google."

	key, emb, err := sm.IndexCompletion(prompt, response, 95.0, []string{"programming", "go"}, 10, 50)
	if err != nil {
		t.Fatalf("IndexCompletion failed: %v", err)
	}
	defer sm.Delete(key)

	if key == "" {
		t.Fatal("expected non-empty key from IndexCompletion")
	}
	if len(emb) != EmbeddingDim {
		t.Fatalf("expected %d-dim embedding, got %d", EmbeddingDim, len(emb))
	}

	// Verify we can search for it
	results, err := sm.Search(emb, 3)
	if err != nil {
		t.Fatalf("Search after IndexCompletion failed: %v", err)
	}
	found := false
	for _, r := range results {
		if r.Key == key {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("indexed completion key %q not found in search results", key)
	}
}

func TestStats(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	sm := NewStoreManager(c)
	stats := sm.Stats()
	if _, ok := stats["total_memories"]; !ok {
		t.Fatal("expected total_memories key in stats")
	}
	// total_memories should be >= 0
	if count, ok := stats["total_memories"].(int); !ok {
		t.Fatalf("total_memories should be int, got %T", stats["total_memories"])
	} else if count < 0 {
		t.Fatalf("total_memories should be >= 0, got %d", count)
	}
}

func TestDelete(t *testing.T) {
	c, err := NewClient("127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer c.Close()

	sm := NewStoreManager(c)
	text := "test delete memory entry"
	key := HashKey(text + "_delete_test")
	emb := Embed(text)

	err = sm.Store(key, emb, text, nil, nil, 50.0)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify it exists
	hashKey := "lintasan:mem:" + key
	exists, err := c.Do("EXISTS", hashKey)
	if err != nil {
		t.Fatalf("EXISTS failed: %v", err)
	}
	if v, ok := exists.(int64); !ok || v != 1 {
		t.Fatal("key should exist after Store")
	}

	// Delete it
	err = sm.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	exists, err = c.Do("EXISTS", hashKey)
	if err != nil {
		t.Fatalf("EXISTS after delete failed: %v", err)
	}
	if v, ok := exists.(int64); ok && v != 0 {
		t.Fatal("key should not exist after Delete")
	}
}

func TestFormatFloat(t *testing.T) {
	if got := formatFloat(3.14); got != "3.14" {
		t.Fatalf("expected %q, got %q", "3.14", got)
	}
	if got := formatFloat(1.0); got != "1" {
		t.Fatalf("expected %q, got %q", "1", got)
	}
	if got := formatFloat(0.123456789); got != "0.123457" {
		t.Fatalf("expected %q, got %q", "0.123457", got)
	}
	if got := formatFloat(0.0); got != "0" {
		t.Fatalf("expected %q, got %q", "0", got)
	}
}
