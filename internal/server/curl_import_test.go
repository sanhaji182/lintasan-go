package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test parseCurlCommand against the curl strings it is expected to handle in
// production. Coverage gap closed: internal/server/curl_import.go is exposed
// via POST /api/connections/import-curl but had no test before.

func TestParseCurlCommand_BasicGET(t *testing.T) {
	got, err := parseCurlCommand("curl https://api.example.com/v1/models")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != "GET" {
		t.Errorf("method = %q, want GET (no body defaults to GET)", got.method)
	}
	if got.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want %q", got.baseURL, "https://api.example.com")
	}
	if got.chatPath != "/v1/models" {
		t.Errorf("chatPath = %q, want %q", got.chatPath, "/v1/models")
	}
	if got.apiKey != "" {
		t.Errorf("apiKey = %q, want empty (no auth header)", got.apiKey)
	}
}

func TestParseCurlCommand_POSTWithBody(t *testing.T) {
	got, err := parseCurlCommand(`curl https://api.openai.com/v1/chat/completions \
  -H "content-type: application/json" \
  -d '{"model":"gpt-4o","messages":[]}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != "POST" {
		t.Errorf("method = %q, want POST (body present, no -X)", got.method)
	}
	if got.baseURL != "https://api.openai.com" {
		t.Errorf("baseURL = %q, want %q", got.baseURL, "https://api.openai.com")
	}
	if got.chatPath != "/v1/chat/completions" {
		t.Errorf("chatPath = %q, want %q", got.chatPath, "/v1/chat/completions")
	}
	if !strings.Contains(got.body, `"model":"gpt-4o"`) {
		t.Errorf("body = %q, should contain model JSON", got.body)
	}
	if got.headers["content-type"] != "application/json" {
		t.Errorf("content-type header = %q, want application/json", got.headers["content-type"])
	}
}

func TestParseCurlCommand_BearerAuth(t *testing.T) {
	got, err := parseCurlCommand(`curl https://api.openai.com/v1/chat/completions \
  -H "authorization: Bearer sk-test-abc123" \
  -H "content-type: application/json" \
  -d '{}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.apiKey != "sk-test-abc123" {
		t.Errorf("apiKey = %q, want %q", got.apiKey, "sk-test-abc123")
	}
	if got.authHeader != "authorization" {
		t.Errorf("authHeader = %q, want authorization (preserved case)", got.authHeader)
	}
	if got.authPrefix != "Bearer " {
		t.Errorf("authPrefix = %q, want %q", got.authPrefix, "Bearer ")
	}
}

func TestParseCurlCommand_XAPIKeyAuth(t *testing.T) {
	got, err := parseCurlCommand(`curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: sk-ant-test" \
  -H "anthropic-version: 2023-06-01" \
  -d '{}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.apiKey != "sk-ant-test" {
		t.Errorf("apiKey = %q, want %q", got.apiKey, "sk-ant-test")
	}
	if got.authHeader != "x-api-key" {
		t.Errorf("authHeader = %q, want x-api-key", got.authHeader)
	}
	if got.authPrefix != "" {
		t.Errorf("authPrefix = %q, want empty (x-api-key has no prefix)", got.authPrefix)
	}
}

func TestParseCurlCommand_ExplicitMethodOverride(t *testing.T) {
	got, err := parseCurlCommand(`curl -X DELETE https://api.example.com/v1/items/42`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != "DELETE" {
		t.Errorf("method = %q, want DELETE (-X override)", got.method)
	}
}

func TestParseCurlCommand_DollarPrefix(t *testing.T) {
	// Some users paste "$ curl ..." with a shell prompt prefix.
	got, err := parseCurlCommand("$ curl https://api.example.com/v1/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want %q", got.baseURL, "https://api.example.com")
	}
}

func TestParseCurlCommand_QuotedBody(t *testing.T) {
	got, err := parseCurlCommand(`curl https://x.test/v1/y -d '{"a":"b","c":1}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.body != `{"a":"b","c":1}` {
		t.Errorf("body = %q, want single-quote stripped", got.body)
	}
}

func TestParseCurlCommand_InvalidInput(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"no URL", "curl --no-url"},
		{"not a curl", "not a curl command"},
		{"whitespace only", "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseCurlCommand(c.in); err == nil {
				t.Errorf("expected error for input %q", c.in)
			}
		})
	}
}

func TestInferNameFromHost(t *testing.T) {
	// These are the host patterns the current implementation handles.
	// Subdomain-style TLDs (e.g. co.uk) and hyphenated names return the
	// component immediately before the TLD — documented as a known best-effort.
	cases := []struct {
		host string
		want string
	}{
		{"api.openai.com", "Openai"},
		{"api.deepseek.com", "Deepseek"},
		{"localhost:8080", "Localhost"},
		{"api.foo-bar.dev", "FooBar"},
	}
	for _, c := range cases {
		t.Run(c.host, func(t *testing.T) {
			if got := inferNameFromHost(c.host); got != c.want {
				t.Errorf("inferNameFromHost(%q) = %q, want %q", c.host, got, c.want)
			}
		})
	}
}

func TestTokenizeCurl(t *testing.T) {
	// tokenizeCurl operates on the part AFTER "curl " has been stripped
	// (parseCurlCommand does that pre-processing).
	got := tokenizeCurl(`https://x.test/v1/y -H "content-type: application/json" -d '{"a":1}'`)
	want := []string{
		"https://x.test/v1/y",
		"-H",
		"content-type: application/json",
		"-d",
		`{"a":1}`,
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("token[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStripQuotes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{"hello", "hello"},
		{`"`, `"`},
		{"", ""},
		{`""`, ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := stripQuotes(c.in); got != c.want {
				t.Errorf("stripQuotes(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// HTTP-level test of the handler. We do not exercise the DB / discoverer
// paths here — those need a real server. The error paths and JSON shape are
// stable and worth covering.

func TestHandleCurlImport_BadJSON(t *testing.T) {
	s := &Server{} // no db / discoverer — handler returns 400 before touching them
	req := httptest.NewRequest(http.MethodPost, "/api/connections/import-curl",
		strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleCurlImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid JSON") {
		t.Errorf("body = %q, want contains 'invalid JSON'", rr.Body.String())
	}
}

func TestHandleCurlImport_EmptyCurl(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/connections/import-curl",
		strings.NewReader(`{"curl":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleCurlImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "curl field is required") {
		t.Errorf("body = %q, want contains 'curl field is required'", rr.Body.String())
	}
}

func TestHandleCurlImport_InvalidCurl(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/connections/import-curl",
		strings.NewReader(`{"curl":"not a curl command"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleCurlImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}
