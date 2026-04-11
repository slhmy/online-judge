package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/judge/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// MockRejudgeClient is a mock implementation of RejudgeClient
type MockRejudgeClient struct {
	CreateRejudgeFunc         func(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error)
	GetRejudgeFunc            func(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error)
	ListRejudgesFunc          func(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error)
	CancelRejudgeFunc         func(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error)
	ApplyRejudgeFunc          func(ctx context.Context, in *pb.ApplyRejudgeRequest, opts ...grpc.CallOption) (*pb.ApplyRejudgeResponse, error)
	RevertRejudgeFunc         func(ctx context.Context, in *pb.RevertRejudgeRequest, opts ...grpc.CallOption) (*pb.RevertRejudgeResponse, error)
	GetRejudgeSubmissionsFunc func(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error)
}

func (m *MockRejudgeClient) CreateRejudge(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error) {
	if m.CreateRejudgeFunc != nil {
		return m.CreateRejudgeFunc(ctx, in, opts...)
	}
	return &pb.CreateRejudgeResponse{
		Id:            "rejudge-1",
		AffectedCount: 10,
		Status:        pb.RejudgeStatus_REJUDGE_STATUS_PENDING,
	}, nil
}

func (m *MockRejudgeClient) GetRejudge(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error) {
	if m.GetRejudgeFunc != nil {
		return m.GetRejudgeFunc(ctx, in, opts...)
	}
	return &pb.GetRejudgeResponse{
		Rejudge: &pb.Rejudge{
			Id:            in.Id,
			Status:        pb.RejudgeStatus_REJUDGE_STATUS_JUDGED,
			AffectedCount: 10,
			JudgedCount:   10,
		},
	}, nil
}

func (m *MockRejudgeClient) ListRejudges(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error) {
	if m.ListRejudgesFunc != nil {
		return m.ListRejudgesFunc(ctx, in, opts...)
	}
	return &pb.ListRejudgesResponse{
		Rejudges: []*pb.Rejudge{
			{Id: "rejudge-1", Status: pb.RejudgeStatus_REJUDGE_STATUS_JUDGED, AffectedCount: 10},
			{Id: "rejudge-2", Status: pb.RejudgeStatus_REJUDGE_STATUS_PENDING, AffectedCount: 5},
		},
		Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
	}, nil
}

func (m *MockRejudgeClient) CancelRejudge(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error) {
	if m.CancelRejudgeFunc != nil {
		return m.CancelRejudgeFunc(ctx, in, opts...)
	}
	return &pb.CancelRejudgeResponse{
		Id:     in.Id,
		Status: pb.RejudgeStatus_REJUDGE_STATUS_CANCELLED,
	}, nil
}

func (m *MockRejudgeClient) ApplyRejudge(ctx context.Context, in *pb.ApplyRejudgeRequest, opts ...grpc.CallOption) (*pb.ApplyRejudgeResponse, error) {
	if m.ApplyRejudgeFunc != nil {
		return m.ApplyRejudgeFunc(ctx, in, opts...)
	}
	return &pb.ApplyRejudgeResponse{
		Id:              in.Id,
		VerdictsChanged: 3,
		Status:          pb.RejudgeStatus_REJUDGE_STATUS_APPLIED,
	}, nil
}

func (m *MockRejudgeClient) RevertRejudge(ctx context.Context, in *pb.RevertRejudgeRequest, opts ...grpc.CallOption) (*pb.RevertRejudgeResponse, error) {
	if m.RevertRejudgeFunc != nil {
		return m.RevertRejudgeFunc(ctx, in, opts...)
	}
	return &pb.RevertRejudgeResponse{
		Id:               in.Id,
		VerdictsRestored: 3,
		Status:           pb.RejudgeStatus_REJUDGE_STATUS_REVERTED,
	}, nil
}

func (m *MockRejudgeClient) GetRejudgeSubmissions(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error) {
	if m.GetRejudgeSubmissionsFunc != nil {
		return m.GetRejudgeSubmissionsFunc(ctx, in, opts...)
	}
	return &pb.GetRejudgeSubmissionsResponse{
		Submissions: []*pb.RejudgeSubmission{
			{
				SubmissionId:    "sub-1",
				OriginalVerdict: "correct",
				NewVerdict:      "wrong-answer",
				VerdictChanged:  true,
				Status:          pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_DONE,
			},
		},
		Pagination: &commonv1.PaginatedResponse{Total: 1, Page: 1, PageSize: 20},
	}, nil
}

func TestAdminHandler_CreateRejudge(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		mockFunc   func(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "create rejudge with submission IDs",
			body: map[string]interface{}{
				"submission_ids": []string{"sub-1", "sub-2", "sub-3"},
				"reason":         "Fix test case bug",
			},
			mockFunc: func(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error) {
				assert.Len(t, in.SubmissionIds, 3)
				return &pb.CreateRejudgeResponse{
					Id:            "rejudge-new",
					AffectedCount: 3,
					Status:        pb.RejudgeStatus_REJUDGE_STATUS_PENDING,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, "rejudge-new", resp["id"])
				assert.Equal(t, float64(3), resp["affected_count"])
			},
		},
		{
			name: "create rejudge with contest and problem filter",
			body: map[string]interface{}{
				"contest_id":   "contest-1",
				"problem_id":   "prob-1",
				"from_verdict": "wrong-answer",
				"reason":       "Rejudge all WA submissions",
			},
			mockFunc: func(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error) {
				assert.Equal(t, "contest-1", in.ContestId)
				assert.Equal(t, "prob-1", in.ProblemId)
				assert.Equal(t, "wrong-answer", in.FromVerdict)
				return &pb.CreateRejudgeResponse{
					Id:            "rejudge-filter",
					AffectedCount: 50,
					Status:        pb.RejudgeStatus_REJUDGE_STATUS_PENDING,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, float64(50), resp["affected_count"])
			},
		},
		{
			name:       "invalid request body",
			body:       "invalid json",
			mockFunc:   nil,
			wantStatus: http.StatusBadRequest,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeClient{CreateRejudgeFunc: tt.mockFunc}
			handler := &AdminHandler{
				db:             nil,
				rejudgeService: mockClient,
			}

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest("POST", "/admin/rejudges", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateRejudge(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestAdminHandler_GetRejudge(t *testing.T) {
	tests := []struct {
		name       string
		rejudgeID  string
		mockFunc   func(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get existing rejudge",
			rejudgeID: "rejudge-1",
			mockFunc: func(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error) {
				return &pb.GetRejudgeResponse{
					Rejudge: &pb.Rejudge{
						Id:            "rejudge-1",
						UserId:        "admin-1",
						ContestId:     "contest-1",
						ProblemId:     "prob-1",
						Status:        pb.RejudgeStatus_REJUDGE_STATUS_JUDGED,
						Reason:        "Bug fix",
						AffectedCount: 10,
						JudgedCount:   10,
						PendingCount:  0,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				rejudge := resp["rejudge"].(map[string]interface{})
				assert.Equal(t, "rejudge-1", rejudge["id"])
				assert.Equal(t, float64(10), rejudge["affected_count"])
			},
		},
		{
			name:      "get non-existent rejudge",
			rejudgeID: "non-existent",
			mockFunc: func(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error) {
				return nil, assert.AnError
			},
			wantStatus: http.StatusNotFound,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeClient{GetRejudgeFunc: tt.mockFunc}
			handler := &AdminHandler{
				db:             nil,
				rejudgeService: mockClient,
			}

			r := chi.NewRouter()
			r.Get("/admin/rejudges/{id}", handler.GetRejudge)

			req := httptest.NewRequest("GET", "/admin/rejudges/"+tt.rejudgeID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestAdminHandler_ListRejudges(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		mockFunc   func(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:  "list all rejudges",
			query: "",
			mockFunc: func(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error) {
				return &pb.ListRejudgesResponse{
					Rejudges: []*pb.Rejudge{
						{Id: "rejudge-1", Status: pb.RejudgeStatus_REJUDGE_STATUS_JUDGED},
						{Id: "rejudge-2", Status: pb.RejudgeStatus_REJUDGE_STATUS_PENDING},
						{Id: "rejudge-3", Status: pb.RejudgeStatus_REJUDGE_STATUS_JUDGING},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 3, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				rejudges := resp["rejudges"].([]interface{})
				assert.Len(t, rejudges, 3)
				pagination := resp["pagination"].(map[string]interface{})
				assert.Equal(t, float64(3), pagination["total"])
			},
		},
		{
			name:  "list rejudges with filters",
			query: "contest_id=contest-1&status=pending&page=1&page_size=10",
			mockFunc: func(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error) {
				assert.Equal(t, "contest-1", in.ContestId)
				return &pb.ListRejudgesResponse{
					Rejudges: []*pb.Rejudge{
						{Id: "rejudge-2", Status: pb.RejudgeStatus_REJUDGE_STATUS_PENDING, ContestId: "contest-1"},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 1, Page: 1, PageSize: 10},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				rejudges := resp["rejudges"].([]interface{})
				assert.Len(t, rejudges, 1)
			},
		},
		{
			name:  "empty list",
			query: "",
			mockFunc: func(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error) {
				return &pb.ListRejudgesResponse{
					Rejudges:   []*pb.Rejudge{},
					Pagination: &commonv1.PaginatedResponse{Total: 0, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				rejudges := resp["rejudges"].([]interface{})
				assert.Len(t, rejudges, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeClient{ListRejudgesFunc: tt.mockFunc}
			handler := &AdminHandler{
				db:             nil,
				rejudgeService: mockClient,
			}

			url := "/admin/rejudges"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.ListRejudges(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestAdminHandler_CancelRejudge(t *testing.T) {
	tests := []struct {
		name       string
		rejudgeID  string
		mockFunc   func(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "cancel pending rejudge",
			rejudgeID: "rejudge-1",
			mockFunc: func(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error) {
				return &pb.CancelRejudgeResponse{
					Id:     "rejudge-1",
					Status: pb.RejudgeStatus_REJUDGE_STATUS_CANCELLED,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, "REJUDGE_STATUS_CANCELLED", resp["status"])
			},
		},
		{
			name:      "cancel non-existent rejudge",
			rejudgeID: "non-existent",
			mockFunc: func(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error) {
				return nil, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeClient{CancelRejudgeFunc: tt.mockFunc}
			handler := &AdminHandler{
				db:             nil,
				rejudgeService: mockClient,
			}

			r := chi.NewRouter()
			r.Delete("/admin/rejudges/{id}", handler.CancelRejudge)

			req := httptest.NewRequest("DELETE", "/admin/rejudges/"+tt.rejudgeID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestAdminHandler_ApplyRejudge(t *testing.T) {
	mockClient := &MockRejudgeClient{
		ApplyRejudgeFunc: func(ctx context.Context, in *pb.ApplyRejudgeRequest, opts ...grpc.CallOption) (*pb.ApplyRejudgeResponse, error) {
			return &pb.ApplyRejudgeResponse{
				Id:              "rejudge-1",
				VerdictsChanged: 5,
				Status:          pb.RejudgeStatus_REJUDGE_STATUS_APPLIED,
			}, nil
		},
	}
	handler := &AdminHandler{
		db:             nil,
		rejudgeService: mockClient,
	}

	r := chi.NewRouter()
	r.Post("/admin/rejudges/{id}/apply", handler.ApplyRejudge)

	req := httptest.NewRequest("POST", "/admin/rejudges/rejudge-1/apply", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "rejudge-1", resp["id"])
	assert.Equal(t, float64(5), resp["verdicts_changed"])
	assert.Equal(t, "REJUDGE_STATUS_APPLIED", resp["status"])
}

func TestAdminHandler_RevertRejudge(t *testing.T) {
	mockClient := &MockRejudgeClient{
		RevertRejudgeFunc: func(ctx context.Context, in *pb.RevertRejudgeRequest, opts ...grpc.CallOption) (*pb.RevertRejudgeResponse, error) {
			return &pb.RevertRejudgeResponse{
				Id:               "rejudge-1",
				VerdictsRestored: 5,
				Status:           pb.RejudgeStatus_REJUDGE_STATUS_REVERTED,
			}, nil
		},
	}
	handler := &AdminHandler{
		db:             nil,
		rejudgeService: mockClient,
	}

	r := chi.NewRouter()
	r.Post("/admin/rejudges/{id}/revert", handler.RevertRejudge)

	req := httptest.NewRequest("POST", "/admin/rejudges/rejudge-1/revert", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "rejudge-1", resp["id"])
	assert.Equal(t, float64(5), resp["verdicts_restored"])
	assert.Equal(t, "REJUDGE_STATUS_REVERTED", resp["status"])
}

func TestAdminHandler_GetRejudgeSubmissions(t *testing.T) {
	tests := []struct {
		name       string
		rejudgeID  string
		query      string
		mockFunc   func(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get rejudge submissions",
			rejudgeID: "rejudge-1",
			query:     "",
			mockFunc: func(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error) {
				return &pb.GetRejudgeSubmissionsResponse{
					Submissions: []*pb.RejudgeSubmission{
						{
							SubmissionId:    "sub-1",
							OriginalVerdict: "correct",
							NewVerdict:      "wrong-answer",
							VerdictChanged:  true,
							Status:          pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_DONE,
						},
						{
							SubmissionId:    "sub-2",
							OriginalVerdict: "wrong-answer",
							NewVerdict:      "wrong-answer",
							VerdictChanged:  false,
							Status:          pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_DONE,
						},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				submissions := resp["submissions"].([]interface{})
				assert.Len(t, submissions, 2)
			},
		},
		{
			name:      "get only changed submissions",
			rejudgeID: "rejudge-1",
			query:     "only_changed=true",
			mockFunc: func(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error) {
				assert.True(t, in.OnlyChanged)
				return &pb.GetRejudgeSubmissionsResponse{
					Submissions: []*pb.RejudgeSubmission{
						{
							SubmissionId:   "sub-1",
							VerdictChanged: true,
						},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 1},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				_ = json.Unmarshal([]byte(body), &resp)
				submissions := resp["submissions"].([]interface{})
				assert.Len(t, submissions, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeClient{GetRejudgeSubmissionsFunc: tt.mockFunc}
			handler := &AdminHandler{
				db:             nil,
				rejudgeService: mockClient,
			}

			r := chi.NewRouter()
			r.Get("/admin/rejudges/{rejudge_id}/submissions", handler.GetRejudgeSubmissions)

			url := "/admin/rejudges/" + tt.rejudgeID + "/submissions"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

// Integration test: Full rejudge workflow
func TestAdminHandler_Integration_RejudgeWorkflow(t *testing.T) {
	mockClient := &MockRejudgeClient{}
	handler := &AdminHandler{
		db:             nil,
		rejudgeService: mockClient,
	}

	r := chi.NewRouter()
	r.Post("/admin/rejudges", handler.CreateRejudge)
	r.Get("/admin/rejudges/{id}", handler.GetRejudge)
	r.Delete("/admin/rejudges/{id}", handler.CancelRejudge)
	r.Post("/admin/rejudges/{id}/apply", handler.ApplyRejudge)
	r.Post("/admin/rejudges/{id}/revert", handler.RevertRejudge)
	r.Get("/admin/rejudges/{rejudge_id}/submissions", handler.GetRejudgeSubmissions)

	// Step 1: Create rejudge
	createBody := `{"submission_ids": ["sub-1", "sub-2"], "reason": "Test rejudge"}`
	req1 := httptest.NewRequest("POST", "/admin/rejudges", bytes.NewReader([]byte(createBody)))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Step 2: Get rejudge status
	req2 := httptest.NewRequest("GET", "/admin/rejudges/rejudge-1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Step 3: Get rejudge submissions
	req3 := httptest.NewRequest("GET", "/admin/rejudges/rejudge-1/submissions", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	// Step 4: Apply rejudge
	req4 := httptest.NewRequest("POST", "/admin/rejudges/rejudge-1/apply", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	// Step 5: Revert if needed
	req5 := httptest.NewRequest("POST", "/admin/rejudges/rejudge-1/revert", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)
}
