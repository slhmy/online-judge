package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/submission/v1"
)

type SubmissionHandler struct {
	client pb.SubmissionServiceClient
}

func NewSubmissionHandler(client pb.SubmissionServiceClient) *SubmissionHandler {
	return &SubmissionHandler{client: client}
}

func (h *SubmissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := grpcContextFromRequest(r)

	var req struct {
		ProblemID string `json:"problem_id"`
		ContestID string `json:"contest_id"`
		Language  string `json:"language"`
		Source    string `json:"source"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.client.CreateSubmission(ctx, &pb.CreateSubmissionRequest{
		ProblemId:  req.ProblemID,
		ContestId:  req.ContestID,
		LanguageId: req.Language,
		SourceCode: req.Source,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SubmissionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	resp, err := h.client.GetSubmission(ctx, &pb.GetSubmissionRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SubmissionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page := int32(1)
	pageSize := int32(20)
	if v := r.URL.Query().Get("page"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 32); err == nil && parsed > 0 {
			page = int32(parsed)
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 32); err == nil && parsed > 0 {
			pageSize = int32(parsed)
		}
	}

	resp, err := h.client.ListSubmissions(ctx, &pb.ListSubmissionsRequest{
		Pagination: &commonv1.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SubmissionHandler) GetJudging(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	submissionID := chi.URLParam(r, "id")

	resp, err := h.client.GetJudging(ctx, &pb.GetJudgingRequest{
		SubmissionId: submissionID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SubmissionHandler) GetRuns(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	submissionID := chi.URLParam(r, "id")

	resp, err := h.client.GetJudgingRuns(ctx, &pb.GetJudgingRunsRequest{
		SubmissionId: submissionID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SubmissionHandler) Rejudge(w http.ResponseWriter, r *http.Request) {
	ctx := grpcContextFromRequest(r)
	submissionID := chi.URLParam(r, "id")

	resp, err := h.client.RejudgeSubmission(ctx, &pb.RejudgeSubmissionRequest{
		SubmissionId: submissionID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}
