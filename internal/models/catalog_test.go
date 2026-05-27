package models

import (
	"testing"
)

func TestCatalog(t *testing.T) {
	catalog := Catalog()

	providerIDs := map[string]bool{}
	for _, p := range catalog {
		providerIDs[p.ID] = true
		if len(p.Models) == 0 {
			t.Errorf("provider %s has no models", p.ID)
		}
		if p.BaseURL == "" {
			t.Errorf("provider %s has no base URL", p.ID)
		}
		if p.Format == "" {
			t.Errorf("provider %s has no format", p.ID)
		}
	}

	expectedProviders := []string{
		"openai", "anthropic", "google", "deepseek", "meta",
		"mistral", "qwen", "xai", "cohere", "ai21", "reka", "perplexity", "commandcode",
	}
	for _, ep := range expectedProviders {
		if !providerIDs[ep] {
			t.Errorf("expected provider %s not found in catalog", ep)
		}
	}

	total := TotalModelCount()
	if total < 65 {
		t.Errorf("expected at least 65 models, got %d", total)
	}
	t.Logf("Total models: %d", total)
}

func TestAllModels(t *testing.T) {
	all := AllModels()
	if len(all) == 0 {
		t.Error("AllModels() returned empty slice")
	}

	// Check for duplicates
	seen := map[string]bool{}
	for _, m := range all {
		if seen[m.ID] {
			t.Errorf("duplicate model ID: %s", m.ID)
		}
		seen[m.ID] = true
	}
}

func TestModelInfo(t *testing.T) {
	for _, m := range AllModels() {
		if m.ContextWindow <= 0 {
			t.Errorf("model %s has invalid context window: %d", m.ID, m.ContextWindow)
		}
		if m.MaxTokens <= 0 {
			t.Errorf("model %s has invalid max tokens: %d", m.ID, m.MaxTokens)
		}
		if len(m.Capabilities) == 0 {
			t.Errorf("model %s has no capabilities", m.ID)
		}
	}
}

func TestFindModel(t *testing.T) {
	tests := []struct {
		id      string
		wantNil bool
		wantID  string
	}{
		{"gpt-4o", false, "gpt-4o"},
		{"claude-opus-4-20250514", false, "claude-opus-4-20250514"},
		{"gemini-2.5-pro", false, "gemini-2.5-pro"},
		{"deepseek-r1", false, "deepseek-r1"},
		{"nonexistent-model", true, ""},
	}

	for _, tt := range tests {
		m := FindModel(tt.id)
		if tt.wantNil {
			if m != nil {
				t.Errorf("FindModel(%q) = %v, want nil", tt.id, m)
			}
		} else {
			if m == nil {
				t.Errorf("FindModel(%q) = nil, want non-nil", tt.id)
			} else if m.ID != tt.wantID {
				t.Errorf("FindModel(%q).ID = %q, want %q", tt.id, m.ID, tt.wantID)
			}
		}
	}
}

func TestFindProvider(t *testing.T) {
	tests := []struct {
		id      string
		wantNil bool
		wantID  string
	}{
		{"openai", false, "openai"},
		{"anthropic", false, "anthropic"},
		{"google", false, "google"},
		{"deepseek", false, "deepseek"},
		{"nonexistent", true, ""},
	}

	for _, tt := range tests {
		p := FindProvider(tt.id)
		if tt.wantNil {
			if p != nil {
				t.Errorf("FindProvider(%q) = %v, want nil", tt.id, p)
			}
		} else {
			if p == nil {
				t.Errorf("FindProvider(%q) = nil, want non-nil", tt.id)
			} else if p.ID != tt.wantID {
				t.Errorf("FindProvider(%q).ID = %q, want %q", tt.id, p.ID, tt.wantID)
			}
		}
	}
}

func TestAllProviders(t *testing.T) {
	providers := AllProviders()
	if len(providers) < 10 {
		t.Errorf("expected at least 10 providers, got %d", len(providers))
	}
	for _, p := range providers {
		if p.ID == "" {
			t.Error("AllProviders() returned provider with empty ID")
		}
		if len(p.Models) == 0 {
			t.Errorf("provider %s has no models", p.ID)
		}
	}
}
