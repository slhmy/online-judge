package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CompileConfig defines detailed compilation settings for each language
type CompileConfig struct {
	ID              string
	LanguageName    string
	Version         string
	SourceFile      string
	BinaryFile      string
	NeedsCompile    bool
	TimeFactor      float64
	MemoryFactor    float64

	// Compilation settings
	CompilerPath    string
	CompileArgs     []string
	CompileTimeout  time.Duration
	CompileMemory   int64 // KB

	// Runtime settings
	RunnerPath      string
	RunArgs         []string

	// Error parsing patterns
	ErrorPatterns   []ErrorPattern
	WarningPatterns []ErrorPattern

	// Additional files/directories needed
	ExtraFiles      []string
}

// ErrorPattern defines how to parse compiler errors
type ErrorPattern struct {
	Pattern     *regexp.Regexp
	Type        string // "error", "warning", "info"
	MessageGroup int   // Which group contains the message
	LineGroup    int   // Which group contains line number (0 if none)
	FileGroup    int   // Which group contains file name (0 if none)
	ColumnGroup  int   // Which group contains column number (0 if none)
}

// CompileError represents a parsed compilation error
type CompileError struct {
	Type     string // "error", "warning"
	Message  string
	File     string
	Line     int
	Column   int
	Raw      string // Original error line
}

// CompileResult contains compilation results with parsed errors
type CompileResult struct {
	Success     bool
	BinaryPath  string
	Errors      []CompileError
	Warnings    []CompileError
	RawOutput   string
	TimeUsed    time.Duration
	MemoryUsed  int64
}

// Language-specific compilation configurations
var compileConfigs = map[string]CompileConfig{
	"cpp": {
		ID:           "cpp",
		LanguageName: "C++",
		Version:      "C++17 (g++)",
		SourceFile:   "main.cpp",
		BinaryFile:   "main",
		NeedsCompile: true,
		TimeFactor:   1.0,
		MemoryFactor: 1.0,

		CompilerPath: "g++",
		CompileArgs: []string{
			"-std=c++17",
			"-O2",
			"-lm",
			"-DONLINE_JUDGE",
			"-pipe",
			"-fno-stack-limit",
			"-o", "main",
			"main.cpp",
		},
		CompileTimeout: 30 * time.Second,
		CompileMemory:  524288, // 512MB

		RunnerPath: "./main",
		RunArgs:    []string{},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`main\.cpp:(\d+):(\d+): error: (.+)$`),
				Type:         "error",
				MessageGroup: 3,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  2,
			},
			{
				Pattern:      regexp.MustCompile(`main\.cpp:(\d+): error: (.+)$`),
				Type:         "error",
				MessageGroup: 2,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`main\.cpp:(\d+):(\d+): warning: (.+)$`),
				Type:         "warning",
				MessageGroup: 3,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  2,
			},
			{
				Pattern:      regexp.MustCompile(`^error: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`undefined reference to .+`),
				Type:         "error",
				MessageGroup: 0,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
		},
	},

	"c": {
		ID:           "c",
		LanguageName: "C",
		Version:      "C11 (gcc)",
		SourceFile:   "main.c",
		BinaryFile:   "main",
		NeedsCompile: true,
		TimeFactor:   1.0,
		MemoryFactor: 1.0,

		CompilerPath: "gcc",
		CompileArgs: []string{
			"-std=c11",
			"-O2",
			"-lm",
			"-DONLINE_JUDGE",
			"-pipe",
			"-o", "main",
			"main.c",
		},
		CompileTimeout: 30 * time.Second,
		CompileMemory:  524288,

		RunnerPath: "./main",
		RunArgs:    []string{},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`main\.c:(\d+):(\d+): error: (.+)$`),
				Type:         "error",
				MessageGroup: 3,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  2,
			},
			{
				Pattern:      regexp.MustCompile(`main\.c:(\d+): error: (.+)$`),
				Type:         "error",
				MessageGroup: 2,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`^error: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
		},
	},

	"python3": {
		ID:           "python3",
		LanguageName: "Python",
		Version:      "Python 3.11",
		SourceFile:   "main.py",
		BinaryFile:   "main.py",
		NeedsCompile: false,
		TimeFactor:   2.0,
		MemoryFactor: 1.5,

		RunnerPath: "python3",
		RunArgs:    []string{"-S", "main.py"},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`File "main\.py", line (\d+)`),
				Type:         "error",
				MessageGroup: 0,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`(\w+Error): (.+)$`),
				Type:         "error",
				MessageGroup: 2,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`SyntaxError: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
		},

		ExtraFiles: []string{"__pycache__"},
	},

	"java": {
		ID:           "java",
		LanguageName: "Java",
		Version:      "Java 17",
		SourceFile:   "Main.java",
		BinaryFile:   "Main.class",
		NeedsCompile: true,
		TimeFactor:   2.0,
		MemoryFactor: 1.5,

		CompilerPath: "javac",
		CompileArgs: []string{
			"-encoding", "UTF8",
			"-source", "17",
			"-target", "17",
			"-Xlint:all",
			"Main.java",
		},
		CompileTimeout: 60 * time.Second,
		CompileMemory:  1048576,

		RunnerPath: "java",
		RunArgs: []string{
			"-Xmx512M",
			"-Xss64M",
			"-DONLINE_JUDGE=true",
			"-enableassertions",
			"Main",
		},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`Main\.java:(\d+): error: (.+)$`),
				Type:         "error",
				MessageGroup: 2,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`Main\.java:(\d+): warning: (.+)$`),
				Type:         "warning",
				MessageGroup: 2,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`^error: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
		},
	},

	"go": {
		ID:           "go",
		LanguageName: "Go",
		Version:      "Go 1.21",
		SourceFile:   "main.go",
		BinaryFile:   "main",
		NeedsCompile: true,
		TimeFactor:   1.0,
		MemoryFactor: 1.0,

		CompilerPath: "go",
		CompileArgs: []string{
			"build",
			"-o", "main",
			"-ldflags", "-s -w",
			"main.go",
		},
		CompileTimeout: 60 * time.Second,
		CompileMemory:  524288,

		RunnerPath: "./main",
		RunArgs:    []string{},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`main\.go:(\d+):(\d+): (.+)$`),
				Type:         "error",
				MessageGroup: 3,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  2,
			},
			{
				Pattern:      regexp.MustCompile(`main\.go:(\d+): (.+)$`),
				Type:         "error",
				MessageGroup: 2,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`^undefined: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`syntax error: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
		},
	},

	"rust": {
		ID:           "rust",
		LanguageName: "Rust",
		Version:      "Rust (stable)",
		SourceFile:   "main.rs",
		BinaryFile:   "main",
		NeedsCompile: true,
		TimeFactor:   1.0,
		MemoryFactor: 1.0,

		CompilerPath: "rustc",
		CompileArgs: []string{
			"-O",
			"-o", "main",
			"main.rs",
		},
		CompileTimeout: 120 * time.Second,
		CompileMemory:  1048576,

		RunnerPath: "./main",
		RunArgs:    []string{},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`error(?:\[\w+\])?: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`--> main\.rs:(\d+):(\d+)$`),
				Type:         "error",
				MessageGroup: 0,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  2,
			},
		},

		WarningPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`warning: (.+)$`),
				Type:         "warning",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
		},
	},

	"nodejs": {
		ID:           "nodejs",
		LanguageName: "Node.js",
		Version:      "Node.js 18",
		SourceFile:   "main.js",
		BinaryFile:   "main.js",
		NeedsCompile: false,
		TimeFactor:   2.0,
		MemoryFactor: 1.5,

		RunnerPath: "node",
		RunArgs:    []string{"--optimize_for_size", "main.js"},

		ErrorPatterns: []ErrorPattern{
			{
				Pattern:      regexp.MustCompile(`SyntaxError: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`ReferenceError: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`TypeError: (.+)$`),
				Type:         "error",
				MessageGroup: 1,
				LineGroup:    0,
				FileGroup:    0,
				ColumnGroup:  0,
			},
			{
				Pattern:      regexp.MustCompile(`at .+ \(main\.js:(\d+):(\d+)\)$`),
				Type:         "error",
				MessageGroup: 0,
				LineGroup:    1,
				FileGroup:    0,
				ColumnGroup:  2,
			},
		},
	},
}

// CompileSource compiles source code with detailed error parsing
func (s *DockerSandbox) CompileSource(ctx context.Context, source string, language string) (*CompileResult, error) {
	cfg, ok := compileConfigs[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Write source file
	sourcePath := filepath.Join(s.workDir, cfg.SourceFile)
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	result := &CompileResult{
		Success:  false,
		Errors:   []CompileError{},
		Warnings: []CompileError{},
	}

	// If no compilation needed, return success
	if !cfg.NeedsCompile {
		result.Success = true
		result.BinaryPath = sourcePath
		return result, nil
	}

	// Run compilation
	startTime := time.Now()
	compileCmd := exec.CommandContext(ctx, cfg.CompilerPath, cfg.CompileArgs...)
	compileCmd.Dir = s.workDir

	var stdout, stderr bytes.Buffer
	compileCmd.Stdout = &stdout
	compileCmd.Stderr = &stderr

	err := compileCmd.Run()
	result.TimeUsed = time.Since(startTime)
	result.RawOutput = stderr.String()

	// Parse errors from output
	result.Errors = ParseCompileErrors(result.RawOutput, cfg.ErrorPatterns)
	result.Warnings = ParseCompileErrors(result.RawOutput, cfg.WarningPatterns)

	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("compilation timeout after %v", cfg.CompileTimeout)
		}

		// Compilation failed
		result.Success = false
		return result, fmt.Errorf("compilation failed: %s", result.RawOutput)
	}

	// Compilation succeeded
	result.Success = true
	result.BinaryPath = filepath.Join(s.workDir, cfg.BinaryFile)
	return result, nil
}

// ParseCompileErrors parses compiler output using patterns
func ParseCompileErrors(output string, patterns []ErrorPattern) []CompileError {
	var errors []CompileError

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		for _, pattern := range patterns {
			matches := pattern.Pattern.FindStringSubmatch(line)
			if matches != nil {
				ce := CompileError{
					Type: pattern.Type,
					Raw:  line,
				}

				// Extract message
				if pattern.MessageGroup > 0 && len(matches) > pattern.MessageGroup {
					ce.Message = matches[pattern.MessageGroup]
				} else if pattern.MessageGroup == 0 {
					ce.Message = line
				}

				// Extract line number
				if pattern.LineGroup > 0 && len(matches) > pattern.LineGroup {
					lineNum, err := strconv.Atoi(matches[pattern.LineGroup])
					if err == nil {
						ce.Line = lineNum
					}
				}

				// Extract column number
				if pattern.ColumnGroup > 0 && len(matches) > pattern.ColumnGroup {
					colNum, err := strconv.Atoi(matches[pattern.ColumnGroup])
					if err == nil {
						ce.Column = colNum
					}
				}

				errors = append(errors, ce)
				break
			}
		}
	}

	return errors
}

// FormatCompileErrors formats compile errors for user display
func FormatCompileErrors(errors []CompileError, maxLines int) string {
	if len(errors) == 0 {
		return ""
	}

	var formatted []string
	count := 0

	for _, err := range errors {
		if count >= maxLines {
			formatted = append(formatted, fmt.Sprintf("... and %d more errors", len(errors)-maxLines))
			break
		}

		var location string
		if err.Line > 0 {
			if err.Column > 0 {
				location = fmt.Sprintf("line %d:%d", err.Line, err.Column)
			} else {
				location = fmt.Sprintf("line %d", err.Line)
			}
		}

		var line string
		if location != "" {
			line = fmt.Sprintf("[%s] %s: %s", err.Type, location, err.Message)
		} else {
			line = fmt.Sprintf("[%s] %s", err.Type, err.Message)
		}

		formatted = append(formatted, line)
		count++
	}

	return strings.Join(formatted, "\n")
}

// GetCompileConfig returns the compile configuration for a language
func GetCompileConfig(language string) (*CompileConfig, error) {
	cfg, ok := compileConfigs[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
	return &cfg, nil
}

// GetSupportedLanguages returns a list of supported language IDs
func GetSupportedLanguages() []string {
	languages := make([]string, 0, len(compileConfigs))
	for lang := range compileConfigs {
		languages = append(languages, lang)
	}
	return languages
}

// GetLanguageInfo returns human-readable information about supported languages
func GetLanguageInfo() []map[string]string {
	info := make([]map[string]string, 0, len(compileConfigs))
	for _, cfg := range compileConfigs {
		info = append(info, map[string]string{
			"id":            cfg.ID,
			"name":          cfg.LanguageName,
			"version":       cfg.Version,
			"time_factor":   fmt.Sprintf("%.1f", cfg.TimeFactor),
			"memory_factor": fmt.Sprintf("%.1f", cfg.MemoryFactor),
			"needs_compile": fmt.Sprintf("%v", cfg.NeedsCompile),
		})
	}
	return info
}

// ValidateSource performs basic validation of source code before compilation
func ValidateSource(source string, language string) []string {
	var warnings []string

	_, ok := compileConfigs[language]
	if !ok {
		warnings = append(warnings, fmt.Sprintf("Unknown language: %s", language))
		return warnings
	}

	// Check source length
	if len(source) == 0 {
		warnings = append(warnings, "Empty source code")
		return warnings
	}

	// Language-specific checks
	switch language {
	case "java":
		// Check for class name (must be Main)
		if !strings.Contains(source, "class Main") && !strings.Contains(source, "public class Main") {
			warnings = append(warnings, "Java class must be named 'Main'")
		}
		// Check for package declaration (should not have one)
		if strings.Contains(source, "package ") {
			warnings = append(warnings, "Java source should not have a package declaration")
		}

	case "go":
		// Check for package declaration
		if !strings.HasPrefix(strings.TrimSpace(source), "package") {
			warnings = append(warnings, "Go source must start with package declaration")
		}

	case "python3":
		// Check for potential encoding issues
		if strings.Contains(source, "coding:") || strings.Contains(source, "encoding:") {
			warnings = append(warnings, "Python encoding declaration may cause issues")
		}

	case "cpp", "c":
		// Check for includes
		if !strings.Contains(source, "#include") {
			warnings = append(warnings, "C/C++ source has no #include statements")
		}

	case "nodejs":
		// Check for potential async issues
		if strings.Contains(source, "await") && !strings.Contains(source, "async") {
			warnings = append(warnings, "await used without async function")
		}
	}

	return warnings
}

// CompileScript generates a shell script for compilation (for documentation/debugging)
func CompileScript(language string) string {
	cfg, ok := compileConfigs[language]
	if !ok {
		return "# Unknown language"
	}

	var script strings.Builder
	script.WriteString(fmt.Sprintf("#!/bin/bash\n# Compilation script for %s (%s)\n\n", cfg.LanguageName, cfg.Version))

	if cfg.NeedsCompile {
		script.WriteString("# Compile\n")
		script.WriteString(fmt.Sprintf("%s %s\n\n", cfg.CompilerPath, strings.Join(cfg.CompileArgs, " ")))

		script.WriteString("# Check if compilation succeeded\n")
		script.WriteString("if [ $? -eq 0 ]; then\n")
		script.WriteString("    echo 'Compilation successful'\n")
		script.WriteString("else\n")
		script.WriteString("    echo 'Compilation failed'\n")
		script.WriteString("    exit 1\n")
		script.WriteString("fi\n\n")
	} else {
		script.WriteString(fmt.Sprintf("# No compilation needed for %s\n", cfg.LanguageName))
	}

	script.WriteString("# Run\n")
	script.WriteString(fmt.Sprintf("%s %s\n", cfg.RunnerPath, strings.Join(cfg.RunArgs, " ")))

	return script.String()
}