// Package memory provides a pure-Go TF-IDF embedder and Redis-backed vector store.
package memory

import "fmt"

// MemoryStore is the top-level memory service wrapping a StoreManager and Client.
type MemoryStore struct {
	Store  *StoreManager
	client *Client
}

// Config holds Redis connection parameters.
type Config struct {
	Addr string
}

// New creates a MemoryStore connected to Redis.
func New(cfg Config) (*MemoryStore, error) {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:6379"
	}
	client, err := NewClient(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("redis connect: %w", err)
	}
	if _, err := client.Do("PING"); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &MemoryStore{
		Store:  NewStoreManager(client),
		client: client,
	}, nil
}

// Close shuts down the Redis connection.
func (ms *MemoryStore) Close() error {
	return ms.client.Close()
}
