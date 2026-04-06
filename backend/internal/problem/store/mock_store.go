package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	pb "github.com/online-judge/backend/gen/go/problem/v1"
)

// ProblemStoreInterface defines the interface for ProblemStore
type ProblemStoreInterface interface {
	List(ctx context.Context, req *pb.ListProblemsRequest) ([]*pb.ProblemSummary, int32, error)
	GetByID(ctx context.Context, id string) (*pb.Problem, error)
	Create(ctx context.Context, req *pb.CreateProblemRequest) (string, error)
	Update(ctx context.Context, id string, req *pb.UpdateProblemRequest) error
	Delete(ctx context.Context, id string) error
	ListTestCases(ctx context.Context, problemID string, samplesOnly bool) ([]*pb.TestCase, error)
	CreateTestCase(ctx context.Context, req *pb.CreateTestCaseRequest) (string, string, string, error)
}

// MockProblemStore is a mock implementation of ProblemStoreInterface for testing
type MockProblemStore struct {
	Problems    map[string]*pb.Problem
	TestCases   map[string][]*pb.TestCase
	CreateError error
	GetError    error
	ListError   error
	UpdateError error
	DeleteError error
}

func NewMockProblemStore() *MockProblemStore {
	return &MockProblemStore{
		Problems:  make(map[string]*pb.Problem),
		TestCases: make(map[string][]*pb.TestCase),
	}
}

func (m *MockProblemStore) List(ctx context.Context, req *pb.ListProblemsRequest) ([]*pb.ProblemSummary, int32, error) {
	if m.ListError != nil {
		return nil, 0, m.ListError
	}

	var problems []*pb.ProblemSummary
	for _, p := range m.Problems {
		// Always filter to published problems (matches real store behavior)
		if !p.IsPublished {
			continue
		}

		// Apply difficulty filter if specified
		if req.GetDifficulty() != "" && p.Difficulty != req.GetDifficulty() {
			continue
		}

		// Apply search filter if specified
		if req.GetSearch() != "" {
			// Simple contains check for mock
			// Real implementation would use ILIKE
			continue
		}

		problems = append(problems, &pb.ProblemSummary{
			Id:           p.Id,
			ExternalId:    p.ExternalId,
			Name:         p.Name,
			Difficulty:   p.Difficulty,
			TimeLimit:    p.TimeLimit,
			MemoryLimit:  p.MemoryLimit,
			Points:       p.Points,
			AllowSubmit:  p.AllowSubmit,
		})
	}

	pageSize := req.GetPagination().GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	page := req.GetPagination().GetPage()
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > int32(len(problems)) {
		end = int32(len(problems))
	}
	if start > int32(len(problems)) {
		return []*pb.ProblemSummary{}, int32(len(problems)), nil
	}

	return problems[start:end], int32(len(problems)), nil
}

func (m *MockProblemStore) GetByID(ctx context.Context, id string) (*pb.Problem, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}

	p, ok := m.Problems[id]
	if !ok {
		return nil, errors.New("problem not found")
	}
	return p, nil
}

func (m *MockProblemStore) Create(ctx context.Context, req *pb.CreateProblemRequest) (string, error) {
	if m.CreateError != nil {
		return "", m.CreateError
	}

	id := uuid.New().String()
	m.Problems[id] = &pb.Problem{
		Id:           id,
		ExternalId:   req.GetExternalId(),
		Name:         req.GetName(),
		TimeLimit:    req.GetTimeLimit(),
		MemoryLimit:  req.GetMemoryLimit(),
		OutputLimit:  req.GetOutputLimit(),
		Difficulty:   req.GetDifficulty(),
		Points:       req.GetPoints(),
		IsPublished:  true,
		AllowSubmit:  true,
	}
	return id, nil
}

func (m *MockProblemStore) Update(ctx context.Context, id string, req *pb.UpdateProblemRequest) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}

	p, ok := m.Problems[id]
	if !ok {
		return errors.New("problem not found")
	}

	p.Name = req.GetName()
	p.TimeLimit = req.GetTimeLimit()
	p.MemoryLimit = req.GetMemoryLimit()
	p.IsPublished = req.GetIsPublished()
	p.AllowSubmit = req.GetAllowSubmit()
	return nil
}

func (m *MockProblemStore) Delete(ctx context.Context, id string) error {
	if m.DeleteError != nil {
		return m.DeleteError
	}

	if _, ok := m.Problems[id]; !ok {
		return errors.New("problem not found")
	}
	delete(m.Problems, id)
	return nil
}

func (m *MockProblemStore) ListTestCases(ctx context.Context, problemID string, samplesOnly bool) ([]*pb.TestCase, error) {
	tcs, ok := m.TestCases[problemID]
	if !ok {
		return []*pb.TestCase{}, nil
	}

	if samplesOnly {
		var samples []*pb.TestCase
		for _, tc := range tcs {
			if tc.IsSample {
				samples = append(samples, tc)
			}
		}
		return samples, nil
	}
	return tcs, nil
}

func (m *MockProblemStore) CreateTestCase(ctx context.Context, req *pb.CreateTestCaseRequest) (string, string, string, error) {
	id := uuid.New().String()
	tc := &pb.TestCase{
		Id:          id,
		ProblemId:   req.GetProblemId(),
		Rank:        req.GetRank(),
		IsSample:    req.GetIsSample(),
		Description: req.GetDescription(),
	}

	m.TestCases[req.GetProblemId()] = append(m.TestCases[req.GetProblemId()], tc)
	return id, "input/path", "output/path", nil
}