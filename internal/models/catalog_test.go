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
