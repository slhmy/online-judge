package validator

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// DOMjudge-style validator exit codes
const (
	ExitCodeCorrect      = 42 // Accept / Correct
	ExitCodeWrongAnswer  = 43 // Wrong Answer
	ExitCodePresentation = 44 // Presentation Error (some validators use this)
)

// SpecialValidatorConfig holds configuration for special validators
type SpecialValidatorConfig struct {
	// Time limit for validator execution (default: 30 seconds)
	TimeLimit time.Duration
	// Memory limit for validator execution in KB (default: 512 MB)
	MemoryLimit int64
	// Cache directory for validator binaries
	CacheDir string
	// Whether to enable caching
	EnableCache bool
}

// DefaultSpecialValidatorConfig returns default configuration
func DefaultSpecialValidatorConfig() SpecialValidatorConfig {
	return SpecialValidatorConfig{
		TimeLimit:   30 * time.Second,
		MemoryLimit: 524288, // 512 MB
		CacheDir:    "/tmp/validator-cache",
		EnableCache: true,
	}
}

// SpecialValidator implements custom validator execution
// Supports DOMjudge-style validators with exit codes 42=correct, 43=wrong-answer
type SpecialValidator struct {
	config           SpecialValidatorConfig
	cache            *ValidatorCache
	validatorFetcher ValidatorFetcher
}

// ValidatorFetcher interface for fetching validator binaries
type ValidatorFetcher interface {
	FetchExecutable(ctx context.Context, executableID string) ([]byte, string, error)
}

// ValidatorBinary represents a cached validator binary
type ValidatorBinary struct {
	ID       string
	Path     string
	MD5Sum   string
	LoadedAt time.Time
}

// NewSpecialValidator creates a new special validator
func NewSpecialValidator(config SpecialValidatorConfig, fetcher ValidatorFetcher) *SpecialValidator {
	v := &SpecialValidator{
		config:           config,
		validatorFetcher: fetcher,
	}

	if config.EnableCache {
		v.cache = NewValidatorCache(config.CacheDir)
	}

	return v
}

// Validate runs a custom validator with input, expected output, and actual output
// Returns verdict and any error message from the validator
func (v *SpecialValidator) Validate(ctx context.Context, validatorID string, args string, input, expected, actual []byte) (Verdict, string) {
	// Get or load validator binary
	binary, err := v.getValidator(ctx, validatorID)
	if err != nil {
		log.Printf("Failed to get validator %s: %v", validatorID, err)
		return VerdictInternalError, fmt.Sprintf("Failed to load validator: %v", err)
	}

	// Create temporary files for validator input
	workDir, err := os.MkdirTemp("", "validator-run-*")
	if err != nil {
		return VerdictInternalError, fmt.Sprintf("Failed to create work directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	// Write test case files
	inputFile := filepath.Join(workDir, "testcase.in")
	answerFile := filepath.Join(workDir, "testcase.out") // Expected output
	outputFile := filepath.Join(workDir, "team.out")     // Actual output (submission output)

	if err := os.WriteFile(inputFile, input, 0644); err != nil {
		return VerdictInternalError, fmt.Sprintf("Failed to write input file: %v", err)
	}
	if err := os.WriteFile(answerFile, expected, 0644); err != nil {
		return VerdictInternalError, fmt.Sprintf("Failed to write answer file: %v", err)
	}
	if err := os.WriteFile(outputFile, actual, 0644); err != nil {
		return VerdictInternalError, fmt.Sprintf("Failed to write output file: %v", err)
	}

	// Build validator command
	// DOMjudge validators receive: <test_input> <test_output> <team_output> [args]
	cmdArgs := []string{inputFile, answerFile, outputFile}
	if args != "" {
		cmdArgs = append(cmdArgs, args)
	}

	// Run validator with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, v.config.TimeLimit)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, binary.Path, cmdArgs...)

	// Capture validator output for feedback
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Check for timeout
	if timeoutCtx.Err() == context.DeadlineExceeded {
		log.Printf("Validator %s timed out", validatorID)
		return VerdictInternalError, "Validator timed out"
	}

	// Get exit code
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	// Combine stdout and stderr for feedback message
	feedback := stdout.String()
	if stderr.Len() > 0 {
		if feedback != "" {
			feedback += "\n"
		}
		feedback += stderr.String()
	}

	// Interpret exit code (DOMjudge-style)
	switch exitCode {
	case ExitCodeCorrect:
		return VerdictCorrect, feedback
	case ExitCodeWrongAnswer:
		return VerdictWrongAnswer, feedback
	case ExitCodePresentation:
		return VerdictPresentation, feedback
	default:
		if runErr != nil {
			log.Printf("Validator %s failed with exit code %d: %v", validatorID, exitCode, runErr)
			return VerdictInternalError, fmt.Sprintf("Validator error (exit %d): %s", exitCode, feedback)
		}
		// Some validators may use non-standard exit codes
		// Treat 0 as correct if not using DOMjudge codes
		if exitCode == 0 {
			return VerdictCorrect, feedback
		}
		return VerdictInternalError, fmt.Sprintf("Validator returned unexpected exit code %d: %s", exitCode, feedback)
	}
}

// getValidator retrieves validator binary, using cache if available
func (v *SpecialValidator) getValidator(ctx context.Context, validatorID string) (*ValidatorBinary, error) {
	// Check cache first
	if v.cache != nil {
		cached, ok := v.cache.Get(validatorID)
		if ok {
			return cached, nil
		}
	}

	// Fetch validator binary from backend
	binaryData, md5sum, err := v.fetchValidator(ctx, validatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch validator: %w", err)
	}

	// Store in cache if enabled
	if v.cache != nil {
		cached, err := v.cache.Store(validatorID, binaryData, md5sum)
		if err != nil {
			log.Printf("Warning: failed to cache validator %s: %v", validatorID, err)
			// Continue anyway - we can still use it
		}
		return cached, nil
	}

	// If no cache, create temporary file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("validator-%s", validatorID))
	if err := os.WriteFile(tmpFile, binaryData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write validator binary: %w", err)
	}

	return &ValidatorBinary{
		ID:     validatorID,
		Path:   tmpFile,
		MD5Sum: md5sum,
	}, nil
}

// fetchValidator downloads validator binary via gRPC
func (v *SpecialValidator) fetchValidator(ctx context.Context, validatorID string) ([]byte, string, error) {
	if v.validatorFetcher == nil {
		return nil, "", fmt.Errorf("no validator fetcher configured")
	}

	return v.validatorFetcher.FetchExecutable(ctx, validatorID)
}

// ValidatorCache manages cached validator binaries
type ValidatorCache struct {
	dir    string
	mu     sync.RWMutex
	binary map[string]*ValidatorBinary
}

// NewValidatorCache creates a new validator cache
func NewValidatorCache(dir string) *ValidatorCache {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Warning: failed to create validator cache directory: %v", err)
	}

	return &ValidatorCache{
		dir:    dir,
		binary: make(map[string]*ValidatorBinary),
	}
}

// Get retrieves a cached validator binary
func (c *ValidatorCache) Get(id string) (*ValidatorBinary, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.binary[id]
	if !ok {
		return nil, false
	}

	// Check if file still exists
	if _, err := os.Stat(cached.Path); err != nil {
		return nil, false
	}

	return cached, true
}

// Store saves a validator binary in the cache
func (c *ValidatorCache) Store(id string, data []byte, md5sum string) (*ValidatorBinary, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we already have this version (by md5sum)
	if existing, ok := c.binary[id]; ok && existing.MD5Sum == md5sum {
		return existing, nil
	}

	// Write binary to cache file
	path := filepath.Join(c.dir, fmt.Sprintf("validator-%s", id))
	if err := os.WriteFile(path, data, 0755); err != nil {
		return nil, fmt.Errorf("failed to write cached binary: %w", err)
	}

	cached := &ValidatorBinary{
		ID:       id,
		Path:     path,
		MD5Sum:   md5sum,
		LoadedAt: time.Now(),
	}

	c.binary[id] = cached
	return cached, nil
}

// Clear removes all cached validators
func (c *ValidatorCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all cached files
	for id, binary := range c.binary {
		if err := os.Remove(binary.Path); err != nil {
			log.Printf("Warning: failed to remove cached validator %s: %v", id, err)
		}
	}

	c.binary = make(map[string]*ValidatorBinary)
	return os.RemoveAll(c.dir)
}

// Size returns the number of cached validators
func (c *ValidatorCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.binary)
}
