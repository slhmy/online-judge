package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
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
	redis   *redis.Client
}

// NewAuth creates a new auth middleware
const (
	sessionCookieName = "oj_session"
	sessionKeyPrefix  = "auth:session:"
)

type userSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewAuth(jwksURL string, redisClient *redis.Client) *Auth {
	return &Auth{
		jwksURL: jwksURL,
		redis:   redisClient,
	}
}

// RequireAuth is a middleware that requires valid authentication
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" && a.redis != nil {
			if cookie, err := r.Cookie(sessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
				raw, redisErr := a.redis.Get(r.Context(), sessionKeyPrefix+strings.TrimSpace(cookie.Value)).Result()
				if redisErr == nil {
					var sess userSession
					if jsonErr := json.Unmarshal([]byte(raw), &sess); jsonErr == nil && strings.TrimSpace(sess.AccessToken) != "" {
						authHeader = "Bearer " + strings.TrimSpace(sess.AccessToken)
						r = r.Clone(r.Context())
						r.Header.Set("Authorization", authHeader)
					}
				}
			}
		}

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
