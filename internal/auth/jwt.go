package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// --- JWT (pure stdlib, no external deps) ---

// JWTClaims represents the payload of a JWT.
type JWTClaims struct {
	Sub      string `json:"sub"`      // user ID
	Username string `json:"username"` // display name
	Role     string `json:"role"`     // "admin" | "user"
	Iat      int64  `json:"iat"`      // issued at
	Exp      int64  `json:"exp"`      // expiration
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// GenerateJWT creates a signed JWT token using HMAC-SHA256.
func GenerateJWT(claims JWTClaims, secret string) (string, error) {
	if claims.Exp == 0 {
		claims.Exp = time.Now().Add(24 * time.Hour).Unix()
	}
	if claims.Iat == 0 {
		claims.Iat = time.Now().Unix()
	}

	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(claims)

	headerB64 := base64URLEncode(headerJSON)
	payloadB64 := base64URLEncode(payloadJSON)

	signingInput := headerB64 + "." + payloadB64
	signature := hmacSHA256(signingInput, secret)

	return signingInput + "." + signature, nil
}

// ValidateJWT verifies a JWT token and returns the claims.
// Returns the claims and nil error if valid, expired=false, or tampered=false.
func ValidateJWT(token, secret string) (*JWTClaims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256(signingInput, secret)

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	payloadJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return &claims, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func hmacSHA256(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64URLEncode(mac.Sum(nil))
}

// --- Password Hashing (SHA-512 with salt + iterations, pure stdlib) ---

const (
	passwordIterations = 200_000
	saltBytes          = 32
)

// HashPassword hashes a password using multi-round SHA-512 with salt.
// Format: $sha512$200000$<hex_salt>$<hex_hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := hashWithSalt(password, salt)

	return fmt.Sprintf("$sha512$%d$%s$%s",
		passwordIterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	), nil
}

// VerifyPassword checks a password against a stored hash.
func VerifyPassword(password, storedHash string) bool {
	parts := strings.SplitN(storedHash, "$", 5)
	if len(parts) != 5 || parts[0] != "" || parts[1] != "sha512" {
		return false
	}

	var iterations int
	fmt.Sscanf(parts[2], "%d", &iterations)

	salt, err := hex.DecodeString(parts[3])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}

	actualHash := hashWithSaltIterations(password, salt, iterations)
	return hmac.Equal(actualHash, expectedHash)
}

func hashWithSalt(password string, salt []byte) []byte {
	return hashWithSaltIterations(password, salt, passwordIterations)
}

func hashWithSaltIterations(password string, salt []byte, iterations int) []byte {
	// First round: SHA-512(salt + password)
	h := sha512.New()
	h.Write(salt)
	h.Write([]byte(password))
	hash := h.Sum(nil)

	// Subsequent rounds: SHA-512(prev_hash + salt + password)
	for i := 1; i < iterations; i++ {
		h.Reset()
		h.Write(hash)
		h.Write(salt)
		h.Write([]byte(password))
		hash = h.Sum(nil)
	}

	return hash
}
