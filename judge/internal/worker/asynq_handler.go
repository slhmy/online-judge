package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/slhmy/online-judge/judge/internal/config"
	"github.com/slhmy/online-judge/judge/internal/queue"
	"github.com/slhmy/online-judge/judge/internal/runner"
	"github.com/slhmy/online-judge/judge/internal/sandbox"
	"github.com/slhmy/online-judge/judge/internal/validator"
)

// AsynqHandler handles judge tasks from asynq queue
type AsynqHandler struct {
	config            *config.Config
	queue             *queue.JudgeQueue
	validator         *validator.DefaultValidator
	specialValidator  *validator.SpecialValidator
	interactiveRunner *runner.InteractiveRunner
	compileCache      *sandbox.CompileCache
	redisClient       *redis.Client

	// Track current job for heartbeat
	mu         sync.Mutex
	currentJob *queue.JudgeJob
}

// NewAsynqHandler creates a new asynq handler
func NewAsynqHandler(cfg *config.Config, redisClient *redis.Client) *AsynqHandler {
	// Set sandbox work directory if configured
	if cfg.SandboxWorkDir != "" {
		sandbox.SetSandboxWorkDir(cfg.SandboxWorkDir)
	}

	// Create judge queue client for fetching submission/problem data
	judgeQueue := queue.NewJudgeQueue(redisClient, cfg.OrchestratorURL)

	// Create special validator config
	specialValidatorConfig := validator.DefaultSpecialValidatorConfig()
	if cfg.SandboxWorkDir != "" {
		specialValidatorConfig.CacheDir = cfg.SandboxWorkDir + "/validator-cache"
	}

	// Create interactive runner config
	interactiveRunnerConfig := runner.DefaultInteractiveRunnerConfig()
	if cfg.SandboxWorkDir != "" {
		interactiveRunnerConfig.CacheDir = cfg.SandboxWorkDir + "/interactor-cache"
	}

	// Create compilation cache
	cacheTTL := time.Duration(cfg.CompileCacheTTL) * time.Hour
	compileCache := sandbox.NewCompileCache(redisClient, cacheTTL, cfg.CompileCacheEnabled)

	return &AsynqHandler{
		config:    cfg,
		queue:     judgeQueue,
		validator: validator.NewDefaultValidator(),
		specialValidator: validator.NewSpecialValidator(
			specialValidatorConfig,
			judgeQueue,
			cfg.OrchestratorURL,
		),
		interactiveRunner: runner.NewInteractiveRunner(interactiveRunnerConfig),
		compileCache:      compileCache,
		redisClient:       redisClient,
	}
}

// ProcessTask implements asynq.Handler interface
func (h *AsynqHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	// Parse task payload
	var payload queue.JudgeJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Printf("Failed to parse task payload: %v", err)
		return fmt.Errorf("failed to parse task payload: %w", err)
	}

	// Convert payload to JudgeJob
	job := &queue.JudgeJob{
		SubmissionID: payload.SubmissionID,
		ProblemID:    payload.ProblemID,
		Language:     payload.Language,
		Priority:     payload.Priority,
		SubmitTime:   time.Now(),
	}

	// Track current job
	h.mu.Lock()
	h.currentJob = job
	h.mu.Unlock()

	log.Printf("Processing submission: %s (language: %s)", job.SubmissionID, job.Language)

	// Process the job using existing logic
	result := h.processJob(ctx, job)

	log.Printf("Submission %s completed: verdict=%s, runtime=%.3f, memory=%d",
		job.SubmissionID, result.Verdict, result.Runtime, result.Memory)

	// Clear current job
	h.mu.Lock()
	h.currentJob = nil
	h.mu.Unlock()

	// Return error if judging failed (for retry logic)
	if result.Verdict == VerdictRunError && result.Error != "" {
		return fmt.Errorf("judging failed: %s", result.Error)
	}

	return nil
}

// processJob handles the complete judging pipeline (copied from worker.go)
func (h *AsynqHandler) processJob(ctx context.Context, job *queue.JudgeJob) *queue.JudgeResult {
	// 1. Fetch submission details
	submission, err := h.queue.FetchSubmission(ctx, job.SubmissionID)
	if err != nil {
		log.Printf("Failed to fetch submission %s: %v", job.SubmissionID, err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to fetch submission: %v", err),
		}
	}

	// 2. Fetch problem details
	problem, err := h.queue.FetchProblem(ctx, submission.ProblemID)
	if err != nil {
		log.Printf("Failed to fetch problem %s: %v", submission.ProblemID, err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to fetch problem: %v", err),
		}
	}

	// 3. Fetch test cases
	testCases, err := h.queue.FetchTestCases(ctx, submission.ProblemID)
	if err != nil {
		log.Printf("Failed to fetch test cases for problem %s: %v", submission.ProblemID, err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to fetch test cases: %v", err),
		}
	}

	if len(testCases) == 0 {
		log.Printf("No test cases found for problem %s", submission.ProblemID)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        "No test cases found",
		}
	}

	// 4. Create sandbox environment
	sb, err := sandbox.NewDockerSandboxWithCache(h.compileCache)
	if err != nil {
		log.Printf("Failed to create sandbox: %v", err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to create sandbox: %v", err),
		}
	}
	defer func() { _ = sb.Cleanup() }()

	// 5. Create judging record (before compilation so CE is recorded)
	judgingID, err := h.queue.CreateJudging(ctx, job.SubmissionID, h.config.JudgehostID)
	if err != nil {
		log.Printf("Failed to create judging record: %v", err)
		// Continue anyway - we can still process
	}

	// 6. Compile source code
	binaryPath, compileErr := h.compile(ctx, sb, submission.SourceCode, submission.LanguageID)
	if compileErr != nil {
		log.Printf("Compilation failed for submission %s: %v", job.SubmissionID, compileErr)
		result := &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictCompilerError,
			CompileError: compileErr.Error(),
		}
		// Push result to database even for CE
		if judgingID != "" {
			_ = h.queue.PushJudgingResult(ctx, result, judgingID, nil)
		}
		return result
	}

	// 7. Run test cases (lazy judging - stop on first non-correct)
	runResults := make([]*queue.TestCaseResult, 0, len(testCases))
	finalVerdict := VerdictCorrect
	maxRuntime := 0.0
	maxMemory := int64(0)

	limits := sandbox.Limits{
		TimeLimit:    time.Duration(problem.TimeLimit) * time.Second,
		MemoryLimit:  int64(problem.MemoryLimit),
		OutputLimit:  int64(problem.OutputLimit),
		ProcessLimit: int(problem.ProcessLimit),
	}

	// Use default limits if not specified
	if limits.TimeLimit == 0 {
		limits.TimeLimit = time.Duration(h.config.DefaultTimeLimit) * time.Second
	}
	if limits.MemoryLimit == 0 {
		limits.MemoryLimit = h.config.DefaultMemoryLimit
	}
	if limits.ProcessLimit == 0 {
		limits.ProcessLimit = h.config.MaxProcesses
	}
	if limits.OutputLimit == 0 {
		limits.OutputLimit = 10240 // 10MB default
	}

	// Check if problem is interactive
	isInteractiveProblem := problem.SpecialRunID != ""

	for i, tc := range testCases {
		log.Printf("Running test case %d/%d for submission %s", i+1, len(testCases), job.SubmissionID)

		// Fetch test case data
		tcData, err := h.queue.FetchTestCaseData(ctx, tc)
		if err != nil {
			log.Printf("Failed to fetch test case %s data: %v", tc.ID, err)
			runResults = append(runResults, &queue.TestCaseResult{
				TestCaseID: tc.ID,
				Rank:       int(tc.Rank),
				Verdict:    VerdictRunError,
				Error:      fmt.Sprintf("Failed to fetch test case data: %v", err),
			})
			finalVerdict = VerdictRunError
			break // Can't continue without test case data
		}

		// Determine if this test case should run interactively
		isInteractive := isInteractiveProblem || tc.IsInteractive
		var tcVerdict string
		var runtimeSeconds float64
		var memoryKB int64
		var tcOutput string

		if isInteractive {
			// Run interactive test
			tcVerdict, runtimeSeconds, memoryKB, err = h.runInteractiveTest(
				ctx, sb, binaryPath, problem.SpecialRunID, tcData.Input, limits,
			)
			if err != nil {
				log.Printf("Failed to run interactive test case %s: %v", tc.ID, err)
				runResults = append(runResults, &queue.TestCaseResult{
					TestCaseID: tc.ID,
					Rank:       int(tc.Rank),
					Verdict:    VerdictRunError,
					Error:      fmt.Sprintf("Interactive execution failed: %v", err),
				})
				finalVerdict = VerdictRunError
				break
			}
			log.Printf("Interactive test case %s verdict: %s", tc.ID, tcVerdict)
		} else {
			// Run standard test
			runResult, err := h.runTest(ctx, sb, binaryPath, submission.LanguageID, tcData.Input, limits)
			if err != nil {
				log.Printf("Failed to run test case %s: %v", tc.ID, err)
				runResults = append(runResults, &queue.TestCaseResult{
					TestCaseID: tc.ID,
					Rank:       int(tc.Rank),
					Verdict:    VerdictRunError,
					Error:      fmt.Sprintf("Execution failed: %v", err),
				})
				finalVerdict = VerdictRunError
				break
			}

			// Determine verdict for this test case
			tcVerdict = runResult.Verdict
			if tcVerdict == VerdictCorrect {
				// Validate output - use special validator if problem has special_compare_id
				if problem.SpecialCompare != "" && h.specialValidator != nil {
					// Run custom validator
					vVerdict, feedback := h.specialValidator.Validate(
						ctx,
						problem.SpecialCompare,
						problem.SpecialCompareArgs,
						tcData.Input,
						tcData.Output,
						runResult.Output,
					)
					tcVerdict = string(vVerdict)
					log.Printf("Special validator %s returned: %s (feedback: %s)", problem.SpecialCompare, tcVerdict, feedback)
				} else {
					// Use default validator
					tcVerdict = string(h.validator.Validate(tcData.Output, runResult.Output))
				}
			}

			// Track statistics
			runtimeSeconds = runResult.TimeUsed.Seconds()
			memoryKB = runResult.MemoryUsed
			tcOutput = string(runResult.Output)
		}

		if runtimeSeconds > maxRuntime {
			maxRuntime = runtimeSeconds
		}
		if memoryKB > maxMemory {
			maxMemory = memoryKB
		}

		runResults = append(runResults, &queue.TestCaseResult{
			TestCaseID: tc.ID,
			Rank:       int(tc.Rank),
			Verdict:    tcVerdict,
			Runtime:    runtimeSeconds,
			Memory:     memoryKB,
			Output:     tcOutput,
		})

		// Lazy judging: stop on first non-correct verdict
		if tcVerdict != VerdictCorrect && tcVerdict != VerdictPresentation {
			finalVerdict = tcVerdict
			log.Printf("Stopping judging at test case %d: verdict=%s", i+1, tcVerdict)
			break
		}
	}

	// 8. Push result to database
	result := &queue.JudgeResult{
		SubmissionID: job.SubmissionID,
		Verdict:      finalVerdict,
		Runtime:      maxRuntime,
		Memory:       maxMemory,
	}

	if judgingID != "" {
		if err := h.queue.PushJudgingResult(ctx, result, judgingID, runResults); err != nil {
			log.Printf("Failed to push judging result: %v", err)
		}

		// If this is a rejudge job, update the rejudging submission
		if job.RejudgeID != "" {
			if err := h.queue.UpdateRejudgeSubmission(ctx, job.RejudgeID, job.SubmissionID, judgingID, finalVerdict); err != nil {
				log.Printf("Failed to update rejudge submission: %v", err)
			}
		}
	} else {
		// Publish result via Redis only
		if err := h.queue.PushResult(ctx, result); err != nil {
			log.Printf("Failed to publish result: %v", err)
		}
	}

	return result
}

// compile compiles source code in the sandbox
func (h *AsynqHandler) compile(ctx context.Context, sb *sandbox.DockerSandbox, source string, language string) (string, error) {
	binaryPath, err := sb.Compile(ctx, source, language)
	if err != nil {
		return "", err
	}
	return binaryPath, nil
}

// runTest runs a test case in the sandbox
func (h *AsynqHandler) runTest(ctx context.Context, sb *sandbox.DockerSandbox, binaryPath string, language string, input []byte, limits sandbox.Limits) (*sandbox.Result, error) {
	result, err := sb.Run(ctx, binaryPath, nil, bytes.NewReader(input), limits)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// runInteractiveTest runs an interactive test case with an interactor
func (h *AsynqHandler) runInteractiveTest(
	ctx context.Context,
	sb *sandbox.DockerSandbox,
	binaryPath string,
	interactorID string,
	testcaseInput []byte,
	limits sandbox.Limits,
) (string, float64, int64, error) {
	// Fetch interactor binary
	interactorBinary, err := h.getInteractorBinary(ctx, interactorID)
	if err != nil {
		return VerdictRunError, 0, 0, fmt.Errorf("failed to get interactor: %w", err)
	}

	// Run interactive execution
	result, err := sb.RunInteractive(ctx, binaryPath, interactorBinary, testcaseInput, limits)
	if err != nil {
		return VerdictRunError, 0, 0, fmt.Errorf("interactive execution failed: %w", err)
	}

	return result.SolutionVerdict, result.TimeUsed.Seconds(), result.MemoryUsed, nil
}

// getInteractorBinary fetches the interactor binary for an interactive problem
func (h *AsynqHandler) getInteractorBinary(ctx context.Context, interactorID string) ([]byte, error) {
	if interactorID == "" {
		return nil, fmt.Errorf("interactor ID is empty")
	}

	// Use the queue's FetchExecutable method to get the binary
	binaryData, _, err := h.queue.FetchExecutable(ctx, interactorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interactor binary: %w", err)
	}

	return binaryData, nil
}

// GetCurrentJob returns the current job being processed
func (h *AsynqHandler) GetCurrentJob() *queue.JudgeJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.currentJob
}

// GetStatus returns current handler status
func (h *AsynqHandler) GetStatus() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.currentJob != nil {
		return "busy"
	}
	return "idle"
}
