package server

import "testing"

func TestIsSensitiveSettingKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"master_key", true},
		{"jwt_secret", true},
		{"some_api_key", true},
		{"openai_apikey", true},
		{"user_password", true},
		{"refresh_token", true},
		{"tls_private_key", true},
		{"JWT_SECRET", true}, // case-insensitive
		// non-sensitive keys must NOT be masked
		{"log_level", false},
		{"lb_strategy", false},
		{"thinking_mode", false},
		{"key", false}, // exact "key" is not a secret marker
		{"aliases", false},
	}
	for _, c := range cases {
		if got := isSensitiveSettingKey(c.key); got != c.want {
			t.Errorf("isSensitiveSettingKey(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestMaskSecretValue(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"short", "***"},          // <= 8 chars fully redacted
		{"12345678", "***"},       // exactly 8 redacted
		{"123456789", "1234...6789"}, // 9 chars -> first4...last4
		{"ttH5verylongmasterkeyzsMb", "ttH5...zsMb"},
	}
	for _, c := range cases {
		if got := maskSecretValue(c.in); got != c.want {
			t.Errorf("maskSecretValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestMaskSecretValueNeverLeaksMiddle guards against a masked value ever
// containing the full original secret.
func TestMaskSecretValueNeverLeaksMiddle(t *testing.T) {
	secret := "jwtSecretAbCdEfGhIjKlMnOpQrStUvWxYz0123456789"
	masked := maskSecretValue(secret)
	if masked == secret {
		t.Fatal("masked value equals original secret")
	}
	if len(masked) >= len(secret) {
		t.Fatalf("masked value not shorter than secret: %q", masked)
	}
}
