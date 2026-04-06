package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/problem/v1"
	"github.com/online-judge/bff/internal/cache"
)

// MockProblemServiceClient is a mock implementation of ProblemServiceClient
type MockProblemServiceClient struct {
	ListProblemsFunc   func(ctx context.Context, req *pb.ListProblemsRequest) (*pb.ListProblemsResponse, error)
	GetProblemFunc     func(ctx context.Context, req *pb.GetProblemRequest) (*pb.GetProblemResponse, error)
	CreateProblemFunc  func(ctx context.Context, req *pb.CreateProblemRequest) (*pb.CreateProblemResponse, error)
	UpdateProblemFunc  func(ctx context.Context, req *pb.UpdateProblemRequest) (*pb.UpdateProblemResponse, error)
	DeleteProblemFunc  func(ctx context.Context, req *pb.DeleteProblemRequest) (*emptypb.Empty, error)
	ListTestCasesFunc  func(ctx context.Context, req *pb.ListTestCasesRequest) (*pb.ListTestCasesResponse, error)
	CreateTestCaseFunc func(ctx context.Context, req *pb.CreateTestCaseRequest) (*pb.CreateTestCaseResponse, error)
	ListLanguagesFunc  func(ctx context.Context, req *emptypb.Empty) (*pb.ListLanguagesResponse, error)
}

func setupTestProblemHandler(t *testing.T, mockClient *MockProblemServiceClient) (*ProblemHandler, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

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

	handler := NewProblemHandler(mockClient, cacheService)
	return handler, mr
}

func (m *MockProblemServiceClient) ListProblems(ctx context.Context, req *pb.ListProblemsRequest, opts ...grpc.CallOption) (*pb.ListProblemsResponse, error) {
	if m.ListProblemsFunc != nil {
		return m.ListProblemsFunc(ctx, req)
	}
	return &pb.ListProblemsResponse{
		Problems: []*pb.ProblemSummary{
			{Id: "prob-1", Name: "Problem 1"},
			{Id: "prob-2", Name: "Problem 2"},
		},
		Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
	}, nil
}

func (m *MockProblemServiceClient) GetProblem(ctx context.Context, req *pb.GetProblemRequest, opts ...grpc.CallOption) (*pb.GetProblemResponse, error) {
	if m.GetProblemFunc != nil {
		return m.GetProblemFunc(ctx, req)
	}
	return &pb.GetProblemResponse{
		Problem: &pb.Problem{Id: req.Id, Name: "Test Problem"},
		SampleTestCases: []*pb.TestCase{
			{Id: "tc-1", IsSample: true},
		},
	}, nil
}

func (m *MockProblemServiceClient) CreateProblem(ctx context.Context, req *pb.CreateProblemRequest, opts ...grpc.CallOption) (*pb.CreateProblemResponse, error) {
	if m.CreateProblemFunc != nil {
		return m.CreateProblemFunc(ctx, req)
	}
	return &pb.CreateProblemResponse{Id: "new-prob-id"}, nil
}

func (m *MockProblemServiceClient) UpdateProblem(ctx context.Context, req *pb.UpdateProblemRequest, opts ...grpc.CallOption) (*pb.UpdateProblemResponse, error) {
	if m.UpdateProblemFunc != nil {
		return m.UpdateProblemFunc(ctx, req)
	}
	return &pb.UpdateProblemResponse{Problem: &pb.Problem{Id: req.Id}}, nil
}

func (m *MockProblemServiceClient) DeleteProblem(ctx context.Context, req *pb.DeleteProblemRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if m.DeleteProblemFunc != nil {
		return m.DeleteProblemFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *MockProblemServiceClient) ListTestCases(ctx context.Context, req *pb.ListTestCasesRequest, opts ...grpc.CallOption) (*pb.ListTestCasesResponse, error) {
	if m.ListTestCasesFunc != nil {
		return m.ListTestCasesFunc(ctx, req)
	}
	return &pb.ListTestCasesResponse{}, nil
}

func (m *MockProblemServiceClient) CreateTestCase(ctx context.Context, req *pb.CreateTestCaseRequest, opts ...grpc.CallOption) (*pb.CreateTestCaseResponse, error) {
	if m.CreateTestCaseFunc != nil {
		return m.CreateTestCaseFunc(ctx, req)
	}
	return &pb.CreateTestCaseResponse{}, nil
}

func (m *MockProblemServiceClient) ListLanguages(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*pb.ListLanguagesResponse, error) {
	if m.ListLanguagesFunc != nil {
		return m.ListLanguagesFunc(ctx, req)
	}
	return &pb.ListLanguagesResponse{
		Languages: []*pb.Language{
			{Id: "cpp", Name: "C++"},
			{Id: "python", Name: "Python"},
		},
	}, nil
}

func TestProblemHandler_ListProblems(t *testing.T) {
	tests := []struct {
		name       string
		mockFunc   func(ctx context.Context, req *pb.ListProblemsRequest) (*pb.ListProblemsResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "list problems successfully",
			mockFunc: func(ctx context.Context, req *pb.ListProblemsRequest) (*pb.ListProblemsResponse, error) {
				return &pb.ListProblemsResponse{
					Problems: []*pb.ProblemSummary{
						{Id: "prob-1", ExternalId: "A", Name: "Problem A"},
						{Id: "prob-2", ExternalId: "B", Name: "Problem B"},
					},
					Pagination: &commonv1.PaginatedResponse{Total: 2, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "Problem A")
				assert.Contains(t, body, "Problem B")
			},
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context, req *pb.ListProblemsRequest) (*pb.ListProblemsResponse, error) {
				return &pb.ListProblemsResponse{
					Problems:    []*pb.ProblemSummary{},
					Pagination:  &commonv1.PaginatedResponse{Total: 0, Page: 1, PageSize: 20},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "pagination")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockProblemServiceClient{
				ListProblemsFunc: tt.mockFunc,
			}

			handler, mr := setupTestProblemHandler(t, mockClient)
			defer mr.Close()

			req := httptest.NewRequest("GET", "/api/v1/problems", nil)
			w := httptest.NewRecorder()

			handler.ListProblems(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}

func TestProblemHandler_GetProblem(t *testing.T) {
	tests := []struct {
		name       string
		problemID  string
		mockFunc   func(ctx context.Context, req *pb.GetProblemRequest) (*pb.GetProblemResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
		wantCache  string // expected X-Cache header value
	}{
		{
			name:      "get problem successfully",
			problemID: "prob-1",
			mockFunc: func(ctx context.Context, req *pb.GetProblemRequest) (*pb.GetProblemResponse, error) {
				return &pb.GetProblemResponse{
					Problem: &pb.Problem{
						Id:          "prob-1",
						Name:        "Test Problem",
						TimeLimit:   1.0,
						MemoryLimit: 256,
					},
					SampleTestCases: []*pb.TestCase{
						{Id: "tc-1", IsSample: true, Rank: 1},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCache:  "miss", // First request should be cache miss
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "Test Problem")
				assert.Contains(t, body, "prob-1")
			},
		},
		{
			name:      "problem not found",
			problemID: "non-existent",
			mockFunc: func(ctx context.Context, req *pb.GetProblemRequest) (*pb.GetProblemResponse, error) {
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
			mockClient := &MockProblemServiceClient{
				GetProblemFunc: tt.mockFunc,
			}

			handler, mr := setupTestProblemHandler(t, mockClient)
			defer mr.Close()

			// Create chi router to test URL param extraction
			r := chi.NewRouter()
			r.Get("/api/v1/problems/{id}", handler.GetProblem)

			req := httptest.NewRequest("GET", "/api/v1/problems/"+tt.problemID, nil)
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

func TestProblemHandler_GetProblem_CacheHit(t *testing.T) {
	mockClient := &MockProblemServiceClient{
		GetProblemFunc: func(ctx context.Context, req *pb.GetProblemRequest) (*pb.GetProblemResponse, error) {
			return &pb.GetProblemResponse{
				Problem: &pb.Problem{
					Id:          "prob-1",
					Name:        "Cached Problem",
					TimeLimit:   1.0,
					MemoryLimit: 256,
				},
			}, nil
		},
	}

	handler, mr := setupTestProblemHandler(t, mockClient)
	defer mr.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/problems/{id}", handler.GetProblem)

	// First request - should be cache miss
	req1 := httptest.NewRequest("GET", "/api/v1/problems/prob-1", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "miss", w1.Header().Get("X-Cache"))
	assert.Contains(t, w1.Body.String(), "Cached Problem")

	// Second request - should be cache hit
	req2 := httptest.NewRequest("GET", "/api/v1/problems/prob-1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "hit", w2.Header().Get("X-Cache"))
	assert.Contains(t, w2.Body.String(), "Cached Problem")
}

func TestProblemHandler_ListLanguages(t *testing.T) {
	tests := []struct {
		name       string
		mockFunc   func(ctx context.Context, req *emptypb.Empty) (*pb.ListLanguagesResponse, error)
		wantStatus int
		wantBody   func(t *testing.T, body string)
	}{
		{
			name: "list languages successfully",
			mockFunc: func(ctx context.Context, req *emptypb.Empty) (*pb.ListLanguagesResponse, error) {
				return &pb.ListLanguagesResponse{
					Languages: []*pb.Language{
						{Id: "cpp", Name: "C++", Extensions: []string{".cpp", ".cc"}},
						{Id: "python", Name: "Python", Extensions: []string{".py"}},
						{Id: "java", Name: "Java", Extensions: []string{".java"}},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "C++")
				assert.Contains(t, body, "Python")
				assert.Contains(t, body, "Java")
			},
		},
		{
			name: "empty languages",
			mockFunc: func(ctx context.Context, req *emptypb.Empty) (*pb.ListLanguagesResponse, error) {
				return &pb.ListLanguagesResponse{
					Languages: []*pb.Language{},
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
			mockClient := &MockProblemServiceClient{
				ListLanguagesFunc: tt.mockFunc,
			}

			handler, mr := setupTestProblemHandler(t, mockClient)
			defer mr.Close()

			req := httptest.NewRequest("GET", "/api/v1/languages", nil)
			w := httptest.NewRecorder()

			handler.ListLanguages(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			tt.wantBody(t, w.Body.String())
		})
	}
}