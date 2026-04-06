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