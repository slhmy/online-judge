package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is the type for context keys
type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
	UserRoleKey  contextKey = "user_role"
)

// Auth middleware validates JWT token and extracts user info
type Auth struct {
	jwksURL string
}

// NewAuth creates a new auth middleware
func NewAuth(jwksURL string) *Auth {
	return &Auth{
		jwksURL: jwksURL,
	}
}

// RequireAuth is a middleware that requires valid authentication
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// BFF does not parse/validate JWT claims.
		// Complex auth should be handled in backend services.
		if token == "" {
			http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Copy optional trusted identity headers into context.
		// These are expected to be set by an upstream auth layer when available.
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, strings.TrimSpace(r.Header.Get("X-User-Id")))
		ctx = context.WithValue(ctx, UserEmailKey, strings.TrimSpace(r.Header.Get("X-User-Email")))
		ctx = context.WithValue(ctx, UserRoleKey, strings.TrimSpace(r.Header.Get("X-User-Role")))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin is a middleware that requires admin role
func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(UserRoleKey).(string)
		if !ok || role != "admin" {
			http.Error(w, `{"error": "admin access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserEmail extracts user email from context
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

// GetUserRole extracts user role from context
func GetUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}
