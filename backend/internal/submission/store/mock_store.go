package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// SubmissionStoreInterface defines the interface for SubmissionStore
type SubmissionStoreInterface interface {
	Create(ctx context.Context, userID, problemID, contestID, languageID, sourceCode string) (string, error)
	GetByID(ctx context.Context, id string) (*Submission, error)
	List(ctx context.Context, userID, problemID, contestID string, page, pageSize int32) ([]*SubmissionSummary, int32, error)
	CreateJudging(ctx context.Context, submissionID, judgehostID string) (string, error)
	UpdateJudging(ctx context.Context, judgingID string, verdict string, maxRuntime float64, maxMemory int64, compileSuccess bool) error
	CreateJudgingRun(ctx context.Context, judgingID, testCaseID string, rank int, verdict string, runtime, wallTime float64, memory int64) error
	GetJudging(ctx context.Context, submissionID string) (*Judging, error)
	GetJudgingByID(ctx context.Context, judgingID string) (*Judging, error)
	GetJudgingRuns(ctx context.Context, judgingID string) ([]*JudgingRun, error)
	UpdateSubmissionStatus(ctx context.Context, submissionID string, status string) error
}

// MockSubmissionStore is a mock implementation of SubmissionStoreInterface for testing
type MockSubmissionStore struct {
	Submissions   map[string]*Submission
	Judgings      map[string]*Judging
	JudgingRuns   map[string][]*JudgingRun
	CreateError   error
	GetError      error
	ListError     error
	JudgingError  error
}

func NewMockSubmissionStore() *MockSubmissionStore {
	return &MockSubmissionStore{
		Submissions: make(map[string]*Submission),
		Judgings:    make(map[string]*Judging),
		JudgingRuns: make(map[string][]*JudgingRun),
	}
}

func (m *MockSubmissionStore) Create(ctx context.Context, userID, problemID, contestID, languageID, sourceCode string) (string, error) {
	if m.CreateError != nil {
		return "", m.CreateError
	}

	id := uuid.New().String()
	m.Submissions[id] = &Submission{
		ID:         id,
		UserID:     userID,
		ProblemID:  problemID,
		ContestID:  contestID,
		LanguageID: languageID,
		SourceCode: sourceCode,
		SubmitTime: time.Now(),
	}
	return id, nil
}

func (m *MockSubmissionStore) GetByID(ctx context.Context, id string) (*Submission, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}

	sub, ok := m.Submissions[id]
	if !ok {
		return nil, errors.New("submission not found")
	}
	return sub, nil
}

func (m *MockSubmissionStore) List(ctx context.Context, userID, problemID, contestID string, page, pageSize int32) ([]*SubmissionSummary, int32, error) {
	if m.ListError != nil {
		return nil, 0, m.ListError
	}

	var submissions []*SubmissionSummary
	for _, sub := range m.Submissions {
		if userID != "" && sub.UserID != userID {
			continue
		}
		if problemID != "" && sub.ProblemID != problemID {
			continue
		}
		if contestID != "" && sub.ContestID != contestID {
			continue
		}
		submissions = append(submissions, &SubmissionSummary{
			ID:         sub.ID,
			UserID:     sub.UserID,
			ProblemID:  sub.ProblemID,
			LanguageID: sub.LanguageID,
			SubmitTime: sub.SubmitTime,
		})
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > int32(len(submissions)) {
		end = int32(len(submissions))
	}
	if start > int32(len(submissions)) {
		return []*SubmissionSummary{}, int32(len(submissions)), nil
	}

	return submissions[start:end], int32(len(submissions)), nil
}

func (m *MockSubmissionStore) CreateJudging(ctx context.Context, submissionID, judgehostID string) (string, error) {
	if m.JudgingError != nil {
		return "", m.JudgingError
	}

	id := uuid.New().String()
	m.Judgings[id] = &Judging{
		ID:           id,
		SubmissionID: submissionID,
		JudgehostID:  judgehostID,
		StartTime:    time.Now(),
		Valid:        true,
	}
	return id, nil
}

func (m *MockSubmissionStore) UpdateJudging(ctx context.Context, judgingID string, verdict string, maxRuntime float64, maxMemory int64, compileSuccess bool) error {
	j, ok := m.Judgings[judgingID]
	if !ok {
		return errors.New("judging not found")
	}
	j.Verdict = verdict
	j.MaxRuntime = maxRuntime
	j.MaxMemory = int32(maxMemory)
	j.CompileSuccess = compileSuccess
	j.EndTime = time.Now()
	return nil
}

func (m *MockSubmissionStore) CreateJudgingRun(ctx context.Context, judgingID, testCaseID string, rank int, verdict string, runtime, wallTime float64, memory int64) error {
	id := uuid.New().String()
	run := &JudgingRun{
		ID:         id,
		JudgingID:  judgingID,
		TestCaseID: testCaseID,
		Rank:       int32(rank),
		Verdict:    verdict,
		Runtime:    runtime,
		WallTime:   wallTime,
		Memory:     int32(memory),
	}
	m.JudgingRuns[judgingID] = append(m.JudgingRuns[judgingID], run)
	return nil
}

func (m *MockSubmissionStore) GetJudging(ctx context.Context, submissionID string) (*Judging, error) {
	if m.JudgingError != nil {
		return nil, m.JudgingError
	}

	for _, j := range m.Judgings {
		if j.SubmissionID == submissionID && j.Valid {
			return j, nil
		}
	}
	return nil, errors.New("judging not found")
}

func (m *MockSubmissionStore) GetJudgingByID(ctx context.Context, judgingID string) (*Judging, error) {
	if m.JudgingError != nil {
		return nil, m.JudgingError
	}

	j, ok := m.Judgings[judgingID]
	if !ok {
		return nil, errors.New("judging not found")
	}
	return j, nil
}

func (m *MockSubmissionStore) GetJudgingRuns(ctx context.Context, judgingID string) ([]*JudgingRun, error) {
	runs, ok := m.JudgingRuns[judgingID]
	if !ok {
		return []*JudgingRun{}, nil
	}
	return runs, nil
}

func (m *MockSubmissionStore) UpdateSubmissionStatus(ctx context.Context, submissionID string, status string) error {
	_, ok := m.Submissions[submissionID]
	if !ok {
		return errors.New("submission not found")
	}
	// Mock implementation - no status field in mock
	return nil
}

// Ensure MockSubmissionStore implements SubmissionStoreInterface
var _ SubmissionStoreInterface = (*MockSubmissionStore)(nil)