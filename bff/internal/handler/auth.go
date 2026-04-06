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
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/jackc/pgx/v5/stdlib"

	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
	"github.com/online-judge/bff/internal/identra"
)

type AuthHandler struct {
	identraClient      *identra.Client
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error": "email and password required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, `{"error": "password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Check if user already exists
	var existingID string
	err := h.identraDB.QueryRowContext(ctx, `
		SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL
	`, req.Email).Scan(&existingID)

	if err == nil {
		http.Error(w, `{"error": "email already registered"}`, http.StatusConflict)
		return
	}
	if err != sql.ErrNoRows {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}

	// Create user in identra database
	userID := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error": "failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	_, err = h.identraDB.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userID, req.Email, string(hashedPassword))
	if err != nil {
		http.Error(w, `{"error": "failed to create user"}`, http.StatusInternalServerError)
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
		h.identraDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", userID)
		http.Error(w, `{"error": "failed to create user profile"}`, http.StatusInternalServerError)
		return
	}

	// Login the user to get tokens
	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		// User created but login failed - still return success
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "registration successful, please login",
			"user": map[string]interface{}{
				"id":       userID,
				"email":    req.Email,
				"username": username,
				"role":     role,
			},
		})
		return
	}

	if resp.Token != nil && resp.Token.AccessToken != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "registration successful, please login",
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Login via identra
	ctx := context.Background()
	resp, err := h.identraClient.LoginByPassword(ctx, req.Email, req.Password)
	if err != nil {
		http.Error(w, `{"error": "invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if resp.Token == nil || resp.Token.AccessToken == nil {
		http.Error(w, `{"error": "login failed"}`, http.StatusUnauthorized)
		return
	}

	// Get user ID from identra database
	var userID string
	err = h.identraDB.QueryRowContext(ctx, `
		SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL
	`, req.Email).Scan(&userID)

	if err != nil {
		http.Error(w, `{"error": "user not found"}`, http.StatusUnauthorized)
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
			http.Error(w, `{"error": "failed to create user profile"}`, http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
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

// OAuthURL returns the OAuth authorization URL
func (h *AuthHandler) OAuthURL(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "github"
	}

	// Check if OAuth is configured
	if h.githubClientID == "" {
		http.Error(w, `{"error": "OAuth not configured"}`, http.StatusBadRequest)
		return
	}

	// Generate random state token
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, `{"error": "failed to generate state"}`, http.StatusInternalServerError)
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Store state in Redis with 5-minute TTL
	ctx := context.Background()
	key := fmt.Sprintf("oauth:state:%s", state)
	if err := h.redis.Set(ctx, key, state, 5*time.Minute).Err(); err != nil {
		http.Error(w, `{"error": "failed to store state"}`, http.StatusInternalServerError)
		return
	}

	// Build GitHub authorization URL
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email",
		h.githubClientID,
		url.QueryEscape(h.oauthRedirectURL),
		state,
	)

	json.NewEncoder(w).Encode(map[string]interface{}{
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
		http.Error(w, `{"error": "missing code or state"}`, http.StatusBadRequest)
		return
	}

	// Validate state from Redis
	ctx := context.Background()
	key := fmt.Sprintf("oauth:state:%s", state)
	storedState, err := h.redis.Get(ctx, key).Result()
	if err != nil || storedState != state {
		http.Error(w, `{"error": "invalid or expired state"}`, http.StatusBadRequest)
		return
	}

	// Delete state after validation (one-time use)
	h.redis.Del(ctx, key)

	// Exchange code for access token
	tokenResp, err := h.exchangeGitHubCode(code)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to exchange code: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Get user info from GitHub
	githubUser, err := h.getGitHubUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to get user info: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Get user email from GitHub (primary verified email)
	email := githubUser.Email
	if email == "" {
		// Fetch emails separately if not in user info
		emails, err := h.getGitHubEmails(tokenResp.AccessToken)
		if err != nil {
			http.Error(w, `{"error": "failed to get user emails"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error": "no verified email found"}`, http.StatusBadRequest)
		return
	}

	// Find or create user in identra database
	userID, randomPassword, err := h.findOrCreateOAuthUser(ctx, email, githubUser.Login, githubUser.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to create user: %s"}`, err.Error()), http.StatusInternalServerError)
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
			http.Error(w, `{"error": "failed to create user profile"}`, http.StatusInternalServerError)
			return
		}
	} else if err == nil {
		// Update avatar if changed
		_, err = h.db.ExecContext(ctx, `
			UPDATE user_profiles SET avatar_url = $1, updated_at = NOW() WHERE user_id = $2
		`, githubUser.AvatarURL, userID)
		if err != nil {
			http.Error(w, `{"error": "failed to update profile"}`, http.StatusInternalServerError)
			return
		}
		username = existingUsername
	} else if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
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
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":         userID,
					"email":      email,
					"username":   username,
					"role":       role,
					"avatar_url": githubUser.AvatarURL,
				},
				"message": "OAuth login successful, please set a password to complete registration",
			})
			return
		}
	} else {
		// Existing user - they should use their existing password
		// Return user info only, they'll need to login manually
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"id":         userID,
				"email":      email,
				"username":   username,
				"role":       role,
				"avatar_url": githubUser.AvatarURL,
			},
			"message": "Account linked successfully. Please login with your password.",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
		_, err = h.identraDB.ExecContext(ctx, `
			UPDATE users SET updated_at = NOW() WHERE id = $1
		`, userID)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := h.identraClient.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		http.Error(w, `{"error": "invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  resp.Token.AccessToken.Token,
		"refresh_token": resp.Token.RefreshToken.Token,
		"expires_in":    resp.Token.AccessToken.ExpiresAt,
	})
}

// Me returns current user info
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Extract token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, `{"error": "invalid authorization header"}`, http.StatusUnauthorized)
		return
	}
	token := parts[1]

	// Get user info from identra
	ctx := context.Background()
	userInfo, err := h.identraClient.GetCurrentUser(ctx, token)
	if err != nil {
		http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       userID,
		"email":    email,
		"username": username,
		"role":     role,
	})
}

// Logout handles logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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