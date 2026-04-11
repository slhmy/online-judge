package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/online-judge/bff/internal/identra"
	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
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
}

type AuthHandler struct {
	identraClient      identraClientInterface
	db                 *sql.DB
	identraDB          *sql.DB
	adminEmail         string
	githubClientID     string
	githubClientSecret string
	oauthRedirectURL   string
	redis              *redis.Client
}

func NewAuthHandler(identraGRPCHost, identraHTTPHost, databaseURL, adminEmail, githubClientID, githubClientSecret, oauthRedirectURL string, redisClient *redis.Client) *AuthHandler {
	// Connect to main database for user profile management
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		panic(err)
	}

	// Connect to identra database for user management
	// identra database is on the same postgres instance
	identraDBURL := strings.Replace(databaseURL, "/oj?", "/identra?", 1)
	identraDB, err := sql.Open("pgx", identraDBURL)
	if err != nil {
		panic(err)
	}

	// Create identra client
	client, err := identra.NewClient(identraGRPCHost)
	if err != nil {
		panic(err)
	}

	return &AuthHandler{
		identraClient:      client,
		db:                 db,
		identraDB:          identraDB,
		adminEmail:         adminEmail,
		githubClientID:     githubClientID,
		githubClientSecret: githubClientSecret,
		oauthRedirectURL:   oauthRedirectURL,
		redis:              redisClient,
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

	// Check if user already exists
	var existingID string
	err := h.identraDB.QueryRowContext(ctx, `
		SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL
	`, req.Email).Scan(&existingID)

	if err == nil {
		writeAuthError(w, AuthErrorCodeEmailExists, "该邮箱已被注册", "email")
		return
	}
	if err != sql.ErrNoRows {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "查询用户失败")
		return
	}

	// Create user in identra database
	userID := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "密码处理失败")
		return
	}

	_, err = h.identraDB.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userID, req.Email, string(hashedPassword))
	if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户失败")
		return
	}

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

	// Create user profile in main database
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO user_profiles (user_id, username, role, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userID, username, role)
	if err != nil {
		// Try to clean up identra user
		_, _ = h.identraDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", userID)
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
		return
	}

	// Login the user to get tokens
	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		// User created but login failed - still return success
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "注册成功，请登录",
			"user": map[string]interface{}{
				"id":       userID,
				"email":    req.Email,
				"username": username,
				"role":     role,
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if resp.Token != nil && resp.Token.AccessToken != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  resp.Token.AccessToken.Token,
			"refresh_token": resp.Token.RefreshToken.Token,
			"expires_in":    resp.Token.AccessToken.ExpiresAt,
			"user": map[string]interface{}{
				"id":       userID,
				"email":    req.Email,
				"username": username,
				"role":     role,
			},
		})
	} else {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "注册成功，请登录",
		})
	}
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

	// Get user ID from identra database
	var userID string
	err = h.identraDB.QueryRowContext(ctx, `
		SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL
	`, req.Email).Scan(&userID)

	if err != nil {
		writeAuthError(w, AuthErrorCodeInvalidCredentials, "用户不存在", "email")
		return
	}

	// Check if user profile exists
	var username string
	var role string
	err = h.db.QueryRowContext(ctx, `
		SELECT username, role FROM user_profiles WHERE user_id = $1
	`, userID).Scan(&username, &role)

	if err == sql.ErrNoRows {
		// Create user profile
		username = strings.Split(req.Email, "@")[0]
		role = "user"
		if req.Email == h.adminEmail {
			role = "admin"
		}

		_, err = h.db.ExecContext(ctx, `
			INSERT INTO user_profiles (user_id, username, role, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, userID, username, role)
		if err != nil {
			writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
			return
		}
	} else if err != nil {
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
			"username": username,
			"role":     role,
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

	// Check if OAuth is configured
	if h.githubClientID == "" {
		writeAuthErrorSimple(w, AuthErrorCodeOAuthNotConfigured, "OAuth未配置")
		return
	}

	// Generate random state token
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "生成状态令牌失败")
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Store state in Redis with 5-minute TTL
	ctx := context.Background()
	key := fmt.Sprintf("oauth:state:%s", state)
	if err := h.redis.Set(ctx, key, state, 5*time.Minute).Err(); err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "存储状态令牌失败")
		return
	}

	// Build GitHub authorization URL
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email",
		h.githubClientID,
		url.QueryEscape(h.oauthRedirectURL),
		state,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authorization_url": authURL,
		"state":             state,
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

	// Validate state from Redis
	ctx := context.Background()
	key := fmt.Sprintf("oauth:state:%s", state)
	storedState, err := h.redis.Get(ctx, key).Result()
	if err != nil || storedState != state {
		writeAuthErrorSimple(w, AuthErrorCodeOAuthStateExpired, "无效或过期的状态令牌")
		return
	}

	// Delete state after validation (one-time use)
	h.redis.Del(ctx, key)

	// Exchange code for access token
	tokenResp, err := h.exchangeGitHubCode(code)
	if err != nil {
		writeAuthError(w, AuthErrorCodeOAuthFailed, "交换授权码失败: "+err.Error(), "")
		return
	}

	// Get user info from GitHub
	githubUser, err := h.getGitHubUserInfo(tokenResp.AccessToken)
	if err != nil {
		writeAuthError(w, AuthErrorCodeOAuthFailed, "获取用户信息失败: "+err.Error(), "")
		return
	}

	// Get user email from GitHub (primary verified email)
	email := githubUser.Email
	if email == "" {
		// Fetch emails separately if not in user info
		emails, err := h.getGitHubEmails(tokenResp.AccessToken)
		if err != nil {
			writeAuthErrorSimple(w, AuthErrorCodeOAuthFailed, "获取用户邮箱失败")
			return
		}
		// Find primary verified email
		for _, e := range emails {
			if e.Primary && e.Verified {
				email = e.Email
				break
			}
		}
		// Fallback to first verified email
		if email == "" {
			for _, e := range emails {
				if e.Verified {
					email = e.Email
					break
				}
			}
		}
	}

	if email == "" {
		writeAuthErrorSimple(w, AuthErrorCodeValidationError, "未找到已验证的邮箱")
		return
	}

	// Find or create user in identra database
	userID, randomPassword, err := h.findOrCreateOAuthUser(ctx, email, githubUser.Login, githubUser.ID)
	if err != nil {
		writeAuthError(w, AuthErrorCodeInternalError, "创建用户失败: "+err.Error(), "")
		return
	}

	// Create or update user profile
	username := githubUser.Login
	role := "user"
	if email == h.adminEmail {
		role = "admin"
	}

	// Check if profile exists
	var existingUsername string
	err = h.db.QueryRowContext(ctx, `
		SELECT username FROM user_profiles WHERE user_id = $1
	`, userID).Scan(&existingUsername)

	if err == sql.ErrNoRows {
		// Create profile
		_, err = h.db.ExecContext(ctx, `
			INSERT INTO user_profiles (user_id, username, role, avatar_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, userID, username, role, githubUser.AvatarURL)
		if err != nil {
			writeAuthErrorSimple(w, AuthErrorCodeInternalError, "创建用户资料失败")
			return
		}
	} else if err == nil {
		// Update avatar if changed
		_, err = h.db.ExecContext(ctx, `
			UPDATE user_profiles SET avatar_url = $1, updated_at = NOW() WHERE user_id = $2
		`, githubUser.AvatarURL, userID)
		if err != nil {
			writeAuthErrorSimple(w, AuthErrorCodeInternalError, "更新用户资料失败")
			return
		}
		username = existingUsername
	} else if err != nil {
		writeAuthErrorSimple(w, AuthErrorCodeInternalError, "获取用户资料失败")
		return
	}

	// Login via identra to get tokens
	// For new OAuth users, we use LoginByPassword with the random password we generated
	// For existing users, they should use their existing password or we return user info only
	var loginResp *pb.LoginByPasswordResponse
	if randomPassword != "" {
		// New user - login with the random password we set
		loginResp, err = h.identraClient.LoginByPassword(ctx, email, randomPassword)
		if err != nil {
			// Fallback: return user info without tokens
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":         userID,
					"email":      email,
					"username":   username,
					"role":       role,
					"avatar_url": githubUser.AvatarURL,
				},
				"message": "OAuth登录成功，请设置密码完成注册",
			})
			return
		}
	} else {
		// Existing user - they should use their existing password
		// Return user info only, they'll need to login manually
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"id":         userID,
				"email":      email,
				"username":   username,
				"role":       role,
				"avatar_url": githubUser.AvatarURL,
			},
			"message": "账号已关联，请使用密码登录",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  loginResp.Token.AccessToken.Token,
		"refresh_token": loginResp.Token.RefreshToken.Token,
		"expires_in":    loginResp.Token.AccessToken.ExpiresAt,
		"user": map[string]interface{}{
			"id":         userID,
			"email":      email,
			"username":   username,
			"role":       role,
			"avatar_url": githubUser.AvatarURL,
		},
	})
}

// GitHubTokenResponse represents GitHub OAuth token response
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// GitHubUser represents GitHub user info
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubEmail represents GitHub email info
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// exchangeGitHubCode exchanges OAuth code for access token
func (h *AuthHandler) exchangeGitHubCode(code string) (*GitHubTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", h.githubClientID)
	data.Set("client_secret", h.githubClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", h.oauthRedirectURL)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp GitHubTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response: %s", string(body))
	}

	return &tokenResp, nil
}

// getGitHubUserInfo fetches user info from GitHub API
func (h *AuthHandler) getGitHubUserInfo(accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// getGitHubEmails fetches user emails from GitHub API
func (h *AuthHandler) getGitHubEmails(accessToken string) ([]GitHubEmail, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var emails []GitHubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, err
	}

	return emails, nil
}

// findOrCreateOAuthUser finds existing user by email or creates new OAuth user
// Returns userID and the random password (for new users only, empty for existing)
func (h *AuthHandler) findOrCreateOAuthUser(ctx context.Context, email, githubLogin string, githubID int) (string, string, error) {
	// Check if user exists with this email
	var userID string
	var hashedPassword string
	err := h.identraDB.QueryRowContext(ctx, `
		SELECT id, hashed_password FROM users WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(&userID, &hashedPassword)

	if err == nil {
		// User exists - return their ID (no password needed for existing users)
		_, updateErr := h.identraDB.ExecContext(ctx, `
			UPDATE users SET updated_at = NOW() WHERE id = $1
		`, userID)
		if updateErr != nil {
			return "", "", updateErr
		}
		return userID, "", nil
	}

	if err != sql.ErrNoRows {
		return "", "", err
	}

	// Create new user
	userID = uuid.New().String()

	// Generate a random password for OAuth users (they won't use it)
	randomPassword := uuid.New().String()
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	_, err = h.identraDB.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userID, email, string(hashedPass))
	if err != nil {
		return "", "", err
	}

	return userID, randomPassword, nil
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

	// Get user profile
	var username string
	var role string
	err = h.db.QueryRowContext(ctx, `
		SELECT username, role FROM user_profiles WHERE user_id = $1
	`, userID).Scan(&username, &role)

	if err != nil {
		username = strings.Split(email, "@")[0]
		role = "user"
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
