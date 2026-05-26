package auth

import (
	"testing"
)

func TestNewOAuthManager(t *testing.T) {
	m := NewOAuthManager(nil)
	if m == nil {
		t.Fatal("NewOAuthManager returned nil")
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc", "a***c"},
		{"sk-abc12345def", "sk-a...5def"},
		{"sk-lintasan-abc123", "sk-l...c123"},
	}
	for _, tt := range tests {
		result := maskToken(tt.input)
		if result != tt.expected {
			t.Errorf("maskToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestValidProviders(t *testing.T) {
	validProviders := []string{"cursor", "codex", "claude-desktop", "copilot", "windsurf", "aider"}
	for _, p := range validProviders {
		if p == "" {
			t.Error("unexpected empty provider")
		}
	}
}

func TestOAuthSessionFields(t *testing.T) {
	s := OAuthSession{
		ID:       "test-id",
		Provider: "cursor",
		Status:   "pending",
	}
	if s.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", s.ID)
	}
	if s.Provider != "cursor" {
		t.Errorf("expected provider 'cursor', got %q", s.Provider)
	}
	if s.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", s.Status)
	}
}
