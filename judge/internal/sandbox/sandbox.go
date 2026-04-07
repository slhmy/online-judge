package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Limits defines resource constraints for sandbox execution
type Limits struct {
	TimeLimit   time.Duration // CPU time limit in seconds
	MemoryLimit int64         // Memory limit in kilobytes
	OutputLimit int64         // Output limit in kilobytes
	ProcessLimit int          // Max number of processes
}

// Result contains the execution result
type Result struct {
	Verdict    string        // correct, wrong-answer, time-limit, memory-limit, run-error, output-limit
	ExitCode   int           // Process exit code
	TimeUsed   time.Duration // Actual CPU time used
	MemoryUsed int64         // Memory used in kilobytes
	Output     []byte        // stdout
	Error      []byte        // stderr
}

// InteractiveResult contains the result of interactive execution
type InteractiveResult struct {
	SolutionVerdict  string        // correct, wrong-answer, time-limit, memory-limit, run-error
	InteractorExit   int           // DOMjudge exit code (42=correct, 43=wrong-answer)
	InteractorOutput []byte        // Interactor stderr for feedback
	TimeUsed         time.Duration // Solution wall time
	MemoryUsed       int64         // Solution memory in KB (approximate)
	Output           []byte        // Any solution output captured
	Error            string        // Error message if any
}

// Sandbox interface for code execution
type Sandbox interface {
	Run(ctx context.Context, binary string, args []string, input io.Reader, limits Limits) (*Result, error)
	RunInteractive(ctx context.Context, solutionBinary string, interactorBinary []byte, testcaseInput []byte, limits Limits) (*InteractiveResult, error)
	Compile(ctx context.Context, source string, language string) (string, error)
	Cleanup() error
}

// LanguageConfig defines compilation and execution for each language
type LanguageConfig struct {
	ID              string
	CompileCmd      []string // Command to compile source
	RunCmd          []string // Command to run binary
	SourceFile      string   // Source file name
	BinaryFile      string   // Output binary name
	NeedsCompile    bool
	Image           string   // Docker image to use
	TimeFactor      float64  // Time multiplier
	MemoryFactor    float64  // Memory multiplier
	ExtraFiles      []string // Additional files needed
}

// Language configurations - comprehensive settings for all supported languages
var languageConfigs = map[string]LanguageConfig{
	"cpp": {
		ID:           "cpp",
		CompileCmd:   []string{"g++", "-std=c++17", "-O2", "-lm", "-DONLINE_JUDGE", "-pipe", "-fno-stack-limit", "-o", "main", "main.cpp"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.cpp",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "judge-runtime:latest", // Use our custom image
		TimeFactor:   1.0,
		MemoryFactor: 1.0,
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
		MemoryFactor: 1.0,
	},
	"python3": {
		ID:           "python3",
		CompileCmd:   nil, // No compilation needed
		RunCmd:       []string{"python3", "-S", "-B", "main.py"},
		SourceFile:   "main.py",
		BinaryFile:   "main.py",
		NeedsCompile: false,
		Image:        "judge-runtime:latest",
		TimeFactor:   2.0, // Python is slower
		MemoryFactor: 1.5, // Python uses more memory
		ExtraFiles:   []string{"__pycache__"},
	},
	"java": {
		ID:           "java",
		CompileCmd:   []string{"javac", "-encoding", "UTF8", "-source", "17", "-target", "17", "Main.java"},
		RunCmd:       []string{"java", "-Xmx512M", "-Xss64M", "-DONLINE_JUDGE=true", "-enableassertions", "Main"},
		SourceFile:   "Main.java",
		BinaryFile:   "Main.class",
		NeedsCompile: true,
		Image:        "judge-runtime:latest",
		TimeFactor:   2.0, // Java startup overhead
		MemoryFactor: 1.5,
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
		MemoryFactor: 1.0,
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
		MemoryFactor: 1.0,
	},
	"nodejs": {
		ID:           "nodejs",
		CompileCmd:   nil, // No compilation needed
		RunCmd:       []string{"node", "--optimize_for_size", "--max-old-space-size=512", "main.js"},
		SourceFile:   "main.js",
		BinaryFile:   "main.js",
		NeedsCompile: false,
		Image:        "judge-runtime:latest",
		TimeFactor:   2.0, // Node.js overhead
		MemoryFactor: 1.5,
	},
}

// DockerSandbox implements Sandbox using Docker containers
type DockerSandbox struct {
	workDir     string
	containerID string
	cache       *CompileCache
}

// sandboxWorkDirBase is the base directory for sandbox work directories
// Can be set via SetSandboxWorkDir before creating sandboxes
var sandboxWorkDirBase string

// SetSandboxWorkDir sets the base directory for sandbox work directories
// This is needed for Docker-in-Docker scenarios where the work directory
// must be accessible from both the host and the container
func SetSandboxWorkDir(base string) {
	sandboxWorkDirBase = base
}

// NewDockerSandbox creates a new Docker sandbox
func NewDockerSandbox() (*DockerSandbox, error) {
	var workDir string
	var err error

	if sandboxWorkDirBase != "" {
		// Use configured base directory
		workDir, err = os.MkdirTemp(sandboxWorkDirBase, "sandbox-*")
	} else {
		// Use default temp directory
		workDir, err = os.MkdirTemp("", "sandbox-*")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	return &DockerSandbox{
		workDir: workDir,
	}, nil
}

// NewDockerSandboxWithCache creates a new Docker sandbox with compilation cache
func NewDockerSandboxWithCache(cache *CompileCache) (*DockerSandbox, error) {
	sb, err := NewDockerSandbox()
	if err != nil {
		return nil, err
	}
	sb.cache = cache
	return sb, nil
}

// Compile compiles source code for the given language
func (s *DockerSandbox) Compile(ctx context.Context, source string, language string) (string, error) {
	cfg, ok := languageConfigs[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	// Write source file
	sourcePath := filepath.Join(s.workDir, cfg.SourceFile)
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return "", fmt.Errorf("failed to write source file: %w", err)
	}

	// If no compilation needed, return source file path
	if !cfg.NeedsCompile || cfg.CompileCmd == nil {
		return sourcePath, nil
	}

	// Check compilation cache
	binaryPath := filepath.Join(s.workDir, cfg.BinaryFile)
	if s.cache != nil {
		cachedBinary, found := s.cache.Get(ctx, language, source)
		if found {
			// Cache hit - write cached binary to work directory
			if err := os.WriteFile(binaryPath, cachedBinary, 0755); err != nil {
				return "", fmt.Errorf("failed to write cached binary: %w", err)
			}
			return binaryPath, nil
		}
	}

	// Cache miss or no cache - run compilation in Docker container
	result, err := s.runDockerWithCopy(ctx, cfg.Image, cfg.CompileCmd, nil, Limits{
		TimeLimit:    30 * time.Second, // Compilation time limit
		MemoryLimit:  524288,            // 512MB for compilation
		OutputLimit:  10240,             // 10KB output limit
		ProcessLimit: 10,
	}, false)
	if err != nil {
		return "", fmt.Errorf("compilation execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("compilation error: %s", string(result.Error))
	}

	// Store compiled binary in cache
	if s.cache != nil {
		binaryData, err := os.ReadFile(binaryPath)
		if err != nil {
			// Log warning but don't fail - compilation succeeded
			fmt.Printf("Warning: failed to read binary for cache: %v\n", err)
		} else {
			if err := s.cache.Set(ctx, language, source, binaryData); err != nil {
				fmt.Printf("Warning: failed to cache compiled binary: %v\n", err)
			}
		}
	}

	// Return path to compiled binary
	return binaryPath, nil
}

// Run executes a binary with given input and limits
func (s *DockerSandbox) Run(ctx context.Context, binary string, args []string, input io.Reader, limits Limits) (*Result, error) {
	// Read input data
	inputData, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Determine language from binary path or use default runner image
	language := detectLanguageFromBinary(binary)
	cfg, ok := languageConfigs[language]
	if !ok {
		cfg = languageConfigs["cpp"] // Default fallback
	}

	// Build run command
	runCmd := cfg.RunCmd
	if len(args) > 0 {
		runCmd = append(runCmd, args...)
	}

	// Apply time factor and memory factor
	effectiveTimeLimit := time.Duration(float64(limits.TimeLimit) * cfg.TimeFactor)
	effectiveMemoryLimit := int64(float64(limits.MemoryLimit) * cfg.MemoryFactor)

	// Run in Docker with network disabled and read-only filesystem
	result, err := s.runDockerWithCopy(ctx, cfg.Image, runCmd, inputData, Limits{
		TimeLimit:    effectiveTimeLimit,
		MemoryLimit:  effectiveMemoryLimit,
		OutputLimit:  limits.OutputLimit,
		ProcessLimit: limits.ProcessLimit,
	}, true)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// runDockerWithCopy executes a command in a Docker container
// For Docker-in-Docker, we use a shared volume mount approach
func (s *DockerSandbox) runDockerWithCopy(ctx context.Context, image string, cmd []string, inputData []byte, limits Limits, isExecution bool) (*Result, error) {
	// Generate a unique container name
	containerName := fmt.Sprintf("sandbox-%d", time.Now().UnixNano())

	// Build Docker run command with volume mount
	// The key for Docker-in-Docker is to use the same path inside and outside
	runArgs := []string{
		"run",
			"-i", // Keep stdin open for input
		"--rm",
		"--name", containerName,
		"-v", s.workDir + ":/workspace",
		"-w", "/workspace",
	}

	// Apply memory limit (convert KB to bytes)
	memoryBytes := limits.MemoryLimit * 1024
	runArgs = append(runArgs,
		"--memory", fmt.Sprintf("%d", memoryBytes),
		"--memory-swap", fmt.Sprintf("%d", memoryBytes), // Disable swap
	)

	// Apply process limit using pids limit (requires cgroups)
	if limits.ProcessLimit > 0 {
		runArgs = append(runArgs, "--pids-limit", fmt.Sprintf("%d", limits.ProcessLimit))
	}

	// For execution mode, apply stricter constraints
	if isExecution {
		// Disable network
		runArgs = append(runArgs, "--network", "none")
		// Read-only filesystem - but we need workspace writable
		// runArgs = append(runArgs, "--read-only")
		// No new privileges
		runArgs = append(runArgs, "--cap-drop", "ALL")
		// Security options
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

	// Set up stdin if we have input
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
		MemoryUsed: 0,
		Output:     stdout.Bytes(),
		Error:      stderr.Bytes(),
	}

	// Determine verdict based on error and exit code
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

// Cleanup removes the work directory
func (s *DockerSandbox) Cleanup() error {
	if s.containerID != "" {
		// Attempt to remove any lingering container
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", s.containerID).Run()
	}
	return os.RemoveAll(s.workDir)
}

// WriteFile writes data to a file in the sandbox work directory
func (s *DockerSandbox) WriteFile(name string, data []byte) error {
	path := filepath.Join(s.workDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadFile reads a file from the sandbox work directory
func (s *DockerSandbox) ReadFile(name string) ([]byte, error) {
	path := filepath.Join(s.workDir, name)
	return os.ReadFile(path)
}

// detectLanguageFromBinary tries to determine language from binary path
func detectLanguageFromBinary(binary string) string {
	ext := filepath.Ext(binary)
	switch ext {
	case ".cpp":
		return "cpp"
	case ".c":
		return "c"
	case ".py":
		return "python3"
	case ".java":
		return "java"
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".js":
		return "nodejs"
	default:
		// Check file name patterns
		base := filepath.Base(binary)
		if base == "Main.class" || strings.HasPrefix(base, "Main") {
			return "java"
		}
		if strings.HasSuffix(binary, ".class") {
			return "java"
		}
		// Default to cpp for compiled binaries
		return "cpp"
	}
}

// GetLanguageConfig returns the configuration for a language
func GetLanguageConfig(language string) (*LanguageConfig, error) {
	cfg, ok := languageConfigs[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
	return &cfg, nil
}

// RunInteractive executes a solution with an interactor for interactive problems
// This method runs two processes connected by pipes:
//   - Interactor stdout -> Solution stdin
//   - Solution stdout -> Interactor stdin
// For Docker-in-Docker, we use a helper script approach
func (s *DockerSandbox) RunInteractive(
	ctx context.Context,
	solutionBinary string,
	interactorBinary []byte,
	testcaseInput []byte,
	limits Limits,
) (*InteractiveResult, error) {
	// DOMjudge-style exit codes
	const (
		ExitCodeCorrect     = 42
		ExitCodeWrongAnswer = 43
		ExitCodePresentation = 44
	)

	// Create work directory for this interactive run
	interactiveDir := filepath.Join(s.workDir, "interactive")
	if err := os.MkdirAll(interactiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create interactive directory: %w", err)
	}

	// Write interactor binary to work directory
	interactorPath := filepath.Join(interactiveDir, "interactor")
	if err := os.WriteFile(interactorPath, interactorBinary, 0755); err != nil {
		return nil, fmt.Errorf("failed to write interactor binary: %w", err)
	}

	// Write test case input to work directory
	inputPath := filepath.Join(interactiveDir, "testcase.in")
	if err := os.WriteFile(inputPath, testcaseInput, 0644); err != nil {
		return nil, fmt.Errorf("failed to write testcase input: %w", err)
	}

	// Determine language from solution binary
	language := detectLanguageFromBinary(solutionBinary)
	cfg, ok := languageConfigs[language]
	if !ok {
		cfg = languageConfigs["cpp"]
	}

	// Apply time factor
	effectiveTimeLimit := time.Duration(float64(limits.TimeLimit) * cfg.TimeFactor)
	effectiveMemoryLimit := int64(float64(limits.MemoryLimit) * cfg.MemoryFactor)

	// Create interactive runner script that will orchestrate the pipe communication
	runnerScript := filepath.Join(interactiveDir, "run_interactive.sh")

	// For container execution, we need simpler paths
	scriptContent := fmt.Sprintf(`#!/bin/bash
INTERACTOR="/workspace/interactive/interactor"
INPUT="/workspace/interactive/testcase.in"
WORKDIR="/workspace/interactive"

# Create named pipes
mkfifo "$WORKDIR/pipe1"
mkfifo "$WORKDIR/pipe2"

# Run interactor: reads from pipe2 (solution output), writes to pipe1 (solution input)
"$INTERACTOR" "$INPUT" < "$WORKDIR/pipe2" > "$WORKDIR/pipe1" 2> "$WORKDIR/interactor_stderr.txt" &
INTERACTOR_PID=$!

# Run solution: reads from pipe1 (interactor output), writes to pipe2 (interactor input)
cd /workspace
%s < "$WORKDIR/pipe1" > "$WORKDIR/pipe2" 2> "$WORKDIR/solution_stderr.txt" &
SOLUTION_PID=$!

# Wait for solution first
wait $SOLUTION_PID 2>/dev/null || true

# Then wait for interactor
wait $INTERACTOR_PID 2>/dev/null
INTERACTOR_EXIT=$?

# Cleanup
rm -f "$WORKDIR/pipe1" "$WORKDIR/pipe2"

# Return interactor exit code
exit $INTERACTOR_EXIT
`, strings.Join(cfg.RunCmd, " "))

	if err := os.WriteFile(runnerScript, []byte(scriptContent), 0755); err != nil {
		return nil, fmt.Errorf("failed to write runner script: %w", err)
	}

	// Run the interactive script in Docker container
	containerName := fmt.Sprintf("interactive-%d", time.Now().UnixNano())

	// Build Docker run command with volume mount
	runArgs := []string{
		"run",
		"-i",
		"--rm",
		"--name", containerName,
		"-v", s.workDir + ":/workspace",
		"-w", "/workspace",
	}

	// Apply memory limit (convert KB to bytes)
	memoryBytes := effectiveMemoryLimit * 1024
	runArgs = append(runArgs,
		"--memory", fmt.Sprintf("%d", memoryBytes),
		"--memory-swap", fmt.Sprintf("%d", memoryBytes),
	)

	// Apply process limit
	if limits.ProcessLimit > 0 {
		runArgs = append(runArgs, "--pids-limit", fmt.Sprintf("%d", limits.ProcessLimit))
	}

	// Security constraints
	runArgs = append(runArgs,
		"--network", "none",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
	)

	// Add image and command
	runArgs = append(runArgs, cfg.Image)
	runArgs = append(runArgs, "/workspace/interactive/run_interactive.sh")

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, effectiveTimeLimit+10*time.Second)
	defer cancel()

	// Execute Docker command
	startTime := time.Now()
	execCmd := exec.CommandContext(timeoutCtx, "docker", runArgs...)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	runErr := execCmd.Run()
	elapsed := time.Since(startTime)

	result := &InteractiveResult{
		TimeUsed: elapsed,
		MemoryUsed: 0, // Approximate - would need cgroups for accurate measurement
	}

	// Read interactor stderr for feedback
	interactorStderrPath := filepath.Join(interactiveDir, "interactor_stderr.txt")
	if data, err := os.ReadFile(interactorStderrPath); err == nil {
		result.InteractorOutput = data
	}

	// Get exit code
	exitCode := -1
	if execCmd.ProcessState != nil {
		exitCode = execCmd.ProcessState.ExitCode()
	}

	// Determine verdict
	if timeoutCtx.Err() == context.DeadlineExceeded {
		result.SolutionVerdict = "time-limit"
		result.Error = "Interactive execution timed out"
		return result, nil
	}

	if runErr != nil {
		// Check Docker error messages for memory limit
		if strings.Contains(stderr.String(), "OOM") || strings.Contains(stderr.String(), "memory") {
			result.SolutionVerdict = "memory-limit"
			result.Error = "Memory limit exceeded"
		} else {
			// Determine from exit code
			switch exitCode {
			case ExitCodeCorrect:
				result.SolutionVerdict = "correct"
			case ExitCodeWrongAnswer:
				result.SolutionVerdict = "wrong-answer"
			case ExitCodePresentation:
				result.SolutionVerdict = "presentation"
			default:
				result.SolutionVerdict = "run-error"
				result.Error = fmt.Sprintf("Execution error (exit %d): %s", exitCode, stderr.String())
			}
		}
	} else {
		// Successful execution - check exit code
		switch exitCode {
		case ExitCodeCorrect:
			result.SolutionVerdict = "correct"
		case ExitCodeWrongAnswer:
			result.SolutionVerdict = "wrong-answer"
		case ExitCodePresentation:
			result.SolutionVerdict = "presentation"
		case 0:
			// Exit code 0 might mean correct in some implementations
			result.SolutionVerdict = "correct"
		default:
			result.SolutionVerdict = "internal-error"
			result.Error = fmt.Sprintf("Unexpected exit code %d: %s", exitCode, string(result.InteractorOutput))
		}
	}

	result.InteractorExit = exitCode
	return result, nil
}