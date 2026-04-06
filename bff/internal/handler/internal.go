package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	pbProblem "github.com/online-judge/backend/gen/go/problem/v1"
	pbSubmission "github.com/online-judge/backend/gen/go/submission/v1"
)

// InternalHandler handles internal API endpoints for judge daemon
type InternalHandler struct {
	submissionClient pbSubmission.SubmissionServiceClient
	problemClient    pbProblem.ProblemServiceClient
	redis            *redis.Client
}

// NewInternalHandler creates a new internal handler
func NewInternalHandler(submissionClient pbSubmission.SubmissionServiceClient, problemClient pbProblem.ProblemServiceClient, redis *redis.Client) *InternalHandler {
	return &InternalHandler{
		submissionClient: submissionClient,
		problemClient:    problemClient,
		redis:            redis,
	}
}

// RegisterRoutes registers internal API routes
func (h *InternalHandler) RegisterRoutes(r chi.Router) {
	r.Route("/internal", func(r chi.Router) {
		// Submission source code
		r.Get("/submissions/{id}/source", h.GetSubmissionSource)

		// Problem test cases (all, not just samples)
		r.Get("/problems/{id}/testcases", h.GetProblemTestCases)

		// Test case input/output data
		r.Get("/testcases/{id}/input", h.GetTestCaseInput)
		r.Get("/testcases/{id}/output", h.GetTestCaseOutput)

		// Executables (validators, run scripts)
		r.Get("/executables/{id}", h.GetExecutable)

		// Judging management
		r.Post("/judgings", h.CreateJudging)
		r.Put("/judgings/{id}", h.UpdateJudging)
		r.Post("/judgings/{id}/runs", h.CreateJudgingRun)

		// Progress updates
		r.Post("/submissions/{id}/progress", h.UpdateProgress)
	})
}

// GetSubmissionSource returns the source code for a submission
func (h *InternalHandler) GetSubmissionSource(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := chi.URLParam(r, "id")

	// Try Redis cache first (faster for judge daemon)
	sourceKey := "submission:" + id + ":source"
	source, err := h.redis.Get(ctx, sourceKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"submission_id": id,
			"source_code":   source,
		})
		return
	}

	// If not in cache, we need to fetch the submission from the database
	// The source code is stored in the submissions table
	// For now, return an error if not in cache
	http.Error(w, "source code not found in cache", http.StatusNotFound)
}

// GetProblemTestCases returns all test cases for a problem (not just samples)
func (h *InternalHandler) GetProblemTestCases(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	problemID := chi.URLParam(r, "id")

	resp, err := h.problemClient.ListTestCases(ctx, &pbProblem.ListTestCasesRequest{
		ProblemId:   problemID,
		SamplesOnly: false,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON-friendly format with input/output data placeholders
	testCases := make([]map[string]interface{}, len(resp.TestCases))
	for i, tc := range resp.TestCases {
		testCases[i] = map[string]interface{}{
			"id":            tc.Id,
			"problem_id":    tc.ProblemId,
			"rank":          tc.Rank,
			"is_sample":     tc.IsSample,
			"input_path":    tc.InputPath,
			"output_path":   tc.OutputPath,
			"description":   tc.Description,
			"is_interactive": tc.IsInteractive,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"test_cases": testCases,
	})
}

// GetTestCaseInput returns the input data for a test case
func (h *InternalHandler) GetTestCaseInput(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	testCaseID := chi.URLParam(r, "id")

	// Try to get from cache
	inputKey := "testcase:" + testCaseID + ":input"
	input, err := h.redis.Get(ctx, inputKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(input))
		return
	}

	// For now, return placeholder until we implement object storage integration
	// In production, this would fetch from MinIO/S3
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Sample input for test case " + testCaseID + "\n"))
}

// GetTestCaseOutput returns the expected output for a test case
func (h *InternalHandler) GetTestCaseOutput(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	testCaseID := chi.URLParam(r, "id")

	// Try to get from cache
	outputKey := "testcase:" + testCaseID + ":output"
	output, err := h.redis.Get(ctx, outputKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(output))
		return
	}

	// For now, return placeholder until we implement object storage integration
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Sample output for test case " + testCaseID + "\n"))
}

// CreateJudging creates a new judging record
func (h *InternalHandler) CreateJudging(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req struct {
		SubmissionID string `json:"submission_id"`
		JudgehostID  string `json:"judgehost_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call gRPC to create judging in database
	resp, err := h.submissionClient.InternalCreateJudging(ctx, &pbSubmission.InternalCreateJudgingRequest{
		SubmissionId: req.SubmissionID,
		JudgehostId:  req.JudgehostID,
	})
	if err != nil {
		log.Printf("Failed to create judging via gRPC: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish progress update to Redis pub/sub
	progressKey := fmt.Sprintf("judging:result:%s", req.SubmissionID)
	progress := map[string]interface{}{
		"submission_id": req.SubmissionID,
		"status":        "judging",
		"progress":      0,
		"verdict":       "Pending",
	}
	progressData, _ := json.Marshal(progress)
	h.redis.Publish(ctx, progressKey, string(progressData))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"judging_id": resp.JudgingId,
	})
}

// UpdateJudging updates a judging record with results
func (h *InternalHandler) UpdateJudging(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	judgingID := chi.URLParam(r, "id")

	var req struct {
		Verdict      string  `json:"verdict"`
		MaxRuntime   float64 `json:"max_runtime"`
		MaxMemory    int64   `json:"max_memory"`
		Error        string  `json:"error"`
		CompileError string  `json:"compile_error"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call gRPC to update judging in database
	_, err := h.submissionClient.InternalUpdateJudging(ctx, &pbSubmission.InternalUpdateJudgingRequest{
		JudgingId:   judgingID,
		Verdict:     req.Verdict,
		MaxRuntime:  req.MaxRuntime,
		MaxMemory:   int32(req.MaxMemory),
	})
	if err != nil {
		log.Printf("Failed to update judging via gRPC: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get submission ID from Redis cache for notification
	judgingKey := "judging:" + judgingID + ":meta"
	submissionID, err := h.redis.HGet(ctx, judgingKey, "submission_id").Result()
	if err == nil {
		// Publish final result via Redis pub/sub
		resultKey := fmt.Sprintf("judging:result:%s", submissionID)
		result := map[string]interface{}{
			"submission_id": submissionID,
			"status":        "completed",
			"verdict":       req.Verdict,
			"runtime":       req.MaxRuntime,
			"memory":        req.MaxMemory,
			"progress":      100,
		}
		resultData, _ := json.Marshal(result)
		h.redis.Publish(ctx, resultKey, string(resultData))

		log.Printf("Published judging result for submission %s: %s", submissionID, req.Verdict)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"judging_id": judgingID,
		"status":     "updated",
	})
}

// CreateJudgingRun creates a record for a test case run
func (h *InternalHandler) CreateJudgingRun(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	judgingID := chi.URLParam(r, "id")

	var req struct {
		TestCaseID string  `json:"test_case_id"`
		Rank       int     `json:"rank"`
		Verdict    string  `json:"verdict"`
		Runtime    float64 `json:"runtime"`
		Memory     int64   `json:"memory"`
		Output     string  `json:"output"`
		Error      string  `json:"error"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call gRPC to create judging run in database
	_, err := h.submissionClient.InternalCreateJudgingRun(ctx, &pbSubmission.InternalCreateJudgingRunRequest{
		JudgingId:  judgingID,
		TestCaseId: req.TestCaseID,
		Rank:       int32(req.Rank),
		Verdict:    req.Verdict,
		Runtime:    req.Runtime,
		Memory:     int32(req.Memory),
	})
	if err != nil {
		log.Printf("Failed to create judging run via gRPC: %v", err)
		// Don't fail the request, just log the error
	}

	// Get submission ID from judging metadata for notification
	judgingKey := "judging:" + judgingID + ":meta"
	submissionID, err := h.redis.HGet(ctx, judgingKey, "submission_id").Result()
	if err == nil {
		// Publish test case run result to Redis pub/sub
		runKey := fmt.Sprintf("judging:run:%s", submissionID)
		run := map[string]interface{}{
			"submission_id": submissionID,
			"test_case_id":  req.TestCaseID,
			"rank":          req.Rank,
			"verdict":       req.Verdict,
			"runtime":       req.Runtime,
			"memory":        req.Memory,
		}
		runData, _ := json.Marshal(run)
		h.redis.Publish(ctx, runKey, string(runData))

		// Also publish progress update
		progressKey := fmt.Sprintf("judging:result:%s", submissionID)
		progress := map[string]interface{}{
			"submission_id": submissionID,
			"status":        "running",
			"current_case":  req.Rank,
		}

		// Get total cases from cache if available
		totalCasesKey := "submission:" + submissionID + ":total_cases"
		totalCases, err := h.redis.Get(ctx, totalCasesKey).Result()
		if err == nil {
			progress["total_cases"] = totalCases
		}

		progressData, _ := json.Marshal(progress)
		h.redis.Publish(ctx, progressKey, string(progressData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"judging_id": judgingID,
		"status":     "created",
	})
}

// UpdateProgress updates the judging progress for a submission
func (h *InternalHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	submissionID := chi.URLParam(r, "id")

	var req struct {
		Status      string  `json:"status"`       // "compiling", "running", "completed"
		Progress    int     `json:"progress"`     // 0-100
		CurrentCase int     `json:"current_case"` // Current test case being judged
		TotalCases  int     `json:"total_cases"`  // Total test cases
		Verdict     string  `json:"verdict"`      // Current verdict (if known)
		Runtime     float64 `json:"runtime"`      // Current runtime
		Memory      int64   `json:"memory"`       // Current memory usage
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Store total cases for later progress updates
	if req.TotalCases > 0 {
		totalCasesKey := "submission:" + submissionID + ":total_cases"
		h.redis.Set(ctx, totalCasesKey, req.TotalCases, 24*60*60)
	}

	// Publish progress update to Redis pub/sub
	progressKey := fmt.Sprintf("judging:result:%s", submissionID)
	progress := map[string]interface{}{
		"submission_id": submissionID,
		"status":        req.Status,
		"progress":      req.Progress,
		"current_case":  req.CurrentCase,
		"total_cases":   req.TotalCases,
		"verdict":       req.Verdict,
		"runtime":       req.Runtime,
		"memory":        req.Memory,
	}
	progressData, _ := json.Marshal(progress)
	h.redis.Publish(ctx, progressKey, string(progressData))

	log.Printf("Published progress update for submission %s: status=%s, progress=%d", submissionID, req.Status, req.Progress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"submission_id": submissionID,
		"status":        "updated",
	})
}

// GetExecutable returns an executable binary (validator, run script)
func (h *InternalHandler) GetExecutable(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	executableID := chi.URLParam(r, "id")

	// Try Redis cache first
	executableKey := "executable:" + executableID + ":binary"
	binaryData, err := h.redis.Get(ctx, executableKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte(binaryData))
		return
	}

	// Try to get metadata from cache
	metaKey := "executable:" + executableID + ":meta"
	md5sum, err := h.redis.HGet(ctx, metaKey, "md5sum").Result()
	if err == nil {
		// Return metadata if binary not available
		execType, _ := h.redis.HGet(ctx, metaKey, "type").Result()
		execPath, _ := h.redis.HGet(ctx, metaKey, "executable_path").Result()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"executable": map[string]interface{}{
				"id":             executableID,
				"type":           execType,
				"executable_path": execPath,
				"md5sum":         md5sum,
			},
		})
		return
	}

	// For now, return a placeholder validator script for testing
	// In production, this would fetch from database and object storage
	log.Printf("Executable %s not found in cache, returning placeholder", executableID)

	// Return a simple shell validator script that compares output
	placeholderValidator := `#!/bin/bash
# DOMjudge-style validator exit codes:
# 42 = correct
# 43 = wrong-answer
# 44 = presentation error

INPUT="$1"
ANSWER="$2"
OUTPUT="$3"

# Read expected output (trim trailing whitespace)
EXPECTED=$(cat "$ANSWER" | sed 's/[[:space:]]*$//')
ACTUAL=$(cat "$OUTPUT" | sed 's/[[:space:]]*$//')

# Compare
if [ "$EXPECTED" = "$ACTUAL" ]; then
	echo "Correct"
	exit 42
else
	# Check if whitespace-only difference (presentation error)
	EXPECTED_NOSPACE=$(echo "$EXPECTED" | tr -d '[:space:]')
	ACTUAL_NOSPACE=$(echo "$ACTUAL" | tr -d '[:space:]')
	if [ "$EXPECTED_NOSPACE" = "$ACTUAL_NOSPACE" ]; then
		echo "Presentation Error: whitespace differences"
		exit 44
	else
		echo "Wrong Answer"
		exit 43
	fi
fi
`

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write([]byte(placeholderValidator))
}