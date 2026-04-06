package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is the type for context keys
type contextKey string

const (
	UserIDKey  contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
	UserRoleKey contextKey = "user_role"
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

		// TODO: Validate JWT using identra JWKS
		// For now, we'll trust the token and extract user info from claims
		// In production, this should validate the token signature

		// For development, we'll accept any non-empty token
		// and use it as user_id (this should be replaced with proper JWT validation)
		if token == "" {
			http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		// In production, this would be extracted from JWT claims
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, "temp-user-id")
		ctx = context.WithValue(ctx, UserEmailKey, "temp@example.com")
		ctx = context.WithValue(ctx, UserRoleKey, "user")

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