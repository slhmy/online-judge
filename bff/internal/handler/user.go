package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/user/v1"
)

type UserHandler struct {
	client pb.UserServiceClient
}

func NewUserHandler(client pb.UserServiceClient) *UserHandler {
	return &UserHandler{client: client}
}

// GetProfile returns a user's profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := chi.URLParam(r, "id")

	resp, err := h.client.GetUserProfile(ctx, &pb.GetUserProfileRequest{UserId: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp.Profile)
}

// GetMyProfile returns the current user's profile
func (h *UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := chi.URLParam(r, "user_id")

	if userID == "" {
		http.Error(w, `{"error": "user_id required"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.client.GetUserProfile(ctx, &pb.GetUserProfileRequest{UserId: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp.Profile)
}

// UpdateProfile updates a user's profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := chi.URLParam(r, "id")

	var req struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
		Bio         string `json:"bio"`
		Country     string `json:"country"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.UpdateUserProfileRequest{
		UserId:      userID,
		DisplayName: req.DisplayName,
		AvatarUrl:   req.AvatarURL,
		Bio:         req.Bio,
		Country:     req.Country,
	}

	resp, err := h.client.UpdateUserProfile(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp.Profile)
}

// GetStats returns user statistics
func (h *UserHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := chi.URLParam(r, "id")

	resp, err := h.client.GetUserStats(ctx, &pb.GetUserStatsRequest{UserId: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp.Stats)
}

// GetSubmissions returns user submissions with pagination
func (h *UserHandler) GetSubmissions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := chi.URLParam(r, "id")

	// Parse pagination params
	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	// Parse filters
	verdict := r.URL.Query().Get("verdict")
	problemID := r.URL.Query().Get("problem_id")

	grpcReq := &pb.ListUserSubmissionsRequest{
		UserId: userID,
		Pagination: &commonv1.Pagination{
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
		Verdict:  verdict,
		ProblemId: problemID,
	}

	resp, err := h.client.ListUserSubmissions(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes registers user routes
func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Get("/users/{id}/profile", h.GetProfile)
	r.Get("/users/{id}/stats", h.GetStats)
	r.Get("/users/{id}/submissions", h.GetSubmissions)

	// Protected routes (require auth)
	r.Put("/users/{id}/profile", h.UpdateProfile)
}