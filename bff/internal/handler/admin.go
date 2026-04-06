package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/online-judge/bff/internal/middleware"
)

type AdminHandler struct {
	db *sql.DB
}

func NewAdminHandler(databaseURL string) *AdminHandler {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		panic(err)
	}

	return &AdminHandler{db: db}
}

// ListUsers returns all users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT user_id, username, role, rating, solved_count, submission_count, created_at
		FROM user_profiles
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type User struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		Role            string `json:"role"`
		Rating          int    `json:"rating"`
		SolvedCount     int    `json:"solved_count"`
		SubmissionCount int    `json:"submission_count"`
		CreatedAt       string `json:"created_at"`
	}

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Rating, &u.SolvedCount, &u.SubmissionCount, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
	})
}

// UpdateUserRole updates a user's role
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Role != "user" && req.Role != "admin" {
		http.Error(w, `{"error": "invalid role"}`, http.StatusBadRequest)
		return
	}

	// Check if current user is admin
	currentRole := middleware.GetUserRole(r.Context())
	if currentRole != "admin" {
		http.Error(w, `{"error": "forbidden"}`, http.StatusForbidden)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		UPDATE user_profiles SET role = $1, updated_at = NOW() WHERE user_id = $2
	`, req.Role, userID)
	if err != nil {
		http.Error(w, `{"error": "failed to update role"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RegisterRoutes registers admin routes
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/users", h.ListUsers)
		r.Put("/users/{id}/role", h.UpdateUserRole)
	})
}