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
		out["hint"] = "Providers with implementation=ready support browser authorize. Others show planned flow type until ported."
	}
	writeJSON(w, out)
}

func startOAuthAuthorize(s *Server, provider, sessionID, publicBase string) (redirectURL string, err error) {
	meta := oauthide.ByID(provider)
	if meta == nil {
		return "", fmt.Errorf("unknown provider %s", provider)
	}
	switch provider {
	case "xai":
		if meta.Impl != oauthide.ImplReady {
			return "", fmt.Errorf("xai not ready")
		}
		pkce, err := oauthide.NewPKCE(oauthide.XAIPKCEBytes)
		if err != nil {
			return "", err
		}
		if err := s.oauthMgr.SetSessionPKCE(sessionID, pkce.Verifier); err != nil {
			return "", err
		}
		redirect := publicBase + "/api/oauth/callback/xai"
		return oauthide.BuildXAIAuthorizeURL(redirect, sessionID, pkce.Challenge), nil
	default:
		return "", fmt.Errorf("provider %s not implemented yet (flow=%s). Catalog lists all 8; port in progress.", provider, meta.Flow)
	}
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