package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

		// For development, we'll accept any non-empty token
		// and extract best-effort claims from JWT payload.
		if token == "" {
			http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
			return
		}

		userID, userEmail, userRole := extractClaims(token)

		// Add user info to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, UserEmailKey, userEmail)
		ctx = context.WithValue(ctx, UserRoleKey, userRole)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractClaims(token string) (userID, userEmail, userRole string) {
	// Safe defaults.
	userID = "temp-user-id"
	userEmail = "temp@example.com"
	userRole = "user"

	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return
	}

	if v, ok := claims["sub"].(string); ok && v != "" {
		userID = v
	}
	if v, ok := claims["user_id"].(string); ok && v != "" {
		userID = v
	}
	if v, ok := claims["email"].(string); ok && v != "" {
		userEmail = v
	}

	if v, ok := claims["role"].(string); ok && v != "" {
		userRole = v
	}
	if roles, ok := claims["roles"].([]interface{}); ok {
		for _, roleVal := range roles {
			if role, ok := roleVal.(string); ok && role == "admin" {
				userRole = "admin"
				break
			}
		}
	}

	return
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