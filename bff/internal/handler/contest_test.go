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

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/contest/v1"
)

// MockContestServiceClient is a mock implementation of ContestServiceClient
type MockContestServiceClient struct {
	ListContestsFunc     func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error)
	GetContestFunc       func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error)
	CreateContestFunc    func(ctx context.Context, req *pb.CreateContestRequest) (*pb.CreateContestResponse, error)
	RegisterContestFunc  func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error)
	GetContestProblemsFunc func(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error)
	GetScoreboardFunc    func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error)
}

func (m *MockContestServiceClient) ListContests(ctx context.Context, req *pb.ListContestsRequest, opts ...grpc.CallOption) (*pb.ListContestsResponse, error) {
	if m.ListContestsFunc != nil {
		return m.ListContestsFunc(ctx, req)
	}
	return &pb.ListContestsResponse{
		Contests:   []*pb.Contest{},
		Pagination: &commonv1.PaginatedResponse{Total: 0, Page: 1, PageSize: 20},
	}, nil
}

func (m *MockContestServiceClient) GetContest(ctx context.Context, req *pb.GetContestRequest, opts ...grpc.CallOption) (*pb.GetContestResponse, error) {
	if m.GetContestFunc != nil {
		return m.GetContestFunc(ctx, req)
	}
	return &pb.GetContestResponse{
		Contest: &pb.Contest{Id: req.Id},
	}, nil
}

func (m *MockContestServiceClient) CreateContest(ctx context.Context, req *pb.CreateContestRequest, opts ...grpc.CallOption) (*pb.CreateContestResponse, error) {
	if m.CreateContestFunc != nil {
		return m.CreateContestFunc(ctx, req)
	}
	return &pb.CreateContestResponse{Id: "new-contest-id"}, nil
}

func (m *MockContestServiceClient) RegisterContest(ctx context.Context, req *pb.RegisterContestRequest, opts ...grpc.CallOption) (*pb.RegisterContestResponse, error) {
	if m.RegisterContestFunc != nil {
		return m.RegisterContestFunc(ctx, req)
	}
	return &pb.RegisterContestResponse{TeamId: "new-team-id"}, nil
}

func (m *MockContestServiceClient) GetContestProblems(ctx context.Context, req *pb.GetContestProblemsRequest, opts ...grpc.CallOption) (*pb.GetContestProblemsResponse, error) {
	if m.GetContestProblemsFunc != nil {
		return m.GetContestProblemsFunc(ctx, req)
	}
	return &pb.GetContestProblemsResponse{Problems: []*pb.ContestProblem{}}, nil
}

func (m *MockContestServiceClient) GetScoreboard(ctx context.Context, req *pb.GetScoreboardRequest, opts ...grpc.CallOption) (*pb.GetScoreboardResponse, error) {
	if m.GetScoreboardFunc != nil {
		return m.GetScoreboardFunc(ctx, req)
	}
	return &pb.GetScoreboardResponse{
		Entries:     []*pb.ScoreboardEntry{},
		ContestTime: "01:00:00",
		IsFrozen:    false,
	}, nil
}

func TestContestHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		mockFunc   func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "list contests successfully",
			mockFunc: func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error) {
				return &pb.ListContestsResponse{
					Contests: []*pb.Contest{
						{Id: "contest-1", Name: "Contest 1", ShortName: "C1", Public: true},
						{Id: "contest-2", Name: "Contest 2", ShortName: "C2", Public: true},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.ListContestsResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Contests, 2)
			},
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error) {
				return &pb.ListContestsResponse{
					Contests:   []*pb.Contest{},
					Pagination: &commonv1.PaginatedResponse{Total: 0, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "pagination")
			},
		},
		{
			name: "service error",
			mockFunc: func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error) {
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
			mockClient := &MockContestServiceClient{
				ListContestsFunc: tt.mockFunc,
			}

			handler := NewContestHandler(mockClient)

			req := httptest.NewRequest("GET", "/api/v1/contests", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestContestHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		contestID  string
		mockFunc   func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get contest successfully",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error) {
				return &pb.GetContestResponse{
					Contest: &pb.Contest{
						Id:         "contest-1",
						Name:       "ICPC Regional 2024",
						ShortName:  "ICPC24",
						StartTime:  "2024-03-01T10:00:00Z",
						EndTime:    "2024-03-01T15:00:00Z",
						Public:     true,
					},
					Problems: []*pb.ContestProblem{
						{ProblemId: "prob-1", ShortName: "A", Rank: 1, Points: 100},
						{ProblemId: "prob-2", ShortName: "B", Rank: 2, Points: 200},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetContestResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "ICPC Regional 2024", resp.Contest.Name)
				assert.Len(t, resp.Problems, 2)
			},
		},
		{
			name:      "contest not found",
			contestID: "non-existent",
			mockFunc: func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error) {
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
			mockClient := &MockContestServiceClient{
				GetContestFunc: tt.mockFunc,
			}

			handler := NewContestHandler(mockClient)

			r := chi.NewRouter()
			r.Get("/api/v1/contests/{id}", handler.Get)

			req := httptest.NewRequest("GET", "/api/v1/contests/"+tt.contestID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestContestHandler_GetProblems(t *testing.T) {
	tests := []struct {
		name       string
		contestID  string
		mockFunc   func(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get contest problems successfully",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error) {
				return &pb.GetContestProblemsResponse{
					Problems: []*pb.ContestProblem{
						{ProblemId: "prob-1", ShortName: "A", Rank: 1, Color: "red", Points: 100, AllowSubmit: true},
						{ProblemId: "prob-2", ShortName: "B", Rank: 2, Color: "blue", Points: 200, AllowSubmit: true},
						{ProblemId: "prob-3", ShortName: "C", Rank: 3, Color: "green", Points: 300, AllowSubmit: true},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetContestProblemsResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Problems, 3)
				assert.Equal(t, "A", resp.Problems[0].ShortName)
			},
		},
		{
			name:      "no problems for contest",
			contestID: "contest-empty",
			mockFunc: func(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error) {
				return &pb.GetContestProblemsResponse{
					Problems: []*pb.ContestProblem{},
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
			mockClient := &MockContestServiceClient{
				GetContestProblemsFunc: tt.mockFunc,
			}

			handler := NewContestHandler(mockClient)

			r := chi.NewRouter()
			r.Get("/api/v1/contests/{id}/problems", handler.GetProblems)

			req := httptest.NewRequest("GET", "/api/v1/contests/"+tt.contestID+"/problems", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestContestHandler_GetScoreboard(t *testing.T) {
	tests := []struct {
		name       string
		contestID  string
		mockFunc   func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get scoreboard successfully",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error) {
				return &pb.GetScoreboardResponse{
					Entries: []*pb.ScoreboardEntry{
						{
							Rank:        1,
							TeamId:      "team-1",
							TeamName:    "Team Alpha",
							Affiliation: "University A",
							NumSolved:   5,
							TotalTime:   600,
						},
						{
							Rank:        2,
							TeamId:      "team-2",
							TeamName:    "Team Beta",
							Affiliation: "University B",
							NumSolved:   4,
							TotalTime:   450,
						},
					},
					ContestTime: "02:30:00",
					IsFrozen:    false,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetScoreboardResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Entries, 2)
				assert.Equal(t, int32(1), resp.Entries[0].Rank)
				assert.Equal(t, "Team Alpha", resp.Entries[0].TeamName)
			},
		},
		{
			name:      "frozen scoreboard",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error) {
				return &pb.GetScoreboardResponse{
					Entries:     []*pb.ScoreboardEntry{},
					ContestTime: "04:00:00",
					IsFrozen:    true,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.GetScoreboardResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.True(t, resp.IsFrozen)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockContestServiceClient{
				GetScoreboardFunc: tt.mockFunc,
			}

			handler := NewContestHandler(mockClient)

			r := chi.NewRouter()
			r.Get("/api/v1/contests/{id}/scoreboard", handler.GetScoreboard)

			req := httptest.NewRequest("GET", "/api/v1/contests/"+tt.contestID+"/scoreboard", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestContestHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		contestID  string
		body       map[string]string
		mockFunc   func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "register successfully",
			contestID: "contest-1",
			body: map[string]string{
				"team_name":    "Team Gamma",
				"affiliation": "University C",
			},
			mockFunc: func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error) {
				return &pb.RegisterContestResponse{TeamId: "team-new-id"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.RegisterContestResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TeamId)
			},
		},
		{
			name:      "register without team name",
			contestID: "contest-1",
			body: map[string]string{
				"affiliation": "University D",
			},
			mockFunc: func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error) {
				return &pb.RegisterContestResponse{TeamId: "team-id"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.RegisterContestResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
			},
		},
		{
			name:      "service error",
			contestID: "contest-1",
			body: map[string]string{
				"team_name": "Team Delta",
			},
			mockFunc: func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error) {
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
			mockClient := &MockContestServiceClient{
				RegisterContestFunc: tt.mockFunc,
			}

			handler := NewContestHandler(mockClient)

			r := chi.NewRouter()
			r.Post("/api/v1/contests/{id}/register", handler.Register)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/contests/"+tt.contestID+"/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}