package queue

import (
	"context"
)

// MockJudgeQueue is a mock implementation of JudgeQueue for testing
type MockJudgeQueue struct {
	Jobs          []*JudgeJob
	Results       []*JudgeResult
	PushError     error
	PopError      error
	PushResultError error

	// For simulating Pop behavior
	PopIndex int
	PopJobs  []*JudgeJob

	// For FetchSubmission
	SubmissionDetails *SubmissionDetails
	FetchSubmissionError error

	// For FetchProblem
	ProblemDetails *ProblemDetails
	FetchProblemError error

	// For FetchTestCases
	TestCasesList []*TestCase
	FetchTestCasesError error

	// For FetchTestCaseData
	TestCaseDataMap map[string]*TestCaseData
	FetchTestCaseDataError error

	// For CreateJudging
	CreatedJudgingID string
	CreateJudgingError error

	// For UpdateJudging
	UpdateJudgingError error

	// For CreateJudgingRun
	CreateJudgingRunError error

	// For PushJudgingResult
	PushJudgingResultError error
}

func NewMockJudgeQueue() *MockJudgeQueue {
	return &MockJudgeQueue{
		Jobs:           make([]*JudgeJob, 0),
		Results:        make([]*JudgeResult, 0),
		TestCaseDataMap: make(map[string]*TestCaseData),
	}
}

func (m *MockJudgeQueue) Push(ctx context.Context, job *JudgeJob) error {
	if m.PushError != nil {
		return m.PushError
	}
	m.Jobs = append(m.Jobs, job)
	return nil
}

func (m *MockJudgeQueue) Pop(ctx context.Context) (*JudgeJob, error) {
	if m.PopError != nil {
		return nil, m.PopError
	}

	// If we have pre-configured PopJobs, use those
	if len(m.PopJobs) > 0 {
		if m.PopIndex >= len(m.PopJobs) {
			return nil, nil
		}
		job := m.PopJobs[m.PopIndex]
		m.PopIndex++
		return job, nil
	}

	// Otherwise use the regular Jobs slice
	if len(m.Jobs) == 0 {
		return nil, nil
	}

	job := m.Jobs[0]
	m.Jobs = m.Jobs[1:]
	return job, nil
}

func (m *MockJudgeQueue) PushResult(ctx context.Context, result *JudgeResult) error {
	if m.PushResultError != nil {
		return m.PushResultError
	}
	m.Results = append(m.Results, result)
	return nil
}

func (m *MockJudgeQueue) SubscribeResults(ctx context.Context) <-chan *JudgeResult {
	ch := make(chan *JudgeResult, 100)
	go func() {
		defer close(ch)
		for _, result := range m.Results {
			ch <- result
		}
	}()
	return ch
}

func (m *MockJudgeQueue) Length(ctx context.Context) (int64, error) {
	return int64(len(m.Jobs)), nil
}

func (m *MockJudgeQueue) Ping(ctx context.Context) error {
	return nil
}

func (m *MockJudgeQueue) FetchSubmission(ctx context.Context, submissionID string) (*SubmissionDetails, error) {
	if m.FetchSubmissionError != nil {
		return nil, m.FetchSubmissionError
	}
	if m.SubmissionDetails != nil {
		return m.SubmissionDetails, nil
	}
	return &SubmissionDetails{
		ID:         submissionID,
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "#include <iostream>\nint main() { return 0; }",
	}, nil
}

func (m *MockJudgeQueue) FetchProblem(ctx context.Context, problemID string) (*ProblemDetails, error) {
	if m.FetchProblemError != nil {
		return nil, m.FetchProblemError
	}
	if m.ProblemDetails != nil {
		return m.ProblemDetails, nil
	}
	return &ProblemDetails{
		ID:           problemID,
		TimeLimit:    1.0,
		MemoryLimit:  256000,
		OutputLimit:  10240,
		ProcessLimit: 1,
	}, nil
}

func (m *MockJudgeQueue) FetchTestCases(ctx context.Context, problemID string) ([]*TestCase, error) {
	if m.FetchTestCasesError != nil {
		return nil, m.FetchTestCasesError
	}
	if m.TestCasesList != nil {
		return m.TestCasesList, nil
	}
	return []*TestCase{
		{
			ID:        "tc-1",
			ProblemID: problemID,
			Rank:      1,
			IsSample:  true,
		},
	}, nil
}

func (m *MockJudgeQueue) FetchTestCaseData(ctx context.Context, testCase *TestCase) (*TestCaseData, error) {
	if m.FetchTestCaseDataError != nil {
		return nil, m.FetchTestCaseDataError
	}
	if data, ok := m.TestCaseDataMap[testCase.ID]; ok {
		return data, nil
	}
	return &TestCaseData{
		Input:  []byte("1 2\n"),
		Output: []byte("3\n"),
	}, nil
}

func (m *MockJudgeQueue) CreateJudging(ctx context.Context, submissionID string, judgehostID string) (string, error) {
	if m.CreateJudgingError != nil {
		return "", m.CreateJudgingError
	}
	if m.CreatedJudgingID != "" {
		return m.CreatedJudgingID, nil
	}
	return "judging-" + submissionID, nil
}

func (m *MockJudgeQueue) UpdateJudging(ctx context.Context, judgingID string, result *JudgeResult) error {
	return m.UpdateJudgingError
}

func (m *MockJudgeQueue) CreateJudgingRun(ctx context.Context, judgingID string, testCaseResult *TestCaseResult) error {
	return m.CreateJudgingRunError
}

func (m *MockJudgeQueue) PushJudgingResult(ctx context.Context, result *JudgeResult, judgingID string, runResults []*TestCaseResult) error {
	return m.PushJudgingResultError
}