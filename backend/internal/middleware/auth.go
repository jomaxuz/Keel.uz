package middleware

import (
	"context"
	"net/http"
	"strings"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
)

type ctxKey string

const userClaimsKey ctxKey = "userClaims"

// RequireAuth validates the Bearer JWT and injects claims into the context.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				httpx.Error(w, http.StatusUnauthorized, "missing token")
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.Parse(secret, token)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole is RequireAuth plus a role check. Without it every valid token —
// including a customer's SMS-login token — would be accepted by admin routes.
func RequireRole(secret string, roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	authed := RequireAuth(secret)
	return func(next http.Handler) http.Handler {
		return authed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFrom(r.Context())
			if claims == nil || !allowed[claims.Role] {
				httpx.Error(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

// ClaimsFrom extracts the auth claims stored by RequireAuth.
func ClaimsFrom(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(userClaimsKey).(*auth.Claims)
	return c
}
