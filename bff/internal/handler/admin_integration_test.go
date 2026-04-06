package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// MockRejudgeServiceClient is a mock implementation of RejudgeServiceClient
type MockRejudgeServiceClient struct {
	CreateRejudgeFunc       func(ctx context.Context, req *CreateRejudgeRequest) (*CreateRejudgeResponse, error)
	GetRejudgeFunc          func(ctx context.Context, req *GetRejudgeRequest) (*GetRejudgeResponse, error)
	ListRejudgesFunc        func(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error)
	CancelRejudgeFunc       func(ctx context.Context, req *CancelRejudgeRequest) (*CancelRejudgeResponse, error)
	ApplyRejudgeFunc        func(ctx context.Context, req *ApplyRejudgeRequest) (*ApplyRejudgeResponse, error)
	RevertRejudgeFunc       func(ctx context.Context, req *RevertRejudgeRequest) (*RevertRejudgeResponse, error)
	GetRejudgeSubmissionsFunc func(ctx context.Context, req *GetRejudgeSubmissionsRequest) (*GetRejudgeSubmissionsResponse, error)
}

func (m *MockRejudgeServiceClient) CreateRejudge(ctx context.Context, req *CreateRejudgeRequest) (*CreateRejudgeResponse, error) {
	if m.CreateRejudgeFunc != nil {
		return m.CreateRejudgeFunc(ctx, req)
	}
	return &CreateRejudgeResponse{
		ID:            "rejudge-1",
		AffectedCount: 10,
		Status:        "pending",
	}, nil
}

func (m *MockRejudgeServiceClient) GetRejudge(ctx context.Context, req *GetRejudgeRequest) (*GetRejudgeResponse, error) {
	if m.GetRejudgeFunc != nil {
		return m.GetRejudgeFunc(ctx, req)
	}
	return &GetRejudgeResponse{
		Rejudge: &Rejudge{
			ID:            req.ID,
			Status:        "completed",
			AffectedCount: 10,
			JudgedCount:   10,
		},
	}, nil
}

func (m *MockRejudgeServiceClient) ListRejudges(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error) {
	if m.ListRejudgesFunc != nil {
		return m.ListRejudgesFunc(ctx, req)
	}
	return &ListRejudgesResponse{
		Rejudges: []*Rejudge{
			{ID: "rejudge-1", Status: "completed", AffectedCount: 10},
			{ID: "rejudge-2", Status: "pending", AffectedCount: 5},
		},
		Total:    2,
		Page:     1,
		PageSize: 20,
	}, nil
}

func (m *MockRejudgeServiceClient) CancelRejudge(ctx context.Context, req *CancelRejudgeRequest) (*CancelRejudgeResponse, error) {
	if m.CancelRejudgeFunc != nil {
		return m.CancelRejudgeFunc(ctx, req)
	}
	return &CancelRejudgeResponse{
		ID:     req.ID,
		Status: "cancelled",
	}, nil
}

func (m *MockRejudgeServiceClient) ApplyRejudge(ctx context.Context, req *ApplyRejudgeRequest) (*ApplyRejudgeResponse, error) {
	if m.ApplyRejudgeFunc != nil {
		return m.ApplyRejudgeFunc(ctx, req)
	}
	return &ApplyRejudgeResponse{
		ID:              req.ID,
		VerdictsChanged: 3,
		Status:          "applied",
	}, nil
}

func (m *MockRejudgeServiceClient) RevertRejudge(ctx context.Context, req *RevertRejudgeRequest) (*RevertRejudgeResponse, error) {
	if m.RevertRejudgeFunc != nil {
		return m.RevertRejudgeFunc(ctx, req)
	}
	return &RevertRejudgeResponse{
		ID:               req.ID,
		VerdictsRestored: 3,
		Status:           "reverted",
	}, nil
}

func (m *MockRejudgeServiceClient) GetRejudgeSubmissions(ctx context.Context, req *GetRejudgeSubmissionsRequest) (*GetRejudgeSubmissionsResponse, error) {
	if m.GetRejudgeSubmissionsFunc != nil {
		return m.GetRejudgeSubmissionsFunc(ctx, req)
	}
	return &GetRejudgeSubmissionsResponse{
		Submissions: []*RejudgeSubmission{
			{
				SubmissionID:    "sub-1",
				OriginalVerdict: "correct",
				NewVerdict:      "wrong-answer",
				VerdictChanged:  true,
				Status:          "judged",
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}, nil
}

func TestAdminHandler_CreateRejudge(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		mockFunc   func(ctx context.Context, req *CreateRejudgeRequest) (*CreateRejudgeResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "create rejudge with submission IDs",
			body: CreateRejudgeRequest{
				SubmissionIDs: []string{"sub-1", "sub-2", "sub-3"},
				Reason:        "Fix test case bug",
			},
			mockFunc: func(ctx context.Context, req *CreateRejudgeRequest) (*CreateRejudgeResponse, error) {
				assert.Len(t, req.SubmissionIDs, 3)
				return &CreateRejudgeResponse{
					ID:            "rejudge-new",
					AffectedCount: 3,
					Status:        "pending",
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody: func(t *testing.T, body string) {
				var resp CreateRejudgeResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, "rejudge-new", resp.ID)
				assert.Equal(t, int32(3), resp.AffectedCount)
			},
		},
		{
			name: "create rejudge with contest and problem filter",
			body: CreateRejudgeRequest{
				ContestID:   "contest-1",
				ProblemID:   "prob-1",
				FromVerdict: "wrong-answer",
				Reason:      "Rejudge all WA submissions",
			},
			mockFunc: func(ctx context.Context, req *CreateRejudgeRequest) (*CreateRejudgeResponse, error) {
				assert.Equal(t, "contest-1", req.ContestID)
				assert.Equal(t, "prob-1", req.ProblemID)
				assert.Equal(t, "wrong-answer", req.FromVerdict)
				return &CreateRejudgeResponse{
					ID:            "rejudge-filter",
					AffectedCount: 50,
					Status:        "pending",
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody: func(t *testing.T, body string) {
				var resp CreateRejudgeResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, int32(50), resp.AffectedCount)
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
			mockClient := &MockRejudgeServiceClient{CreateRejudgeFunc: tt.mockFunc}
			// Use empty DB URL since we're mocking the service
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
		mockFunc   func(ctx context.Context, req *GetRejudgeRequest) (*GetRejudgeResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get existing rejudge",
			rejudgeID: "rejudge-1",
			mockFunc: func(ctx context.Context, req *GetRejudgeRequest) (*GetRejudgeResponse, error) {
				return &GetRejudgeResponse{
					Rejudge: &Rejudge{
						ID:            "rejudge-1",
						UserID:        "admin-1",
						ContestID:     "contest-1",
						ProblemID:     "prob-1",
						Status:        "completed",
						Reason:        "Bug fix",
						AffectedCount: 10,
						JudgedCount:   10,
						PendingCount:  0,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp GetRejudgeResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, "rejudge-1", resp.Rejudge.ID)
				assert.Equal(t, "completed", resp.Rejudge.Status)
				assert.Equal(t, int32(10), resp.Rejudge.AffectedCount)
			},
		},
		{
			name:      "get non-existent rejudge",
			rejudgeID: "non-existent",
			mockFunc: func(ctx context.Context, req *GetRejudgeRequest) (*GetRejudgeResponse, error) {
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
			mockClient := &MockRejudgeServiceClient{GetRejudgeFunc: tt.mockFunc}
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
		mockFunc   func(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:  "list all rejudges",
			query: "",
			mockFunc: func(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error) {
				return &ListRejudgesResponse{
					Rejudges: []*Rejudge{
						{ID: "rejudge-1", Status: "completed"},
						{ID: "rejudge-2", Status: "pending"},
						{ID: "rejudge-3", Status: "judging"},
					},
					Total:    3,
					Page:     1,
					PageSize: 20,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp ListRejudgesResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Len(t, resp.Rejudges, 3)
				assert.Equal(t, int32(3), resp.Total)
			},
		},
		{
			name:  "list rejudges with filters",
			query: "contest_id=contest-1&status=pending&page=1&page_size=10",
			mockFunc: func(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error) {
				assert.Equal(t, "contest-1", req.ContestID)
				assert.Equal(t, "pending", req.Status)
				return &ListRejudgesResponse{
					Rejudges: []*Rejudge{
						{ID: "rejudge-2", Status: "pending", ContestID: "contest-1"},
					},
					Total:    1,
					Page:     1,
					PageSize: 10,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp ListRejudgesResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Len(t, resp.Rejudges, 1)
			},
		},
		{
			name:  "empty list",
			query: "",
			mockFunc: func(ctx context.Context, req *ListRejudgesRequest) (*ListRejudgesResponse, error) {
				return &ListRejudgesResponse{
					Rejudges: []*Rejudge{},
					Total:    0,
					Page:     1,
					PageSize: 20,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp ListRejudgesResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Len(t, resp.Rejudges, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeServiceClient{ListRejudgesFunc: tt.mockFunc}
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
		mockFunc   func(ctx context.Context, req *CancelRejudgeRequest) (*CancelRejudgeResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "cancel pending rejudge",
			rejudgeID: "rejudge-1",
			mockFunc: func(ctx context.Context, req *CancelRejudgeRequest) (*CancelRejudgeResponse, error) {
				return &CancelRejudgeResponse{
					ID:     "rejudge-1",
					Status: "cancelled",
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp CancelRejudgeResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Equal(t, "cancelled", resp.Status)
			},
		},
		{
			name:      "cancel non-existent rejudge",
			rejudgeID: "non-existent",
			mockFunc: func(ctx context.Context, req *CancelRejudgeRequest) (*CancelRejudgeResponse, error) {
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
			mockClient := &MockRejudgeServiceClient{CancelRejudgeFunc: tt.mockFunc}
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
	mockClient := &MockRejudgeServiceClient{
		ApplyRejudgeFunc: func(ctx context.Context, req *ApplyRejudgeRequest) (*ApplyRejudgeResponse, error) {
			return &ApplyRejudgeResponse{
				ID:              "rejudge-1",
				VerdictsChanged: 5,
				Status:          "applied",
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

	var resp ApplyRejudgeResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "rejudge-1", resp.ID)
	assert.Equal(t, int32(5), resp.VerdictsChanged)
	assert.Equal(t, "applied", resp.Status)
}

func TestAdminHandler_RevertRejudge(t *testing.T) {
	mockClient := &MockRejudgeServiceClient{
		RevertRejudgeFunc: func(ctx context.Context, req *RevertRejudgeRequest) (*RevertRejudgeResponse, error) {
			return &RevertRejudgeResponse{
				ID:               "rejudge-1",
				VerdictsRestored: 5,
				Status:           "reverted",
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

	var resp RevertRejudgeResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "rejudge-1", resp.ID)
	assert.Equal(t, int32(5), resp.VerdictsRestored)
	assert.Equal(t, "reverted", resp.Status)
}

func TestAdminHandler_GetRejudgeSubmissions(t *testing.T) {
	tests := []struct {
		name       string
		rejudgeID  string
		query      string
		mockFunc   func(ctx context.Context, req *GetRejudgeSubmissionsRequest) (*GetRejudgeSubmissionsResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get rejudge submissions",
			rejudgeID: "rejudge-1",
			query:     "",
			mockFunc: func(ctx context.Context, req *GetRejudgeSubmissionsRequest) (*GetRejudgeSubmissionsResponse, error) {
				return &GetRejudgeSubmissionsResponse{
					Submissions: []*RejudgeSubmission{
						{
							SubmissionID:      "sub-1",
							OriginalVerdict:   "correct",
							NewVerdict:        "wrong-answer",
							VerdictChanged:    true,
							Status:            "judged",
						},
						{
							SubmissionID:      "sub-2",
							OriginalVerdict:   "wrong-answer",
							NewVerdict:        "wrong-answer",
							VerdictChanged:    false,
							Status:            "judged",
						},
					},
					Total:    2,
					Page:     1,
					PageSize: 20,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp GetRejudgeSubmissionsResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Len(t, resp.Submissions, 2)
			},
		},
		{
			name:      "get only changed submissions",
			rejudgeID: "rejudge-1",
			query:     "only_changed=true",
			mockFunc: func(ctx context.Context, req *GetRejudgeSubmissionsRequest) (*GetRejudgeSubmissionsResponse, error) {
				assert.True(t, req.OnlyChanged)
				return &GetRejudgeSubmissionsResponse{
					Submissions: []*RejudgeSubmission{
						{
							SubmissionID:   "sub-1",
							VerdictChanged: true,
						},
					},
					Total: 1,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp GetRejudgeSubmissionsResponse
				json.Unmarshal([]byte(body), &resp)
				assert.Len(t, resp.Submissions, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockRejudgeServiceClient{GetRejudgeSubmissionsFunc: tt.mockFunc}
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
	mockClient := &MockRejudgeServiceClient{}
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