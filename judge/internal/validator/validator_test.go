package validator

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultValidator_Correct(t *testing.T) {
	v := NewDefaultValidator()

	expected := []byte("Hello World\n")
	actual := []byte("Hello World\n")

	verdict := v.Validate(expected, actual)
	if verdict != VerdictCorrect {
		t.Errorf("Expected VerdictCorrect, got %s", verdict)
	}
}

func TestDefaultValidator_WrongAnswer(t *testing.T) {
	v := NewDefaultValidator()

	expected := []byte("Hello World\n")
	actual := []byte("Wrong Answer\n")

	verdict := v.Validate(expected, actual)
	if verdict != VerdictWrongAnswer {
		t.Errorf("Expected VerdictWrongAnswer, got %s", verdict)
	}
}

func TestDefaultValidator_PresentationError(t *testing.T) {
	v := NewDefaultValidator()

	expected := []byte("Hello World\n")
	actual := []byte("Hello   World\n") // Extra spaces

	verdict := v.Validate(expected, actual)
	if verdict != VerdictPresentation {
		t.Errorf("Expected VerdictPresentation, got %s", verdict)
	}
}

func TestDefaultValidator_TrailingWhitespace(t *testing.T) {
	v := NewDefaultValidator()
	v.IgnoreTrailingWhitespace = true

	expected := []byte("Hello World   \n")
	actual := []byte("Hello World\n")

	verdict := v.Validate(expected, actual)
	if verdict != VerdictCorrect {
		t.Errorf("Expected VerdictCorrect with trailing whitespace ignored, got %s", verdict)
	}
}

func TestDefaultValidator_IgnoreCase(t *testing.T) {
	v := NewDefaultValidator()
	v.IgnoreCase = true

	expected := []byte("Hello World\n")
	actual := []byte("HELLO WORLD\n")

	verdict := v.Validate(expected, actual)
	if verdict != VerdictCorrect {
		t.Errorf("Expected VerdictCorrect with case ignored, got %s", verdict)
	}
}

// Test helper validator script execution
func createTestValidatorScript(t *testing.T, script string) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "validator.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write validator script: %v", err)
	}
	return scriptPath
}

func TestSpecialValidator_ExitCodeCorrect(t *testing.T) {
	// Create a simple validator that returns exit code 42
	script := `#!/bin/bash
exit 42
`
	scriptPath := createTestValidatorScript(t, script)

	// Create a mock special validator
	cfg := DefaultSpecialValidatorConfig()
	cfg.EnableCache = false

	// Create test files
	workDir := t.TempDir()
	inputFile := filepath.Join(workDir, "test.in")
	answerFile := filepath.Join(workDir, "test.out")
	outputFile := filepath.Join(workDir, "team.out")

	os.WriteFile(inputFile, []byte("input"), 0644)
	os.WriteFile(answerFile, []byte("expected"), 0644)
	os.WriteFile(outputFile, []byte("actual"), 0644)

	// Manually run validator
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmdArgs := []string{inputFile, answerFile, outputFile}
	cmd := exec.CommandContext(ctx, scriptPath, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	if cmd.ProcessState != nil {
		exitCode := cmd.ProcessState.ExitCode()
		if exitCode != ExitCodeCorrect {
			t.Errorf("Expected exit code %d (Correct), got %d", ExitCodeCorrect, exitCode)
		}
	}
}

func TestSpecialValidator_ExitCodeWrongAnswer(t *testing.T) {
	// Create a simple validator that returns exit code 43
	script := `#!/bin/bash
exit 43
`
	scriptPath := createTestValidatorScript(t, script)

	workDir := t.TempDir()
	inputFile := filepath.Join(workDir, "test.in")
	answerFile := filepath.Join(workDir, "test.out")
	outputFile := filepath.Join(workDir, "team.out")

	os.WriteFile(inputFile, []byte("input"), 0644)
	os.WriteFile(answerFile, []byte("expected"), 0644)
	os.WriteFile(outputFile, []byte("actual"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmdArgs := []string{inputFile, answerFile, outputFile}
	cmd := exec.CommandContext(ctx, scriptPath, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	if cmd.ProcessState != nil {
		exitCode := cmd.ProcessState.ExitCode()
		if exitCode != ExitCodeWrongAnswer {
			t.Errorf("Expected exit code %d (Wrong Answer), got %d", ExitCodeWrongAnswer, exitCode)
		}
	}
}

func TestValidatorCache_StoreAndGet(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewValidatorCache(cacheDir)

	validatorID := "test-validator-123"
	binaryData := []byte("#!/bin/bash\nexit 42\n")
	md5sum := "abc123"

	// Store validator
	binary, err := cache.Store(validatorID, binaryData, md5sum)
	if err != nil {
		t.Fatalf("Failed to store validator: %v", err)
	}

	if binary.ID != validatorID {
		t.Errorf("Expected ID %s, got %s", validatorID, binary.ID)
	}

	// Get validator from cache
	retrieved, ok := cache.Get(validatorID)
	if !ok {
		t.Error("Expected to find validator in cache")
	}

	if retrieved.ID != validatorID {
		t.Errorf("Expected ID %s, got %s", validatorID, retrieved.ID)
	}

	// Verify file exists
	if _, err := os.Stat(retrieved.Path); err != nil {
		t.Errorf("Validator file should exist: %v", err)
	}
}

func TestValidatorCache_NotFound(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewValidatorCache(cacheDir)

	_, ok := cache.Get("non-existent")
	if ok {
		t.Error("Expected validator not to be found")
	}
}

func TestValidatorCache_Size(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewValidatorCache(cacheDir)

	if cache.Size() != 0 {
		t.Errorf("Expected empty cache to have size 0, got %d", cache.Size())
	}

	// Store multiple validators
	for i := 0; i < 3; i++ {
		cache.Store("validator-"+string(rune('A'+i)), []byte("test"), "md5")
	}

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}
}

// MockHTTPClient for testing special validator fetching
type mockHTTPClient struct {
	data []byte
	err  error
}

func (m *mockHTTPClient) Get(ctx context.Context, url string) ([]byte, error) {
	return m.data, m.err
}

func TestSpecialValidator_WithMockHTTPClient(t *testing.T) {
	// Create a validator script that will be returned by mock HTTP client
	validatorScript := `#!/bin/bash
INPUT="$1"
ANSWER="$2"
OUTPUT="$3"

EXPECTED=$(cat "$ANSWER")
ACTUAL=$(cat "$OUTPUT")

if [ "$EXPECTED" = "$ACTUAL" ]; then
	echo "Correct"
	exit 42
else
	echo "Wrong Answer"
	exit 43
fi
`

	// Set up mock client
	mockClient := &mockHTTPClient{
		data: []byte(validatorScript),
		err:  nil,
	}

	cfg := DefaultSpecialValidatorConfig()
	cfg.EnableCache = false
	cfg.CacheDir = t.TempDir()

	// Create special validator with mock client
	sv := NewSpecialValidator(cfg, mockClient, "http://test")

	ctx := context.Background()
	validatorID := "test-validator"
	args := ""

	// Test correct output
	input := []byte("test input")
	expected := []byte("Hello World\n")
	actualCorrect := []byte("Hello World\n")

	verdict, feedback := sv.Validate(ctx, validatorID, args, input, expected, actualCorrect)
	if verdict != VerdictCorrect {
		t.Errorf("Expected VerdictCorrect, got %s (feedback: %s)", verdict, feedback)
	}

	// Test wrong answer
	actualWrong := []byte("Wrong Answer\n")
	verdict, feedback = sv.Validate(ctx, validatorID, args, input, expected, actualWrong)
	if verdict != VerdictWrongAnswer {
		t.Errorf("Expected VerdictWrongAnswer, got %s (feedback: %s)", verdict, feedback)
	}
}

func TestSpecialValidator_CacheIntegration(t *testing.T) {
	validatorScript := `#!/bin/bash
exit 42
`

	mockClient := &mockHTTPClient{
		data: []byte(validatorScript),
		err:  nil,
	}

	cacheDir := t.TempDir()
	cfg := DefaultSpecialValidatorConfig()
	cfg.EnableCache = true
	cfg.CacheDir = cacheDir

	sv := NewSpecialValidator(cfg, mockClient, "http://test")

	ctx := context.Background()
	validatorID := "cached-validator"

	// First call should fetch and cache
	verdict1, _ := sv.Validate(ctx, validatorID, "", []byte("input"), []byte("expected"), []byte("actual"))
	if verdict1 != VerdictCorrect {
		t.Errorf("First call: Expected VerdictCorrect, got %s", verdict1)
	}

	// Second call should use cache (mock client won't be called again)
	verdict2, _ := sv.Validate(ctx, validatorID, "", []byte("input"), []byte("expected"), []byte("actual"))
	if verdict2 != VerdictCorrect {
		t.Errorf("Second call (cached): Expected VerdictCorrect, got %s", verdict2)
	}

	// Verify cache has the validator
	if sv.cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", sv.cache.Size())
	}
}