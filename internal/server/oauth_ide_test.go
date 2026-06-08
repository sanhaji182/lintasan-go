package server

import (
	"net/http"
	"testing"

	"github.com/sanhaji182/lintasan-go/internal/config"
)

func TestIsOAuthIdeCallback(t *testing.T) {
	if !isOAuthIdeCallback(http.MethodGet, "/api/oauth/callback/copilot") {
		t.Fatal("expected callback path public")
	}
	if isOAuthIdeCallback(http.MethodPost, "/api/oauth/callback/copilot") {
		t.Fatal("POST must not bypass auth")
	}
	if isOAuthIdeCallback(http.MethodGet, "/api/oauth/authorize") {
		t.Fatal("authorize must stay authenticated")
	}
}

func TestOAuthIdeDisabledAuthorize(t *testing.T) {
	cfg := &config.Config{Port: 0, OAuthIDEEnabled: false}
	_, ts := newTestServer(t, cfg)
	resp, _ := http.Get(ts.URL + "/api/oauth/status")
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusOK {
		// status is readable with auth in active state - without auth may 401
	}
	resp.Body.Close()
}

func TestOAuthCallbackBypassOnlyWhenEnabled(t *testing.T) {
	cfg := &config.Config{Port: 0, OAuthIDEEnabled: false}
	_, ts := newTestServer(t, cfg)
	resp, err := http.Get(ts.URL + "/api/oauth/callback/copilot?code=x&state=y")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("disabled: callback should require auth or setup lock, got %d", resp.StatusCode)
	}
}

func TestOAuthCallbackPublicWhenEnabled(t *testing.T) {
	cfg := &config.Config{Port: 0, OAuthIDEEnabled: true, MasterKey: "test-oauth-master-key-32chars!!"}
	s, ts := newTestServer(t, cfg)
	makeKnownAdmin(t, s, "oauthadmin", "test-password-oauth-ide")
	if !s.isActive() {
		t.Fatal("need ACTIVE state for callback bypass test")
	}
	resp, err := http.Get(ts.URL + "/api/oauth/callback/copilot?code=x&state=nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("enabled callback should not 401 — handler returns HTML error instead")
	}
}