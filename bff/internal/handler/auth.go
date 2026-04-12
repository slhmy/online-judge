package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
	"github.com/slhmy/online-judge/bff/internal/identra"
	userpb "github.com/slhmy/online-judge/gen/go/user/v1"
)

// AuthErrorCode represents structured error codes for auth operations.
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
	sessionCookieName      = "oj_session"
	sessionKeyPrefix       = "auth:session:"
	sessionTTL             = 7 * 24 * time.Hour
	loginMaxFailedAttempts = 5
	loginLockoutDuration   = 15 * time.Minute
)

// AuthErrorResponse represents a structured error response.
type AuthErrorResponse struct {
	ErrorCode AuthErrorCode `json:"error_code"`
	Message   string        `json:"message"`
	Field     string        `json:"field,omitempty"`
}

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

func writeAuthError(w http.ResponseWriter, errorCode AuthErrorCode, message, field string) {
	resp := AuthErrorResponse{ErrorCode: errorCode, Message: message, Field: field}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(authErrorHTTPStatus[errorCode])
	_ = json.NewEncoder(w).Encode(resp)
}

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

type userSession struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
}

type AuthHandler struct {
	identraClient    identraClientInterface
	userClient       userpb.UserServiceClient
	adminEmail       string
	oauthRedirectURL string
	redis            *redis.Client
}

func NewAuthHandler(identraGRPCHost string, userClient userpb.UserServiceClient, adminEmail, oauthRedirectURL string, redisClient *redis.Client) *AuthHandler {
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

func (h *AuthHandler) newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, sessionID string) {
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

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *AuthHandler) createSession(ctx context.Context, session userSession) (string, error) {
	if h.redis == nil {
		return "", fmt.Errorf("redis unavailable")
	}
	id, err := h.newSessionID()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	if err := h.redis.Set(ctx, sessionKeyPrefix+id, payload, sessionTTL).Err(); err != nil {
		return "", err
	}
	return id, nil
}

func (h *AuthHandler) getSessionFromRequest(r *http.Request) (string, *userSession, error) {
	if h.redis == nil {
		return "", nil, fmt.Errorf("redis unavailable")
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", nil, fmt.Errorf("session missing")
	}
	id := strings.TrimSpace(cookie.Value)
	raw, err := h.redis.Get(r.Context(), sessionKeyPrefix+id).Result()
	if err != nil {
		return id, nil, err
	}
	var session userSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return id, nil, err
	}
	return id, &session, nil
}

func (h *AuthHandler) saveSession(ctx context.Context, sessionID string, session *userSession) error {
	if h.redis == nil {
		return fmt.Errorf("redis unavailable")
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return h.redis.Set(ctx, sessionKeyPrefix+sessionID, payload, sessionTTL).Err()
}

func (h *AuthHandler) issueSessionAndRespond(w http.ResponseWriter, r *http.Request, accessToken, refreshToken string, user map[string]interface{}, expiresAt int64) {
	sessionID, err := h.createSession(r.Context(), userSession{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建会话失败")
		return
	}

	h.setSessionCookie(w, sessionID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user":       user,
		"expires_in": expiresAt,
	})
}

func loginFailureKey(email string) string {
	return "auth:login:fail:" + strings.ToLower(strings.TrimSpace(email))
}

func loginLockKey(email string) string {
	return "auth:login:lock:" + strings.ToLower(strings.TrimSpace(email))
}

func (h *AuthHandler) recordFailedLogin(ctx context.Context, email string) {
	if h.redis == nil {
		return
	}
	failKey := loginFailureKey(email)
	count, _ := h.redis.Incr(ctx, failKey).Result()
	_ = h.redis.Expire(ctx, failKey, loginLockoutDuration).Err()
	if count >= loginMaxFailedAttempts {
		_ = h.redis.Set(ctx, loginLockKey(email), "1", loginLockoutDuration).Err()
	}
}

func (h *AuthHandler) clearFailedLogin(ctx context.Context, email string) {
	if h.redis == nil {
		return
	}
	_ = h.redis.Del(ctx, loginFailureKey(email), loginLockKey(email)).Err()
}

func (h *AuthHandler) isLoginLocked(ctx context.Context, email string) (bool, int64) {
	if h.redis == nil {
		return false, 0
	}
	ttl, err := h.redis.TTL(ctx, loginLockKey(email)).Result()
	if err != nil || ttl <= 0 {
		return false, 0
	}
	return true, int64(ttl.Seconds())
}

// Register handles new user registration.
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
	if req.Email == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "邮箱不能为空", "email")
		return
	}
	if req.Password == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "密码不能为空", "password")
		return
	}

	ctx := context.Background()
	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		writeAuthError(w, AuthErrorCodeEmailExists, "该邮箱已被注册", "email")
		return
	}
	if resp.Token == nil || resp.Token.AccessToken == nil || resp.Token.RefreshToken == nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "注册失败")
		return
	}

	userInfo, err := h.identraClient.GetCurrentUser(ctx, resp.Token.AccessToken.Token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户信息失败")
		return
	}

	role := "user"
	if req.Email == h.adminEmail {
		role = "admin"
	}
	username := req.Username
	if strings.TrimSpace(username) == "" {
		username = strings.Split(req.Email, "@")[0]
	}

	profileResp, err := h.userClient.EnsureUserProfile(ctx, &userpb.EnsureUserProfileRequest{
		UserId:   userInfo.UserId,
		Email:    req.Email,
		Username: username,
		Role:     role,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
		return
	}
	if !profileResp.Created {
		writeAuthError(w, AuthErrorCodeEmailExists, "该邮箱已被注册", "email")
		return
	}

	h.issueSessionAndRespond(w, r, resp.Token.AccessToken.Token, resp.Token.RefreshToken.Token, map[string]interface{}{
		"id":       userInfo.UserId,
		"email":    req.Email,
		"username": profileResp.Profile.Username,
		"role":     profileResp.Profile.Role,
	}, resp.Token.AccessToken.ExpiresAt)
}

// Login handles password login via Identra.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "请求格式错误")
		return
	}
	if req.Email == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "邮箱不能为空", "email")
		return
	}
	if req.Password == "" {
		writeAuthError(w, AuthErrorCodeValidationError, "密码不能为空", "password")
		return
	}

	ctx := context.Background()
	if locked, retryAfter := h.isLoginLocked(ctx, req.Email); locked {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		writeAuthErrorSimple(w, AuthErrorCodeTooManyAttempts, fmt.Sprintf("Too many failed login attempts. Please try again in %d seconds.", retryAfter))
		return
	}

	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		h.recordFailedLogin(ctx, req.Email)
		writeAuthError(w, AuthErrorCodeInvalidCredentials, "邮箱或密码错误", "email")
		return
	}
	if resp.Token == nil || resp.Token.AccessToken == nil || resp.Token.RefreshToken == nil {
		h.recordFailedLogin(ctx, req.Email)
		writeAuthError(w, AuthErrorCodeInvalidCredentials, "登录失败", "")
		return
	}
	h.clearFailedLogin(ctx, req.Email)

	userInfo, err := h.identraClient.GetCurrentUser(ctx, resp.Token.AccessToken.Token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户信息失败")
		return
	}

	role := "user"
	if req.Email == h.adminEmail {
		role = "admin"
	}

	profileResp, err := h.userClient.EnsureUserProfile(ctx, &userpb.EnsureUserProfileRequest{
		UserId:   userInfo.UserId,
		Email:    req.Email,
		Username: strings.Split(req.Email, "@")[0],
		Role:     role,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "登录失败")
		return
	}

	h.issueSessionAndRespond(w, r, resp.Token.AccessToken.Token, resp.Token.RefreshToken.Token, map[string]interface{}{
		"id":       userInfo.UserId,
		"email":    req.Email,
		"username": profileResp.Profile.Username,
		"role":     profileResp.Profile.Role,
	}, resp.Token.AccessToken.ExpiresAt)
}

// OAuthURL returns the OAuth authorization URL.
func (h *AuthHandler) OAuthURL(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.oauthRedirectURL) == "" {
		writeAuthErrorSimple(w, AuthErrorCodeOAuthNotConfigured, "OAuth回调地址未配置")
		return
	}

	ctx := context.Background()
	resp, err := h.identraClient.GetOAuthAuthorizationURL(ctx, "github", &h.oauthRedirectURL)
	if err != nil {
		writeAuthError(w, AuthErrorCodeOAuthFailed, "获取OAuth链接失败: "+err.Error(), "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": resp.Url})
}

// OAuthCallback handles OAuth callback flow.
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || state == "" {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "缺少code或state参数")
		return
	}

	ctx := context.Background()
	oauthResp, err := h.identraClient.LoginByOAuth(ctx, code, state)
	if err != nil {
		writeAuthError(w, AuthErrorCodeOAuthFailed, "OAuth登录失败: "+err.Error(), "")
		return
	}
	if oauthResp.Token == nil || oauthResp.Token.AccessToken == nil || oauthResp.Token.RefreshToken == nil {
		writeAuthErrorSimple(w, AuthErrorCodeOAuthFailed, "OAuth登录失败")
		return
	}

	userInfo, err := h.identraClient.GetCurrentUser(ctx, oauthResp.Token.AccessToken.Token)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户信息失败")
		return
	}

	role := "user"
	if oauthResp.Email == h.adminEmail {
		role = "admin"
	}

	profileResp, err := h.userClient.EnsureUserProfile(ctx, &userpb.EnsureUserProfileRequest{
		UserId:    userInfo.UserId,
		Email:     oauthResp.Email,
		Username:  oauthResp.Username,
		Role:      role,
		AvatarUrl: oauthResp.AvatarUrl,
	})
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
		return
	}

	h.issueSessionAndRespond(w, r, oauthResp.Token.AccessToken.Token, oauthResp.Token.RefreshToken.Token, map[string]interface{}{
		"id":         userInfo.UserId,
		"email":      oauthResp.Email,
		"username":   profileResp.Profile.Username,
		"role":       profileResp.Profile.Role,
		"avatar_url": profileResp.Profile.AvatarUrl,
	}, oauthResp.Token.AccessToken.ExpiresAt)
}

// Refresh refreshes the access token.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "请求格式错误")
		return
	}

	sessionID, session, sessionErr := h.getSessionFromRequest(r)
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		if sessionErr != nil || session == nil || strings.TrimSpace(session.RefreshToken) == "" {
			writeAuthError(w, AuthErrorCodeValidationError, "刷新令牌不能为空", "refresh_token")
			return
		}
		refreshToken = session.RefreshToken
	}

	ctx := context.Background()
	resp, err := h.identraClient.RefreshToken(ctx, refreshToken)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeTokenInvalid, "无效的刷新令牌")
		return
	}

	if session != nil && sessionErr == nil {
		session.AccessToken = resp.Token.AccessToken.Token
		session.RefreshToken = resp.Token.RefreshToken.Token
		_ = h.saveSession(ctx, sessionID, session)
		h.setSessionCookie(w, sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  resp.Token.AccessToken.Token,
		"refresh_token": resp.Token.RefreshToken.Token,
		"expires_in":    resp.Token.AccessToken.ExpiresAt,
	})
}

// Me returns current user info.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if _, session, err := h.getSessionFromRequest(r); err == nil && session != nil && session.User != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(session.User)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeAuthErrorSimple(w, AuthErrorCodeUnauthorized, "未授权访问")
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		writeAuthErrorSimple(w, AuthErrorCodeTokenInvalid, "无效的授权头格式")
		return
	}
	accessToken := parts[1]

	ctx := context.Background()
	userInfo, err := h.identraClient.GetCurrentUser(ctx, accessToken)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeTokenInvalid, "无效的令牌")
		return
	}

	profileResp, err := h.userClient.GetUserProfile(ctx, &userpb.GetUserProfileRequest{UserId: userInfo.UserId})
	username := strings.Split(userInfo.Email, "@")[0]
	role := "user"
	if err == nil && profileResp != nil && profileResp.Profile != nil {
		username = profileResp.Profile.Username
		role = profileResp.Profile.Role
	}

	userPayload := map[string]interface{}{
		"id":       userInfo.UserId,
		"email":    userInfo.Email,
		"username": username,
		"role":     role,
	}

	if sessionID, session, err := h.getSessionFromRequest(r); err == nil && session != nil {
		session.User = userPayload
		_ = h.saveSession(ctx, sessionID, session)
		h.setSessionCookie(w, sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userPayload)
}

// Logout clears server-side session and cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.redis != nil {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			sessionID := strings.TrimSpace(cookie.Value)
			if sessionID != "" {
				_ = h.redis.Del(r.Context(), sessionKeyPrefix+sessionID).Err()
			}
		}
	}
	h.clearSessionCookie(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "已退出登录"})
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	r.Get("/auth/oauth/url", h.OAuthURL)
	r.Get("/auth/oauth/callback", h.OAuthCallback)
	r.Post("/auth/refresh", h.Refresh)
	r.Get("/auth/me", h.Me)
	r.Post("/auth/logout", h.Logout)
}
