package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload for authenticated admin users.
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	// Revocation counter, used by the branch kiosk token: bumping the branch's
	// version invalidates every token issued before it. Passwords can be
	// changed, but a token pinned to a tablet on a wall has no password to
	// change — this is how it gets taken away.
	Ver int `json:"ver,omitempty"`
	jwt.RegisteredClaims
}

// Generate issues a signed JWT valid for 7 days.
func Generate(secret, userID, role string) (string, error) {
	return generate(secret, userID, role, 0, 7*24*time.Hour)
}

// GenerateLong issues a token for a device rather than a person — the branch
// kiosk screen, which nobody signs into every morning. It carries a version so
// it can be revoked; see Claims.Ver.
func GenerateLong(secret, userID, role string, ver int, ttl time.Duration) (string, error) {
	return generate(secret, userID, role, ver, ttl)
}

func generate(secret, userID, role string, ver int, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		Ver:    ver,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse validates a token string and returns its claims.
func Parse(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
