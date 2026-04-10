package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	pbProblem "github.com/online-judge/gen/go/problem/v1"
	"github.com/online-judge/bff/internal/sandbox"
)

// TestRunHandler handles test run requests
type TestRunHandler struct {
	problemClient pbProblem.ProblemServiceClient
}

// NewTestRunHandler creates a new test run handler
func NewTestRunHandler(problemClient pbProblem.ProblemServiceClient) *TestRunHandler {
	return &TestRunHandler{
		problemClient: problemClient,
	}
}

// TestRunRequest represents a test run request
type TestRunRequest struct {
	ProblemID string `json:"problem_id"`
	Language  string `json:"language"`
	Source    string `json:"source"`
}

// TestRunResult represents the complete test run result
type TestRunResult struct {
	ID           string                  `json:"id"`
	Status       string                  `json:"status"`
	Verdict      string                  `json:"verdict"`
	Runtime      float64                 `json:"runtime"`
	Memory       int64                   `json:"memory"`
	CompileError string                  `json:"compile_error,omitempty"`
	TestCases    []TestCaseRunResult     `json:"test_cases"`
}

// TestCaseRunResult represents result for a single test case
type TestCaseRunResult struct {
	TestCaseID string  `json:"test_case_id"`
	Rank       int     `json:"rank"`
	Verdict    string  `json:"verdict"`
	Input      string  `json:"input"`
	Expected   string  `json:"expected"`
	Output     string  `json:"output"`
	Runtime    float64 `json:"runtime"`
	Memory     int64   `json:"memory"`
	Pass       bool    `json:"pass"`
}

// RegisterRoutes registers test run routes
func (h *TestRunHandler) RegisterRoutes(r chi.Router) {
	r.Post("/test-runs", h.Create)
}

// Create handles test run creation
// Endpoint: POST /api/v1/test-runs
func (h *TestRunHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req TestRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ProblemID == "" || req.Language == "" || req.Source == "" {
		http.Error(w, "problem_id, language, and source are required", http.StatusBadRequest)
		return
	}

	// Check language support
	if _, err := sandbox.GetLanguageConfig(req.Language); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate ID for tracking
	testRunID := uuid.New().String()

	// Fetch problem details
	problemResp, err := h.problemClient.GetProblem(ctx, &pbProblem.GetProblemRequest{
		Id: req.ProblemID,
	})
	if err != nil {
		http.Error(w, "failed to fetch problem: "+err.Error(), http.StatusInternalServerError)
		return
	}

	problem := problemResp.Problem
	sampleTestCases := problemResp.SampleTestCases

	// Check if we have sample test cases
	if len(sampleTestCases) == 0 {
		http.Error(w, "no sample test cases available for this problem", http.StatusBadRequest)
		return
	}

	// Convert to sandbox test case data
	testCaseData := make([]sandbox.TestCaseData, len(sampleTestCases))
	for i, tc := range sampleTestCases {
		// Get input/output content
		inputContent := tc.InputContent
		outputContent := tc.OutputContent

		// Truncate for display if too long
		if len(inputContent) > 5000 {
			inputContent = inputContent[:5000] + "... (truncated)"
		}
		if len(outputContent) > 5000 {
			outputContent = outputContent[:5000] + "... (truncated)"
		}

		testCaseData[i] = sandbox.TestCaseData{
			ID:     tc.Id,
			Rank:   int(tc.Rank),
			Input:  inputContent,
			Output: outputContent,
		}
	}

	// Set up limits from problem
	limits := sandbox.Limits{
		TimeLimit:    time.Duration(problem.TimeLimit) * time.Second,
		MemoryLimit:  int64(problem.MemoryLimit),
		OutputLimit:  int64(problem.OutputLimit),
		ProcessLimit: 50, // Default process limit
	}

	// Use defaults if not specified
	if limits.TimeLimit == 0 {
		limits.TimeLimit = 5 * time.Second
	}
	if limits.MemoryLimit == 0 {
		limits.MemoryLimit = 262144 // 256 MB
	}
	if limits.OutputLimit == 0 {
		limits.OutputLimit = 10240 // 10 MB
	}

	// Create sandbox and run test cases
	sb, err := sandbox.NewMiniSandbox()
	if err != nil {
		http.Error(w, "failed to create sandbox: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = sb.Cleanup() }()

	// Run test cases (with extended timeout for HTTP request)
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	result, tcResults, err := sb.RunTestCases(runCtx, req.Source, req.Language, testCaseData, limits)
	if err != nil {
		// Check if it's a compilation error
		if result != nil && result.Verdict == "compiler-error" {
			// Return compilation error result
			_ = json.NewEncoder(w).Encode(TestRunResult{
				ID:           testRunID,
				Status:       "completed",
				Verdict:      "compiler-error",
				CompileError: string(result.Error),
				TestCases:    []TestCaseRunResult{},
			})
			return
		}
		http.Error(w, "execution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert results
	testCaseResults := make([]TestCaseRunResult, len(tcResults))
	for i, tc := range tcResults {
		testCaseResults[i] = TestCaseRunResult{
			TestCaseID: tc.TestCaseID,
			Rank:       tc.Rank,
			Verdict:    tc.Verdict,
			Input:      tc.Input,
			Expected:   tc.Expected,
			Output:     tc.Output,
			Runtime:    tc.Runtime,
			Memory:     tc.Memory,
			Pass:       tc.Pass,
		}
	}

	// Build response
	response := TestRunResult{
		ID:        testRunID,
		Status:    "completed",
		Verdict:   result.Verdict,
		Runtime:   result.TimeUsed.Seconds(),
		Memory:    result.MemoryUsed,
		TestCases: testCaseResults,
	}

	// Return results
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}