package auth

import (
	"database/sql"
	"fmt"
	"time"
)

// IdeOAuthDisclaimer is shown in the dashboard before starting an IDE OAuth flow.
const IdeOAuthDisclaimer = `Experimental — personal / lab use only.

• Tokens are stored in your Lintasan database (treat backups as sensitive).
• Upstream providers (Cursor, GitHub Copilot, etc.) may prohibit routing their OAuth tokens through a third-party gateway; account suspension is possible.
• This is NOT a substitute for official API keys or billing. Use at your own risk.
• Lintasan does not grant rights to resell or multi-tenant host someone else's subscription.`

// IdeOAuthProviders lists IDE integrations supported by the lab OAuth flow.
var IdeOAuthProviders = map[string]bool{
	"cursor":         true,
	"codex":          true,
	"claude-desktop": true,
	"copilot":        true,
	"windsurf":       true,
	"aider":          true,
}

// IsIdeOAuthProvider reports whether name is a known IDE OAuth provider slug.
func IsIdeOAuthProvider(name string) bool {
	return IdeOAuthProviders[name]
}

// GetPendingSession loads a session that must still be in pending status (callback gate).
func (m *OAuthManager) GetPendingSession(id string) (*OAuthSession, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("oauth manager not configured")
	}
	var s OAuthSession
	var accessToken, refreshToken, expiresAtStr, createdAt sql.NullString
	err := m.db.Conn().QueryRow(
		`SELECT id, provider, access_token, refresh_token, expires_at, status, created_at FROM oauth_sessions WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.Provider, &accessToken, &refreshToken, &expiresAtStr, &s.Status, &createdAt)
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