package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const userKey ctxKey = "keelUser"

// Claims is who is holding the dashboard open.
type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Sign issues a dashboard token.
func Sign(secret, userID, username string) (string, error) {
	c := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// RequireAuth guards everything the dashboard reads and writes.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				http.Error(w, `{"error":"avtorizatsiya kerak"}`, http.StatusUnauthorized)
				return
			}
			var c Claims
			tok, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !tok.Valid {
				http.Error(w, `{"error":"sessiya tugagan"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, &c)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// From returns the signed-in user, if any.
func From(ctx context.Context) *Claims {
	c, _ := ctx.Value(userKey).(*Claims)
	return c
}
