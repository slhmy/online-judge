package sandbox

import (
	"archive/tar"
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

// Sandbox interface for code execution
type Sandbox interface {
	Run(ctx context.Context, binary string, args []string, input io.Reader, limits Limits) (*Result, error)
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
}

// Language configurations
var languageConfigs = map[string]LanguageConfig{
	"cpp": {
		ID:           "cpp",
		CompileCmd:   []string{"g++", "-O2", "-std=c++17", "-o", "main", "main.cpp"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.cpp",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "gcc:latest",
		TimeFactor:   1.0,
	},
	"c": {
		ID:           "c",
		CompileCmd:   []string{"gcc", "-O2", "-std=c11", "-o", "main", "main.c"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.c",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "gcc:latest",
		TimeFactor:   1.0,
	},
	"python3": {
		ID:           "python3",
		CompileCmd:   nil, // No compilation needed
		RunCmd:       []string{"python3", "main.py"},
		SourceFile:   "main.py",
		BinaryFile:   "main.py",
		NeedsCompile: false,
		Image:        "python:3.11-slim",
		TimeFactor:   2.0, // Python is slower
	},
	"java": {
		ID:           "java",
		CompileCmd:   []string{"javac", "Main.java"},
		RunCmd:       []string{"java", "Main"},
		SourceFile:   "Main.java",
		BinaryFile:   "Main.class",
		NeedsCompile: true,
		Image:        "openjdk:17-slim",
		TimeFactor:   2.0, // Java startup overhead
	},
	"go": {
		ID:           "go",
		CompileCmd:   []string{"go", "build", "-o", "main", "main.go"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.go",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "golang:1.21-alpine",
		TimeFactor:   1.0,
	},
	"rust": {
		ID:           "rust",
		CompileCmd:   []string{"rustc", "-O", "-o", "main", "main.rs"},
		RunCmd:       []string{"./main"},
		SourceFile:   "main.rs",
		BinaryFile:   "main",
		NeedsCompile: true,
		Image:        "rust:latest",
		TimeFactor:   1.0,
	},
	"nodejs": {
		ID:           "nodejs",
		CompileCmd:   nil, // No compilation needed
		RunCmd:       []string{"node", "main.js"},
		SourceFile:   "main.js",
		BinaryFile:   "main.js",
		NeedsCompile: false,
		Image:        "node:18-alpine",
		TimeFactor:   2.0, // Node.js overhead
	},
}

// DockerSandbox implements Sandbox using Docker containers
type DockerSandbox struct {
	workDir     string
	containerID string
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

	// Run compilation in Docker container
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

	// Return path to compiled binary
	binaryPath := filepath.Join(s.workDir, cfg.BinaryFile)
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

	// Apply time factor
	effectiveTimeLimit := time.Duration(float64(limits.TimeLimit) * cfg.TimeFactor)

	// Run in Docker with network disabled and read-only filesystem
	result, err := s.runDockerWithCopy(ctx, cfg.Image, runCmd, inputData, Limits{
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

// createTarArchive creates a tar archive of the work directory
func (s *DockerSandbox) createTarArchive(tarPath string) error {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	// Walk the work directory and add files to the tar
	return filepath.Walk(s.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(s.workDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write content
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(content); err != nil {
				return err
			}
		}

		return nil
	})
}

// Cleanup removes the work directory
func (s *DockerSandbox) Cleanup() error {
	if s.containerID != "" {
		// Attempt to remove any lingering container
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		exec.CommandContext(ctx, "docker", "rm", "-f", s.containerID).Run()
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