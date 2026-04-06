package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/online-judge/bff/internal/identra"
)

type AuthHandler struct {
	identraClient *identra.Client
	db            *sql.DB
	identraDB     *sql.DB
	adminEmail    string
}

func NewAuthHandler(identraGRPCHost, identraHTTPHost, databaseURL, adminEmail string) *AuthHandler {
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
		identraClient: client,
		db:            db,
		identraDB:     identraDB,
		adminEmail:    adminEmail,
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

// OAuthURL returns the OAuth authorization URL (placeholder)
func (h *AuthHandler) OAuthURL(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "OAuth not configured",
	})
}

// OAuthCallback handles OAuth callback (placeholder)
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "OAuth not configured",
	})
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