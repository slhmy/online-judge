package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slhmy/online-judge/judge/internal/validator"
)

// TestInteractiveRunnerConfig tests the default configuration
func TestInteractiveRunnerConfig(t *testing.T) {
	config := DefaultInteractiveRunnerConfig()
	if config.InteractorTimeLimit != 60*time.Second {
		t.Errorf("Expected interactor time limit 60s, got %v", config.InteractorTimeLimit)
	}
	if config.InteractorMemoryLimit != 524288 {
		t.Errorf("Expected interactor memory limit 524288 KB, got %v", config.InteractorMemoryLimit)
	}
	if config.CacheDir != "/tmp/interactor-cache" {
		t.Errorf("Expected cache dir /tmp/interactor-cache, got %v", config.CacheDir)
	}
	if !config.EnableCache {
		t.Errorf("Expected EnableCache to be true")
	}
}

// TestNewInteractiveRunner tests the creation of an interactive runner
func TestNewInteractiveRunner(t *testing.T) {
	config := DefaultInteractiveRunnerConfig()
	runner := NewInteractiveRunner(config)
	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
	if runner.config.InteractorTimeLimit != config.InteractorTimeLimit {
		t.Errorf("Runner config not set correctly")
	}
}

// TestInteractiveResult tests the interactive result structure
func TestInteractiveResult(t *testing.T) {
	result := &InteractiveResult{
		SolutionVerdict:  "correct",
		InteractorExit:   ExitCodeCorrect,
		InteractorOutput: []byte("feedback"),
		TimeUsed:         time.Second,
		MemoryUsed:       1024,
	}

	if result.SolutionVerdict != "correct" {
		t.Errorf("Expected verdict correct, got %v", result.SolutionVerdict)
	}
	if result.InteractorExit != ExitCodeCorrect {
		t.Errorf("Expected interactor exit code %d, got %d", ExitCodeCorrect, result.InteractorExit)
	}
}

// TestGetInteractor tests fetching an interactor binary
func TestGetInteractor(t *testing.T) {
	// Create temporary directory for cache
	cacheDir, err := os.MkdirTemp("", "interactor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()

	config := InteractiveRunnerConfig{
		InteractorTimeLimit:   30 * time.Second,
		InteractorMemoryLimit: 524288,
		CacheDir:              cacheDir,
		EnableCache:           true,
	}

	r := NewInteractiveRunner(config)

	// Mock fetch function that returns a simple shell script
	mockFetch := func(ctx context.Context, id string) ([]byte, string, error) {
		script := `#!/bin/bash
echo "test interactor"
exit 42
`
		return []byte(script), "test-md5", nil
	}

	// Get interactor
	binary, err := r.GetInteractor(context.Background(), "test-interactor", mockFetch)
	if err != nil {
		t.Fatalf("Failed to get interactor: %v", err)
	}

	if binary == nil {
		t.Fatal("Expected non-nil binary")
	}
	if binary.ID != "test-interactor" {
		t.Errorf("Expected ID test-interactor, got %v", binary.ID)
	}

	// Verify the file exists
	if _, err := os.Stat(binary.Path); os.IsNotExist(err) {
		t.Errorf("Interactor binary file does not exist at %s", binary.Path)
	}

	// Get from cache (should use cached version)
	binary2, err := r.GetInteractor(context.Background(), "test-interactor", mockFetch)
	if err != nil {
		t.Fatalf("Failed to get cached interactor: %v", err)
	}
	if binary.Path != binary2.Path {
		t.Errorf("Expected same cached path, got %s vs %s", binary.Path, binary2.Path)
	}
}

// TestRunInteractive tests the interactive execution (requires a solution and interactor)
func TestRunInteractive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory
	workDir, err := os.MkdirTemp("", "interactive-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	// Create a simple solution that echoes back
	solutionScript := filepath.Join(workDir, "solution")
	solutionContent := `#!/bin/bash
while read line; do
    echo "Response: $line"
done
`
	if err := os.WriteFile(solutionScript, []byte(solutionContent), 0755); err != nil {
		t.Fatalf("Failed to write solution: %v", err)
	}

	// Create a simple interactor
	interactorScript := filepath.Join(workDir, "interactor")
	interactorContent := `#!/bin/bash
# Simple interactor that sends a query and checks response
echo "Hello"
read response
if [ "$response" = "Response: Hello" ]; then
    echo "Correct!" >&2
    exit 42
else
    echo "Wrong: $response" >&2
    exit 43
fi
`
	if err := os.WriteFile(interactorScript, []byte(interactorContent), 0755); err != nil {
		t.Fatalf("Failed to write interactor: %v", err)
	}

	config := InteractiveRunnerConfig{
		InteractorTimeLimit:   10 * time.Second,
		InteractorMemoryLimit: 524288,
		CacheDir:              workDir,
		EnableCache:           false,
	}

	r := NewInteractiveRunner(config)

	// Create validator binary wrapper
	interactorBinary := &validator.ValidatorBinary{
		ID:     "test",
		Path:   interactorScript,
		MD5Sum: "test",
	}

	// Run interactive test
	testcaseInput := []byte("test input\n")
	result, err := r.RunInteractive(
		context.Background(),
		interactorBinary,
		solutionScript,
		testcaseInput,
		5*time.Second,
		262144,
	)

	if err != nil {
		t.Fatalf("RunInteractive failed: %v", err)
	}

	// Check result
	if result.SolutionVerdict != "correct" {
		t.Errorf("Expected verdict 'correct', got '%s'", result.SolutionVerdict)
		t.Errorf("Interactor output: %s", string(result.InteractorOutput))
		t.Errorf("Error: %s", result.Error)
	}
	if result.InteractorExit != ExitCodeCorrect {
		t.Errorf("Expected interactor exit code %d, got %d", ExitCodeCorrect, result.InteractorExit)
	}
}

// TestRunInteractiveTimeout tests timeout handling
func TestRunInteractiveTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory
	workDir, err := os.MkdirTemp("", "interactive-timeout-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	// Create a solution that loops forever
	solutionScript := filepath.Join(workDir, "solution")
	solutionContent := `#!/bin/bash
while true; do
    sleep 1
done
`
	if err := os.WriteFile(solutionScript, []byte(solutionContent), 0755); err != nil {
		t.Fatalf("Failed to write solution: %v", err)
	}

	// Create an interactor that waits forever
	interactorScript := filepath.Join(workDir, "interactor")
	interactorContent := `#!/bin/bash
sleep 60
exit 42
`
	if err := os.WriteFile(interactorScript, []byte(interactorContent), 0755); err != nil {
		t.Fatalf("Failed to write interactor: %v", err)
	}

	config := InteractiveRunnerConfig{
		InteractorTimeLimit:   10 * time.Second,
		InteractorMemoryLimit: 524288,
		CacheDir:              workDir,
		EnableCache:           false,
	}

	r := NewInteractiveRunner(config)

	interactorBinary := &validator.ValidatorBinary{
		ID:     "test",
		Path:   interactorScript,
		MD5Sum: "test",
	}

	// Run with very short timeout
	testcaseInput := []byte("test input\n")
	result, err := r.RunInteractive(
		context.Background(),
		interactorBinary,
		solutionScript,
		testcaseInput,
		2*time.Second, // Short timeout
		262144,
	)

	if err != nil {
		t.Fatalf("RunInteractive failed: %v", err)
	}

	// Should get time-limit verdict
	if result.SolutionVerdict != "time-limit" {
		t.Errorf("Expected verdict 'time-limit', got '%s'", result.SolutionVerdict)
	}
}

// TestRunInteractiveWrongAnswer tests wrong answer handling
func TestRunInteractiveWrongAnswer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory
	workDir, err := os.MkdirTemp("", "interactive-wa-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	// Create a solution that returns wrong response
	solutionScript := filepath.Join(workDir, "solution")
	solutionContent := `#!/bin/bash
while read line; do
    echo "Wrong: $line"
done
`
	if err := os.WriteFile(solutionScript, []byte(solutionContent), 0755); err != nil {
		t.Fatalf("Failed to write solution: %v", err)
	}

	// Create an interactor that expects specific response
	interactorScript := filepath.Join(workDir, "interactor")
	interactorContent := `#!/bin/bash
echo "Query"
read response
if [ "$response" = "Expected Response" ]; then
    exit 42
else
    exit 43
fi
`
	if err := os.WriteFile(interactorScript, []byte(interactorContent), 0755); err != nil {
		t.Fatalf("Failed to write interactor: %v", err)
	}

	config := InteractiveRunnerConfig{
		InteractorTimeLimit:   10 * time.Second,
		InteractorMemoryLimit: 524288,
		CacheDir:              workDir,
		EnableCache:           false,
	}

	r := NewInteractiveRunner(config)

	interactorBinary := &validator.ValidatorBinary{
		ID:     "test",
		Path:   interactorScript,
		MD5Sum: "test",
	}

	testcaseInput := []byte("test input\n")
	result, err := r.RunInteractive(
		context.Background(),
		interactorBinary,
		solutionScript,
		testcaseInput,
		5*time.Second,
		262144,
	)

	if err != nil {
		t.Fatalf("RunInteractive failed: %v", err)
	}

	// Should get wrong-answer verdict
	if result.SolutionVerdict != "wrong-answer" {
		t.Errorf("Expected verdict 'wrong-answer', got '%s'", result.SolutionVerdict)
	}
	if result.InteractorExit != ExitCodeWrongAnswer {
		t.Errorf("Expected interactor exit code %d, got %d", ExitCodeWrongAnswer, result.InteractorExit)
	}
}
