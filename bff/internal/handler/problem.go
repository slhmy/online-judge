package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/problem/v1"
	"github.com/online-judge/bff/internal/cache"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProblemHandler struct {
	client pb.ProblemServiceClient
	cache  *cache.Service
}

func NewProblemHandler(client pb.ProblemServiceClient, cacheService *cache.Service) *ProblemHandler {
	return &ProblemHandler{
		client: client,
		cache:  cacheService,
	}
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
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	// Try cache first
	cacheKey := "problem:" + id
	cached, err := h.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "hit")
		w.Write(cached)
		return
	}

	// Fetch from gRPC
	resp, err := h.client.GetProblem(ctx, &pb.GetProblemRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache the response
	data, _ := json.Marshal(resp)
	h.cache.Set(ctx, cacheKey, data, h.cache.GetConfig().ProblemTTL, "problem:"+id)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "miss")
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
		ExternalID  string  `json:"external_id"`
		Name        string  `json:"name"`
		TimeLimit   float64 `json:"time_limit"`
		MemoryLimit int32   `json:"memory_limit"`
		OutputLimit int32   `json:"output_limit"`
		Difficulty  string  `json:"difficulty"`
		Points      int32   `json:"points"`
		Description string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.CreateProblemRequest{
		ExternalId:  req.ExternalID,
		Name:        req.Name,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		OutputLimit: req.OutputLimit,
		Difficulty:  req.Difficulty,
		Points:      req.Points,
	}

	resp, err := h.client.CreateProblem(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If description is provided, set the problem statement
	if req.Description != "" && resp.Id != "" {
		_, err = h.client.SetProblemStatement(ctx, &pb.SetProblemStatementRequest{
			ProblemId: resp.Id,
			Language:  "en",
			Format:    "markdown",
			Content:   req.Description,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"id": resp.Id})
}

func (h *ProblemHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req struct {
		Name        string  `json:"name"`
		TimeLimit   float64 `json:"time_limit"`
		MemoryLimit int32   `json:"memory_limit"`
		OutputLimit int32   `json:"output_limit"`
		Difficulty  string  `json:"difficulty"`
		Points      int32   `json:"points"`
		IsPublished bool    `json:"is_published"`
		AllowSubmit bool    `json:"allow_submit"`
		Description string  `json:"description"`
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

	// If description is provided, update the problem statement
	if req.Description != "" {
		_, err = h.client.SetProblemStatement(ctx, &pb.SetProblemStatementRequest{
			ProblemId: id,
			Language:  "en",
			Format:    "markdown",
			Content:   req.Description,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Invalidate problem cache
	h.cache.InvalidateProblemCache(ctx, id)

	json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	_, err := h.client.DeleteProblem(ctx, &pb.DeleteProblemRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate problem cache
	h.cache.InvalidateProblemCache(ctx, id)

	w.WriteHeader(http.StatusNoContent)
}

// GetProblemStatement returns the problem statement content (markdown)
func (h *ProblemHandler) GetProblemStatement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	// Get language from query param, default to "en"
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	resp, err := h.client.GetProblemStatement(ctx, &pb.GetProblemStatementRequest{
		ProblemId: problemID,
		Language:  language,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return just the content string for markdown rendering
	if resp.Statement == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Statement.Content)
}

// SetProblemStatement updates the problem statement content (markdown)
func (h *ProblemHandler) SetProblemStatement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	var req struct {
		Language string `json:"language"`
		Format   string `json:"format"`
		Title    string `json:"title"`
		Content  string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default format to markdown
	if req.Format == "" {
		req.Format = "markdown"
	}

	// Default language to "en"
	if req.Language == "" {
		req.Language = "en"
	}

	resp, err := h.client.SetProblemStatement(ctx, &pb.SetProblemStatementRequest{
		ProblemId: problemID,
		Language:  req.Language,
		Format:    req.Format,
		Title:     req.Title,
		Content:   req.Content,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate problem cache
	h.cache.InvalidateProblemCache(ctx, problemID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Statement)
}

// RegisterRoutes registers problem routes
func (h *ProblemHandler) RegisterRoutes(r chi.Router) {
	r.Get("/problems", h.ListProblems)
	r.Get("/problems/{id}", h.GetProblem)
	r.Get("/problems/{id}/statement", h.GetProblemStatement)
	r.Get("/languages", h.ListLanguages)

	// Admin routes for problem CRUD
	r.Post("/problems", h.CreateProblem)
	r.Put("/problems/{id}", h.UpdateProblem)
	r.Delete("/problems/{id}", h.DeleteProblem)
	r.Put("/problems/{id}/statement", h.SetProblemStatement)
}
