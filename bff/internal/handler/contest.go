package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/slhmy/online-judge/bff/internal/cache"
	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/contest/v1"
)

type ContestHandler struct {
	client pb.ContestServiceClient
	cache  *cache.Service
}

func NewContestHandler(client pb.ContestServiceClient, cacheService *cache.Service) *ContestHandler {
	return &ContestHandler{
		client: client,
		cache:  cacheService,
	}
}

func (h *ContestHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp, err := h.client.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{
			Page:     1,
			PageSize: 20,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	// Try cache first
	cacheKey := "contest:" + id
	cached, err := h.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "hit")
		_, _ = w.Write(cached)
		return
	}

	// Fetch from gRPC
	resp, err := h.client.GetContest(ctx, &pb.GetContestRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache the response
	data, _ := json.Marshal(resp)
	_ = h.cache.Set(ctx, cacheKey, data, h.cache.GetConfig().ContestTTL, "contest:"+id)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "miss")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) GetProblems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestID := chi.URLParam(r, "id")

	resp, err := h.client.GetContestProblems(ctx, &pb.GetContestProblemsRequest{
		ContestId: contestID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) GetScoreboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestID := chi.URLParam(r, "id")

	// Try cache first (short TTL for scoreboard during contest)
	cacheKey := "scoreboard:" + contestID
	cached, err := h.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "hit")
		_, _ = w.Write(cached)
		return
	}

	// Fetch from gRPC
	resp, err := h.client.GetScoreboard(ctx, &pb.GetScoreboardRequest{
		ContestId: contestID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache the response with short TTL (scoreboard changes frequently during contest)
	data, _ := json.Marshal(resp)
	_ = h.cache.Set(ctx, cacheKey, data, h.cache.GetConfig().ScoreboardTTL, "contest:"+contestID, "scoreboard:"+contestID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "miss")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := grpcContextFromRequest(r)
	contestID := chi.URLParam(r, "id")

	var req struct {
		TeamName    string `json:"team_name"`
		Affiliation string `json:"affiliation"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.client.RegisterContest(ctx, &pb.RegisterContestRequest{
		ContestId:   contestID,
		TeamName:    req.TeamName,
		Affiliation: req.Affiliation,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}
