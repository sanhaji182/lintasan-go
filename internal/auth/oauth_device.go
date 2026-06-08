package auth

import (
	"database/sql"
	"fmt"
	"time"
)

// GetSessionByID loads session regardless of status (device poll).
func (m *OAuthManager) GetSessionByID(id string) (*OAuthSession, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("oauth manager not configured")
	}
	var s OAuthSession
	var accessToken, refreshToken, expiresAtStr, createdAt, pkce, device, flowMeta sql.NullString
	err := m.db.Conn().QueryRow(
		`SELECT id, provider, access_token, refresh_token, expires_at, status, created_at, pkce_verifier, device_code, flow_meta FROM oauth_sessions WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.Provider, &accessToken, &refreshToken, &expiresAtStr, &s.Status, &createdAt, &pkce, &device, &flowMeta)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
	if device.Valid {
		s.DeviceCode = device.String
	}
	if flowMeta.Valid {
		s.FlowMeta = flowMeta.String
	}
	if expiresAtStr.Valid && expiresAtStr.String != "" {
		s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr.String)
	}
	if createdAt.Valid {
		s.CreatedAt = createdAt.String
	}
	return &s, nil
}

// SetSessionDevice stores GitHub (etc.) device_code + UI meta JSON.
func (m *OAuthManager) SetSessionDevice(id, deviceCode, flowMetaJSON string) error {
	_, err := m.db.Conn().Exec(
		`UPDATE oauth_sessions SET device_code = ?, flow_meta = ?, expires_at = ? WHERE id = ? AND status = 'pending'`,
		deviceCode, flowMetaJSON, time.Now().Add(15*time.Minute).Format(time.RFC3339), id,
	)
	return err
}

// UpdateSessionTokensWithMeta activates session with optional flow_meta (copilot json).
func (m *OAuthManager) UpdateSessionTokensWithMeta(id, accessToken, refreshToken string, expiresAt time.Time, flowMeta string) error {
	_, err := m.db.Conn().Exec(
		`UPDATE oauth_sessions SET access_token = ?, refresh_token = ?, expires_at = ?, status = 'active', flow_meta = ? WHERE id = ?`,
		accessToken, refreshToken, expiresAt.Format(time.RFC3339), flowMeta, id,
	)
	return err
}