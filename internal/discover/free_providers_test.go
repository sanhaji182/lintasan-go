package discover

import (
	"testing"
)

func TestGetFreeProviders(t *testing.T) {
	providers := GetFreeProviders()
	if len(providers) < 5 {
		t.Fatalf("expected at least 5 free providers, got %d", len(providers))
	}

	// Check each provider has required fields
	for _, p := range providers {
		if p.Name == "" {
			t.Error("provider name should not be empty")
		}
		if p.Prefix == "" {
			t.Errorf("%s: prefix should not be empty", p.Name)
		}
		if p.BaseURL == "" {
			t.Errorf("%s: base_url should not be empty", p.Name)
		}
		if len(p.Models) == 0 {
			t.Errorf("%s: should have at least 1 model", p.Name)
		}
	}
}

func TestGetFreeProviderByPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		wantName string
		wantNil  bool
	}{
		{"kr/", "Kiro AI", false},
		{"pol/", "Pollinations", false},
		{"cf/", "Cloudflare AI", false},
		{"nvidia/", "NVIDIA NIM", false},
		{"cerebras/", "Cerebras", false},
		{"invalid/", "", true},
	}

	for _, tt := range tests {
		p := GetFreeProviderByPrefix(tt.prefix)
		if tt.wantNil {
			if p != nil {
				t.Errorf("prefix %s: expected nil, got %s", tt.prefix, p.Name)
			}
			continue
		}
		if p == nil {
			t.Errorf("prefix %s: expected %s, got nil", tt.prefix, tt.wantName)
			continue
		}
		if p.Name != tt.wantName {
			t.Errorf("prefix %s: expected name %s, got %s", tt.prefix, tt.wantName, p.Name)
		}
	}
}

func TestGetFreeProviderByName(t *testing.T) {
	p := GetFreeProviderByName("Kiro AI")
	if p == nil {
		t.Fatal("expected Kiro AI, got nil")
	}
	if p.Prefix != "kr/" {
		t.Errorf("expected prefix kr/, got %s", p.Prefix)
	}

	if p := GetFreeProviderByName("NonExistent"); p != nil {
		t.Error("expected nil for non-existent provider")
	}
}

func TestGetAllFreeModels(t *testing.T) {
	models := GetAllFreeModels()
	if len(models) < 5 {
		t.Fatalf("expected at least 5 providers in model map, got %d", len(models))
	}

	// Check Kiro has models
	kiroModels, ok := models["kr/"]
	if !ok {
		t.Fatal("expected kr/ in model map")
	}
	if len(kiroModels) == 0 {
		t.Error("Kiro should have at least 1 model")
	}
}

func TestFreeProviderAuthTypes(t *testing.T) {
	providers := GetFreeProviders()
	authTypes := make(map[string]bool)
	for _, p := range providers {
		authTypes[p.AuthType] = true
	}

	// Should have both "none" and "apikey" auth types
	if !authTypes["none"] {
		t.Error("expected at least one provider with auth_type=none")
	}
	if !authTypes["apikey"] {
		t.Error("expected at least one provider with auth_type=apikey")
	}
}

func TestFreeProviderPrefixes(t *testing.T) {
	providers := GetFreeProviders()
	seen := make(map[string]bool)
	for _, p := range providers {
		if seen[p.Prefix] {
			t.Errorf("duplicate prefix: %s", p.Prefix)
		}
		seen[p.Prefix] = true

		// Prefix should end with /
		if len(p.Prefix) == 0 || p.Prefix[len(p.Prefix)-1] != '/' {
			t.Errorf("%s: prefix should end with /, got %s", p.Name, p.Prefix)
		}
	}
}
