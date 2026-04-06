package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/online-judge/bff/internal/middleware"
)

type AdminHandler struct {
	db             *sql.DB
	rejudgeService RejudgeServiceClient
}

// RejudgeServiceClient is the interface for the rejudge gRPC service
type RejudgeServiceClient interface {
	CreateRejudge(ctx context.Context, req *CreateRejudgeRequest) (*CreateRejudgeResponse, error)
	GetRejudge(ctx context.Context, req *GetRejudgeRequest) (*GetRejudgeResponse, error)
	ListRejudges(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error)
	CancelRejudge(ctx context.Context, req *CancelRejudgeRequest) (*CancelRejudgeResponse, error)
	ApplyRejudge(ctx context.Context, req *ApplyRejudgeRequest) (*ApplyRejudgeResponse, error)
	RevertRejudge(ctx context.Context, req *RevertRejudgeRequest) (*RevertRejudgeResponse, error)
	GetRejudgeSubmissions(ctx context.Context, req *GetRejudgeSubmissionsRequest) (*GetRejudgeSubmissionsResponse, error)
}

// Request/Response types for rejudge operations
type CreateRejudgeRequest struct {
	SubmissionIDs []string `json:"submission_ids"`
	ContestID     string   `json:"contest_id"`
	ProblemID     string   `json:"problem_id"`
	FromVerdict   string   `json:"from_verdict"`
	Reason        string   `json:"reason"`
}

type CreateRejudgeResponse struct {
	ID            string `json:"id"`
	AffectedCount int32  `json:"affected_count"`
	Status        string `json:"status"`
}

type GetRejudgeRequest struct {
	ID string `json:"id"`
}

type GetRejudgeResponse struct {
	Rejudge *Rejudge `json:"rejudge"`
}

type Rejudge struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	ContestID     string `json:"contest_id"`
	ProblemID     string `json:"problem_id"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	AffectedCount int32  `json:"affected_count"`
	JudgedCount   int32  `json:"judged_count"`
	PendingCount  int32  `json:"pending_count"`
	CreatedAt     string `json:"created_at"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
	AppliedAt     string `json:"applied_at"`
	RevertedAt    string `json:"reverted_at"`
}

type ListRejudgesRequest struct {
	Page      int32  `json:"page"`
	PageSize  int32  `json:"page_size"`
	ContestID string `json:"contest_id"`
	ProblemID string `json:"problem_id"`
	Status    string `json:"status"`
	UserID    string `json:"user_id"`
}

type ListRejudgesResponse struct {
	Rejudges   []*Rejudge `json:"rejudges"`
	Total      int32      `json:"total"`
	Page       int32      `json:"page"`
	PageSize   int32      `json:"page_size"`
}

type CancelRejudgeRequest struct {
	ID string `json:"id"`
}

type CancelRejudgeResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ApplyRejudgeRequest struct {
	ID string `json:"id"`
}

type ApplyRejudgeResponse struct {
	ID             string `json:"id"`
	VerdictsChanged int32  `json:"verdicts_changed"`
	Status         string `json:"status"`
}

type RevertRejudgeRequest struct {
	ID string `json:"id"`
}

type RevertRejudgeResponse struct {
	ID              string `json:"id"`
	VerdictsRestored int32  `json:"verdicts_restored"`
	Status          string `json:"status"`
}

type GetRejudgeSubmissionsRequest struct {
	RejudgeID string `json:"rejudge_id"`
	OnlyChanged bool  `json:"only_changed"`
	Status     string `json:"status"`
	Page       int32  `json:"page"`
	PageSize   int32  `json:"page_size"`
}

type GetRejudgeSubmissionsResponse struct {
	Submissions []*RejudgeSubmission `json:"submissions"`
	Total       int32                `json:"total"`
	Page        int32                `json:"page"`
	PageSize    int32                `json:"page_size"`
}

type RejudgeSubmission struct {
	SubmissionID      string `json:"submission_id"`
	OriginalJudgingID string `json:"original_judging_id"`
	NewJudgingID      string `json:"new_judging_id"`
	OriginalVerdict   string `json:"original_verdict"`
	NewVerdict        string `json:"new_verdict"`
	VerdictChanged    bool   `json:"verdict_changed"`
	Status            string `json:"status"`
}

func NewAdminHandler(databaseURL string, rejudgeService RejudgeServiceClient) *AdminHandler {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		panic(err)
	}

	return &AdminHandler{
		db:             db,
		rejudgeService: rejudgeService,
	}
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

// CreateRejudge creates a new rejudging operation
func (h *AdminHandler) CreateRejudge(w http.ResponseWriter, r *http.Request) {
	var req CreateRejudgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.rejudgeService.CreateRejudge(r.Context(), &req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetRejudge retrieves a rejudging operation
func (h *AdminHandler) GetRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.GetRejudge(r.Context(), &GetRejudgeRequest{ID: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// ListRejudges lists rejudging operations
func (h *AdminHandler) ListRejudges(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	req := &ListRejudgesRequest{
		ContestID: query.Get("contest_id"),
		ProblemID: query.Get("problem_id"),
		Status:    query.Get("status"),
		UserID:    query.Get("user_id"),
	}

	if page := query.Get("page"); page != "" {
		json.NewDecoder(strings.NewReader(page)).Decode(&req.Page)
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		json.NewDecoder(strings.NewReader(pageSize)).Decode(&req.PageSize)
	}

	resp, err := h.rejudgeService.ListRejudges(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// CancelRejudge cancels a pending rejudging operation
func (h *AdminHandler) CancelRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.CancelRejudge(r.Context(), &CancelRejudgeRequest{ID: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// ApplyRejudge applies the rejudge results
func (h *AdminHandler) ApplyRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.ApplyRejudge(r.Context(), &ApplyRejudgeRequest{ID: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// RevertRejudge reverts the rejudge results
func (h *AdminHandler) RevertRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.RevertRejudge(r.Context(), &RevertRejudgeRequest{ID: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// GetRejudgeSubmissions retrieves submissions for a rejudging operation
func (h *AdminHandler) GetRejudgeSubmissions(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "rejudge_id")
	query := r.URL.Query()

	req := &GetRejudgeSubmissionsRequest{
		RejudgeID:  rejudgeID,
		OnlyChanged: query.Get("only_changed") == "true",
		Status:     query.Get("status"),
	}

	if page := query.Get("page"); page != "" {
		json.NewDecoder(strings.NewReader(page)).Decode(&req.Page)
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		json.NewDecoder(strings.NewReader(pageSize)).Decode(&req.PageSize)
	}

	resp, err := h.rejudgeService.GetRejudgeSubmissions(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes registers admin routes
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/users", h.ListUsers)
		r.Put("/users/{id}/role", h.UpdateUserRole)

		// Rejudge routes
		r.Route("/rejudges", func(r chi.Router) {
			r.Get("/", h.ListRejudges)
			r.Post("/", h.CreateRejudge)
			r.Get("/{id}", h.GetRejudge)
			r.Delete("/{id}", h.CancelRejudge)
			r.Post("/{id}/apply", h.ApplyRejudge)
			r.Post("/{id}/revert", h.RevertRejudge)
			r.Get("/{rejudge_id}/submissions", h.GetRejudgeSubmissions)
		})
	})
}