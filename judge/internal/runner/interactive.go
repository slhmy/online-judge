package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/online-judge/judge/internal/validator"
)

// DOMjudge-style exit codes (same as validator)
const (
	ExitCodeCorrect     = 42
	ExitCodeWrongAnswer = 43
	ExitCodePresentation = 44
)

// InteractiveRunnerConfig holds configuration for interactive runner
type InteractiveRunnerConfig struct {
	// Time limit for interactor execution (default: 60 seconds)
	InteractorTimeLimit time.Duration
	// Memory limit for interactor in KB (default: 512 MB)
	InteractorMemoryLimit int64
	// Cache directory for interactor binaries
	CacheDir string
	// Whether to enable caching
	EnableCache bool
}

// DefaultInteractiveRunnerConfig returns default configuration
func DefaultInteractiveRunnerConfig() InteractiveRunnerConfig {
	return InteractiveRunnerConfig{
		InteractorTimeLimit:    60 * time.Second,
		InteractorMemoryLimit:  524288, // 512 MB
		CacheDir:               "/tmp/interactor-cache",
		EnableCache:            true,
	}
}

// InteractiveRunner manages interactive problem execution
type InteractiveRunner struct {
	config InteractiveRunnerConfig
	cache  *validator.ValidatorCache // Reuse validator cache pattern
}

// NewInteractiveRunner creates a new interactive runner
func NewInteractiveRunner(config InteractiveRunnerConfig) *InteractiveRunner {
	r := &InteractiveRunner{
		config: config,
	}

	if config.EnableCache {
		r.cache = validator.NewValidatorCache(config.CacheDir)
	}

	return r
}

// InteractiveResult contains the result of interactive execution
type InteractiveResult struct {
	SolutionVerdict  string        // correct, time-limit, memory-limit, run-error
	InteractorExit   int           // DOMjudge exit code (42=correct, 43=wrong)
	InteractorOutput []byte        // Interactor stderr for feedback
	TimeUsed         time.Duration // Solution CPU time
	MemoryUsed       int64         // Solution memory in KB
	Error            string        // Error message if any
}

// RunInteractive executes a solution with an interactor (interactive runner)
// The interactor communicates with the solution via bidirectional pipes:
//   - Interactor stdout -> Solution stdin
//   - Solution stdout -> Interactor stdin
// Test case input is passed to the interactor as a file argument
func (r *InteractiveRunner) RunInteractive(
	ctx context.Context,
	interactorBinary *validator.ValidatorBinary,
	solutionBinaryPath string,
	testcaseInput []byte,
	solutionTimeLimit time.Duration,
	solutionMemoryLimit int64,
) (*InteractiveResult, error) {
	// Create work directory
	workDir, err := os.MkdirTemp("", "interactive-run-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Write test case input file
	inputFile := filepath.Join(workDir, "testcase.in")
	if err := os.WriteFile(inputFile, testcaseInput, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Set up pipe connections:
	// interactorStdout -> solutionStdin
	// solutionStdout -> interactorStdin
	interactorStdinReader, interactorStdinWriter := io.Pipe()
	interactorStdoutReader, interactorStdoutWriter := io.Pipe()
	solutionStdinReader, solutionStdinWriter := io.Pipe()
	solutionStdoutReader, solutionStdoutWriter := io.Pipe()

	// Create interactor command
	// DOMjudge-style: interactor receives testcase.in as first argument
	interactorCmd := exec.CommandContext(ctx, interactorBinary.Path, inputFile)
	interactorCmd.Stdin = interactorStdinReader
	interactorCmd.Stdout = interactorStdoutWriter
	var interactorStderr bytes.Buffer
	interactorCmd.Stderr = &interactorStderr

	// Create solution command
	solutionCmd := exec.CommandContext(ctx, solutionBinaryPath)
	solutionCmd.Stdin = solutionStdinReader
	solutionCmd.Stdout = solutionStdoutWriter
	var solutionStderr bytes.Buffer
	solutionCmd.Stderr = &solutionStderr

	// Copy data between pipes:
	// interactor stdout -> solution stdin
	// solution stdout -> interactor stdin
	var copyWg sync.WaitGroup
	copyErrChan := make(chan error, 2)

	// Copy from interactor stdout to solution stdin
	copyWg.Add(1)
	go func() {
		defer copyWg.Done()
		defer solutionStdinWriter.Close()
		_, err := io.Copy(solutionStdinWriter, interactorStdoutReader)
		if err != nil && err != io.ErrClosedPipe {
			copyErrChan <- fmt.Errorf("interactor->solution copy error: %w", err)
		}
	}()

	// Copy from solution stdout to interactor stdin
	copyWg.Add(1)
	go func() {
		defer copyWg.Done()
		defer interactorStdinWriter.Close()
		_, err := io.Copy(interactorStdinWriter, solutionStdoutReader)
		if err != nil && err != io.ErrClosedPipe {
			copyErrChan <- fmt.Errorf("solution->interactor copy error: %w", err)
		}
	}()

	// Start both processes
	startTime := time.Now()

	// Start interactor first (it will drive the interaction)
	if err := interactorCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start interactor: %w", err)
	}

	// Start solution
	if err := solutionCmd.Start(); err != nil {
		// Kill interactor if solution fails to start
		interactorCmd.Process.Kill()
		return nil, fmt.Errorf("failed to start solution: %w", err)
	}

	// Wait for both processes to complete
	var interactorWaitErr, solutionWaitErr error
	var interactorExitCode, solutionExitCode int

	waitWg := sync.WaitGroup{}
	waitWg.Add(2)

	// Wait for interactor
	go func() {
		defer waitWg.Done()
		interactorWaitErr = interactorCmd.Wait()
		if interactorCmd.ProcessState != nil {
			interactorExitCode = interactorCmd.ProcessState.ExitCode()
		} else {
			interactorExitCode = -1
		}
		// Close interactor stdout to signal EOF to solution
		interactorStdoutWriter.Close()
	}()

	// Wait for solution with timeout monitoring
	go func() {
		defer waitWg.Done()
		// Create a separate timeout context for solution
		solutionTimeoutCtx, solutionCancel := context.WithTimeout(context.Background(), solutionTimeLimit)
		defer solutionCancel()

		done := make(chan error, 1)
		go func() {
			done <- solutionCmd.Wait()
		}()

		select {
		case err := <-done:
			solutionWaitErr = err
			if solutionCmd.ProcessState != nil {
				solutionExitCode = solutionCmd.ProcessState.ExitCode()
			} else {
				solutionExitCode = -1
			}
		case <-solutionTimeoutCtx.Done():
			// Solution timed out
			solutionCmd.Process.Kill()
			solutionWaitErr = fmt.Errorf("solution timed out")
			solutionExitCode = -1
		}
	}()

	// Wait for all copy operations to finish
	copyWg.Wait()

	// Wait for both processes
	waitWg.Wait()

	elapsed := time.Since(startTime)

	// Check for copy errors
	select {
	case err := <-copyErrChan:
		log.Printf("Pipe copy error: %v", err)
	default:
	}

	// Determine result
	result := &InteractiveResult{
		TimeUsed: elapsed,
		InteractorOutput: interactorStderr.Bytes(),
	}

	// Get solution memory (approximate - would need cgroups for accurate measurement)
	result.MemoryUsed = 0 // TODO: Implement proper memory tracking

	// Determine verdict based on interactor exit code and solution status
	if ctx.Err() == context.DeadlineExceeded {
		// Overall timeout
		result.SolutionVerdict = "time-limit"
		result.Error = "Interactive execution timed out"
		return result, nil
	}

	// Check if solution had a runtime error first
	if solutionWaitErr != nil && solutionExitCode != 0 {
		if solutionExitCode == -1 {
			// Check if it was a timeout
			if elapsed > solutionTimeLimit {
				result.SolutionVerdict = "time-limit"
			} else {
				result.SolutionVerdict = "run-error"
				result.Error = solutionStderr.String()
			}
		} else {
			result.SolutionVerdict = "run-error"
			result.Error = fmt.Sprintf("Solution exit code %d: %s", solutionExitCode, solutionStderr.String())
		}
		return result, nil
	}

	// Determine verdict from interactor exit code (DOMjudge-style)
	switch interactorExitCode {
	case ExitCodeCorrect:
		result.SolutionVerdict = "correct"
	case ExitCodeWrongAnswer:
		result.SolutionVerdict = "wrong-answer"
	case ExitCodePresentation:
		result.SolutionVerdict = "presentation"
	default:
		if interactorWaitErr != nil {
			result.SolutionVerdict = "internal-error"
			result.Error = fmt.Sprintf("Interactor failed (exit %d): %s", interactorExitCode, interactorStderr.String())
		} else {
			// Interactor exited with non-standard code
			result.SolutionVerdict = "internal-error"
			result.Error = fmt.Sprintf("Interactor returned unexpected exit code %d: %s", interactorExitCode, interactorStderr.String())
		}
	}

	return result, nil
}

// GetInteractor retrieves interactor binary from cache or fetches it
func (r *InteractiveRunner) GetInteractor(ctx context.Context, interactorID string, fetchFunc func(ctx context.Context, id string) ([]byte, string, error)) (*validator.ValidatorBinary, error) {
	// Check cache first
	if r.cache != nil {
		cached, ok := r.cache.Get(interactorID)
		if ok {
			return cached, nil
		}
	}

	// Fetch interactor binary
	binaryData, md5sum, err := fetchFunc(ctx, interactorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interactor: %w", err)
	}

	// Store in cache if enabled
	if r.cache != nil {
		cached, err := r.cache.Store(interactorID, binaryData, md5sum)
		if err != nil {
			log.Printf("Warning: failed to cache interactor %s: %v", interactorID, err)
			// Create temporary file anyway
			tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("interactor-%s", interactorID))
			if err := os.WriteFile(tmpFile, binaryData, 0755); err != nil {
				return nil, fmt.Errorf("failed to write interactor binary: %w", err)
			}
			return &validator.ValidatorBinary{
				ID:     interactorID,
				Path:   tmpFile,
				MD5Sum: md5sum,
			}, nil
		}
		return cached, nil
	}

	// No cache - create temporary file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("interactor-%s", interactorID))
	if err := os.WriteFile(tmpFile, binaryData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write interactor binary: %w", err)
	}

	return &validator.ValidatorBinary{
		ID:     interactorID,
		Path:   tmpFile,
		MD5Sum: md5sum,
	}, nil
}