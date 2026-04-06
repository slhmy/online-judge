package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

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
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	ProblemID   string `json:"problem_id"`
	ContestID   string `json:"contest_id,omitempty"`
	LanguageID  string `json:"language_id"`
	SourceCode  string `json:"source_code"`
	SubmitTime  string `json:"submit_time"`
}

// ProblemDetails contains problem configuration for judging
type ProblemDetails struct {
	ID              string  `json:"id"`
	TimeLimit       float64 `json:"time_limit"`  // in seconds
	MemoryLimit     int32   `json:"memory_limit"` // in kilobytes
	OutputLimit     int32   `json:"output_limit"` // in kilobytes
	ProcessLimit    int32   `json:"process_limit"`
	SpecialCompare  string  `json:"special_compare_id,omitempty"`
	SpecialCompareArgs string `json:"special_compare_args,omitempty"`
}

// TestCase contains test case data
type TestCase struct {
	ID           string `json:"id"`
	ProblemID    string `json:"problem_id"`
	Rank         int32  `json:"rank"`
	IsSample     bool   `json:"is_sample"`
	InputPath    string `json:"input_path"`
	OutputPath   string `json:"output_path"`
	Description  string `json:"description,omitempty"`
	IsInteractive bool  `json:"is_interactive"`
	InputData    string `json:"input_data,omitempty"`  // Base64 encoded or direct
	OutputData   string `json:"output_data,omitempty"` // Base64 encoded or direct
}

// TestCaseData contains the actual test case input/output
type TestCaseData struct {
	Input  []byte `json:"input"`
	Output []byte `json:"output"`
}

// JudgeQueue manages the judging job queue
type JudgeQueue struct {
	redis          *redis.Client
	httpClient     *http.Client
	orchestratorURL string
}

// NewJudgeQueue creates a new judge queue client
func NewJudgeQueue(redis *redis.Client, orchestratorURL string) *JudgeQueue {
	return &JudgeQueue{
		redis: redis,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		orchestratorURL: orchestratorURL,
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
	memberStr := result[0].Member
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

// FetchSubmission fetches submission details from the BFF API
func (q *JudgeQueue) FetchSubmission(ctx context.Context, submissionID string) (*SubmissionDetails, error) {
	url := fmt.Sprintf("%s/api/v1/submissions/%s", q.orchestratorURL, submissionID)

	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submission: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch submission: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse the response - it's wrapped in a proto-like structure
	var rawResp struct {
		Submission struct {
			ID         string `json:"id"`
			UserId     string `json:"user_id"`
			ProblemId  string `json:"problem_id"`
			ContestId  string `json:"contest_id"`
			LanguageId string `json:"language_id"`
			SourcePath string `json:"source_path"`
			SubmitTime string `json:"submit_time"`
		} `json:"submission"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode submission response: %w", err)
	}

	// Fetch source code from internal API
	sourceCode, err := q.fetchSourceCode(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source code: %w", err)
	}

	return &SubmissionDetails{
		ID:         rawResp.Submission.ID,
		UserID:     rawResp.Submission.UserId,
		ProblemID:  rawResp.Submission.ProblemId,
		ContestID:  rawResp.Submission.ContestId,
		LanguageID: rawResp.Submission.LanguageId,
		SourceCode: sourceCode,
		SubmitTime: rawResp.Submission.SubmitTime,
	}, nil
}

// fetchSourceCode fetches the source code for a submission
func (q *JudgeQueue) fetchSourceCode(ctx context.Context, submissionID string) (string, error) {
	// Try to get from internal endpoint first
	url := fmt.Sprintf("%s/internal/submissions/%s/source", q.orchestratorURL, submissionID)

	resp, err := q.httpClient.Get(url)
	if err != nil {
		// Fallback: try to get from cache or queue data
		return q.getSourceFromQueue(ctx, submissionID)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return q.getSourceFromQueue(ctx, submissionID)
	}

	var sourceResp struct {
		SourceCode string `json:"source_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sourceResp); err != nil {
		return "", fmt.Errorf("failed to decode source response: %w", err)
	}

	return sourceResp.SourceCode, nil
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

// FetchProblem fetches problem details from the BFF API
func (q *JudgeQueue) FetchProblem(ctx context.Context, problemID string) (*ProblemDetails, error) {
	url := fmt.Sprintf("%s/api/v1/problems/%s", q.orchestratorURL, problemID)

	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch problem: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch problem: status %d, body: %s", resp.StatusCode, string(body))
	}

	var rawResp struct {
		Problem struct {
			Id              string  `json:"id"`
			TimeLimit       float64 `json:"time_limit"`
			MemoryLimit     int32   `json:"memory_limit"`
			OutputLimit     int32   `json:"output_limit"`
			ProcessLimit    int32   `json:"process_limit"`
			SpecialCompareId string `json:"special_compare_id"`
			SpecialCompareArgs string `json:"special_compare_args"`
		} `json:"problem"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode problem response: %w", err)
	}

	return &ProblemDetails{
		ID:            rawResp.Problem.Id,
		TimeLimit:     rawResp.Problem.TimeLimit,
		MemoryLimit:   rawResp.Problem.MemoryLimit,
		OutputLimit:   rawResp.Problem.OutputLimit,
		ProcessLimit:  rawResp.Problem.ProcessLimit,
		SpecialCompare: rawResp.Problem.SpecialCompareId,
		SpecialCompareArgs: rawResp.Problem.SpecialCompareArgs,
	}, nil
}

// FetchTestCases fetches all test cases for a problem from the BFF API
func (q *JudgeQueue) FetchTestCases(ctx context.Context, problemID string) ([]*TestCase, error) {
	url := fmt.Sprintf("%s/internal/problems/%s/testcases", q.orchestratorURL, problemID)

	resp, err := q.httpClient.Get(url)
	if err != nil {
		// Fallback to public API (sample test cases only)
		return q.fetchSampleTestCases(ctx, problemID)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to public API
		return q.fetchSampleTestCases(ctx, problemID)
	}

	var rawResp struct {
		TestCases []struct {
			Id            string `json:"id"`
			ProblemId     string `json:"problem_id"`
			Rank          int32  `json:"rank"`
			IsSample      bool   `json:"is_sample"`
			InputPath     string `json:"input_path"`
			OutputPath    string `json:"output_path"`
			Description   string `json:"description"`
			IsInteractive bool   `json:"is_interactive"`
			InputData     string `json:"input_data"`
			OutputData    string `json:"output_data"`
		} `json:"test_cases"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode testcases response: %w", err)
	}

	testCases := make([]*TestCase, len(rawResp.TestCases))
	for i, tc := range rawResp.TestCases {
		testCases[i] = &TestCase{
			ID:            tc.Id,
			ProblemID:     tc.ProblemId,
			Rank:          tc.Rank,
			IsSample:      tc.IsSample,
			InputPath:     tc.InputPath,
			OutputPath:    tc.OutputPath,
			Description:   tc.Description,
			IsInteractive: tc.IsInteractive,
			InputData:     tc.InputData,
			OutputData:    tc.OutputData,
		}
	}

	return testCases, nil
}

// fetchSampleTestCases fetches only sample test cases from public API
func (q *JudgeQueue) fetchSampleTestCases(ctx context.Context, problemID string) ([]*TestCase, error) {
	url := fmt.Sprintf("%s/api/v1/problems/%s", q.orchestratorURL, problemID)

	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch problem with samples: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch problem: status %d", resp.StatusCode)
	}

	var rawResp struct {
		SampleTestCases []struct {
			Id            string `json:"id"`
			ProblemId     string `json:"problem_id"`
			Rank          int32  `json:"rank"`
			IsSample      bool   `json:"is_sample"`
			InputPath     string `json:"input_path"`
			OutputPath    string `json:"output_path"`
			Description   string `json:"description"`
			IsInteractive bool   `json:"is_interactive"`
		} `json:"sample_test_cases"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode sample testcases: %w", err)
	}

	testCases := make([]*TestCase, len(rawResp.SampleTestCases))
	for i, tc := range rawResp.SampleTestCases {
		testCases[i] = &TestCase{
			ID:            tc.Id,
			ProblemID:     tc.ProblemId,
			Rank:          tc.Rank,
			IsSample:      tc.IsSample,
			InputPath:     tc.InputPath,
			OutputPath:    tc.OutputPath,
			Description:   tc.Description,
			IsInteractive: tc.IsInteractive,
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

	// Fetch from storage paths via internal API
	inputURL := fmt.Sprintf("%s/internal/testcases/%s/input", q.orchestratorURL, testCase.ID)
	outputURL := fmt.Sprintf("%s/internal/testcases/%s/output", q.orchestratorURL, testCase.ID)

	inputResp, err := q.httpClient.Get(inputURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch test case input: %w", err)
	}
	defer inputResp.Body.Close()

	if inputResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch input: status %d", inputResp.StatusCode)
	}

	inputData, err := io.ReadAll(inputResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read input data: %w", err)
	}

	outputResp, err := q.httpClient.Get(outputURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch test case output: %w", err)
	}
	defer outputResp.Body.Close()

	if outputResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch output: status %d", outputResp.StatusCode)
	}

	outputData, err := io.ReadAll(outputResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read output data: %w", err)
	}

	return &TestCaseData{
		Input:  inputData,
		Output: outputData,
	}, nil
}

// CreateJudging creates a new judging record
func (q *JudgeQueue) CreateJudging(ctx context.Context, submissionID string, judgehostID string) (string, error) {
	url := fmt.Sprintf("%s/internal/judgings", q.orchestratorURL)

	body := map[string]string{
		"submission_id": submissionID,
		"judgehost_id":  judgehostID,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	resp, err := q.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create judging: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create judging: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		JudgingId string `json:"judging_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode judging response: %w", err)
	}

	return result.JudgingId, nil
}

// UpdateJudging updates the judging record with final result
func (q *JudgeQueue) UpdateJudging(ctx context.Context, judgingID string, result *JudgeResult) error {
	url := fmt.Sprintf("%s/internal/judgings/%s", q.orchestratorURL, judgingID)

	body := map[string]interface{}{
		"verdict":     result.Verdict,
		"max_runtime": result.Runtime,
		"max_memory":  result.Memory,
		"error":       result.Error,
		"compile_error": result.CompileError,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update judging: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update judging: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CreateJudgingRun creates a record for a single test case run
func (q *JudgeQueue) CreateJudgingRun(ctx context.Context, judgingID string, testCaseResult *TestCaseResult) error {
	url := fmt.Sprintf("%s/internal/judgings/%s/runs", q.orchestratorURL, judgingID)

	body := map[string]interface{}{
		"test_case_id": testCaseResult.TestCaseID,
		"rank":         testCaseResult.Rank,
		"verdict":      testCaseResult.Verdict,
		"runtime":      testCaseResult.Runtime,
		"memory":       testCaseResult.Memory,
		"output":       testCaseResult.Output,
		"error":        testCaseResult.Error,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := q.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create judging run: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create judging run: status %d, body: %s", resp.StatusCode, string(respBody))
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

// UpdateRejudgeSubmission notifies the backend about a completed rejudge submission
func (q *JudgeQueue) UpdateRejudgeSubmission(ctx context.Context, rejudgeID, submissionID, judgingID, verdict string) error {
	url := fmt.Sprintf("%s/internal/rejudges/%s/submissions", q.orchestratorURL, rejudgeID)

	body := map[string]string{
		"submission_id":  submissionID,
		"new_judging_id": judgingID,
		"new_verdict":    verdict,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := q.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to update rejudge submission: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update rejudge submission: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// FetchExecutable fetches an executable binary (validator, run script) from the backend
func (q *JudgeQueue) FetchExecutable(ctx context.Context, executableID string) ([]byte, string, error) {
	url := fmt.Sprintf("%s/internal/executables/%s", q.orchestratorURL, executableID)

	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch executable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("failed to fetch executable: status %d, body: %s", resp.StatusCode, string(body))
	}

	var rawResp struct {
		Executable struct {
			Id            string `json:"id"`
			Type          string `json:"type"`
			ExecutablePath string `json:"executable_path"`
			Md5sum        string `json:"md5sum"`
			BinaryData    string `json:"binary_data"` // Base64 encoded binary
		} `json:"executable"`
	}

	// Try to parse as JSON response
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		// If JSON parsing fails, try reading as raw binary
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read executable data: %w", err)
		}
		return data, "", nil
	}

	// Decode base64 binary if present
	if rawResp.Executable.BinaryData != "" {
		// Simple base64 decoding
		data := []byte(rawResp.Executable.BinaryData) // In real implementation, decode base64
		return data, rawResp.Executable.Md5sum, nil
	}

	// If no binary data in response, we need to fetch from the path
	// This would require a separate endpoint or direct storage access
	return nil, rawResp.Executable.Md5sum, fmt.Errorf("executable binary not in response, need to fetch from path: %s", rawResp.Executable.ExecutablePath)
}

// GetExecutable implements the HTTPClient interface for validator fetching
func (q *JudgeQueue) Get(ctx context.Context, url string) ([]byte, error) {
	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}