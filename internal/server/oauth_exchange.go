package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// exchangeIdeOAuthCode exchanges an authorization code for tokens using env-configured OAuth apps.
// No stub tokens: without client_id + client_secret the flow fails closed.
func exchangeIdeOAuthCode(provider, code, publicBase string) (*oauthTokenResponse, error) {
	clientID, clientSecret, tokenURL, err := oauthProviderEndpoints(provider)
	if err != nil {
		return nil, err
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("OAuth app not configured for %s (set LINTASAN_OAUTH_IDE_%s_CLIENT_ID and _CLIENT_SECRET)", provider, envProviderKey(provider))
	}

	redirectURI := publicBase + "/api/oauth/callback/" + provider
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token endpoint HTTP %d: %s", resp.StatusCode, truncateOAuthErr(string(body)))
	}

	var out oauthTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	return &out, nil
}

func envProviderKey(provider string) string {
	return strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
}

func oauthProviderEndpoints(provider string) (clientID, clientSecret, tokenURL string, err error) {
	pk := envProviderKey(provider)
	clientID = os.Getenv("LINTASAN_OAUTH_IDE_" + pk + "_CLIENT_ID")
	clientSecret = os.Getenv("LINTASAN_OAUTH_IDE_" + pk + "_CLIENT_SECRET")
	if v := os.Getenv("LINTASAN_OAUTH_IDE_" + pk + "_TOKEN_URL"); v != "" {
		tokenURL = v
	}
	switch provider {
	case "copilot":
		if tokenURL == "" {
			tokenURL = "https://github.com/login/oauth/access_token"
		}
	case "cursor", "windsurf", "aider", "codex", "claude-desktop":
		if tokenURL == "" {
			return clientID, clientSecret, "", fmt.Errorf("set LINTASAN_OAUTH_IDE_%s_TOKEN_URL to the provider token endpoint (lab — no public Lintasan OAuth app)", pk)
		}
	default:
		return "", "", "", fmt.Errorf("unsupported provider %s", provider)
	}
	return clientID, clientSecret, tokenURL, nil
}

func truncateOAuthErr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}