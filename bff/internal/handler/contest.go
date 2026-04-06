package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/contest/v1"
)

type ContestHandler struct {
	client pb.ContestServiceClient
}

func NewContestHandler(client pb.ContestServiceClient) *ContestHandler {
	return &ContestHandler{client: client}
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

	json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	resp, err := h.client.GetContest(ctx, &pb.GetContestRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
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

	json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) GetScoreboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contestID := chi.URLParam(r, "id")

	resp, err := h.client.GetScoreboard(ctx, &pb.GetScoreboardRequest{
		ContestId: contestID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *ContestHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	json.NewEncoder(w).Encode(resp)
}