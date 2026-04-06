package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Limits defines resource constraints for sandbox execution
type Limits struct {
	TimeLimit   time.Duration // CPU time limit
	MemoryLimit int64         // Memory limit in kilobytes
	OutputLimit int64         // Output limit in kilobytes
	ProcessLimit int          // Max number of processes
}

// Result contains the execution result
type Result struct {
	Verdict    string        // correct, wrong-answer, time-limit, memory-limit, run-error, output-limit, compiler-error
	ExitCode   int           // Process exit code
	TimeUsed   time.Duration // Actual CPU time used
	MemoryUsed int64         // Memory used in kilobytes (approximate)
	Output     []byte        // stdout
	Error      []byte        // stderr
}

// TestCaseResult represents result for a single test case
type TestCaseResult struct {
	TestCaseID string
	Rank       int
	Verdict    string
	Input      string
	Expected   string
	Output     string
	Runtime    float64
	Memory     int64
	Pass       bool
}

// LanguageConfig defines compilation and execution for each language
type LanguageConfig struct {
	ID           string
	CompileCmd   []string
	RunCmd       []string
	SourceFile   string
	BinaryFile   string
	NeedsCompile bool
	Image        string
	TimeFactor   float64
}

// Language configurations
var languageConfigs = map[string]LanguageConfig{
	"cpp": {
		ID:           "cpp",
		CompileCmd:   []string{"g++", "-std=c++17", "-O2", "-lm", "-DONLINE_JUDGE", "-pipe", "-o", "main", "main.cpp"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.cpp",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "judge-runtime:latest",
		TimeFactor:   1.0,
	},
	"c": {
		ID:           "c",
		CompileCmd:   []string{"gcc", "-std=c11", "-O2", "-lm", "-DONLINE_JUDGE", "-pipe", "-o", "main", "main.c"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.c",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "judge-runtime:latest",
		TimeFactor:   1.0,
	},
	"python3": {
		ID:           "python3",
		CompileCmd:   nil,
		RunCmd:       []string{"python3", "-S", "main.py"},
		SourceFile:   "main.py",
		BinaryFile:   "main.py",
		NeedsCompile: false,
		Image:        "judge-runtime:latest",
		TimeFactor:   2.0,
	},
	"java": {
		ID:           "java",
		CompileCmd:   []string{"javac", "-encoding", "UTF8", "-source", "17", "-target", "17", "Main.java"},
		RunCmd:       []string{"java", "-Xmx512M", "-Xss64M", "-DONLINE_JUDGE=true", "-enableassertions", "Main"},
		SourceFile:   "Main.java",
		BinaryFile:   "Main.class",
		NeedsCompile: true,
		Image:        "judge-runtime:latest",
		TimeFactor:   2.0,
	},
	"go": {
		ID:           "go",
		CompileCmd:   []string{"go", "build", "-o", "main", "-ldflags", "-s -w", "main.go"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.go",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "judge-runtime:latest",
		TimeFactor:   1.0,
	},
	"rust": {
		ID:           "rust",
		CompileCmd:   []string{"rustc", "-O", "-o", "main", "main.rs"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.rs",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "judge-runtime:latest",
		TimeFactor:   1.0,
	},
	"nodejs": {
		ID:           "nodejs",
		CompileCmd:   nil,
		RunCmd:       []string{"node", "--optimize_for_size", "main.js"},
		SourceFile:   "main.js",
		BinaryFile:   "main.js",
		NeedsCompile: false,
		Image:        "judge-runtime:latest",
		TimeFactor:   2.0,
	},
}

// MiniSandbox implements a lightweight sandbox for test runs
type MiniSandbox struct {
	workDir string
}

// NewMiniSandbox creates a new mini sandbox
func NewMiniSandbox() (*MiniSandbox, error) {
	workDir, err := os.MkdirTemp("", "testrun-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	return &MiniSandbox{
		workDir: workDir,
	}, nil
}

// Cleanup removes the work directory
func (s *MiniSandbox) Cleanup() error {
	return os.RemoveAll(s.workDir)
}

// Compile compiles source code for the given language
func (s *MiniSandbox) Compile(ctx context.Context, source string, language string) error {
	cfg, ok := languageConfigs[language]
	if !ok {
		return fmt.Errorf("unsupported language: %s", language)
	}

	// Write source file
	sourcePath := filepath.Join(s.workDir, cfg.SourceFile)
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return fmt.Errorf("failed to write source file: %w", err)
	}

	// If no compilation needed, return early
	if !cfg.NeedsCompile || cfg.CompileCmd == nil {
		return nil
	}

	// Run compilation in Docker container
	result, err := s.runDocker(ctx, cfg.Image, cfg.CompileCmd, nil, Limits{
		TimeLimit:    30 * time.Second,
		MemoryLimit:  524288, // 512MB
		OutputLimit:  10240,  // 10KB
		ProcessLimit: 10,
	}, false)
	if err != nil {
		return fmt.Errorf("compilation execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("compilation error: %s", string(result.Error))
	}

	return nil
}

// Run executes code with given input and limits
func (s *MiniSandbox) Run(ctx context.Context, language string, input []byte, limits Limits) (*Result, error) {
	cfg, ok := languageConfigs[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Build run command
	runCmd := cfg.RunCmd

	// Apply time factor
	effectiveTimeLimit := time.Duration(float64(limits.TimeLimit) * cfg.TimeFactor)

	// Run in Docker
	result, err := s.runDocker(ctx, cfg.Image, runCmd, input, Limits{
		TimeLimit:    effectiveTimeLimit,
		MemoryLimit:  limits.MemoryLimit,
		OutputLimit:  limits.OutputLimit,
		ProcessLimit: limits.ProcessLimit,
	}, true)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RunTestCases compiles code and runs it against multiple test cases
func (s *MiniSandbox) RunTestCases(ctx context.Context, source string, language string, testCases []TestCaseData, limits Limits) (*Result, []TestCaseResult, error) {
	// Compile first
	compileErr := s.Compile(ctx, source, language)
	if compileErr != nil {
		return &Result{
			Verdict:    "compiler-error",
			ExitCode:   -1,
			Error:      []byte(compileErr.Error()),
		}, nil, compileErr
	}

	// Run each test case
	results := make([]TestCaseResult, len(testCases))
	maxRuntime := 0.0
	maxMemory := int64(0)
	finalVerdict := "correct"

	for i, tc := range testCases {
		runResult, err := s.Run(ctx, language, []byte(tc.Input), limits)
		if err != nil {
			results[i] = TestCaseResult{
				TestCaseID: tc.ID,
				Rank:       tc.Rank,
				Verdict:    "run-error",
				Input:      tc.Input,
				Expected:   tc.Output,
				Output:     "",
				Runtime:    0,
				Memory:     0,
				Pass:       false,
			}
			finalVerdict = "run-error"
			continue
		}

		// Validate output
		tcVerdict := validateOutput(tc.Output, string(runResult.Output))
		pass := tcVerdict == "correct"

		runtimeSeconds := runResult.TimeUsed.Seconds()
		if runtimeSeconds > maxRuntime {
			maxRuntime = runtimeSeconds
		}
		if runResult.MemoryUsed > maxMemory {
			maxMemory = runResult.MemoryUsed
		}

		// Truncate output if too long for display
		outputStr := string(runResult.Output)
		if len(outputStr) > 10000 {
			outputStr = outputStr[:10000] + "... (truncated)"
		}

		results[i] = TestCaseResult{
			TestCaseID: tc.ID,
			Rank:       tc.Rank,
			Verdict:    tcVerdict,
			Input:      tc.Input,
			Expected:   tc.Output,
			Output:     outputStr,
			Runtime:    runtimeSeconds,
			Memory:     runResult.MemoryUsed,
			Pass:       pass,
		}

		// If any test case fails, mark as wrong-answer (but continue running all samples)
		if !pass && finalVerdict == "correct" {
			finalVerdict = tcVerdict
		}
	}

	return &Result{
		Verdict:    finalVerdict,
		TimeUsed:   time.Duration(maxRuntime) * time.Second,
		MemoryUsed: maxMemory,
	}, results, nil
}

// runDocker executes a command in a Docker container
func (s *MiniSandbox) runDocker(ctx context.Context, image string, cmd []string, inputData []byte, limits Limits, isExecution bool) (*Result, error) {
	containerName := fmt.Sprintf("testrun-%d", time.Now().UnixNano())

	// Build Docker run command
	runArgs := []string{
		"run",
		"-i",
		"--rm",
		"--name", containerName,
		"-v", s.workDir + ":/workspace",
		"-w", "/workspace",
	}

	// Apply memory limit (convert KB to bytes)
	memoryBytes := limits.MemoryLimit * 1024
	runArgs = append(runArgs,
		"--memory", fmt.Sprintf("%d", memoryBytes),
		"--memory-swap", fmt.Sprintf("%d", memoryBytes),
	)

	// Apply process limit
	if limits.ProcessLimit > 0 {
		runArgs = append(runArgs, "--pids-limit", fmt.Sprintf("%d", limits.ProcessLimit))
	}

	// For execution mode, apply stricter constraints
	if isExecution {
		runArgs = append(runArgs, "--network", "none")
		runArgs = append(runArgs, "--cap-drop", "ALL")
		runArgs = append(runArgs, "--security-opt", "no-new-privileges")
	}

	// Add image and command
	runArgs = append(runArgs, image)
	runArgs = append(runArgs, cmd...)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, limits.TimeLimit+10*time.Second)
	defer cancel()

	// Execute Docker command
	startTime := time.Now()
	execCmd := exec.CommandContext(timeoutCtx, "docker", runArgs...)

	if len(inputData) > 0 {
		execCmd.Stdin = bytes.NewReader(inputData)
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	runErr := execCmd.Run()
	elapsed := time.Since(startTime)

	result := &Result{
		TimeUsed:   elapsed,
		MemoryUsed: 0, // Docker doesn't report memory easily, we'd need stats API
		Output:     stdout.Bytes(),
		Error:      stderr.Bytes(),
	}

	// Determine verdict
	if runErr != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			result.Verdict = "time-limit"
			result.ExitCode = -1
		} else if execCmd.ProcessState != nil {
			result.ExitCode = execCmd.ProcessState.ExitCode()
			if strings.Contains(stderr.String(), "memory") || strings.Contains(stderr.String(), "OOM") {
				result.Verdict = "memory-limit"
			} else {
				result.Verdict = "run-error"
			}
		} else {
			result.Verdict = "run-error"
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
		result.Verdict = "correct"
	}

	// Check output limit
	if limits.OutputLimit > 0 && int64(len(result.Output)) > limits.OutputLimit*1024 {
		result.Verdict = "output-limit"
	}

	return result, nil
}

// validateOutput compares actual output with expected output
func validateOutput(expected, actual string) string {
	// Normalize whitespace for comparison
	expectedNorm := normalizeWhitespace(expected)
	actualNorm := normalizeWhitespace(actual)

	if expectedNorm == actualNorm {
		return "correct"
	}

	// Check if whitespace-only difference
	expectedNoSpace := strings.ReplaceAll(expectedNorm, " ", "")
	expectedNoSpace = strings.ReplaceAll(expectedNoSpace, "\n", "")
	actualNoSpace := strings.ReplaceAll(actualNorm, " ", "")
	actualNoSpace = strings.ReplaceAll(actualNoSpace, "\n", "")

	if expectedNoSpace == actualNoSpace {
		return "presentation"
	}

	return "wrong-answer"
}

// normalizeWhitespace trims trailing whitespace and normalizes newlines
func normalizeWhitespace(s string) string {
	// Trim trailing whitespace from each line and the whole string
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	result := strings.Join(lines, "\n")
	return strings.TrimRight(result, " \t\r\n")
}

// TestCaseData represents input and expected output for a test case
type TestCaseData struct {
	ID     string
	Rank   int
	Input  string
	Output string
}

// GetLanguageConfig returns the configuration for a language
func GetLanguageConfig(language string) (*LanguageConfig, error) {
	cfg, ok := languageConfigs[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
	return &cfg, nil
}

// SupportedLanguages returns list of supported language IDs
func SupportedLanguages() []string {
	langs := make([]string, 0, len(languageConfigs))
	for id := range languageConfigs {
		langs = append(langs, id)
	}
	return langs
}