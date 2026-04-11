package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
	"github.com/slhmy/online-judge/bff/internal/identra"
	userpb "github.com/slhmy/online-judge/gen/go/user/v1"
)

// AuthErrorCode represents structured error codes for auth operations
type AuthErrorCode string

const (
	AuthErrorCodeInvalidCredentials AuthErrorCode = "INVALID_CREDENTIALS"
	AuthErrorCodeEmailExists        AuthErrorCode = "EMAIL_EXISTS"
	AuthErrorCodeValidationError    AuthErrorCode = "VALIDATION_ERROR"
	AuthErrorCodeUnauthorized       AuthErrorCode = "UNAUTHORIZED"
	AuthErrorCodeOAuthNotConfigured AuthErrorCode = "OAUTH_NOT_CONFIGURED"
	AuthErrorCodeOAuthStateExpired  AuthErrorCode = "OAUTH_STATE_EXPIRED"
	AuthErrorCodeOAuthFailed        AuthErrorCode = "OAUTH_FAILED"
	AuthErrorCodeTokenInvalid       AuthErrorCode = "TOKEN_INVALID"
	AuthErrorCodeDatabaseError      AuthErrorCode = "DATABASE_ERROR"
	AuthErrorCodeInternalError      AuthErrorCode = "INTERNAL_ERROR"
	AuthErrorCodeTooManyAttempts    AuthErrorCode = "TOO_MANY_ATTEMPTS"
)

const (
	loginMaxFailedAttempts = 5
	loginLockoutDuration   = 15 * time.Minute
)

// AuthErrorResponse represents a structured error response
type AuthErrorResponse struct {
	ErrorCode AuthErrorCode `json:"error_code"`
	Message   string        `json:"message"`
	Field     string        `json:"field,omitempty"`
}

// authErrorHTTPStatus maps error codes to HTTP status codes
var authErrorHTTPStatus = map[AuthErrorCode]int{
	AuthErrorCodeInvalidCredentials: http.StatusUnauthorized,
	AuthErrorCodeEmailExists:        http.StatusConflict,
	AuthErrorCodeValidationError:    http.StatusBadRequest,
	AuthErrorCodeUnauthorized:       http.StatusUnauthorized,
	AuthErrorCodeOAuthNotConfigured: http.StatusBadRequest,
	AuthErrorCodeOAuthStateExpired:  http.StatusBadRequest,
	AuthErrorCodeOAuthFailed:        http.StatusInternalServerError,
	AuthErrorCodeTokenInvalid:       http.StatusUnauthorized,
	AuthErrorCodeDatabaseError:      http.StatusInternalServerError,
	AuthErrorCodeInternalError:      http.StatusInternalServerError,
	AuthErrorCodeTooManyAttempts:    http.StatusTooManyRequests,
}

// writeAuthError writes a structured auth error response
func writeAuthError(w http.ResponseWriter, errorCode AuthErrorCode, message string, field string) {
	resp := AuthErrorResponse{
		ErrorCode: errorCode,
		Message:   message,
		Field:     field,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(authErrorHTTPStatus[errorCode])
	_ = json.NewEncoder(w).Encode(resp)
}

// writeAuthErrorSimple writes a structured auth error response without a field
func writeAuthErrorSimple(w http.ResponseWriter, errorCode AuthErrorCode, message string) {
	writeAuthError(w, errorCode, message, "")
}

// identraClientInterface abstracts the identra gRPC client for testability.
type identraClientInterface interface {
	LoginByPassword(ctx context.Context, email, password string) (*pb.LoginByPasswordResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error)
	GetCurrentUser(ctx context.Context, accessToken string) (*pb.GetCurrentUserLoginInfoResponse, error)
	GetOAuthAuthorizationURL(ctx context.Context, provider string, redirectURL *string) (*pb.GetOAuthAuthorizationURLResponse, error)
	LoginByOAuth(ctx context.Context, code, state string) (*pb.LoginByOAuthResponse, error)
}

type AuthHandler struct {
	identraClient    identraClientInterface
	userClient       userpb.UserServiceClient
	adminEmail       string
	oauthRedirectURL string
	redis            *redis.Client
}

func NewAuthHandler(identraGRPCHost string, userClient userpb.UserServiceClient, adminEmail, oauthRedirectURL string, redisClient *redis.Client) *AuthHandler {
	// Create identra client
	client, err := identra.NewClient(identraGRPCHost)
	if err != nil {
		panic(err)
	}

	return &AuthHandler{
		identraClient:    client,
		userClient:       userClient,
		adminEmail:       adminEmail,
		oauthRedirectURL: oauthRedirectURL,
		redis:            redisClient,
	}
}

// Register handles new user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "请求格式错误")
		return
	}

	// Validate input
	if req.Email == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "邮箱不能为空", "email")
		return
	}
	if req.Password == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "密码不能为空", "password")
		return
	}

	if len(req.Password) < 6 {
		writeAuthError(w, AuthErrorCodeValidationError, "密码长度至少为6位", "password")
		return
	}

	ctx := context.Background()

	// Call LoginByPassword which auto-creates user if not exists
	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		// If login fails, the user may already exist with a different password
		writeAuthError(w, AuthErrorCodeEmailExists, "该邮箱已被注册", "email")
		return
	}

	if resp.Token == nil || resp.Token.AccessToken == nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "注册失败")
		return
	}

	// Get user ID from identra via token
	userInfo, err := h.identraClient.GetCurrentUser(ctx, resp.Token.AccessToken.Token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户信息失败")
		return
	}
	userID := userInfo.UserId

	// Determine role
	role := "user"
	if req.Email == h.adminEmail {
		role = "admin"
	}

	// Create username
	username := req.Username
	if username == "" {
		username = strings.Split(req.Email, "@")[0]
	}

	// Ensure user profile exists via backend
	profileResp, err := h.userClient.EnsureUserProfile(ctx, &userpb.EnsureUserProfileRequest{
		UserId:   userID,
		Email:    req.Email,
		Username: username,
		Role:     role,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
		return
	}

	if !profileResp.Created {
		// Profile already existed — user was already registered
		writeAuthError(w, AuthErrorCodeEmailExists, "该邮箱已被注册", "email")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  resp.Token.AccessToken.Token,
		"refresh_token": resp.Token.RefreshToken.Token,
		"expires_in":    resp.Token.AccessToken.ExpiresAt,
		"user": map[string]interface{}{
			"id":       userID,
			"email":    req.Email,
			"username": profileResp.Profile.Username,
			"role":     profileResp.Profile.Role,
		},
	})
}

// Login handles password login via Identra
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "请求格式错误")
		return
	}

	// Validate input
	if req.Email == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "邮箱不能为空", "email")
		return
	}
	if req.Password == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "密码不能为空", "password")
		return
	}

	ctx := context.Background()

	// Check if this account is temporarily locked due to too many failed attempts
	if locked, retryAfter := h.isLoginLocked(ctx, req.Email); locked {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		writeAuthErrorSimple(w, AuthErrorCodeTooManyAttempts, fmt.Sprintf("Too many failed login attempts. Please try again in %d seconds.", retryAfter))
		return
	}

	// Login via identra
	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		h.recordFailedLogin(ctx, req.Email)
		writeAuthError(w, AuthErrorCodeInvalidCredentials, "邮箱或密码错误", "email")
		return
	}

	if resp.Token == nil || resp.Token.AccessToken == nil {
		h.recordFailedLogin(ctx, req.Email)
		writeAuthError(w, AuthErrorCodeInvalidCredentials, "登录失败", "")
		return
	}

	// Login succeeded — clear any recorded failures
	h.clearFailedLogin(ctx, req.Email)

	// Get user ID from identra via token
	userInfo, err := h.identraClient.GetCurrentUser(ctx, resp.Token.AccessToken.Token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户信息失败")
		return
	}
	userID := userInfo.UserId

	// Determine role
	role := "user"
	if req.Email == h.adminEmail {
		role = "admin"
	}

	// Ensure user profile exists via backend
	profileResp, err := h.userClient.EnsureUserProfile(ctx, &userpb.EnsureUserProfileRequest{
		UserId:   userID,
		Email:    req.Email,
		Username: strings.Split(req.Email, "@")[0],
		Role:     role,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户资料失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  resp.Token.AccessToken.Token,
		"refresh_token": resp.Token.RefreshToken.Token,
		"expires_in":    resp.Token.AccessToken.ExpiresAt,
		"user": map[string]interface{}{
			"id":       userID,
			"email":    req.Email,
			"username": profileResp.Profile.Username,
			"role":     profileResp.Profile.Role,
		},
	})
}

// loginAttemptsKey returns the Redis key used to track failed login attempts for an email.
func loginAttemptsKey(email string) string {
	return fmt.Sprintf("login:attempts:%s", email)
}

// isLoginLocked checks whether the given email is temporarily locked due to too many
// failed login attempts. It returns (true, retryAfterSeconds) when locked.
func (h *AuthHandler) isLoginLocked(ctx context.Context, email string) (bool, int) {
	if h.redis == nil {
		return false, 0
	}

	key := loginAttemptsKey(email)
	val, err := h.redis.Get(ctx, key).Int()
	if err != nil {
		// Key not found or Redis error — allow the request
		return false, 0
	}

	if val >= loginMaxFailedAttempts {
		ttl, err := h.redis.TTL(ctx, key).Result()
		if err != nil || ttl <= 0 {
			return true, int(loginLockoutDuration.Seconds())
		}
		return true, int(ttl.Seconds())
	}

	return false, 0
}

// recordFailedLogin increments the failed-attempt counter for an email.
// Each failed attempt refreshes the lockout window so that a persistent attacker
// must wait the full lockout duration from the last failure.
func (h *AuthHandler) recordFailedLogin(ctx context.Context, email string) {
	if h.redis == nil {
		return
	}

	key := loginAttemptsKey(email)
	pipe := h.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, loginLockoutDuration)
	_, _ = pipe.Exec(ctx)
}

// clearFailedLogin removes the failed-attempt counter after a successful login.
func (h *AuthHandler) clearFailedLogin(ctx context.Context, email string) {
	if h.redis == nil {
		return
	}

	h.redis.Del(ctx, loginAttemptsKey(email))
}

// OAuthURL returns the OAuth authorization URL
func (h *AuthHandler) OAuthURL(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "github"
	}

	ctx := context.Background()
	redirectURL := h.oauthRedirectURL
	resp, err := h.identraClient.GetOAuthAuthorizationURL(ctx, provider, &redirectURL)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeOAuthNotConfigured, "OAuth未配置: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authorization_url": resp.Url,
		"state":             resp.State,
		"provider":          provider,
	})
}

// OAuthCallback handles OAuth callback
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "缺少code或state参数")
		return
	}

	ctx := context.Background()

	// Delegate OAuth login to Identra
	oauthResp, err := h.identraClient.LoginByOAuth(ctx, code, state)
	if err != nil {
		writeAuthError(w, AuthErrorCodeOAuthFailed, "OAuth登录失败: "+err.Error(), "")
		return
	}

	if oauthResp.Token == nil || oauthResp.Token.AccessToken == nil {
		writeAuthErrorSimple(w, AuthErrorCodeOAuthFailed, "OAuth登录失败")
		return
	}

	// Get user ID from identra via token
	userInfo, err := h.identraClient.GetCurrentUser(ctx, oauthResp.Token.AccessToken.Token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户信息失败")
		return
	}
	userID := userInfo.UserId
	email := oauthResp.Email
	username := oauthResp.Username

	// Determine role
	role := "user"
	if email == h.adminEmail {
		role = "admin"
	}

	// Ensure user profile exists via backend
	profileResp, err := h.userClient.EnsureUserProfile(ctx, &userpb.EnsureUserProfileRequest{
		UserId:    userID,
		Email:     email,
		Username:  username,
		Role:      role,
		AvatarUrl: oauthResp.AvatarUrl,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  oauthResp.Token.AccessToken.Token,
		"refresh_token": oauthResp.Token.RefreshToken.Token,
		"expires_in":    oauthResp.Token.AccessToken.ExpiresAt,
		"user": map[string]interface{}{
			"id":         userID,
			"email":      email,
			"username":   profileResp.Profile.Username,
			"role":       profileResp.Profile.Role,
			"avatar_url": profileResp.Profile.AvatarUrl,
		},
	})
}

// Refresh refreshes the access token
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "请求格式错误")
		return
	}

	if req.RefreshToken == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "刷新令牌不能为空", "refresh_token")
		return
	}

	ctx := context.Background()
	resp, err := h.identraClient.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeTokenInvalid, "无效的刷新令牌")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  resp.Token.AccessToken.Token,
		"refresh_token": resp.Token.RefreshToken.Token,
		"expires_in":    resp.Token.AccessToken.ExpiresAt,
	})
}

// Me returns current user info
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeAuthErrorSimple(w, AuthErrorCodeUnauthorized, "未授权访问")
		return
	}

	// Extract token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		writeAuthErrorSimple(w, AuthErrorCodeTokenInvalid, "无效的授权头格式")
		return
	}
	token := parts[1]

	// Get user info from identra
	ctx := context.Background()
	userInfo, err := h.identraClient.GetCurrentUser(ctx, token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeTokenInvalid, "无效的令牌")
		return
	}

	userID := userInfo.UserId
	email := userInfo.Email

	// Get user profile via backend
	profileResp, err := h.userClient.GetUserProfile(ctx, &userpb.GetUserProfileRequest{UserId: userID})

	var username, role string
	if err != nil {
		username = strings.Split(email, "@")[0]
		role = "user"
	} else {
		username = profileResp.Profile.Username
		role = profileResp.Profile.Role
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       userID,
		"email":    email,
		"username": username,
		"role":     role,
	})
}

// Logout handles logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RegisterRoutes registers auth routes
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	r.Get("/auth/oauth/url", h.OAuthURL)
	r.Get("/auth/oauth/callback", h.OAuthCallback)
	r.Post("/auth/refresh", h.Refresh)
	r.Get("/auth/me", h.Me)
	r.Post("/auth/logout", h.Logout)
}
