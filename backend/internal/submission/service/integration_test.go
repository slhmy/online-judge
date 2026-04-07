package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/submission/v1"
	"github.com/online-judge/backend/internal/submission/store"
)

func setupTestWithRedis(t *testing.T) (*SubmissionService, *miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, rdb)

	return service, mr, rdb
}

func TestSubmissionService_Integration_CreateSubmission(t *testing.T) {
	service, mr, rdb := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Create submission
	resp, err := service.CreateSubmission(ctx, &pb.CreateSubmissionRequest{
		ProblemId:  "prob-1",
		LanguageId: "cpp",
		SourceCode: "#include <iostream>\nint main() { return 0; }",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
	assert.Equal(t, pb.SubmissionStatus_SUBMISSION_STATUS_QUEUED, resp.Status)

	// Verify source code was cached in Redis
	sourceKey := "submission:" + resp.Id + ":source"
	source, err := mr.Get(sourceKey)
	require.NoError(t, err)
	assert.Equal(t, "#include <iostream>\nint main() { return 0; }", source)

	// Verify job was added to queue using Redis client
	queueLen, err := rdb.ZCard(ctx, "judge:queue").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), queueLen)
}

func TestSubmissionService_Integration_CreateSubmissionWithContest(t *testing.T) {
	service, mr, _ := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	resp, err := service.CreateSubmission(ctx, &pb.CreateSubmissionRequest{
		ProblemId:  "prob-1",
		ContestId:  "contest-1",
		LanguageId: "python",
		SourceCode: "print('hello')",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
}

func TestSubmissionService_Integration_GetSubmission(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	// Setup mock data
	mockStore.Submissions["sub-1"] = &store.Submission{
		ID:         "sub-1",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "int main() { return 0; }",
		SubmitTime: time.Now(),
	}
	mockStore.Judgings["jud-1"] = &store.Judging{
		ID:           "jud-1",
		SubmissionID: "sub-1",
		JudgehostID:  "host-1",
		Verdict:      "correct",
		MaxRuntime:   0.5,
		MaxMemory:    1024,
		Valid:        true,
	}

	resp, err := service.GetSubmission(ctx, &pb.GetSubmissionRequest{Id: "sub-1"})
	require.NoError(t, err)
	assert.Equal(t, "sub-1", resp.Submission.Id)
	assert.Equal(t, "prob-1", resp.Submission.ProblemId)
	assert.NotNil(t, resp.LatestJudging)
	assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.LatestJudging.Verdict)
}

func TestSubmissionService_Integration_GetSubmissionWithRedisCache(t *testing.T) {
	service, mr, rdb := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Setup mock submission
	mockStore := store.NewMockSubmissionStore()
	service.store = mockStore

	mockStore.Submissions["sub-1"] = &store.Submission{
		ID:         "sub-1",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "code",
		SubmitTime: time.Now(),
	}

	// Set judging info in Redis cache using Redis client
	judgingKey := "judging:judging-sub-1:meta"
	rdb.HSet(ctx, judgingKey, map[string]interface{}{
		"verdict":      "wrong-answer",
		"max_runtime":  "1.5",
		"max_memory":   "2048",
		"judgehost_id": "host-1",
		"status":       "completed",
	})

	resp, err := service.GetSubmission(ctx, &pb.GetSubmissionRequest{Id: "sub-1"})
	require.NoError(t, err)
	assert.Equal(t, "sub-1", resp.Submission.Id)
	// Should have judging from Redis cache
	assert.NotNil(t, resp.LatestJudging)
	assert.Equal(t, pb.Verdict_VERDICT_WRONG_ANSWER, resp.LatestJudging.Verdict)
}

func TestSubmissionService_Integration_ListSubmissions(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	// Create multiple submissions
	mockStore.Submissions["sub-1"] = &store.Submission{
		ID:         "sub-1",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SubmitTime: time.Now(),
	}
	mockStore.Submissions["sub-2"] = &store.Submission{
		ID:         "sub-2",
		UserID:     "user-1",
		ProblemID:  "prob-2",
		LanguageID: "python",
		SubmitTime: time.Now().Add(-time.Hour),
	}
	mockStore.Submissions["sub-3"] = &store.Submission{
		ID:         "sub-3",
		UserID:     "user-2",
		ProblemID:  "prob-1",
		LanguageID: "java",
		SubmitTime: time.Now(),
	}

	// Add judgings for verdict lookup
	mockStore.Judgings["jud-1"] = &store.Judging{
		SubmissionID: "sub-1",
		Verdict:      "correct",
		Valid:        true,
	}
	mockStore.Judgings["jud-2"] = &store.Judging{
		SubmissionID: "sub-2",
		Verdict:      "wrong-answer",
		Valid:        true,
	}

	// List all
	resp, err := service.ListSubmissions(ctx, &pb.ListSubmissionsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, resp.Submissions, 3)

	// List by user
	userResp, err := service.ListSubmissions(ctx, &pb.ListSubmissionsRequest{
		UserId:     "user-1",
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, userResp.Submissions, 2)

	// List by problem
	probResp, err := service.ListSubmissions(ctx, &pb.ListSubmissionsRequest{
		ProblemId:  "prob-1",
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, probResp.Submissions, 2)
}

func TestSubmissionService_Integration_GetJudging(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	// Setup judging
	mockStore.Judgings["jud-1"] = &store.Judging{
		ID:             "jud-1",
		SubmissionID:   "sub-1",
		JudgehostID:    "host-1",
		Verdict:        "timelimit",
		MaxRuntime:     2.0,
		MaxMemory:      512,
		CompileSuccess: true,
		Valid:          true,
		StartTime:      time.Now().Add(-time.Minute),
		EndTime:        time.Now(),
	}

	// Get by submission ID
	resp, err := service.GetJudging(ctx, &pb.GetJudgingRequest{SubmissionId: "sub-1"})
	require.NoError(t, err)
	assert.Equal(t, "jud-1", resp.Judging.Id)
	assert.Equal(t, pb.Verdict_VERDICT_TIMELIMIT, resp.Judging.Verdict)
	assert.Equal(t, 2.0, resp.Judging.MaxRuntime)

	// Get by judging ID
	resp2, err := service.GetJudging(ctx, &pb.GetJudgingRequest{JudgingId: "jud-1"})
	require.NoError(t, err)
	assert.Equal(t, "jud-1", resp2.Judging.Id)
}

func TestSubmissionService_Integration_GetJudgingRuns(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	// Setup judging and runs
	mockStore.Judgings["jud-1"] = &store.Judging{
		ID:           "jud-1",
		SubmissionID: "sub-1",
		Valid:        true,
	}
	mockStore.JudgingRuns["jud-1"] = []*store.JudgingRun{
		{
			ID:         "run-1",
			JudgingID:  "jud-1",
			TestCaseID: "tc-1",
			Rank:       1,
			Verdict:    "correct",
			Runtime:    0.1,
			Memory:     512,
		},
		{
			ID:         "run-2",
			JudgingID:  "jud-1",
			TestCaseID: "tc-2",
			Rank:       2,
			Verdict:    "correct",
			Runtime:    0.2,
			Memory:     768,
		},
		{
			ID:         "run-3",
			JudgingID:  "jud-1",
			TestCaseID: "tc-3",
			Rank:       3,
			Verdict:    "wrong-answer",
			Runtime:    0.3,
			Memory:     1024,
		},
	}

	resp, err := service.GetJudgingRuns(ctx, &pb.GetJudgingRunsRequest{SubmissionId: "sub-1"})
	require.NoError(t, err)
	assert.Len(t, resp.Runs, 3)
	assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.Runs[0].Verdict)
	assert.Equal(t, pb.Verdict_VERDICT_WRONG_ANSWER, resp.Runs[2].Verdict)
}

func TestSubmissionService_Integration_InternalCreateJudging(t *testing.T) {
	service, mr, rdb := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	resp, err := service.InternalCreateJudging(ctx, &pb.InternalCreateJudgingRequest{
		SubmissionId: "sub-1",
		JudgehostId:  "host-1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.JudgingId)

	// Verify judging was cached in Redis
	judgingKey := "judging:judging-sub-1:meta"
	status, err := rdb.HGet(ctx, judgingKey, "status").Result()
	require.NoError(t, err)
	assert.Equal(t, "judging", status)
}

func TestSubmissionService_Integration_InternalUpdateJudging(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	// Setup judging
	mockStore.Judgings["jud-1"] = &store.Judging{
		ID:           "jud-1",
		SubmissionID: "sub-1",
	}

	resp, err := service.InternalUpdateJudging(ctx, &pb.InternalUpdateJudgingRequest{
		JudgingId:  "jud-1",
		Verdict:    "correct",
		MaxRuntime: 0.5,
		MaxMemory:  1024,
	})
	require.NoError(t, err)
	assert.Equal(t, "jud-1", resp.JudgingId)
	assert.Equal(t, "updated", resp.Status)

	// Verify judging was updated
	updated := mockStore.Judgings["jud-1"]
	assert.Equal(t, "correct", updated.Verdict)
	assert.Equal(t, 0.5, updated.MaxRuntime)
	assert.True(t, updated.CompileSuccess) // correct verdict should set this to true
}

func TestSubmissionService_Integration_InternalUpdateJudging_CompilerError(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	mockStore.Judgings["jud-1"] = &store.Judging{
		ID:           "jud-1",
		SubmissionID: "sub-1",
	}

	_, err := service.InternalUpdateJudging(ctx, &pb.InternalUpdateJudgingRequest{
		JudgingId:    "jud-1",
		Verdict:      "compiler-error",
		MaxRuntime:   0,
		MaxMemory:    0,
		CompileError: "undefined variable x",
	})
	require.NoError(t, err)

	// Verify compile_success is false for compiler error
	updated := mockStore.Judgings["jud-1"]
	assert.False(t, updated.CompileSuccess)
}

func TestSubmissionService_Integration_RejudgeSubmission(t *testing.T) {
	service, mr, rdb := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Setup submission and existing judging
	mockStore := store.NewMockSubmissionStore()
	service.store = mockStore

	mockStore.Submissions["sub-1"] = &store.Submission{
		ID:         "sub-1",
		UserID:     "user-1",
		ProblemID:  "prob-1",
		LanguageID: "cpp",
		SourceCode: "code",
		SubmitTime: time.Now(),
	}
	mockStore.Judgings["jud-1"] = &store.Judging{
		ID:           "jud-1",
		SubmissionID: "sub-1",
		Verdict:      "wrong-answer",
		Valid:        true,
	}

	// Set Redis cache for old judging using Redis client
	rdb.HSet(ctx, "judging:judging-sub-1:meta", map[string]interface{}{
		"verdict": "wrong-answer",
		"status":  "completed",
	})

	// Rejudge
	resp, err := service.RejudgeSubmission(ctx, &pb.RejudgeSubmissionRequest{SubmissionId: "sub-1"})
	require.NoError(t, err)
	assert.Equal(t, "sub-1", resp.SubmissionId)
	assert.Equal(t, "queued", resp.Status)

	// Verify old judging was invalidated
	oldJudging := mockStore.Judgings["jud-1"]
	assert.False(t, oldJudging.Valid)

	// Verify new job was added to queue using Redis client
	queueLen, err := rdb.ZCard(ctx, "judge:queue").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), queueLen)
}

func TestSubmissionService_Integration_GetSourceCode(t *testing.T) {
	service, mr, rdb := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Setup submission
	mockStore := store.NewMockSubmissionStore()
	service.store = mockStore

	mockStore.Submissions["sub-1"] = &store.Submission{
		ID:         "sub-1",
		SourceCode: "#include <iostream>\nint main() { return 0; }",
	}

	// Get from Redis cache first
	rdb.Set(ctx, "submission:sub-1:source", "cached code", 0)

	source, err := service.GetSourceCode(ctx, "sub-1")
	require.NoError(t, err)
	assert.Equal(t, "cached code", source)

	// Get from database when not in cache
	source, err = service.GetSourceCode(ctx, "sub-1")
	require.NoError(t, err)
	// Should have been re-cached
}

func TestSubmissionService_Integration_VerdictMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.Verdict
	}{
		{"correct", pb.Verdict_VERDICT_CORRECT},
		{"wrong-answer", pb.Verdict_VERDICT_WRONG_ANSWER},
		{"timelimit", pb.Verdict_VERDICT_TIMELIMIT},
		{"time-limit", pb.Verdict_VERDICT_TIMELIMIT},
		{"memory-limit", pb.Verdict_VERDICT_MEMORY_LIMIT},
		{"run-error", pb.Verdict_VERDICT_RUN_ERROR},
		{"compiler-error", pb.Verdict_VERDICT_COMPILER_ERROR},
		{"output-limit", pb.Verdict_VERDICT_OUTPUT_LIMIT},
		{"presentation", pb.Verdict_VERDICT_PRESENTATION},
		{"unknown", pb.Verdict_VERDICT_UNSPECIFIED},
		{"", pb.Verdict_VERDICT_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := mapVerdictToProto(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSubmissionService_Integration_ErrorHandling(t *testing.T) {
	mockStore := store.NewMockSubmissionStore()
	service := NewSubmissionService(mockStore, nil)
	ctx := context.Background()

	t.Run("GetNonExistentSubmission", func(t *testing.T) {
		_, err := service.GetSubmission(ctx, &pb.GetSubmissionRequest{Id: "non-existent"})
		require.Error(t, err)
	})

	t.Run("GetJudgingNonExistent", func(t *testing.T) {
		_, err := service.GetJudging(ctx, &pb.GetJudgingRequest{SubmissionId: "non-existent"})
		require.Error(t, err)
	})

	t.Run("GetJudgingRunsNonExistent", func(t *testing.T) {
		_, err := service.GetJudgingRuns(ctx, &pb.GetJudgingRunsRequest{SubmissionId: "non-existent"})
		require.Error(t, err)
	})

	t.Run("UpdateJudgingNonExistent", func(t *testing.T) {
		_, err := service.InternalUpdateJudging(ctx, &pb.InternalUpdateJudgingRequest{
			JudgingId: "non-existent",
		})
		require.Error(t, err)
	})

	t.Run("RejudgeNonExistentSubmission", func(t *testing.T) {
		_, err := service.RejudgeSubmission(ctx, &pb.RejudgeSubmissionRequest{
			SubmissionId: "non-existent",
		})
		require.Error(t, err)
	})
}

// Full integration flow test
func TestSubmissionService_Integration_FullFlow(t *testing.T) {
	service, mr, _ := setupTestWithRedis(t)
	defer mr.Close()
	ctx := context.Background()

	// Step 1: Create submission
	createResp, err := service.CreateSubmission(ctx, &pb.CreateSubmissionRequest{
		ProblemId:  "prob-1",
		LanguageId: "cpp",
		SourceCode: "#include <iostream>\nint main() { std::cout << \"Hello\"; return 0; }",
	})
	require.NoError(t, err)
	subID := createResp.Id

	// Step 2: Create judging record
	judgingResp, err := service.InternalCreateJudging(ctx, &pb.InternalCreateJudgingRequest{
		SubmissionId: subID,
		JudgehostId:  "host-1",
	})
	require.NoError(t, err)
	judgingID := judgingResp.JudgingId

	// Step 3: Create judging runs
	_, err = service.InternalCreateJudgingRun(ctx, &pb.InternalCreateJudgingRunRequest{
		JudgingId:  judgingID,
		TestCaseId: "tc-1",
		Rank:       1,
		Verdict:    "correct",
		Runtime:    0.1,
		WallTime:   0.15,
		Memory:     512,
	})
	require.NoError(t, err)

	_, err = service.InternalCreateJudgingRun(ctx, &pb.InternalCreateJudgingRunRequest{
		JudgingId:  judgingID,
		TestCaseId: "tc-2",
		Rank:       2,
		Verdict:    "correct",
		Runtime:    0.2,
		WallTime:   0.25,
		Memory:     768,
	})
	require.NoError(t, err)

	// Step 4: Update judging with final result
	_, err = service.InternalUpdateJudging(ctx, &pb.InternalUpdateJudgingRequest{
		JudgingId:  judgingID,
		Verdict:    "correct",
		MaxRuntime: 0.2,
		MaxMemory:  768,
	})
	require.NoError(t, err)

	// Step 5: Get submission with judging result
	getResp, err := service.GetSubmission(ctx, &pb.GetSubmissionRequest{Id: subID})
	require.NoError(t, err)
	assert.NotNil(t, getResp.LatestJudging)
	assert.Equal(t, pb.Verdict_VERDICT_CORRECT, getResp.LatestJudging.Verdict)

	// Step 6: Get judging runs
	runsResp, err := service.GetJudgingRuns(ctx, &pb.GetJudgingRunsRequest{SubmissionId: subID})
	require.NoError(t, err)
	assert.Len(t, runsResp.Runs, 2)
}
