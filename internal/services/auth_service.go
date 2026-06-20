package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims extends the standard JWT registered claims with our custom fields.
// The Site field is the routing key that every middleware uses to find the
// correct per-site database — no extra headers or context required.
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Site      string    `json:"site"` // e.g. "site1.com" — DB routing key
	TokenType string    `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// GenerateTokenPair creates both an access token and a refresh token.
func GenerateTokenPair(userID uuid.UUID, email, site, secret string, accessExp, refreshExp time.Duration) (accessToken, refreshToken string, err error) {
	// 1. Generate Access Token
	accessClaims := Claims{
		UserID:    userID,
		Email:     email,
		Site:      site,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	accToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", fmt.Errorf("signing access token: %w", err)
	}

	// 2. Generate Refresh Token
	refreshClaims := Claims{
		UserID:    userID,
		Email:     email,
		Site:      site,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(), // Unique jti for the refresh token
		},
	}
	refToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", fmt.Errorf("signing refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// GenerateJWT creates and signs an HS256 access JWT. (Deprecated: prefer GenerateTokenPair)
func GenerateJWT(userID uuid.UUID, email, site, secret string) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Site:      site,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}
	return signed, nil
}

// ValidateJWT parses and validates a JWT string, returning the embedded Claims.
func ValidateJWT(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Important: This function is called by JWTAuth middleware which expects an access token.
	// If the user tries to pass a refresh token to an access-protected route, reject it here.
	if claims.TokenType == "refresh" {
		return nil, fmt.Errorf("invalid token type: expected access token, got refresh token")
	}

	return claims, nil
}

// ValidateRefreshToken parses and validates a JWT string specifically as a refresh token.
func ValidateRefreshToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type: expected refresh token, got %s", claims.TokenType)
	}

	return claims, nil
}

// ─── Verification Code ────────────────────────────────────────────────────────

// GenerateVerifyCode creates a cryptographically random numeric string of
// exactly `length` digits (e.g. length=8 → "04829173").
// Uses crypto/rand so the output is safe for security-sensitive OTP use.
func GenerateVerifyCode(length int) (string, error) {
	if length < 1 {
		return "", fmt.Errorf("code length must be at least 1")
	}
	digits := make([]byte, length)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generating random digit: %w", err)
		}
		digits[i] = byte('0') + byte(n.Int64())
	}
	return string(digits), nil
}

// ─── JWT Blocklist (in-memory) ────────────────────────────────────────────────

// blocklist maps jti → expiry time. Entries are purged lazily after they expire.
// In production, replace with Redis for multi-instance deployments.
var (
	blocklist   = make(map[string]time.Time)
	blocklistMu sync.Mutex
)

func init() {
	// Background goroutine prunes expired entries every hour.
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			blocklistMu.Lock()
			now := time.Now()
			for jti, exp := range blocklist {
				if now.After(exp) {
					delete(blocklist, jti)
				}
			}
			blocklistMu.Unlock()
		}
	}()
}

// RevokeToken adds a JWT ID (jti) to the blocklist until its natural expiry.
func RevokeToken(jti string, expiresAt time.Time) {
	blocklistMu.Lock()
	defer blocklistMu.Unlock()
	blocklist[jti] = expiresAt
}

// IsTokenRevoked returns true if the given jti has been revoked via logout.
func IsTokenRevoked(jti string) bool {
	blocklistMu.Lock()
	defer blocklistMu.Unlock()
	_, revoked := blocklist[jti]
	return revoked
}

