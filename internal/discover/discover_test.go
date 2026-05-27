package discover

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseModelsResponse_OpenAIFormat(t *testing.T) {
	body := `{"object":"list","data":[{"id":"gpt-4o","object":"model","owned_by":"openai"},{"id":"gpt-4o-mini","object":"model","owned_by":"openai"}]}`
	models := parseModelsResponse([]byte(body))
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", models[0].ID)
	}
	if models[0].OwnedBy != "openai" {
		t.Errorf("expected openai owned_by, got %s", models[0].OwnedBy)
	}
	if models[1].ID != "gpt-4o-mini" {
		t.Errorf("expected gpt-4o-mini, got %s", models[1].ID)
	}
}

func TestParseModelsResponse_ModelsKey(t *testing.T) {
	body := `{"models":[{"id":"claude-sonnet-4","name":"Claude Sonnet 4","owned_by":"anthropic"}]}`
	models := parseModelsResponse([]byte(body))
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "claude-sonnet-4" {
		t.Errorf("expected claude-sonnet-4, got %s", models[0].ID)
	}
	if models[0].Name != "Claude Sonnet 4" {
		t.Errorf("expected 'Claude Sonnet 4', got %s", models[0].Name)
	}
}

func TestParseModelsResponse_PlainArray(t *testing.T) {
	body := `[{"id":"a"},{"id":"b"}]`
	models := parseModelsResponse([]byte(body))
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
}

func TestParseModelsResponse_Empty(t *testing.T) {
	body := `{"data":[]}`
	models := parseModelsResponse([]byte(body))
	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestParseModelsResponse_InvalidJSON(t *testing.T) {
	models := parseModelsResponse([]byte(`not json`))
	if models != nil {
		t.Errorf("expected nil for invalid JSON, got %v", models)
	}
}

func TestParseModelsResponse_ModelField(t *testing.T) {
	// Some providers use "model" instead of "id"
	body := `{"data":[{"model":"llama-3","name":"Llama 3"}]}`
	models := parseModelsResponse([]byte(body))
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "llama-3" {
		t.Errorf("expected 'llama-3', got %s", models[0].ID)
	}
}

func TestParseModelsResponse_NameFallback(t *testing.T) {
	// Some providers only have "name"
	body := `{"data":[{"name":"gemini-pro"}]}`
	models := parseModelsResponse([]byte(body))
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "gemini-pro" {
		t.Errorf("expected 'gemini-pro', got %s", models[0].ID)
	}
	if models[0].Name != "gemini-pro" {
		t.Errorf("expected name 'gemini-pro', got %s", models[0].Name)
	}
}

func TestParseModelsResponse_OwnerField(t *testing.T) {
	body := `{"data":[{"id":"x","owner":"org"}]}`
	models := parseModelsResponse([]byte(body))
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].OwnedBy != "org" {
		t.Errorf("expected owned_by 'org', got %s", models[0].OwnedBy)
	}
}

func TestKnownModels(t *testing.T) {
	km := knownModels()

	// commandcode
	if cc, ok := km["commandcode"]; !ok || len(cc) < 7 {
		t.Errorf("expected at least 7 commandcode models, got %d", len(cc))
	}

	// openai
	if oa, ok := km["openai"]; !ok || len(oa) < 5 {
		t.Errorf("expected at least 5 openai models, got %d", len(oa))
	}

	// anthropic
	if an, ok := km["anthropic"]; !ok || len(an) < 3 {
		t.Errorf("expected at least 3 anthropic models, got %d", len(an))
	}
}

func TestKnownModels_Contents(t *testing.T) {
	km := knownModels()

	// Verify specific required models exist
	cc := km["commandcode"]
	ccIDs := make(map[string]bool)
	for _, m := range cc {
		ccIDs[m.ID] = true
	}
	requiredCC := []string{
		"deepseek/deepseek-v4-pro", "deepseek/deepseek-v3", "deepseek/deepseek-r1",
		"kimi/kimi-k2.6", "glm/glm-4.7", "minimax/minimax-m2.7", "qwen/qwen3-coder",
	}
	for _, id := range requiredCC {
		if !ccIDs[id] {
			t.Errorf("missing required commandcode model: %s", id)
		}
	}

	oa := km["openai"]
	oaIDs := make(map[string]bool)
	for _, m := range oa {
		oaIDs[m.ID] = true
	}
	requiredOA := []string{"gpt-4o", "gpt-4o-mini", "o3-mini", "o1", "o1-mini"}
	for _, id := range requiredOA {
		if !oaIDs[id] {
			t.Errorf("missing required openai model: %s", id)
		}
	}

	an := km["anthropic"]
	anIDs := make(map[string]bool)
	for _, m := range an {
		anIDs[m.ID] = true
	}
	requiredAN := []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-haiku-3-5"}
	for _, id := range requiredAN {
		if !anIDs[id] {
			t.Errorf("missing required anthropic model: %s", id)
		}
	}
}

func TestDiscoverer_NewDiscoverer(t *testing.T) {
	d := NewDiscoverer(nil)
	if d == nil {
		t.Fatal("NewDiscoverer returned nil")
	}
	if d.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if d.knownModels == nil {
		t.Error("knownModels is nil")
	}
	if len(d.knownModels) < 3 {
		t.Errorf("expected at least 3 known model formats, got %d", len(d.knownModels))
	}
}

func TestFallbackModels_ValidFormat(t *testing.T) {
	d := NewDiscoverer(nil)
	conn := map[string]any{"format": "openai"}
	models, err := d.fallbackModels(conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) < 5 {
		t.Errorf("expected at least 5 openai fallback models, got %d", len(models))
	}
}

func TestFallbackModels_UnknownFormat(t *testing.T) {
	d := NewDiscoverer(nil)
	conn := map[string]any{"format": "nonexistent"}
	models, err := d.fallbackModels(conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if models != nil {
		t.Errorf("expected nil models for unknown format, got %v", models)
	}
}

func TestFetchModelsFromProvider_RealHTTPServer(t *testing.T) {
	// Integration test with a real httptest server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "test-model-1", "name": "Test Model 1", "owned_by": "test-org"},
				{"id": "test-model-2", "name": "Test Model 2", "owned_by": "test-org"},
			},
		})
	}))
	defer server.Close()

	d := NewDiscoverer(nil)
	conn := map[string]any{
		"base_url":    server.URL,
		"models_path": "/v1/models",
		"format":      "openai",
	}
	models, err := d.fetchModelsFromProvider(conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "test-model-1" {
		t.Errorf("expected test-model-1, got %s", models[0].ID)
	}
}

func TestFetchModelsFromProvider_HTTPErrorFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	d := NewDiscoverer(nil)
	conn := map[string]any{
		"base_url":    server.URL,
		"models_path": "/v1/models",
		"format":      "openai",
	}
	models, err := d.fetchModelsFromProvider(conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return fallback models for openai
	if len(models) < 5 {
		t.Errorf("expected at least 5 fallback models on error, got %d", len(models))
	}
}

func TestFetchModelsFromProvider_EmptyResponseFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer server.Close()

	d := NewDiscoverer(nil)
	conn := map[string]any{
		"base_url":    server.URL,
		"models_path": "/v1/models",
		"format":      "openai",
	}
	models, err := d.fetchModelsFromProvider(conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) < 5 {
		t.Errorf("expected at least 5 fallback models on empty response, got %d", len(models))
	}
}
