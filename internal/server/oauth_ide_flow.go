package server

import (
	"fmt"
	"net/http"

	"github.com/sanhaji182/lintasan-go/internal/auth"
	"github.com/sanhaji182/lintasan-go/internal/oauthide"
)

func (s *Server) handleOAuthStatus(w http.ResponseWriter, r *http.Request) {
	catalog := oauthide.Catalog()
	enabled := s.oauthIdeEnabled()
	out := map[string]any{
		"enabled":      enabled,
		"experimental": true,
		"catalog":      catalog,
		"disclaimer":   auth.IdeOAuthDisclaimer,
		"proxy_wired":  false,
		"source":       "9router OAUTH_PROVIDERS v0.4.71 (Go rewrite)",
	}
	if enabled {
		out["public_base"] = s.oauthPublicBaseURL()
		out["hint"] = "ready: xai (browser), github (device code + poll). Others planned."
	}
	writeJSON(w, out)
}

func startOAuthAuthorize(s *Server, provider, sessionID, publicBase string) (redirectURL string, err error) {
	res, err := startOAuthAuthorizeFull(s, provider, sessionID, publicBase)
	if err != nil {
		return "", err
	}
	if res.Flow != "browser_redirect" {
		return "", fmt.Errorf("use device_code response for %s", provider)
	}
	return res.RedirectURL, nil
}

func exchangeOAuthCallback(provider, code, redirectURI, pkceVerifier string) (access, refresh string, expiresIn int, err error) {
	switch provider {
	case "xai":
		tok, err := oauthide.ExchangeXAIToken(code, redirectURI, pkceVerifier)
		if err != nil {
			return "", "", 0, err
		}
		return tok.AccessToken, tok.RefreshToken, tok.ExpiresIn, nil
	default:
		return exchangeIdeOAuthCodeLegacy(provider, code, redirectURI)
	}
}