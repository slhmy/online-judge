package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockJudgeQueue_PushPop(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Push jobs
	job1 := &JudgeJob{
		SubmissionID: "sub-1",
		ProblemID:    "prob-1",
		Language:     "cpp",
		Priority:     0,
		SubmitTime:   time.Now(),
	}
	job2 := &JudgeJob{
		SubmissionID: "sub-2",
		ProblemID:    "prob-2",
		Language:     "python",
		Priority:     10, // Higher priority
		SubmitTime:   time.Now(),
	}

	err := mockQueue.Push(ctx, job1)
	require.NoError(t, err)

	err = mockQueue.Push(ctx, job2)
	require.NoError(t, err)

	// Check queue length
	length, err := mockQueue.Length(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), length)

	// Pop job (FIFO for mock)
	poppedJob, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "sub-1", poppedJob.SubmissionID)

	// Pop remaining job
	poppedJob2, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "sub-2", poppedJob2.SubmissionID)

	// Empty queue
	emptyPop, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Nil(t, emptyPop)
}

func TestMockJudgeQueue_PopWithPreConfiguredJobs(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Pre-configure PopJobs
	mockQueue.PopJobs = []*JudgeJob{
		{SubmissionID: "pre-1", ProblemID: "prob-1"},
		{SubmissionID: "pre-2", ProblemID: "prob-2"},
	}

	// Pop first
	job1, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "pre-1", job1.SubmissionID)

	// Pop second
	job2, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "pre-2", job2.SubmissionID)

	// Empty
	job3, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Nil(t, job3)
}

func TestMockJudgeQueue_Integration_PushResult(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	result := &JudgeResult{
		SubmissionID: "sub-1",
		Verdict:      "correct",
		Runtime:      0.5,
		Memory:       1024,
	}

	err := mockQueue.PushResult(ctx, result)
	require.NoError(t, err)

	assert.Len(t, mockQueue.Results, 1)
	assert.Equal(t, "correct", mockQueue.Results[0].Verdict)
}

func TestMockJudgeQueue_SubscribeResults(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Add some results first
	mockQueue.Results = []*JudgeResult{
		{SubmissionID: "sub-1", Verdict: "correct"},
		{SubmissionID: "sub-2", Verdict: "wrong-answer"},
	}

	// Subscribe
	ch := mockQueue.SubscribeResults(ctx)

	// Read results
	var results []*JudgeResult
	for result := range ch {
		results = append(results, result)
	}

	assert.Len(t, results, 2)
}

func TestMockJudgeQueue_Integration_FetchSubmission(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Default behavior
	details, err := mockQueue.FetchSubmission(ctx, "sub-1")
	require.NoError(t, err)
	assert.Equal(t, "sub-1", details.ID)
	assert.Equal(t, "cpp", details.LanguageID)
	assert.NotEmpty(t, details.SourceCode)

	// Custom submission details
	mockQueue.SubmissionDetails = &SubmissionDetails{
		ID:         "sub-custom",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "python",
		SourceCode: "print('hello')",
	}

	details, err = mockQueue.FetchSubmission(ctx, "sub-custom")
	require.NoError(t, err)
	assert.Equal(t, "python", details.LanguageID)
	assert.Equal(t, "print('hello')", details.SourceCode)

	// Error case
	mockQueue.FetchSubmissionError = assert.AnError
	_, err = mockQueue.FetchSubmission(ctx, "sub-error")
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_FetchProblem(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Default behavior
	details, err := mockQueue.FetchProblem(ctx, "prob-1")
	require.NoError(t, err)
	assert.Equal(t, "prob-1", details.ID)
	assert.Equal(t, 1.0, details.TimeLimit)
	assert.Equal(t, int32(256000), details.MemoryLimit)

	// Custom problem details
	mockQueue.ProblemDetails = &ProblemDetails{
		ID:              "prob-custom",
		TimeLimit:       2.5,
		MemoryLimit:     512000,
		OutputLimit:     20480,
		ProcessLimit:    2,
		SpecialCompare:  "custom_validator",
	}

	details, err = mockQueue.FetchProblem(ctx, "prob-custom")
	require.NoError(t, err)
	assert.Equal(t, 2.5, details.TimeLimit)
	assert.Equal(t, "custom_validator", details.SpecialCompare)

	// Error case
	mockQueue.FetchProblemError = assert.AnError
	_, err = mockQueue.FetchProblem(ctx, "prob-error")
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_FetchTestCases(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Default behavior
	testCases, err := mockQueue.FetchTestCases(ctx, "prob-1")
	require.NoError(t, err)
	assert.Len(t, testCases, 1)
	assert.Equal(t, "tc-1", testCases[0].ID)
	assert.True(t, testCases[0].IsSample)

	// Custom test cases
	mockQueue.TestCasesList = []*TestCase{
		{ID: "tc-1", ProblemID: "prob-1", Rank: 1, IsSample: true},
		{ID: "tc-2", ProblemID: "prob-1", Rank: 2, IsSample: false},
		{ID: "tc-3", ProblemID: "prob-1", Rank: 3, IsSample: false},
	}

	testCases, err = mockQueue.FetchTestCases(ctx, "prob-1")
	require.NoError(t, err)
	assert.Len(t, testCases, 3)

	// Error case
	mockQueue.FetchTestCasesError = assert.AnError
	_, err = mockQueue.FetchTestCases(ctx, "prob-error")
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_FetchTestCaseData(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	tc := &TestCase{ID: "tc-1", ProblemID: "prob-1"}

	// Default behavior
	data, err := mockQueue.FetchTestCaseData(ctx, tc)
	require.NoError(t, err)
	assert.Equal(t, []byte("1 2\n"), data.Input)
	assert.Equal(t, []byte("3\n"), data.Output)

	// Custom test case data
	mockQueue.TestCaseDataMap["tc-custom"] = &TestCaseData{
		Input:  []byte("5\n1 2 3 4 5\n"),
		Output: []byte("15\n"),
	}

	tcCustom := &TestCase{ID: "tc-custom"}
	data, err = mockQueue.FetchTestCaseData(ctx, tcCustom)
	require.NoError(t, err)
	assert.Equal(t, []byte("5\n1 2 3 4 5\n"), data.Input)

	// Error case
	mockQueue.FetchTestCaseDataError = assert.AnError
	_, err = mockQueue.FetchTestCaseData(ctx, &TestCase{ID: "tc-error"})
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_CreateJudging(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Default behavior
	judgingID, err := mockQueue.CreateJudging(ctx, "sub-1", "host-1")
	require.NoError(t, err)
	assert.Equal(t, "judging-sub-1", judgingID)

	// Custom ID
	mockQueue.CreatedJudgingID = "custom-judging-123"
	judgingID, err = mockQueue.CreateJudging(ctx, "sub-2", "host-1")
	require.NoError(t, err)
	assert.Equal(t, "custom-judging-123", judgingID)

	// Error case
	mockQueue.CreateJudgingError = assert.AnError
	_, err = mockQueue.CreateJudging(ctx, "sub-error", "host-1")
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_UpdateJudging(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	result := &JudgeResult{
		SubmissionID: "sub-1",
		Verdict:      "correct",
		Runtime:      0.5,
		Memory:       1024,
	}

	// Default behavior (no error)
	err := mockQueue.UpdateJudging(ctx, "judging-1", result)
	require.NoError(t, err)

	// Error case
	mockQueue.UpdateJudgingError = assert.AnError
	err = mockQueue.UpdateJudging(ctx, "judging-error", result)
	require.Error(t, err)
}

func TestMockJudgeQueue_CreateJudgingRun(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	tcResult := &TestCaseResult{
		TestCaseID: "tc-1",
		Rank:       1,
		Verdict:    "correct",
		Runtime:    0.1,
		Memory:     512,
	}

	// Default behavior (no error)
	err := mockQueue.CreateJudgingRun(ctx, "judging-1", tcResult)
	require.NoError(t, err)

	// Error case
	mockQueue.CreateJudgingRunError = assert.AnError
	err = mockQueue.CreateJudgingRun(ctx, "judging-error", tcResult)
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_PushJudgingResult(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	result := &JudgeResult{
		SubmissionID: "sub-1",
		Verdict:      "correct",
	}

	runResults := []*TestCaseResult{
		{TestCaseID: "tc-1", Verdict: "correct"},
		{TestCaseID: "tc-2", Verdict: "correct"},
	}

	// Default behavior (no error)
	err := mockQueue.PushJudgingResult(ctx, result, "judging-1", runResults)
	require.NoError(t, err)

	// Error case
	mockQueue.PushJudgingResultError = assert.AnError
	err = mockQueue.PushJudgingResult(ctx, result, "judging-error", runResults)
	require.Error(t, err)
}

func TestMockJudgeQueue_Integration_Ping(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	err := mockQueue.Ping(ctx)
	require.NoError(t, err)
}

// Integration test: Full queue workflow
func TestMockJudgeQueue_Integration_FullWorkflow(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	ctx := context.Background()

	// Setup
	mockQueue.SubmissionDetails = &SubmissionDetails{
		ID:         "sub-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "#include <iostream>\nint main() { return 0; }",
	}
	mockQueue.ProblemDetails = &ProblemDetails{
		ID:           "prob-1",
		TimeLimit:    2.0,
		MemoryLimit:  256000,
		OutputLimit:  10240,
		ProcessLimit: 1,
	}
	mockQueue.TestCasesList = []*TestCase{
		{ID: "tc-1", ProblemID: "prob-1", Rank: 1, IsSample: true},
		{ID: "tc-2", ProblemID: "prob-1", Rank: 2, IsSample: false},
	}
	mockQueue.TestCaseDataMap["tc-1"] = &TestCaseData{
		Input:  []byte("1 2\n"),
		Output: []byte("3\n"),
	}
	mockQueue.TestCaseDataMap["tc-2"] = &TestCaseData{
		Input:  []byte("5 7\n"),
		Output: []byte("12\n"),
	}

	// Step 1: Push job to queue
	job := &JudgeJob{
		SubmissionID: "sub-1",
		ProblemID:    "prob-1",
		Language:     "cpp",
		Priority:     0,
		SubmitTime:   time.Now(),
	}
	err := mockQueue.Push(ctx, job)
	require.NoError(t, err)

	// Step 2: Pop job
	poppedJob, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "sub-1", poppedJob.SubmissionID)

	// Step 3: Fetch submission
	submission, err := mockQueue.FetchSubmission(ctx, poppedJob.SubmissionID)
	require.NoError(t, err)
	assert.Equal(t, "cpp", submission.LanguageID)

	// Step 4: Fetch problem
	problem, err := mockQueue.FetchProblem(ctx, submission.ProblemID)
	require.NoError(t, err)
	assert.Equal(t, 2.0, problem.TimeLimit)

	// Step 5: Fetch test cases
	testCases, err := mockQueue.FetchTestCases(ctx, submission.ProblemID)
	require.NoError(t, err)
	assert.Len(t, testCases, 2)

	// Step 6: Fetch test case data
	tcData, err := mockQueue.FetchTestCaseData(ctx, testCases[0])
	require.NoError(t, err)
	assert.NotEmpty(t, tcData.Input)

	// Step 7: Create judging record
	judgingID, err := mockQueue.CreateJudging(ctx, submission.ID, "host-1")
	require.NoError(t, err)
	assert.NotEmpty(t, judgingID)

	// Step 8: Create test case run results
	tcResult := &TestCaseResult{
		TestCaseID: testCases[0].ID,
		Rank:       int(testCases[0].Rank),
		Verdict:    "correct",
		Runtime:    0.1,
		Memory:     512,
	}
	err = mockQueue.CreateJudgingRun(ctx, judgingID, tcResult)
	require.NoError(t, err)

	// Step 9: Push final result
	result := &JudgeResult{
		SubmissionID: submission.ID,
		Verdict:      "correct",
		Runtime:      0.2,
		Memory:       1024,
	}
	err = mockQueue.PushResult(ctx, result)
	require.NoError(t, err)

	// Verify
	assert.Len(t, mockQueue.Results, 1)
	assert.Equal(t, "correct", mockQueue.Results[0].Verdict)
}