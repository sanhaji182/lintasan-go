package memory

import "time"

// Memory represents a stored vector memory entry.
type Memory struct {
	Key        string            `json:"key"`
	Text       string            `json:"text"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Embedding  []float64         `json:"embedding,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Score      float64           `json:"score"`
	Similarity float64           `json:"similarity"`
	Hits       int               `json:"hits"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Message is a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Prompt represents a completed LLM request for indexing.
type Prompt struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
