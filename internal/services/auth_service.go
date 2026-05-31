package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims extends the standard JWT registered claims with our custom fields.
// The Site field is the routing key that every middleware uses to find the
// correct per-site database — no extra headers or context required.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Site   string    `json:"site"` // e.g. "site1.com" — DB routing key
	jwt.RegisteredClaims
}

// GenerateJWT creates and signs an HS256 JWT with a 7-day expiry.
// The site claim is baked into the token so any downstream middleware can
// call GetDB(claims.Site) without any other context.
func GenerateJWT(userID uuid.UUID, email, site, secret string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Site:   site,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(), // jti — unique token ID (useful for blocklisting)
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
	return claims, nil
}
