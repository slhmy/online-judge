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
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/submission/v1"
)

// MockSubmissionServiceClient is a mock implementation of SubmissionServiceClient
type MockSubmissionServiceClient struct {
	CreateSubmissionFunc         func(ctx context.Context, req *pb.CreateSubmissionRequest) (*pb.CreateSubmissionResponse, error)
	GetSubmissionFunc            func(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error)
	ListSubmissionsFunc          func(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error)
	GetJudgingFunc               func(ctx context.Context, req *pb.GetJudgingRequest) (*pb.GetJudgingResponse, error)
	GetJudgingRunsFunc           func(ctx context.Context, req *pb.GetJudgingRunsRequest) (*pb.GetJudgingRunsResponse, error)
	RejudgeSubmissionFunc        func(ctx context.Context, req *pb.RejudgeSubmissionRequest) (*pb.RejudgeSubmissionResponse, error)
	InternalCreateJudgingFunc    func(ctx context.Context, req *pb.InternalCreateJudgingRequest) (*pb.InternalCreateJudgingResponse, error)
	InternalUpdateJudgingFunc    func(ctx context.Context, req *pb.InternalUpdateJudgingRequest) (*pb.InternalUpdateJudgingResponse, error)
	InternalCreateJudgingRunFunc func(ctx context.Context, req *pb.InternalCreateJudgingRunRequest) (*pb.InternalCreateJudgingRunResponse, error)
}

func (m *MockSubmissionServiceClient) CreateSubmission(ctx context.Context, req *pb.CreateSubmissionRequest, opts ...grpc.CallOption) (*pb.CreateSubmissionResponse, error) {
	if m.CreateSubmissionFunc != nil {
		return m.CreateSubmissionFunc(ctx, req)
	}
	return &pb.CreateSubmissionResponse{Id: "new-sub-id", Status: pb.SubmissionStatus_SUBMISSION_STATUS_QUEUED}, nil
}

func (m *MockSubmissionServiceClient) GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest, opts ...grpc.CallOption) (*pb.GetSubmissionResponse, error) {
	if m.GetSubmissionFunc != nil {
		return m.GetSubmissionFunc(ctx, req)
	}
	return &pb.GetSubmissionResponse{
		Submission: &pb.Submission{Id: req.Id, ProblemId: "prob-1"},
	}, nil
}

func (m *MockSubmissionServiceClient) ListSubmissions(ctx context.Context, req *pb.ListSubmissionsRequest, opts ...grpc.CallOption) (*pb.ListSubmissionsResponse, error) {
	if m.ListSubmissionsFunc != nil {
		return m.ListSubmissionsFunc(ctx, req)
	}
	return &pb.ListSubmissionsResponse{
		Submissions: []*pb.SubmissionSummary{},
		Pagination:  &commonv1.PaginatedResponse{Total: 0, Page: 1, PageSize: 20},
	}, nil
}

func (m *MockSubmissionServiceClient) GetJudging(ctx context.Context, req *pb.GetJudgingRequest, opts ...grpc.CallOption) (*pb.GetJudgingResponse, error) {
	if m.GetJudgingFunc != nil {
		return m.GetJudgingFunc(ctx, req)
	}
	return &pb.GetJudgingResponse{
		Judging: &pb.Judging{SubmissionId: req.SubmissionId, Verdict: pb.Verdict_VERDICT_CORRECT},
	}, nil
}

func (m *MockSubmissionServiceClient) GetJudgingRuns(ctx context.Context, req *pb.GetJudgingRunsRequest, opts ...grpc.CallOption) (*pb.GetJudgingRunsResponse, error) {
	if m.GetJudgingRunsFunc != nil {
		return m.GetJudgingRunsFunc(ctx, req)
	}
	return &pb.GetJudgingRunsResponse{Runs: []*pb.JudgingRun{}}, nil
}

func (m *MockSubmissionServiceClient) RejudgeSubmission(ctx context.Context, req *pb.RejudgeSubmissionRequest, opts ...grpc.CallOption) (*pb.RejudgeSubmissionResponse, error) {
	if m.RejudgeSubmissionFunc != nil {
		return m.RejudgeSubmissionFunc(ctx, req)
	}
	return &pb.RejudgeSubmissionResponse{SubmissionId: req.SubmissionId, Status: "queued"}, nil
}

func (m *MockSubmissionServiceClient) InternalCreateJudging(ctx context.Context, req *pb.InternalCreateJudgingRequest, opts ...grpc.CallOption) (*pb.InternalCreateJudgingResponse, error) {
	if m.InternalCreateJudgingFunc != nil {
		return m.InternalCreateJudgingFunc(ctx, req)
	}
	return &pb.InternalCreateJudgingResponse{JudgingId: "jud-id"}, nil
}

func (m *MockSubmissionServiceClient) InternalUpdateJudging(ctx context.Context, req *pb.InternalUpdateJudgingRequest, opts ...grpc.CallOption) (*pb.InternalUpdateJudgingResponse, error) {
	if m.InternalUpdateJudgingFunc != nil {
		return m.InternalUpdateJudgingFunc(ctx, req)
	}
	return &pb.InternalUpdateJudgingResponse{JudgingId: req.JudgingId, Status: "updated"}, nil
}

func (m *MockSubmissionServiceClient) InternalCreateJudgingRun(ctx context.Context, req *pb.InternalCreateJudgingRunRequest, opts ...grpc.CallOption) (*pb.InternalCreateJudgingRunResponse, error) {
	if m.InternalCreateJudgingRunFunc != nil {
		return m.InternalCreateJudgingRunFunc(ctx, req)
	}
	return &pb.InternalCreateJudgingRunResponse{RunId: "run-id", Status: "created"}, nil
}

func TestSubmissionHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]string
		mockFunc   func(ctx context.Context, req *pb.CreateSubmissionRequest) (*pb.CreateSubmissionResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "create submission successfully",
			body: map[string]string{
				"problem_id": "prob-1",
				"language":   "cpp",
				"source":     "#include <iostream>",
			},
			mockFunc: func(ctx context.Context, req *pb.CreateSubmissionRequest) (*pb.CreateSubmissionResponse, error) {
				return &pb.CreateSubmissionResponse{
					Id:     "sub-123",
					Status: pb.SubmissionStatus_SUBMISSION_STATUS_QUEUED,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.CreateSubmissionResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "sub-123", resp.Id)
			},
		},
		{
			name: "create submission with contest",
			body: map[string]string{
				"problem_id": "prob-1",
				"contest_id": "contest-1",
				"language":   "python",
				"source":     "print('hello')",
			},
			mockFunc: func(ctx context.Context, req *pb.CreateSubmissionRequest) (*pb.CreateSubmissionResponse, error) {
				assert.Equal(t, "contest-1", req.ContestId)
				return &pb.CreateSubmissionResponse{
					Id:     "sub-456",
					Status: pb.SubmissionStatus_SUBMISSION_STATUS_QUEUED,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.CreateSubmissionResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "sub-456", resp.Id)
			},
		},
		{
			name: "service error",
			body: map[string]string{
				"problem_id": "prob-1",
				"language":   "cpp",
				"source":     "code",
			},
			mockFunc: func(ctx context.Context, req *pb.CreateSubmissionRequest) (*pb.CreateSubmissionResponse, error) {
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
			mockClient := &MockSubmissionServiceClient{
				CreateSubmissionFunc: tt.mockFunc,
			}

			handler := NewSubmissionHandler(mockClient)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/submissions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestSubmissionHandler_Get(t *testing.T) {
	tests := []struct {
		name         string
		submissionID string
		mockFunc     func(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error)
		wantStatus   int
		wantBody     func(t *testing.T, body string)
	}{
		{
			name:         "get submission successfully",
			submissionID: "sub-1",
			mockFunc: func(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) {
				return &pb.GetSubmissionResponse{
					Submission: &pb.Submission{
						Id:         "sub-1",
						UserId:     "user-1",
						ProblemId:  "prob-1",
						LanguageId: "cpp",
						SubmitTime: "2024-01-01T10:00:00Z",
					},
					LatestJudging: &pb.Judging{
						Id:           "jud-1",
						SubmissionId: "sub-1",
						Verdict:      pb.Verdict_VERDICT_CORRECT,
						MaxRuntime:   0.5,
						MaxMemory:    1024,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetSubmissionResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "sub-1", resp.Submission.Id)
				assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.LatestJudging.Verdict)
			},
		},
		{
			name:         "submission not found",
			submissionID: "non-existent",
			mockFunc: func(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) {
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
			mockClient := &MockSubmissionServiceClient{
				GetSubmissionFunc: tt.mockFunc,
			}

			handler := NewSubmissionHandler(mockClient)

			r := chi.NewRouter()
			r.Get("/api/v1/submissions/{id}", handler.Get)

			req := httptest.NewRequest("GET", "/api/v1/submissions/"+tt.submissionID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestSubmissionHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		mockFunc   func(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "list submissions successfully",
			mockFunc: func(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error) {
				return &pb.ListSubmissionsResponse{
					Submissions: []*pb.SubmissionSummary{
						{
							Id:          "sub-1",
							ProblemId:   "prob-1",
							ProblemName: "Problem A",
							User:        &commonv1.UserRef{Id: "user-1"},
							Verdict:     pb.Verdict_VERDICT_CORRECT,
							Runtime:     0.5,
							Memory:      1024,
						},
						{
							Id:          "sub-2",
							ProblemId:   "prob-2",
							ProblemName: "Problem B",
							User:        &commonv1.UserRef{Id: "user-2"},
							Verdict:     pb.Verdict_VERDICT_WRONG_ANSWER,
							Runtime:     1.0,
							Memory:      2048,
						},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.ListSubmissionsResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 2)
			},
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error) {
				return &pb.ListSubmissionsResponse{
					Submissions: []*pb.SubmissionSummary{},
					Pagination:  &commonv1.PaginatedResponse{Total: 0, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "pagination")
			},
		},
		{
			name: "service error",
			mockFunc: func(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error) {
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
			mockClient := &MockSubmissionServiceClient{
				ListSubmissionsFunc: tt.mockFunc,
			}

			handler := NewSubmissionHandler(mockClient)

			req := httptest.NewRequest("GET", "/api/v1/submissions", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestSubmissionHandler_GetJudging(t *testing.T) {
	tests := []struct {
		name         string
		submissionID string
		mockFunc     func(ctx context.Context, req *pb.GetJudgingRequest) (*pb.GetJudgingResponse, error)
		wantStatus   int
		wantBody     func(t *testing.T, body string)
	}{
		{
			name:         "get judging successfully",
			submissionID: "sub-1",
			mockFunc: func(ctx context.Context, req *pb.GetJudgingRequest) (*pb.GetJudgingResponse, error) {
				return &pb.GetJudgingResponse{
					Judging: &pb.Judging{
						Id:           "jud-1",
						SubmissionId: "sub-1",
						JudgehostId:  "host-1",
						Verdict:      pb.Verdict_VERDICT_CORRECT,
						MaxRuntime:   0.5,
						MaxMemory:    1024,
						Valid:        true,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetJudgingResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.Judging.Verdict)
			},
		},
		{
			name:         "judging not found",
			submissionID: "non-existent",
			mockFunc: func(ctx context.Context, req *pb.GetJudgingRequest) (*pb.GetJudgingResponse, error) {
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
			mockClient := &MockSubmissionServiceClient{
				GetJudgingFunc: tt.mockFunc,
			}

			handler := NewSubmissionHandler(mockClient)

			r := chi.NewRouter()
			r.Get("/api/v1/submissions/{id}/judging", handler.GetJudging)

			req := httptest.NewRequest("GET", "/api/v1/submissions/"+tt.submissionID+"/judging", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestSubmissionHandler_GetRuns(t *testing.T) {
	tests := []struct {
		name         string
		submissionID string
		mockFunc     func(ctx context.Context, req *pb.GetJudgingRunsRequest) (*pb.GetJudgingRunsResponse, error)
		wantStatus   int
		wantBody     func(t *testing.T, body string)
	}{
		{
			name:         "get runs successfully",
			submissionID: "sub-1",
			mockFunc: func(ctx context.Context, req *pb.GetJudgingRunsRequest) (*pb.GetJudgingRunsResponse, error) {
				return &pb.GetJudgingRunsResponse{
					Runs: []*pb.JudgingRun{
						{
							Id:         "run-1",
							JudgingId:  "jud-1",
							TestCaseId: "tc-1",
							Rank:       1,
							Verdict:    pb.Verdict_VERDICT_CORRECT,
							Runtime:    0.1,
							Memory:     512,
						},
						{
							Id:         "run-2",
							JudgingId:  "jud-1",
							TestCaseId: "tc-2",
							Rank:       2,
							Verdict:    pb.Verdict_VERDICT_CORRECT,
							Runtime:    0.2,
							Memory:     768,
						},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetJudgingRunsResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Runs, 2)
			},
		},
		{
			name:         "no runs",
			submissionID: "sub-2",
			mockFunc: func(ctx context.Context, req *pb.GetJudgingRunsRequest) (*pb.GetJudgingRunsResponse, error) {
				return &pb.GetJudgingRunsResponse{
					Runs: []*pb.JudgingRun{},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Equal(t, http.StatusOK, http.StatusOK)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSubmissionServiceClient{
				GetJudgingRunsFunc: tt.mockFunc,
			}

			handler := NewSubmissionHandler(mockClient)

			r := chi.NewRouter()
			r.Get("/api/v1/submissions/{id}/runs", handler.GetRuns)

			req := httptest.NewRequest("GET", "/api/v1/submissions/"+tt.submissionID+"/runs", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}
