package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	pbJudge "github.com/slhmy/online-judge/gen/go/judge/v1"
	pbProblem "github.com/slhmy/online-judge/gen/go/problem/v1"
	pbSubmission "github.com/slhmy/online-judge/gen/go/submission/v1"
)

// Task type constant (matches backend)
const TaskTypeJudge = "task:type_judge"

// JudgeJobPayload is the payload for asynq judge tasks
type JudgeJobPayload struct {
	SubmissionID string `json:"submission_id"`
	ProblemID    string `json:"problem_id"`
	Language     string `json:"language"`
	Priority     int    `json:"priority"`
}

// JudgeJob represents a judging job from the queue
type JudgeJob struct {
	SubmissionID string    `json:"submission_id"`
	ProblemID    string    `json:"problem_id"`
	Language     string    `json:"language"`
	Priority     int       `json:"priority"`
	SubmitTime   time.Time `json:"submit_time"`
	RejudgeID    string    `json:"rejudge_id,omitempty"`
}

// JudgeResult represents the final judging result
type JudgeResult struct {
	SubmissionID string  `json:"submission_id"`
	Verdict      string  `json:"verdict"`
	Runtime      float64 `json:"runtime"` // in seconds
	Memory       int64   `json:"memory"`  // in kilobytes
	Error        string  `json:"error,omitempty"`
	CompileError string  `json:"compile_error,omitempty"`
}

// TestCaseResult represents result for a single test case
type TestCaseResult struct {
	TestCaseID string  `json:"test_case_id"`
	Rank       int     `json:"rank"`
	Verdict    string  `json:"verdict"`
	Runtime    float64 `json:"runtime"`
	Memory     int64   `json:"memory"`
	Output     string  `json:"output,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// SubmissionDetails contains submission information needed for judging
type SubmissionDetails struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	ProblemID  string `json:"problem_id"`
	ContestID  string `json:"contest_id,omitempty"`
	LanguageID string `json:"language_id"`
	SourceCode string `json:"source_code"`
	SubmitTime string `json:"submit_time"`
}

// ProblemDetails contains problem configuration for judging
type ProblemDetails struct {
	ID                 string  `json:"id"`
	TimeLimit          float64 `json:"time_limit"`   // in seconds
	MemoryLimit        int32   `json:"memory_limit"` // in kilobytes
	OutputLimit        int32   `json:"output_limit"` // in kilobytes
	ProcessLimit       int32   `json:"process_limit"`
	SpecialRunID       string  `json:"special_run_id,omitempty"` // For interactive problems
	SpecialCompare     string  `json:"special_compare_id,omitempty"`
	SpecialCompareArgs string  `json:"special_compare_args,omitempty"`
}

// TestCase contains test case data
type TestCase struct {
	ID            string `json:"id"`
	ProblemID     string `json:"problem_id"`
	Rank          int32  `json:"rank"`
	IsSample      bool   `json:"is_sample"`
	InputPath     string `json:"input_path"`
	OutputPath    string `json:"output_path"`
	Description   string `json:"description,omitempty"`
	IsInteractive bool   `json:"is_interactive"`
	InputData     string `json:"input_data,omitempty"`  // Base64 encoded or direct
	OutputData    string `json:"output_data,omitempty"` // Base64 encoded or direct
}

// TestCaseData contains the actual test case input/output
type TestCaseData struct {
	Input  []byte `json:"input"`
	Output []byte `json:"output"`
}

// JudgeQueue manages the judging job queue
type JudgeQueue struct {
	redis            *redis.Client
	submissionClient pbSubmission.SubmissionServiceClient
	problemClient    pbProblem.ProblemServiceClient
	judgeClient      pbJudge.JudgeServiceClient
}

// NewJudgeQueue creates a new judge queue client
func NewJudgeQueue(redis *redis.Client, submissionClient pbSubmission.SubmissionServiceClient, problemClient pbProblem.ProblemServiceClient, judgeClient pbJudge.JudgeServiceClient) *JudgeQueue {
	return &JudgeQueue{
		redis:            redis,
		submissionClient: submissionClient,
		problemClient:    problemClient,
		judgeClient:      judgeClient,
	}
}

// Push adds a job to the priority queue (Redis sorted set)
func (q *JudgeQueue) Push(ctx context.Context, job *JudgeJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Use Redis sorted set for priority queue
	// Lower score = higher priority
	// Priority is inverted so lower values are processed first
	// Submit time breaks ties (earlier = lower score = higher priority)
	score := float64(job.Priority*1000000) - float64(job.SubmitTime.Unix())

	return q.redis.ZAdd(ctx, "judge:queue", redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

// Pop gets the highest priority job (lowest score)
func (q *JudgeQueue) Pop(ctx context.Context) (*JudgeJob, error) {
	// Get lowest score (highest priority)
	result, err := q.redis.ZPopMin(ctx, "judge:queue").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	var job JudgeJob
	memberStr := fmt.Sprint(result[0].Member)
	if err := json.Unmarshal([]byte(memberStr), &job); err != nil {
		return nil, err
	}

	return &job, nil
}

// PushResult publishes judging result via Redis pub/sub
func (q *JudgeQueue) PushResult(ctx context.Context, result *JudgeResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// Publish to result channel
	return q.redis.Publish(ctx, "judge:results", data).Err()
}

// SubscribeResults subscribes to judging result updates
func (q *JudgeQueue) SubscribeResults(ctx context.Context) <-chan *JudgeResult {
	ch := make(chan *JudgeResult, 100)

	pubsub := q.redis.Subscribe(ctx, "judge:results")

	go func() {
		defer close(ch)
		for msg := range pubsub.Channel() {
			var result JudgeResult
			if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
				continue
			}
			ch <- &result
		}
	}()

	return ch
}

// Length returns queue length
func (q *JudgeQueue) Length(ctx context.Context) (int64, error) {
	return q.redis.ZCard(ctx, "judge:queue").Result()
}

// Ping checks Redis connection
func (q *JudgeQueue) Ping(ctx context.Context) error {
	return q.redis.Ping(ctx).Err()
}

// FetchSubmission fetches submission details via gRPC
func (q *JudgeQueue) FetchSubmission(ctx context.Context, submissionID string) (*SubmissionDetails, error) {
	resp, err := q.submissionClient.GetSubmission(ctx, &pbSubmission.GetSubmissionRequest{
		Id: submissionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submission: %w", err)
	}

	sub := resp.GetSubmission()

	// Fetch source code via internal RPC
	sourceResp, err := q.submissionClient.InternalGetSourceCode(ctx, &pbSubmission.InternalGetSourceCodeRequest{
		SubmissionId: submissionID,
	})
	if err != nil {
		// Fallback: try to get from Redis cache
		sourceCode, cacheErr := q.getSourceFromQueue(ctx, submissionID)
		if cacheErr != nil {
			return nil, fmt.Errorf("failed to fetch source code: %w", err)
		}
		return &SubmissionDetails{
			ID:         sub.GetId(),
			UserID:     sub.GetUserId(),
			ProblemID:  sub.GetProblemId(),
			ContestID:  sub.GetContestId(),
			LanguageID: sub.GetLanguageId(),
			SourceCode: sourceCode,
			SubmitTime: sub.GetSubmitTime(),
		}, nil
	}

	return &SubmissionDetails{
		ID:         sub.GetId(),
		UserID:     sub.GetUserId(),
		ProblemID:  sub.GetProblemId(),
		ContestID:  sub.GetContestId(),
		LanguageID: sub.GetLanguageId(),
		SourceCode: sourceResp.GetSourceCode(),
		SubmitTime: sub.GetSubmitTime(),
	}, nil
}

// getSourceFromQueue tries to retrieve source code from Redis cache
func (q *JudgeQueue) getSourceFromQueue(ctx context.Context, submissionID string) (string, error) {
	key := fmt.Sprintf("submission:%s:source", submissionID)
	source, err := q.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("source code not found for submission %s", submissionID)
		}
		return "", err
	}
	return source, nil
}

// FetchProblem fetches problem details via gRPC
func (q *JudgeQueue) FetchProblem(ctx context.Context, problemID string) (*ProblemDetails, error) {
	resp, err := q.problemClient.GetProblem(ctx, &pbProblem.GetProblemRequest{
		Id: problemID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch problem: %w", err)
	}

	problem := resp.GetProblem()
	return &ProblemDetails{
		ID:                 problem.GetId(),
		TimeLimit:          problem.GetTimeLimit(),
		MemoryLimit:        problem.GetMemoryLimit(),
		OutputLimit:        problem.GetOutputLimit(),
		ProcessLimit:       problem.GetProcessLimit(),
		SpecialRunID:       problem.GetSpecialRunId(),
		SpecialCompare:     problem.GetSpecialCompareId(),
		SpecialCompareArgs: problem.GetSpecialCompareArgs(),
	}, nil
}

// FetchTestCases fetches all test cases for a problem via gRPC
func (q *JudgeQueue) FetchTestCases(ctx context.Context, problemID string) ([]*TestCase, error) {
	resp, err := q.problemClient.ListTestCases(ctx, &pbProblem.ListTestCasesRequest{
		ProblemId:   problemID,
		SamplesOnly: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch test cases: %w", err)
	}

	testCases := make([]*TestCase, len(resp.GetTestCases()))
	for i, tc := range resp.GetTestCases() {
		testCases[i] = &TestCase{
			ID:            tc.GetId(),
			ProblemID:     tc.GetProblemId(),
			Rank:          tc.GetRank(),
			IsSample:      tc.GetIsSample(),
			InputPath:     tc.GetInputPath(),
			OutputPath:    tc.GetOutputPath(),
			Description:   tc.GetDescription(),
			IsInteractive: tc.GetIsInteractive(),
			InputData:     tc.GetInputContent(),
			OutputData:    tc.GetOutputContent(),
		}
	}

	return testCases, nil
}

// FetchTestCaseData fetches the actual input/output data for a test case
func (q *JudgeQueue) FetchTestCaseData(ctx context.Context, testCase *TestCase) (*TestCaseData, error) {
	// If data is already included
	if testCase.InputData != "" && testCase.OutputData != "" {
		return &TestCaseData{
			Input:  []byte(testCase.InputData),
			Output: []byte(testCase.OutputData),
		}, nil
	}

	// Fetch from backend via gRPC
	resp, err := q.problemClient.InternalGetTestCaseContent(ctx, &pbProblem.InternalGetTestCaseContentRequest{
		TestCaseId: testCase.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch test case content: %w", err)
	}

	return &TestCaseData{
		Input:  []byte(resp.GetInput()),
		Output: []byte(resp.GetOutput()),
	}, nil
}

// CreateJudging creates a new judging record via gRPC
func (q *JudgeQueue) CreateJudging(ctx context.Context, submissionID string, judgehostID string) (string, error) {
	resp, err := q.submissionClient.InternalCreateJudging(ctx, &pbSubmission.InternalCreateJudgingRequest{
		SubmissionId: submissionID,
		JudgehostId:  judgehostID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create judging: %w", err)
	}

	return resp.GetJudgingId(), nil
}

// UpdateJudging updates the judging record with final result via gRPC
func (q *JudgeQueue) UpdateJudging(ctx context.Context, judgingID string, result *JudgeResult) error {
	_, err := q.submissionClient.InternalUpdateJudging(ctx, &pbSubmission.InternalUpdateJudgingRequest{
		JudgingId:      judgingID,
		Verdict:        result.Verdict,
		MaxRuntime:     result.Runtime,
		MaxMemory:      int32(result.Memory),
		CompileSuccess: result.CompileError == "",
		CompileError:   result.CompileError,
	})
	if err != nil {
		return fmt.Errorf("failed to update judging: %w", err)
	}

	return nil
}

// CreateJudgingRun creates a record for a single test case run via gRPC
func (q *JudgeQueue) CreateJudgingRun(ctx context.Context, judgingID string, testCaseResult *TestCaseResult) error {
	_, err := q.submissionClient.InternalCreateJudgingRun(ctx, &pbSubmission.InternalCreateJudgingRunRequest{
		JudgingId:  judgingID,
		TestCaseId: testCaseResult.TestCaseID,
		Rank:       int32(testCaseResult.Rank),
		Verdict:    testCaseResult.Verdict,
		Runtime:    testCaseResult.Runtime,
		Memory:     int32(testCaseResult.Memory),
	})
	if err != nil {
		return fmt.Errorf("failed to create judging run: %w", err)
	}

	return nil
}

// PushJudgingResult pushes the complete judging result to database and notifies via Redis
func (q *JudgeQueue) PushJudgingResult(ctx context.Context, result *JudgeResult, judgingID string, runResults []*TestCaseResult) error {
	// Update the judging record
	if err := q.UpdateJudging(ctx, judgingID, result); err != nil {
		return fmt.Errorf("failed to update judging: %w", err)
	}

	// Create records for each test case run
	for _, run := range runResults {
		if err := q.CreateJudgingRun(ctx, judgingID, run); err != nil {
			// Log but continue - the main result is more important
			fmt.Printf("Warning: failed to create judging run: %v\n", err)
		}
	}

	// Publish result via Redis for real-time updates
	if err := q.PushResult(ctx, result); err != nil {
		fmt.Printf("Warning: failed to publish result via Redis: %v\n", err)
	}

	return nil
}

// UpdateRejudgeSubmission notifies the backend about a completed rejudge submission via gRPC
func (q *JudgeQueue) UpdateRejudgeSubmission(ctx context.Context, rejudgeID, submissionID, judgingID, verdict string) error {
	_, err := q.judgeClient.InternalUpdateRejudgeSubmission(ctx, &pbJudge.InternalUpdateRejudgeSubmissionRequest{
		RejudgeId:    rejudgeID,
		SubmissionId: submissionID,
		NewJudgingId: judgingID,
		NewVerdict:   verdict,
	})
	if err != nil {
		return fmt.Errorf("failed to update rejudge submission: %w", err)
	}

	return nil
}

// FetchExecutable fetches an executable binary (validator, run script) via gRPC
func (q *JudgeQueue) FetchExecutable(ctx context.Context, executableID string) ([]byte, string, error) {
	resp, err := q.problemClient.InternalGetExecutable(ctx, &pbProblem.InternalGetExecutableRequest{
		Id: executableID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch executable: %w", err)
	}

	return resp.GetBinaryData(), resp.GetMd5Sum(), nil
}
