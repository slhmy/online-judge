package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/contest/v1"
	"github.com/online-judge/bff/internal/cache"
)

// MockContestServiceClientIntegration is a mock implementation for integration tests
type MockContestServiceClientIntegration struct {
	ListContestsFunc      func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error)
	GetContestFunc        func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error)
	GetContestProblemsFunc func(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error)
	GetScoreboardFunc     func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error)
	RegisterContestFunc   func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error)
	CreateContestFunc     func(ctx context.Context, req *pb.CreateContestRequest) (*pb.CreateContestResponse, error)
}

func (m *MockContestServiceClientIntegration) ListContests(ctx context.Context, req *pb.ListContestsRequest, opts ...grpc.CallOption) (*pb.ListContestsResponse, error) {
	if m.ListContestsFunc != nil {
		return m.ListContestsFunc(ctx, req)
	}
	return &pb.ListContestsResponse{
		Contests: []*pb.Contest{
			{Id: "contest-1", Name: "Contest 1", Public: true},
			{Id: "contest-2", Name: "Contest 2", Public: false},
		},
		Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
	}, nil
}

func (m *MockContestServiceClientIntegration) GetContest(ctx context.Context, req *pb.GetContestRequest, opts ...grpc.CallOption) (*pb.GetContestResponse, error) {
	if m.GetContestFunc != nil {
		return m.GetContestFunc(ctx, req)
	}
	return &pb.GetContestResponse{
		Contest: &pb.Contest{Id: req.Id, Name: "Test Contest", ShortName: "TC"},
		Problems: []*pb.ContestProblem{
			{ProblemId: "prob-1", ShortName: "A", Points: 100},
		},
	}, nil
}

func (m *MockContestServiceClientIntegration) GetContestProblems(ctx context.Context, req *pb.GetContestProblemsRequest, opts ...grpc.CallOption) (*pb.GetContestProblemsResponse, error) {
	if m.GetContestProblemsFunc != nil {
		return m.GetContestProblemsFunc(ctx, req)
	}
	return &pb.GetContestProblemsResponse{
		Problems: []*pb.ContestProblem{
			{ProblemId: "prob-1", ShortName: "A", Points: 100},
			{ProblemId: "prob-2", ShortName: "B", Points: 200},
		},
	}, nil
}

func (m *MockContestServiceClientIntegration) GetScoreboard(ctx context.Context, req *pb.GetScoreboardRequest, opts ...grpc.CallOption) (*pb.GetScoreboardResponse, error) {
	if m.GetScoreboardFunc != nil {
		return m.GetScoreboardFunc(ctx, req)
	}
	return &pb.GetScoreboardResponse{
		Entries: []*pb.ScoreboardEntry{
			{Rank: 1, TeamName: "Team Alpha", NumSolved: 3, TotalTime: 300},
			{Rank: 2, TeamName: "Team Beta", NumSolved: 2, TotalTime: 250},
		},
		ContestTime: "01:00:00",
		IsFrozen:    false,
	}, nil
}

func (m *MockContestServiceClientIntegration) RegisterContest(ctx context.Context, req *pb.RegisterContestRequest, opts ...grpc.CallOption) (*pb.RegisterContestResponse, error) {
	if m.RegisterContestFunc != nil {
		return m.RegisterContestFunc(ctx, req)
	}
	return &pb.RegisterContestResponse{TeamId: "team-1"}, nil
}

func (m *MockContestServiceClientIntegration) CreateContest(ctx context.Context, req *pb.CreateContestRequest, opts ...grpc.CallOption) (*pb.CreateContestResponse, error) {
	if m.CreateContestFunc != nil {
		return m.CreateContestFunc(ctx, req)
	}
	return &pb.CreateContestResponse{Id: "new-contest-id"}, nil
}

func setupTestContestHandlerIntegration(t *testing.T, mockClient *MockContestServiceClientIntegration) (*ContestHandler, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cacheConfig := cache.Config{
		Enabled:       true,
		ProblemTTL:    5 * time.Minute,
		ContestTTL:    2 * time.Minute,
		ScoreboardTTL: 10 * time.Second,
	}
	cacheService := cache.NewService(rdb, cacheConfig)

	handler := NewContestHandler(mockClient, cacheService)
	return handler, mr
}

func TestContestHandler_Integration_List(t *testing.T) {
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
						{Id: "contest-1", Name: "ICPC 2024", ShortName: "ICPC24", Public: true},
						{Id: "contest-2", Name: "Codeforces Round", ShortName: "CF", Public: true},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "ICPC 2024")
				assert.Contains(t, body, "Codeforces Round")
			},
		},
		{
			name: "empty contest list",
			mockFunc: func(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error) {
				return &pb.ListContestsResponse{
					Contests:    []*pb.Contest{},
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
			mockClient := &MockContestServiceClientIntegration{ListContestsFunc: tt.mockFunc}
			handler, mr := setupTestContestHandlerIntegration(t, mockClient)
			defer mr.Close()

			req := httptest.NewRequest("GET", "/api/v1/contests", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestContestHandler_Integration_Get(t *testing.T) {
	tests := []struct {
		name       string
		contestID  string
		mockFunc   func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error)
		wantStatus int
		wantCache  string
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "get contest successfully",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error) {
				return &pb.GetContestResponse{
					Contest: &pb.Contest{
						Id:        "contest-1",
						Name:      "ICPC Regional 2024",
						ShortName: "ICPC24",
						Public:    true,
					},
					Problems: []*pb.ContestProblem{
						{ProblemId: "prob-1", ShortName: "A", Points: 100},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCache:  "miss",
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "ICPC Regional 2024")
				assert.Contains(t, body, "prob-1")
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
			mockClient := &MockContestServiceClientIntegration{GetContestFunc: tt.mockFunc}
			handler, mr := setupTestContestHandlerIntegration(t, mockClient)
			defer mr.Close()

			r := chi.NewRouter()
			r.Get("/api/v1/contests/{id}", handler.Get)

			req := httptest.NewRequest("GET", "/api/v1/contests/"+tt.contestID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantCache != "" {
				assert.Equal(t, tt.wantCache, w.Header().Get("X-Cache"))
			}
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestContestHandler_Integration_Get_CacheHit(t *testing.T) {
	mockClient := &MockContestServiceClientIntegration{
		GetContestFunc: func(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error) {
			return &pb.GetContestResponse{
				Contest: &pb.Contest{
					Id:   "contest-1",
					Name: "Cached Contest",
				},
			}, nil
		},
	}

	handler, mr := setupTestContestHandlerIntegration(t, mockClient)
	defer mr.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/contests/{id}", handler.Get)

	// First request - cache miss
	req1 := httptest.NewRequest("GET", "/api/v1/contests/contest-1", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "miss", w1.Header().Get("X-Cache"))

	// Second request - cache hit
	req2 := httptest.NewRequest("GET", "/api/v1/contests/contest-1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "hit", w2.Header().Get("X-Cache"))
}

func TestContestHandler_Integration_GetScoreboard(t *testing.T) {
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
						{Rank: 1, TeamName: "Team Alpha", Affiliation: "MIT", NumSolved: 5, TotalTime: 500},
						{Rank: 2, TeamName: "Team Beta", Affiliation: "Stanford", NumSolved: 4, TotalTime: 450},
						{Rank: 3, TeamName: "Team Gamma", Affiliation: "Berkeley", NumSolved: 4, TotalTime: 520},
					},
					ContestTime: "02:30:00",
					IsFrozen:    false,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "Team Alpha")
				assert.Contains(t, body, "Team Beta")
				assert.Contains(t, body, "02:30:00")
			},
		},
		{
			name:      "empty scoreboard",
			contestID: "contest-empty",
			mockFunc: func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error) {
				return &pb.GetScoreboardResponse{
					Entries:     []*pb.ScoreboardEntry{},
					ContestTime: "00:00:00",
					IsFrozen:    false,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "contest_time")
			},
		},
		{
			name:      "scoreboard error",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error) {
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
			mockClient := &MockContestServiceClientIntegration{GetScoreboardFunc: tt.mockFunc}
			handler, mr := setupTestContestHandlerIntegration(t, mockClient)
			defer mr.Close()

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

func TestContestHandler_Integration_GetScoreboard_CacheHit(t *testing.T) {
	mockClient := &MockContestServiceClientIntegration{
		GetScoreboardFunc: func(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error) {
			return &pb.GetScoreboardResponse{
				Entries:     []*pb.ScoreboardEntry{{Rank: 1, TeamName: "Team A"}},
				ContestTime: "01:00:00",
			}, nil
		},
	}

	handler, mr := setupTestContestHandlerIntegration(t, mockClient)
	defer mr.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/contests/{id}/scoreboard", handler.GetScoreboard)

	// First request - cache miss
	req1 := httptest.NewRequest("GET", "/api/v1/contests/contest-1/scoreboard", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "miss", w1.Header().Get("X-Cache"))

	// Second request - cache hit
	req2 := httptest.NewRequest("GET", "/api/v1/contests/contest-1/scoreboard", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "hit", w2.Header().Get("X-Cache"))
}

func TestContestHandler_Integration_GetProblems(t *testing.T) {
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
						{ProblemId: "prob-1", ShortName: "A", Color: "red", Points: 100, AllowSubmit: true},
						{ProblemId: "prob-2", ShortName: "B", Color: "blue", Points: 200, AllowSubmit: true},
						{ProblemId: "prob-3", ShortName: "C", Color: "green", Points: 300, AllowSubmit: true},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "prob-1")
				assert.Contains(t, body, "prob-2")
				assert.Contains(t, body, "prob-3")
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
				var resp pb.GetContestProblemsResponse
				_ = json.Unmarshal([]byte(body), &resp)
				assert.Len(t, resp.Problems, 0)
			},
		},
		{
			name:      "service error",
			contestID: "contest-1",
			mockFunc: func(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error) {
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
			mockClient := &MockContestServiceClientIntegration{GetContestProblemsFunc: tt.mockFunc}
			handler, mr := setupTestContestHandlerIntegration(t, mockClient)
			defer mr.Close()

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

func TestContestHandler_Integration_Register(t *testing.T) {
	tests := []struct {
		name       string
		contestID  string
		body       interface{}
		mockFunc   func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name:      "register successfully",
			contestID: "contest-1",
			body: map[string]string{
				"team_name":    "Team Alpha",
				"affiliation":  "MIT",
			},
			mockFunc: func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error) {
				assert.Equal(t, "Team Alpha", req.TeamName)
				assert.Equal(t, "MIT", req.Affiliation)
				return &pb.RegisterContestResponse{TeamId: "team-123"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.RegisterContestResponse
				_ = json.Unmarshal([]byte(body), &resp)
				assert.NotEmpty(t, resp.TeamId)
			},
		},
		{
			name:      "register without team name",
			contestID: "contest-1",
			body: map[string]string{
				"affiliation": "Stanford",
			},
			mockFunc: func(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error) {
				return &pb.RegisterContestResponse{TeamId: "team-456"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				var resp pb.RegisterContestResponse
				_ = json.Unmarshal([]byte(body), &resp)
				assert.NotEmpty(t, resp.TeamId)
			},
		},
		{
			name:      "service error",
			contestID: "contest-1",
			body: map[string]string{
				"team_name": "Team Test",
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
			mockClient := &MockContestServiceClientIntegration{RegisterContestFunc: tt.mockFunc}
			handler, mr := setupTestContestHandlerIntegration(t, mockClient)
			defer mr.Close()

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

// Integration test: Full contest flow
func TestContestHandler_Integration_FullFlow(t *testing.T) {
	mockClient := &MockContestServiceClientIntegration{}
	handler, mr := setupTestContestHandlerIntegration(t, mockClient)
	defer mr.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/contests", handler.List)
	r.Get("/api/v1/contests/{id}", handler.Get)
	r.Get("/api/v1/contests/{id}/problems", handler.GetProblems)
	r.Get("/api/v1/contests/{id}/scoreboard", handler.GetScoreboard)
	r.Post("/api/v1/contests/{id}/register", handler.Register)

	// Step 1: List contests
	req1 := httptest.NewRequest("GET", "/api/v1/contests", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Step 2: Get contest (cache miss)
	req2 := httptest.NewRequest("GET", "/api/v1/contests/contest-1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "miss", w2.Header().Get("X-Cache"))

	// Step 3: Get contest (cache hit)
	req3 := httptest.NewRequest("GET", "/api/v1/contests/contest-1", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, "hit", w3.Header().Get("X-Cache"))

	// Step 4: Get problems
	req4 := httptest.NewRequest("GET", "/api/v1/contests/contest-1/problems", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	// Step 5: Get scoreboard
	req5 := httptest.NewRequest("GET", "/api/v1/contests/contest-1/scoreboard", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)
}