package auth

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/oauthide"
)

// IdeOAuthDisclaimer is shown in the dashboard before starting an IDE OAuth flow.
const IdeOAuthDisclaimer = oauthide.RiskNotice + `

Experimental — personal / lab use only.
• Tokens are stored in your Lintasan database (treat backups as sensitive).
• This is NOT a substitute for official API keys or billing.
• Lintasan does not grant rights to resell or multi-tenant host someone else's subscription.`

// IsIdeOAuthProvider reports whether name is in the 9router OAuth catalog.
func IsIdeOAuthProvider(name string) bool {
	return oauthide.IsKnownProvider(name)
}

// GetPendingSession loads a session that must still be in pending status (callback gate).
func (m *OAuthManager) GetPendingSession(id string) (*OAuthSession, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("oauth manager not configured")
	}
	var s OAuthSession
	var accessToken, refreshToken, expiresAtStr, createdAt, pkce sql.NullString
	err := m.db.Conn().QueryRow(
		`SELECT id, provider, access_token, refresh_token, expires_at, status, created_at, pkce_verifier FROM oauth_sessions WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.Provider, &accessToken, &refreshToken, &expiresAtStr, &s.Status, &createdAt, &pkce)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if accessToken.Valid {
		s.AccessToken = accessToken.String
	}
	if refreshToken.Valid {
		s.RefreshToken = refreshToken.String
	}
	if pkce.Valid {
		s.PKCEVerifier = pkce.String
	}
	if expiresAtStr.Valid && expiresAtStr.String != "" {
		s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr.String)
	}
	if createdAt.Valid {
		s.CreatedAt = createdAt.String
	}
	if s.Status != "pending" {
		return nil, fmt.Errorf("session %s is not pending (status=%s)", id, s.Status)
	}
	if !s.ExpiresAt.IsZero() && time.Now().After(s.ExpiresAt) {
		return nil, fmt.Errorf("session %s expired", id)
	}
	return &s, nil
}

// SetSessionPKCE stores PKCE verifier for pending PKCE flows.
func (m *OAuthManager) SetSessionPKCE(id, verifier string) error {
	_, err := m.db.Conn().Exec(`UPDATE oauth_sessions SET pkce_verifier = ? WHERE id = ? AND status = 'pending'`, verifier, id)
	return err
}