package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/online-judge/judge/internal/config"
	"github.com/online-judge/judge/internal/queue"
	"github.com/online-judge/judge/internal/sandbox"
	"github.com/online-judge/judge/internal/validator"
)

// Verdict constants
const (
	VerdictCorrect       = "correct"
	VerdictWrongAnswer   = "wrong-answer"
	VerdictTimeLimit     = "time-limit"
	VerdictMemoryLimit   = "memory-limit"
	VerdictRunError      = "run-error"
	VerdictCompilerError = "compiler-error"
	VerdictOutputLimit   = "output-limit"
	VerdictPresentation  = "presentation"
)

// JudgeWorker handles judging jobs from the queue
type JudgeWorker struct {
	id              string
	queue           *queue.JudgeQueue
	config          *config.Config
	validator       *validator.DefaultValidator
	specialValidator *validator.SpecialValidator
	mu              sync.Mutex
	currentJob      *queue.JudgeJob
}

// NewJudgeWorker creates a new judge worker
func NewJudgeWorker(id string, cfg *config.Config, judgeQueue *queue.JudgeQueue) *JudgeWorker {
	// Set sandbox work directory if configured
	if cfg.SandboxWorkDir != "" {
		sandbox.SetSandboxWorkDir(cfg.SandboxWorkDir)
	}

	// Create special validator config
	specialValidatorConfig := validator.DefaultSpecialValidatorConfig()
	if cfg.SandboxWorkDir != "" {
		specialValidatorConfig.CacheDir = cfg.SandboxWorkDir + "/validator-cache"
	}

	return &JudgeWorker{
		id:        id,
		queue:     judgeQueue,
		config:    cfg,
		validator: validator.NewDefaultValidator(),
		specialValidator: validator.NewSpecialValidator(
			specialValidatorConfig,
			judgeQueue,
			cfg.OrchestratorURL,
		),
	}
}

// Run starts the worker loop
func (w *JudgeWorker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			job, err := w.queue.Pop(ctx)
			if err != nil {
				log.Printf("Error popping from queue: %v", err)
				time.Sleep(time.Second)
				continue
			}

			if job == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			w.mu.Lock()
			w.currentJob = job
			w.mu.Unlock()

			log.Printf("Processing submission: %s (language: %s)", job.SubmissionID, job.Language)

			result := w.processJob(ctx, job)

			log.Printf("Submission %s completed: verdict=%s, runtime=%.3f, memory=%d",
				job.SubmissionID, result.Verdict, result.Runtime, result.Memory)

			w.mu.Lock()
			w.currentJob = nil
			w.mu.Unlock()
		}
	}
}

// processJob handles the complete judging pipeline
func (w *JudgeWorker) processJob(ctx context.Context, job *queue.JudgeJob) *queue.JudgeResult {
	// 1. Fetch submission details
	submission, err := w.queue.FetchSubmission(ctx, job.SubmissionID)
	if err != nil {
		log.Printf("Failed to fetch submission %s: %v", job.SubmissionID, err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to fetch submission: %v", err),
		}
	}

	// 2. Fetch problem details
	problem, err := w.queue.FetchProblem(ctx, submission.ProblemID)
	if err != nil {
		log.Printf("Failed to fetch problem %s: %v", submission.ProblemID, err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to fetch problem: %v", err),
		}
	}

	// 3. Fetch test cases
	testCases, err := w.queue.FetchTestCases(ctx, submission.ProblemID)
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
	sb, err := sandbox.NewDockerSandbox()
	if err != nil {
		log.Printf("Failed to create sandbox: %v", err)
		return &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictRunError,
			Error:        fmt.Sprintf("Failed to create sandbox: %v", err),
		}
	}
	defer sb.Cleanup()

	// 5. Create judging record (before compilation so CE is recorded)
	judgingID, err := w.queue.CreateJudging(ctx, job.SubmissionID, w.id)
	if err != nil {
		log.Printf("Failed to create judging record: %v", err)
		// Continue anyway - we can still process
	}

	// 6. Compile source code
	binaryPath, compileErr := w.compile(ctx, sb, submission.SourceCode, submission.LanguageID)
	if compileErr != nil {
		log.Printf("Compilation failed for submission %s: %v", job.SubmissionID, compileErr)
		result := &queue.JudgeResult{
			SubmissionID: job.SubmissionID,
			Verdict:      VerdictCompilerError,
			CompileError: compileErr.Error(),
		}
		// Push result to database even for CE
		if judgingID != "" {
			w.queue.PushJudgingResult(ctx, result, judgingID, nil)
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
		limits.TimeLimit = time.Duration(w.config.DefaultTimeLimit) * time.Second
	}
	if limits.MemoryLimit == 0 {
		limits.MemoryLimit = w.config.DefaultMemoryLimit
	}
	if limits.ProcessLimit == 0 {
		limits.ProcessLimit = w.config.MaxProcesses
	}
	if limits.OutputLimit == 0 {
		limits.OutputLimit = 10240 // 10MB default
	}

	for i, tc := range testCases {
		log.Printf("Running test case %d/%d for submission %s", i+1, len(testCases), job.SubmissionID)

		// Fetch test case data
		tcData, err := w.queue.FetchTestCaseData(ctx, tc)
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

		// Run the binary with test case input
		runResult, err := w.runTest(ctx, sb, binaryPath, submission.LanguageID, tcData.Input, limits)
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
		tcVerdict := runResult.Verdict
		if tcVerdict == VerdictCorrect {
			// Validate output - use special validator if problem has special_compare_id
			if problem.SpecialCompare != "" && w.specialValidator != nil {
				// Run custom validator
				vVerdict, feedback := w.specialValidator.Validate(
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
				tcVerdict = string(w.validator.Validate(tcData.Output, runResult.Output))
			}
		}

		// Track statistics
		runtimeSeconds := runResult.TimeUsed.Seconds()
		memoryKB := runResult.MemoryUsed

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
			Output:     string(runResult.Output),
			Error:      string(runResult.Error),
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
		if err := w.queue.PushJudgingResult(ctx, result, judgingID, runResults); err != nil {
			log.Printf("Failed to push judging result: %v", err)
		}
	} else {
		// Publish result via Redis only
		if err := w.queue.PushResult(ctx, result); err != nil {
			log.Printf("Failed to publish result: %v", err)
		}
	}

	return result
}

// compile compiles source code in the sandbox
func (w *JudgeWorker) compile(ctx context.Context, sb *sandbox.DockerSandbox, source string, language string) (string, error) {
	binaryPath, err := sb.Compile(ctx, source, language)
	if err != nil {
		return "", err
	}
	return binaryPath, nil
}

// runTest runs a test case in the sandbox
func (w *JudgeWorker) runTest(ctx context.Context, sb *sandbox.DockerSandbox, binaryPath string, language string, input []byte, limits sandbox.Limits) (*sandbox.Result, error) {
	result, err := sb.Run(ctx, binaryPath, nil, bytes.NewReader(input), limits)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetCurrentJob returns the current job being processed
func (w *JudgeWorker) GetCurrentJob() *queue.JudgeJob {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentJob
}

// GetStatus returns current worker status
func (w *JudgeWorker) GetStatus() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentJob != nil {
		return "judging"
	}
	return "idle"
}