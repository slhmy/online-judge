package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/problem/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProblemHandler struct {
	client pb.ProblemServiceClient
}

func NewProblemHandler(client pb.ProblemServiceClient) *ProblemHandler {
	return &ProblemHandler{client: client}
}

func (h *ProblemHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	resp, err := h.client.ListProblems(ctx, &pb.ListProblemsRequest{
		Pagination: &commonv1.Pagination{
			Page:     1,
			PageSize: 20,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	resp, err := h.client.GetProblem(ctx, &pb.GetProblemRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) ListLanguages(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	resp, err := h.client.ListLanguages(ctx, &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// Admin handlers for problem CRUD

func (h *ProblemHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req struct {
		ExternalID    string  `json:"external_id"`
		Name          string  `json:"name"`
		TimeLimit     float64 `json:"time_limit"`
		MemoryLimit   int32   `json:"memory_limit"`
		OutputLimit   int32   `json:"output_limit"`
		Difficulty    string  `json:"difficulty"`
		Points        int32   `json:"points"`
		Description   string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.CreateProblemRequest{
		ExternalId:    req.ExternalID,
		Name:          req.Name,
		TimeLimit:     req.TimeLimit,
		MemoryLimit:   req.MemoryLimit,
		OutputLimit:   req.OutputLimit,
		Difficulty:    req.Difficulty,
		Points:        req.Points,
	}

	resp, err := h.client.CreateProblem(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If description is provided, update the problem statement
	if req.Description != "" && resp.Id != "" {
		// Update the problem with the description
		updateReq := &pb.UpdateProblemRequest{
			Id: resp.Id,
			// Note: The proto doesn't have description field in UpdateProblemRequest
			// We need to handle this separately if the backend supports it
		}
		h.client.UpdateProblem(ctx, updateReq)
	}

	json.NewEncoder(w).Encode(map[string]string{"id": resp.Id})
}

func (h *ProblemHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	var req struct {
		Name          string  `json:"name"`
		TimeLimit     float64 `json:"time_limit"`
		MemoryLimit   int32   `json:"memory_limit"`
		OutputLimit   int32   `json:"output_limit"`
		Difficulty    string  `json:"difficulty"`
		Points        int32   `json:"points"`
		IsPublished   bool    `json:"is_published"`
		AllowSubmit   bool    `json:"allow_submit"`
		Description   string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.UpdateProblemRequest{
		Id:          id,
		Name:        req.Name,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		IsPublished: req.IsPublished,
		AllowSubmit: req.AllowSubmit,
	}

	resp, err := h.client.UpdateProblem(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	_, err := h.client.DeleteProblem(ctx, &pb.DeleteProblemRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes registers problem routes
func (h *ProblemHandler) RegisterRoutes(r chi.Router) {
	r.Get("/problems", h.ListProblems)
	r.Get("/problems/{id}", h.GetProblem)
	r.Get("/languages", h.ListLanguages)

	// Admin routes for problem CRUD
	r.Post("/problems", h.CreateProblem)
	r.Put("/problems/{id}", h.UpdateProblem)
	r.Delete("/problems/{id}", h.DeleteProblem)
}