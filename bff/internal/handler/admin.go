package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/judge/v1"
	pbUser "github.com/slhmy/online-judge/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type AdminHandler struct {
	rejudgeService RejudgeClient
	userService    UserAdminClient
}

// RejudgeClient is the subset of the generated JudgeServiceClient used for rejudge operations.
// The generated pb.JudgeServiceClient satisfies this interface.
type RejudgeClient interface {
	CreateRejudge(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error)
	GetRejudge(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error)
	ListRejudges(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error)
	CancelRejudge(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error)
	ApplyRejudge(ctx context.Context, in *pb.ApplyRejudgeRequest, opts ...grpc.CallOption) (*pb.ApplyRejudgeResponse, error)
	RevertRejudge(ctx context.Context, in *pb.RevertRejudgeRequest, opts ...grpc.CallOption) (*pb.RevertRejudgeResponse, error)
	GetRejudgeSubmissions(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error)
}

// UserAdminClient is the subset of UserServiceClient used for admin user operations.
type UserAdminClient interface {
	ListUsers(ctx context.Context, in *pbUser.ListUsersRequest, opts ...grpc.CallOption) (*pbUser.ListUsersResponse, error)
	UpdateUserRole(ctx context.Context, in *pbUser.UpdateUserRoleRequest, opts ...grpc.CallOption) (*pbUser.UpdateUserRoleResponse, error)
}

func NewAdminHandler(rejudgeService RejudgeClient, userService UserAdminClient) *AdminHandler {
	return &AdminHandler{
		rejudgeService: rejudgeService,
		userService:    userService,
	}
}

// ListUsers returns all users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	resp, err := h.userService.ListUsers(grpcContextFromRequest(r), &pbUser.ListUsersRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 200},
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

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
	for _, p := range resp.Users {
		u := User{
			ID:              p.UserId,
			Username:        p.Username,
			Role:            p.Role,
			Rating:          int(p.Rating),
			SolvedCount:     int(p.SolvedCount),
			SubmissionCount: int(p.SubmissionCount),
			CreatedAt:       p.CreatedAt,
		}
		users = append(users, u)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

	_, err := h.userService.UpdateUserRole(grpcContextFromRequest(r), &pbUser.UpdateUserRoleRequest{
		UserId: userID,
		Role:   req.Role,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// protoJSON is the shared protojson marshaler for writing gRPC responses as JSON.
var protoJSON = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}

// CreateRejudge creates a new rejudging operation
func (h *AdminHandler) CreateRejudge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubmissionIDs []string `json:"submission_ids"`
		ContestID     string   `json:"contest_id"`
		ProblemID     string   `json:"problem_id"`
		FromVerdict   string   `json:"from_verdict"`
		Reason        string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error": "invalid request"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.rejudgeService.CreateRejudge(grpcContextFromRequest(r), &pb.CreateRejudgeRequest{
		SubmissionIds: body.SubmissionIDs,
		ContestId:     body.ContestID,
		ProblemId:     body.ProblemID,
		FromVerdict:   body.FromVerdict,
		Reason:        body.Reason,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// GetRejudge retrieves a rejudging operation
func (h *AdminHandler) GetRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.GetRejudge(grpcContextFromRequest(r), &pb.GetRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// ListRejudges lists rejudging operations
func (h *AdminHandler) ListRejudges(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	req := &pb.ListRejudgesRequest{
		ContestId: query.Get("contest_id"),
		ProblemId: query.Get("problem_id"),
		UserId:    query.Get("user_id"),
	}

	if statusStr := query.Get("status"); statusStr != "" {
		if v, ok := pb.RejudgeStatus_value[statusStr]; ok {
			req.Status = pb.RejudgeStatus(v)
		}
	}

	pagination := &commonv1.Pagination{}
	if page := query.Get("page"); page != "" {
		if v, err := strconv.ParseInt(page, 10, 32); err == nil {
			pagination.Page = int32(v)
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if v, err := strconv.ParseInt(pageSize, 10, 32); err == nil {
			pagination.PageSize = int32(v)
		}
	}
	req.Pagination = pagination

	resp, err := h.rejudgeService.ListRejudges(grpcContextFromRequest(r), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// CancelRejudge cancels a pending rejudging operation
func (h *AdminHandler) CancelRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.CancelRejudge(grpcContextFromRequest(r), &pb.CancelRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// ApplyRejudge applies the rejudge results
func (h *AdminHandler) ApplyRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.ApplyRejudge(grpcContextFromRequest(r), &pb.ApplyRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// RevertRejudge reverts the rejudge results
func (h *AdminHandler) RevertRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.RevertRejudge(grpcContextFromRequest(r), &pb.RevertRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// GetRejudgeSubmissions retrieves submissions for a rejudging operation
func (h *AdminHandler) GetRejudgeSubmissions(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "rejudge_id")
	query := r.URL.Query()

	req := &pb.GetRejudgeSubmissionsRequest{
		RejudgeId:   rejudgeID,
		OnlyChanged: query.Get("only_changed") == "true",
	}

	if statusStr := query.Get("status"); statusStr != "" {
		if v, ok := pb.RejudgeSubmissionStatus_value[statusStr]; ok {
			req.Status = pb.RejudgeSubmissionStatus(v)
		}
	}

	pagination := &commonv1.Pagination{}
	if page := query.Get("page"); page != "" {
		if v, err := strconv.ParseInt(page, 10, 32); err == nil {
			pagination.Page = int32(v)
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if v, err := strconv.ParseInt(pageSize, 10, 32); err == nil {
			pagination.PageSize = int32(v)
		}
	}
	req.Pagination = pagination

	resp, err := h.rejudgeService.GetRejudgeSubmissions(grpcContextFromRequest(r), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
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
