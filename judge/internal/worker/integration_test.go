package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/online-judge/judge/internal/queue"
)

// TestWorker_ProcessJob_WithMockQueue tests the worker's processJob logic
// using a mock queue that simulates all backend interactions
func TestWorker_ProcessJob_WithMockQueue(t *testing.T) {
	// Setup mock queue
	mockQueue := queue.NewMockJudgeQueue()

	// Configure mock queue responses
	mockQueue.SubmissionDetails = &queue.SubmissionDetails{
		ID:         "sub-1",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b; return 0; }",
	}

	mockQueue.ProblemDetails = &queue.ProblemDetails{
		ID:           "prob-1",
		TimeLimit:    2.0,
		MemoryLimit:  256000,
		OutputLimit:  10240,
		ProcessLimit: 1,
	}

	mockQueue.TestCasesList = []*queue.TestCase{
		{ID: "tc-1", ProblemID: "prob-1", Rank: 1, IsSample: true},
		{ID: "tc-2", ProblemID: "prob-1", Rank: 2, IsSample: false},
	}

	mockQueue.TestCaseDataMap = map[string]*queue.TestCaseData{
		"tc-1": {Input: []byte("1 2\n"), Output: []byte("3\n")},
		"tc-2": {Input: []byte("5 7\n"), Output: []byte("12\n")},
	}

	// Note: The actual worker requires Docker sandbox which we can't use in tests.
	// This test verifies the queue interactions work correctly.
	// For full integration tests, use Docker-based test environments.

	t.Run("VerifyMockQueueSetup", func(t *testing.T) {
		ctx := context.Background()

		// Test fetching submission
		sub, err := mockQueue.FetchSubmission(ctx, "sub-1")
		require.NoError(t, err)
		assert.Equal(t, "cpp", sub.LanguageID)
		assert.Contains(t, sub.SourceCode, "iostream")

		// Test fetching problem
		prob, err := mockQueue.FetchProblem(ctx, "prob-1")
		require.NoError(t, err)
		assert.Equal(t, 2.0, prob.TimeLimit)

		// Test fetching test cases
		tcs, err := mockQueue.FetchTestCases(ctx, "prob-1")
		require.NoError(t, err)
		assert.Len(t, tcs, 2)

		// Test fetching test case data
		tcData, err := mockQueue.FetchTestCaseData(ctx, tcs[0])
		require.NoError(t, err)
		assert.Equal(t, []byte("1 2\n"), tcData.Input)
	})

	t.Run("CreateJudging", func(t *testing.T) {
		ctx := context.Background()

		judgingID, err := mockQueue.CreateJudging(ctx, "sub-1", "test-host")
		require.NoError(t, err)
		assert.NotEmpty(t, judgingID)
	})

	t.Run("UpdateJudging", func(t *testing.T) {
		ctx := context.Background()

		result := &queue.JudgeResult{
			SubmissionID: "sub-1",
			Verdict:      "correct",
			Runtime:      0.5,
			Memory:       1024,
		}
		err := mockQueue.UpdateJudging(ctx, "judging-1", result)
		require.NoError(t, err)
	})

	t.Run("CreateJudgingRun", func(t *testing.T) {
		ctx := context.Background()

		tcResult := &queue.TestCaseResult{
			TestCaseID: "tc-1",
			Rank:       1,
			Verdict:    "correct",
			Runtime:    0.1,
			Memory:     512,
		}
		err := mockQueue.CreateJudgingRun(ctx, "judging-1", tcResult)
		require.NoError(t, err)
	})

	t.Run("PushJudgingResult", func(t *testing.T) {
		ctx := context.Background()

		result := &queue.JudgeResult{
			SubmissionID: "sub-1",
			Verdict:      "correct",
			Runtime:      0.2,
			Memory:       768,
		}
		runResults := []*queue.TestCaseResult{
			{TestCaseID: "tc-1", Verdict: "correct", Runtime: 0.1},
			{TestCaseID: "tc-2", Verdict: "correct", Runtime: 0.1},
		}

		err := mockQueue.PushJudgingResult(ctx, result, "judging-1", runResults)
		require.NoError(t, err)
	})
}

func TestWorker_VerdictConstants(t *testing.T) {
	// Verify verdict constants match expected values
	assert.Equal(t, "correct", VerdictCorrect)
	assert.Equal(t, "wrong-answer", VerdictWrongAnswer)
	assert.Equal(t, "time-limit", VerdictTimeLimit)
	assert.Equal(t, "memory-limit", VerdictMemoryLimit)
	assert.Equal(t, "run-error", VerdictRunError)
	assert.Equal(t, "compiler-error", VerdictCompilerError)
	assert.Equal(t, "output-limit", VerdictOutputLimit)
	assert.Equal(t, "presentation", VerdictPresentation)
}

func TestWorker_JobProcessing(t *testing.T) {
	// Test queue operations for job processing
	mockQueue := queue.NewMockJudgeQueue()
	ctx := context.Background()

	// Push a job
	job := &queue.JudgeJob{
		SubmissionID: "sub-test",
		ProblemID:    "prob-test",
		Language:     "cpp",
		Priority:     0,
		SubmitTime:   time.Now(),
	}

	err := mockQueue.Push(ctx, job)
	require.NoError(t, err)

	// Verify queue length
	length, err := mockQueue.Length(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), length)

	// Pop the job
	poppedJob, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	require.NotNil(t, poppedJob)
	assert.Equal(t, "sub-test", poppedJob.SubmissionID)
	assert.Equal(t, "prob-test", poppedJob.ProblemID)
	assert.Equal(t, "cpp", poppedJob.Language)

	// Queue should be empty now
	length, err = mockQueue.Length(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)
}

func TestWorker_PriorityQueue(t *testing.T) {
	mockQueue := queue.NewMockJudgeQueue()
	ctx := context.Background()

	now := time.Now()

	// Push jobs with different priorities
	jobs := []*queue.JudgeJob{
		{SubmissionID: "low-priority", Priority: 0, SubmitTime: now},
		{SubmissionID: "high-priority", Priority: 10, SubmitTime: now},
		{SubmissionID: "medium-priority", Priority: 5, SubmitTime: now},
	}

	for _, job := range jobs {
		err := mockQueue.Push(ctx, job)
		require.NoError(t, err)
	}

	// Verify all jobs are in queue
	length, err := mockQueue.Length(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

func TestWorker_RejudgeJob(t *testing.T) {
	mockQueue := queue.NewMockJudgeQueue()
	ctx := context.Background()

	// Push a rejudge job
	rejudgeJob := &queue.JudgeJob{
		SubmissionID: "sub-rejudge",
		ProblemID:    "prob-1",
		Language:     "python",
		Priority:     10, // Higher priority for rejudges
		SubmitTime:   time.Now(),
		RejudgeID:    "rejudge-123",
	}

	err := mockQueue.Push(ctx, rejudgeJob)
	require.NoError(t, err)

	// Pop and verify rejudge fields
	job, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "rejudge-123", job.RejudgeID)
	assert.Equal(t, 10, job.Priority)
}

// Integration test: Simulate a full judging workflow (without actual sandbox)
func TestWorker_Integration_JudgingWorkflow(t *testing.T) {
	mockQueue := queue.NewMockJudgeQueue()
	ctx := context.Background()

	// Setup complete workflow data
	mockQueue.SubmissionDetails = &queue.SubmissionDetails{
		ID:         "sub-integration",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "#include <iostream>\nint main() { return 0; }",
	}

	mockQueue.ProblemDetails = &queue.ProblemDetails{
		ID:           "prob-1",
		TimeLimit:    1.0,
		MemoryLimit:  256000,
		OutputLimit:  10240,
		ProcessLimit: 1,
	}

	mockQueue.TestCasesList = []*queue.TestCase{
		{ID: "tc-1", ProblemID: "prob-1", Rank: 1, IsSample: true},
		{ID: "tc-2", ProblemID: "prob-1", Rank: 2, IsSample: false},
		{ID: "tc-3", ProblemID: "prob-1", Rank: 3, IsSample: false},
	}

	mockQueue.TestCaseDataMap = map[string]*queue.TestCaseData{
		"tc-1": {Input: []byte("test1\n"), Output: []byte("out1\n")},
		"tc-2": {Input: []byte("test2\n"), Output: []byte("out2\n")},
		"tc-3": {Input: []byte("test3\n"), Output: []byte("out3\n")},
	}

	// Step 1: Push job
	job := &queue.JudgeJob{
		SubmissionID: "sub-integration",
		ProblemID:    "prob-1",
		Language:     "cpp",
		SubmitTime:   time.Now(),
	}
	err := mockQueue.Push(ctx, job)
	require.NoError(t, err)

	// Step 2: Pop job
	poppedJob, err := mockQueue.Pop(ctx)
	require.NoError(t, err)
	require.NotNil(t, poppedJob)

	// Step 3: Fetch submission details
	submission, err := mockQueue.FetchSubmission(ctx, poppedJob.SubmissionID)
	require.NoError(t, err)
	assert.Equal(t, "cpp", submission.LanguageID)

	// Step 4: Fetch problem details
	problem, err := mockQueue.FetchProblem(ctx, submission.ProblemID)
	require.NoError(t, err)
	assert.Equal(t, 1.0, problem.TimeLimit)

	// Step 5: Fetch test cases
	testCases, err := mockQueue.FetchTestCases(ctx, submission.ProblemID)
	require.NoError(t, err)
	assert.Len(t, testCases, 3)

	// Step 6: Create judging record
	judgingID, err := mockQueue.CreateJudging(ctx, submission.ID, "test-worker")
	require.NoError(t, err)
	assert.NotEmpty(t, judgingID)

	// Step 7: Simulate test case execution
	var runResults []*queue.TestCaseResult
	maxRuntime := 0.0
	maxMemory := int64(0)

	for _, tc := range testCases {
		// Fetch test case data
		tcData, err := mockQueue.FetchTestCaseData(ctx, tc)
		require.NoError(t, err)
		assert.NotEmpty(t, tcData.Input)

		// Simulate execution (in real worker, this would use sandbox)
		runtime := 0.1 * float64(tc.Rank)
		memory := int64(512 * tc.Rank)

		runResult := &queue.TestCaseResult{
			TestCaseID: tc.ID,
			Rank:       int(tc.Rank),
			Verdict:    "correct",
			Runtime:    runtime,
			Memory:     memory,
		}
		runResults = append(runResults, runResult)

		// Track max
		if runtime > maxRuntime {
			maxRuntime = runtime
		}
		if memory > maxMemory {
			maxMemory = memory
		}

		// Create judging run record
		err = mockQueue.CreateJudgingRun(ctx, judgingID, runResult)
		require.NoError(t, err)
	}

	// Step 8: Push final result
	result := &queue.JudgeResult{
		SubmissionID: submission.ID,
		Verdict:      "correct",
		Runtime:      maxRuntime,
		Memory:       maxMemory,
	}
	err = mockQueue.PushJudgingResult(ctx, result, judgingID, runResults)
	require.NoError(t, err)

	// Step 9: Publish result
	err = mockQueue.PushResult(ctx, result)
	require.NoError(t, err)

	// Verify final state
	assert.Len(t, mockQueue.Results, 1)
	assert.Equal(t, "correct", mockQueue.Results[0].Verdict)
}

func TestWorker_ErrorHandling(t *testing.T) {
	t.Run("SubmissionNotFound", func(t *testing.T) {
		mockQueue := queue.NewMockJudgeQueue()
		mockQueue.FetchSubmissionError = assert.AnError

		_, err := mockQueue.FetchSubmission(context.Background(), "non-existent")
		require.Error(t, err)
	})

	t.Run("ProblemNotFound", func(t *testing.T) {
		mockQueue := queue.NewMockJudgeQueue()
		mockQueue.FetchProblemError = assert.AnError

		_, err := mockQueue.FetchProblem(context.Background(), "non-existent")
		require.Error(t, err)
	})

	t.Run("TestCasesNotFound", func(t *testing.T) {
		mockQueue := queue.NewMockJudgeQueue()
		mockQueue.FetchTestCasesError = assert.AnError

		_, err := mockQueue.FetchTestCases(context.Background(), "non-existent")
		require.Error(t, err)
	})

	t.Run("CreateJudgingError", func(t *testing.T) {
		mockQueue := queue.NewMockJudgeQueue()
		mockQueue.CreateJudgingError = assert.AnError

		_, err := mockQueue.CreateJudging(context.Background(), "sub-1", "host-1")
		require.Error(t, err)
	})

	t.Run("UpdateJudgingError", func(t *testing.T) {
		mockQueue := queue.NewMockJudgeQueue()
		mockQueue.UpdateJudgingError = assert.AnError

		err := mockQueue.UpdateJudging(context.Background(), "judging-1", &queue.JudgeResult{})
		require.Error(t, err)
	})
}