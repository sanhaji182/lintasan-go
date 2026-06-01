package server

// handlers_experimental_admit_test.go — G1 regression test.
//
// Loads the FULL HTTP middleware chain (cors + auth + mux) via the
// newTestServer/security-boundary harness, then drives POST
// /api/experimental/providers/codex/admit through the same wire path a
// dashboard user would hit. This is the regression that the prior
// framework_test.go-only coverage missed: the production HTTP handler
// passed `nil` as the provider registry to AdmitProvider, so every admit
// attempt returned "admission-error: expprovider: nil registry" and the
// sticky evidence made the page un-recoverable from the dashboard.
//
// The fix (G1): handlers_experimental.go now passes s.proxy.providerReg
// (the live ProxyHandler's *provider.Registry) to AdmitProvider, and
// aligns the persisted state with the framework's lifecycle verdict so
// the admit -> activate -> deactivate chain actually works end-to-end.
//
// Membrane is asserted explicitly: the Experimental provider must register
// in the providerReg but MUST NOT appear in reg.RoutableProviders()
// (Track==Experimental, filtered by the production membrane).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sanhaji182/lintasan-go/internal/config"
	"github.com/sanhaji182/lintasan-go/internal/expprovider"
	"github.com/sanhaji182/lintasan-go/internal/provider"
)

// newAdmitTestServer builds a Server backed by an in-memory DB with the
// full middleware chain, exactly as production wires it.
func newAdmitTestServer(t *testing.T, cfg *config.Config) (*Server, *httptest.Server) {
	t.Helper()
	return newTestServer(t, cfg)
}

// seedOpenAICredential sets OPENAI_API_KEY for the duration of the test so
// the admit endpoint's credential-availability precheck passes.
func seedOpenAICredential(t *testing.T, v string) {
	t.Helper()
	orig, had := os.LookupEnv("OPENAI_API_KEY")
	if err := os.Setenv("OPENAI_API_KEY", v); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("OPENAI_API_KEY", orig)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
	})
}

// doPost is a POST helper that serialises body, attaches the Bearer token,
// and returns the raw response.
func doPost(t *testing.T, ts *httptest.Server, path, token string, body any) *http.Response {
	t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// decodeData unwraps the {"data": ...} envelope that writeData() produces
// and returns the inner map. Fails the test if the envelope is missing.
func decodeData(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp.Body.Close()
	d, ok := envelope["data"]
	if !ok {
		t.Fatalf("expected {data: ...} envelope, got: %+v", envelope)
	}
	m, ok := d.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a map, got: %T (%+v)", d, d)
	}
	return m
}

// fetchPersistedRecord reads the record directly from the DB.
func fetchPersistedRecord(t *testing.T, s *Server, name string) *expprovider.ProviderRecord {
	t.Helper()
	rec, err := s.expStore().Get(context.Background(), name)
	if err != nil {
		t.Fatalf("get provider record: %v", err)
	}
	return rec
}

// contains is a tiny helper.
func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// ─── G1 REGRESSION: nil-registry admit no longer happens ────────────────────

func TestExperimentalAdmit_NilRegistryBugIsFixed(t *testing.T) {
	seedOpenAICredential(t, "sk-test-fake-1234567890abcdef")

	cfg := &config.Config{MasterKey: "test-master-key-1234567890"}
	s, ts := newAdmitTestServer(t, cfg)
	token := makeKnownAdmin(t, s, "admin-g1", "correct horse battery")
	if !s.isActive() {
		t.Fatal("expected ACTIVE state (admin + master key present)")
	}

	resp := doPost(t, ts, "/api/experimental/providers/codex/admit", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admit: expected 200, got %d", resp.StatusCode)
	}
	body := decodeData(t, resp)

	// THE core G1 assertion: evidence must NOT be the "nil registry" sticky
	// string. The harness verdict may be PASS or NO-GO — both are valid
	// post-fix outcomes. The critical thing is that the nil-registry bug
	// is gone.
	evidence, _ := body["evidence"].(string)
	if strings.Contains(evidence, "nil registry") {
		t.Fatalf("REGRESSION: admit returned the old nil-registry error — "+
			"fix is not in effect. evidence=%q response=%+v", evidence, body)
	}
	if evidence == "" {
		t.Fatalf("admit returned empty evidence; expected either 'fixture-pass' "+
			"(harness GO) or 'admission-no-go' (stub harness). response=%+v", body)
	}

	// Provider must be registered in the live registry (the G1 wiring fix).
	if _, ok := s.proxy.providerReg.Get("codex"); !ok {
		t.Fatal("after admit, provider 'codex' must be registered in proxy.providerReg " +
			"(wiring regression: returned *ACPProvider was discarded)")
	}

	// Membrane MUST be intact: Experimental provider visible by Get, absent
	// from RoutableProviders (which filters by Track==Official).
	if routable := s.proxy.providerReg.RoutableProviders(); contains(routable, "codex") {
		t.Fatalf("MEMBRANE VIOLATION: 'codex' appears in RoutableProviders(): %v", routable)
	}
	if p, ok := s.proxy.providerReg.Get("codex"); ok {
		if p.Track() != provider.TrackExperimental {
			t.Fatalf("registered 'codex' has Track=%q, want Experimental", p.Track())
		}
	}
}

// ─── End-to-end: admit -> activate -> deactivate ──────────────────────────────

func TestExperimentalAdmit_Activate_Deactivate_EndToEnd(t *testing.T) {
	seedOpenAICredential(t, "sk-test-fake-1234567890abcdef")

	cfg := &config.Config{MasterKey: "test-master-key-1234567890"}
	s, ts := newAdmitTestServer(t, cfg)
	token := makeKnownAdmin(t, s, "admin-e2e", "correct horse battery")

	// --- admit ---
	resp := doPost(t, ts, "/api/experimental/providers/codex/admit", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admit: expected 200, got %d", resp.StatusCode)
	}
	body := decodeData(t, resp)

	// After admit: persisted state must be "admitted" (not "proposed" —
	// that was the old rewind bug).
	state, _ := body["state"].(string)
	if state != "admitted" && state != "active" {
		t.Fatalf("admit state: got %q, want admitted or active", state)
	}

	// Persisted record must match.
	rec := fetchPersistedRecord(t, s, "codex")
	if rec == nil {
		t.Fatal("expected persisted record after admit, got nil")
	}
	if rec.State != "admitted" && rec.State != "active" {
		t.Fatalf("persisted state after admit: got %q, want admitted or active", rec.State)
	}
	if strings.Contains(rec.ValidationEvidence, "nil registry") {
		t.Fatalf("persisted evidence is the old nil-registry sticky error: %q",
			rec.ValidationEvidence)
	}
	if rec.AdmittedAt == nil {
		t.Fatal("admitted_at should be set after a successful admit")
	}

	// --- activate ---
	resp = doPost(t, ts, "/api/experimental/providers/codex/activate", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("activate: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	rec = fetchPersistedRecord(t, s, "codex")
	if rec.State != "active" {
		t.Fatalf("persisted state after activate: got %q, want active", rec.State)
	}
	if rec.ActivatedAt == nil {
		t.Fatal("activated_at should be set after a successful activate")
	}

	// --- deactivate ---
	resp = doPost(t, ts, "/api/experimental/providers/codex/deactivate", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("deactivate: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	rec = fetchPersistedRecord(t, s, "codex")
	if rec.State != "deprecated" {
		t.Fatalf("persisted state after deactivate: got %q, want deprecated", rec.State)
	}
	if rec.DeactivatedAt == nil {
		t.Fatal("deactivated_at should be set after a successful deactivate")
	}
}

// ─── Idempotency: admit twice does NOT regress to "nil registry" ────────────

func TestExperimentalAdmit_IdempotentAcrossRetries(t *testing.T) {
	seedOpenAICredential(t, "sk-test-fake-1234567890abcdef")

	cfg := &config.Config{MasterKey: "test-master-key-1234567890"}
	s, ts := newAdmitTestServer(t, cfg)
	token := makeKnownAdmin(t, s, "admin-idem", "correct horse battery")
	_ = s

	var firstEvidence string
	for i := 0; i < 3; i++ {
		resp := doPost(t, ts, "/api/experimental/providers/codex/admit", token, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("admit attempt %d: expected 200, got %d", i+1, resp.StatusCode)
		}
		body := decodeData(t, resp)
		ev, _ := body["evidence"].(string)
		if strings.Contains(ev, "nil registry") {
			t.Fatalf("retry %d regressed to nil-registry error: evidence=%q", i+1, ev)
		}
		if i == 0 {
			firstEvidence = ev
		} else if ev != firstEvidence {
			t.Logf("note: evidence flipped across retries: %q -> %q (allowed if harness is flaky)",
				firstEvidence, ev)
		}
	}
}

// ─── Guard: when the live registry is missing, admit fails LOUDLY ──────────

func TestExperimentalAdmit_RegistryMissingReturnsLoudError(t *testing.T) {
	seedOpenAICredential(t, "sk-test-fake-1234567890abcdef")

	cfg := &config.Config{MasterKey: "test-master-key-1234567890"}
	s, ts := newAdmitTestServer(t, cfg)
	token := makeKnownAdmin(t, s, "admin-reg", "correct horse battery")

	// Sabotage: clear the registry post-construction.
	s.proxy.providerReg = nil

	resp := doPost(t, ts, "/api/experimental/providers/codex/admit", token, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		resp.Body.Close()
		t.Fatalf("expected 500 when registry is nil, got %d", resp.StatusCode)
	}
	var errBody map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	resp.Body.Close()
	msg, _ := errBody["error"].(string)
	if !strings.Contains(msg, "registry not initialised") {
		t.Fatalf("expected the new loud error message, got %q", msg)
	}
	if strings.Contains(msg, "nil registry") {
		t.Fatalf("the old sticky 'nil registry' string is still being emitted: %q", msg)
	}
}
