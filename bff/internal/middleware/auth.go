package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
	"github.com/redis/go-redis/v9"
	"github.com/slhmy/online-judge/bff/internal/identra"
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
	identra identraRefreshClient
}

// NewAuth creates a new auth middleware
const (
	sessionCookieName = "oj_session"
	sessionKeyPrefix  = "auth:session:"
	sessionTTL        = 7 * 24 * time.Hour
)

type userSession struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user,omitempty"`
}

type identraRefreshClient interface {
	RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error)
}

func NewAuth(jwksURL string, redisClient *redis.Client, identraGRPCHost string) *Auth {
	a := &Auth{
		jwksURL: jwksURL,
		redis:   redisClient,
	}

	if strings.TrimSpace(identraGRPCHost) != "" {
		if client, err := identra.NewClient(identraGRPCHost); err == nil {
			a.identra = client
		}
	}

	return a
}

func (a *Auth) maybeRefreshSession(ctx context.Context, sessionID string, sess *userSession) {
	if a.identra == nil || a.redis == nil {
		return
	}
	if strings.TrimSpace(sess.RefreshToken) == "" {
		return
	}

	resp, err := a.identra.RefreshToken(ctx, strings.TrimSpace(sess.RefreshToken))
	if err != nil || resp == nil || resp.Token == nil || resp.Token.AccessToken == nil {
		return
	}

	sess.AccessToken = strings.TrimSpace(resp.Token.AccessToken.Token)
	if resp.Token.RefreshToken != nil && strings.TrimSpace(resp.Token.RefreshToken.Token) != "" {
		sess.RefreshToken = strings.TrimSpace(resp.Token.RefreshToken.Token)
	}

	payload, err := json.Marshal(sess)
	if err != nil {
		return
	}
	_ = a.redis.Set(ctx, sessionKeyPrefix+sessionID, payload, sessionTTL).Err()
}

// RequireAuth is a middleware that requires valid authentication
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" && a.redis != nil {
			if cookie, err := r.Cookie(sessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
				sessionID := strings.TrimSpace(cookie.Value)
				raw, redisErr := a.redis.Get(r.Context(), sessionKeyPrefix+sessionID).Result()
				if redisErr == nil {
					var sess userSession
					if jsonErr := json.Unmarshal([]byte(raw), &sess); jsonErr == nil {
						a.maybeRefreshSession(r.Context(), sessionID, &sess)

						if strings.TrimSpace(sess.AccessToken) != "" {
							authHeader = "Bearer " + strings.TrimSpace(sess.AccessToken)
							r = r.Clone(r.Context())
							r.Header.Set("Authorization", authHeader)

							http.SetCookie(w, &http.Cookie{
								Name:     sessionCookieName,
								Value:    sessionID,
								Path:     "/",
								HttpOnly: true,
								Secure:   false,
								SameSite: http.SameSiteLaxMode,
								MaxAge:   int(sessionTTL.Seconds()),
							})
						}
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
