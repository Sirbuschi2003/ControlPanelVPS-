package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
)

type contextKey string

const ClaimsKey contextKey = "claims"

// Auth validates the Bearer token and injects the claims into the request context.
// For WebSocket connections the browser cannot set headers, so the token may
// also be passed as ?token=... query parameter.
func Auth(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			header := r.Header.Get("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimPrefix(header, "Bearer ")
			} else if t := r.URL.Query().Get("token"); t != "" {
				// WebSocket fallback: token passed as query parameter
				token = t
			}
			if token == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims, err := authSvc.ValidateToken(token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims extracts JWT claims from the request context.
// Returns nil if no claims are present (i.e. the Auth middleware wasn't applied).
func GetClaims(r *http.Request) *services.Claims {
	claims, _ := r.Context().Value(ClaimsKey).(*services.Claims)
	return claims
}

// AdminOnly rejects requests from non-admin users with 403 Forbidden.
// Must be used inside an Auth-protected route group.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsKey).(*services.Claims)
		if !ok || claims.Role != "admin" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
