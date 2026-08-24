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
	// Which machine this is, for a till device token.
	//
	// ⚠️ **Added because "how many registers does this branch have" had no
	// answer.** Every monoblock used to carry an identical branch token, so the
	// registers a plan is sold by could not be counted, and the panel's own
	// rotate button had to kill every till in the building because there was
	// nothing finer to revoke. A device id makes both possible without changing
	// what the token authorises: it still says "this machine belongs to that
	// branch", now it also says which machine.
	//
	// Empty on every other token, and on till tokens issued before this — see
	// tillDeviceBranch, which treats an unnamed device as one that predates the
	// registry rather than as an invalid one.
	Dev string `json:"dev,omitempty"`
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

// GenerateDevice issues a long-lived token that also names the machine holding
// it, so one till can be counted and revoked without touching its neighbours.
func GenerateDevice(secret, userID, role, dev string, ver int, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		Ver:    ver,
		Dev:    dev,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
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
