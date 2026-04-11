package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/slhmy/online-judge/bff/internal/cache"
	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/problem/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProblemHandler struct {
	client pb.ProblemServiceClient
	cache  *cache.Service
}

func NewProblemHandler(client pb.ProblemServiceClient, cacheService *cache.Service) *ProblemHandler {
	return &ProblemHandler{
		client: client,
		cache:  cacheService,
	}
}

func (h *ProblemHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	resp, err := h.client.ListProblems(ctx, &pb.ListProblemsRequest{
		Pagination: &commonv1.Pagination{
			Page:     1,
			PageSize: 20,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	// Try cache first
	cacheKey := "problem:" + id
	cached, err := h.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "hit")
		_, _ = w.Write(cached)
		return
	}

	// Fetch from gRPC
	resp, err := h.client.GetProblem(ctx, &pb.GetProblemRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache the response
	data, _ := json.Marshal(resp)
	_ = h.cache.Set(ctx, cacheKey, data, h.cache.GetConfig().ProblemTTL, "problem:"+id)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "miss")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) ListLanguages(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	resp, err := h.client.ListLanguages(ctx, &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Admin handlers for problem CRUD

func (h *ProblemHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req struct {
		ExternalID  string  `json:"external_id"`
		Name        string  `json:"name"`
		TimeLimit   float64 `json:"time_limit"`
		MemoryLimit int32   `json:"memory_limit"`
		OutputLimit int32   `json:"output_limit"`
		Difficulty  string  `json:"difficulty"`
		Points      int32   `json:"points"`
		Description string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.CreateProblemRequest{
		ExternalId:  req.ExternalID,
		Name:        req.Name,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		OutputLimit: req.OutputLimit,
		Difficulty:  req.Difficulty,
		Points:      req.Points,
	}

	resp, err := h.client.CreateProblem(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If description is provided, set the problem statement
	if req.Description != "" && resp.Id != "" {
		_, err = h.client.SetProblemStatement(ctx, &pb.SetProblemStatementRequest{
			ProblemId: resp.Id,
			Language:  "en",
			Format:    "markdown",
			Content:   req.Description,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"id": resp.Id})
}

func (h *ProblemHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req struct {
		Name        string  `json:"name"`
		TimeLimit   float64 `json:"time_limit"`
		MemoryLimit int32   `json:"memory_limit"`
		OutputLimit int32   `json:"output_limit"`
		Difficulty  string  `json:"difficulty"`
		Points      int32   `json:"points"`
		IsPublished bool    `json:"is_published"`
		AllowSubmit bool    `json:"allow_submit"`
		Description string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.UpdateProblemRequest{
		Id:          id,
		Name:        req.Name,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		IsPublished: req.IsPublished,
		AllowSubmit: req.AllowSubmit,
	}

	resp, err := h.client.UpdateProblem(ctx, grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If description is provided, update the problem statement
	if req.Description != "" {
		_, err = h.client.SetProblemStatement(ctx, &pb.SetProblemStatementRequest{
			ProblemId: id,
			Language:  "en",
			Format:    "markdown",
			Content:   req.Description,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Invalidate problem cache
	_ = h.cache.InvalidateProblemCache(ctx, id)

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	_, err := h.client.DeleteProblem(ctx, &pb.DeleteProblemRequest{Id: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate problem cache
	_ = h.cache.InvalidateProblemCache(ctx, id)

	w.WriteHeader(http.StatusNoContent)
}

// GetProblemStatement returns the problem statement content (markdown)
func (h *ProblemHandler) GetProblemStatement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	// Get language from query param, default to "en"
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	resp, err := h.client.GetProblemStatement(ctx, &pb.GetProblemStatementRequest{
		ProblemId: problemID,
		Language:  language,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the full statement object with format information
	w.Header().Set("Content-Type", "application/json")
	if resp.Statement == nil {
		_ = json.NewEncoder(w).Encode(nil)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{
		"format":  resp.Statement.Format,
		"content": resp.Statement.Content,
	})
}

// SetProblemStatement updates the problem statement content (markdown)
func (h *ProblemHandler) SetProblemStatement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	var req struct {
		Language string `json:"language"`
		Format   string `json:"format"`
		Title    string `json:"title"`
		Content  string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default format to markdown
	if req.Format == "" {
		req.Format = "markdown"
	}

	// Default language to "en"
	if req.Language == "" {
		req.Language = "en"
	}

	resp, err := h.client.SetProblemStatement(ctx, &pb.SetProblemStatementRequest{
		ProblemId: problemID,
		Language:  req.Language,
		Format:    req.Format,
		Title:     req.Title,
		Content:   req.Content,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate problem cache
	_ = h.cache.InvalidateProblemCache(ctx, problemID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp.Statement)
}

// RegisterRoutes registers problem routes
func (h *ProblemHandler) RegisterRoutes(r chi.Router) {
	r.Get("/problems", h.ListProblems)
	r.Get("/problems/{id}", h.GetProblem)
	r.Get("/problems/{id}/statement", h.GetProblemStatement)
	r.Get("/languages", h.ListLanguages)

	// Admin routes for problem CRUD
	r.Post("/problems", h.CreateProblem)
	r.Put("/problems/{id}", h.UpdateProblem)
	r.Delete("/problems/{id}", h.DeleteProblem)
	r.Put("/problems/{id}/statement", h.SetProblemStatement)

	// Test case management
	r.Get("/problems/{id}/testcases", h.ListTestCases)
	r.Post("/problems/{id}/testcases", h.CreateTestCase)
	r.Post("/problems/{id}/testcases/batch", h.BatchUploadTestCases)
	r.Put("/testcases/{id}", h.UpdateTestCase)
	r.Delete("/testcases/{id}", h.DeleteTestCase)
	r.Put("/testcases/{id}/toggle-sample", h.ToggleTestCaseSample)
}

// ListTestCases returns all test cases for a problem
func (h *ProblemHandler) ListTestCases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	resp, err := h.client.ListTestCases(ctx, &pb.ListTestCasesRequest{
		ProblemId: problemID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// CreateTestCase creates a new test case for a problem
func (h *ProblemHandler) CreateTestCase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	var req struct {
		Rank         int32  `json:"rank"`
		IsSample     bool   `json:"is_sample"`
		InputContent string `json:"input_content"`
		OutputContent string `json:"output_content"`
		Description  string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.client.CreateTestCase(ctx, &pb.CreateTestCaseRequest{
		ProblemId:   problemID,
		Rank:        req.Rank,
		IsSample:    req.IsSample,
		Input:       req.InputContent,
		Output:      req.OutputContent,
		Description: req.Description,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateTestCase updates a test case
func (h *ProblemHandler) UpdateTestCase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	testCaseID := chi.URLParam(r, "id")

	var req struct {
		Rank        int32  `json:"rank"`
		IsSample    bool   `json:"is_sample"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.client.UpdateTestCase(ctx, &pb.UpdateTestCaseRequest{
		Id:          testCaseID,
		Rank:        req.Rank,
		IsSample:    req.IsSample,
		Description: req.Description,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// DeleteTestCase deletes a test case
func (h *ProblemHandler) DeleteTestCase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	testCaseID := chi.URLParam(r, "id")

	_, err := h.client.DeleteTestCase(ctx, &pb.DeleteTestCaseRequest{
		Id: testCaseID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ToggleTestCaseSample toggles whether a test case is a sample
func (h *ProblemHandler) ToggleTestCaseSample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	testCaseID := chi.URLParam(r, "id")

	resp, err := h.client.ToggleTestCaseSample(ctx, &pb.ToggleTestCaseSampleRequest{
		Id: testCaseID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// BatchUploadTestCases handles batch upload of test cases via multipart form data
func (h *ProblemHandler) BatchUploadTestCases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	problemID := chi.URLParam(r, "id")

	// Parse multipart form (max 64MB)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		// Fall back to JSON body
		var req struct {
			TestCases []struct {
				Rank          int32  `json:"rank"`
				IsSample      bool   `json:"is_sample"`
				InputContent  string `json:"input_content"`
				OutputContent string `json:"output_content"`
				Description   string `json:"description"`
			} `json:"test_cases"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var testCases []*pb.TestCaseData
		for _, tc := range req.TestCases {
			testCases = append(testCases, &pb.TestCaseData{
				Rank:          tc.Rank,
				IsSample:      tc.IsSample,
				InputContent:  tc.InputContent,
				OutputContent: tc.OutputContent,
				Description:   tc.Description,
			})
		}

		resp, err := h.client.BatchUploadTestCases(ctx, &pb.BatchUploadTestCasesRequest{
			ProblemId: problemID,
			TestCases: testCases,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	isSample := r.FormValue("is_sample") == "true"

	var testCases []*pb.TestCaseData

	// Check for zip file upload
	zipFile, _, err := r.FormFile("zip_file")
	if err == nil {
		defer func() { _ = zipFile.Close() }()
		zipData, err := io.ReadAll(zipFile)
		if err != nil {
			http.Error(w, "failed to read zip file", http.StatusBadRequest)
			return
		}

		zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
		if err != nil {
			http.Error(w, "invalid zip file", http.StatusBadRequest)
			return
		}

		// Extract files and pair by name (e.g., 1.in/1.out or 1.in/1.ans)
		inputs := make(map[string]string)
		outputs := make(map[string]string)
		for _, f := range zipReader.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(rc)
			if closeErr := rc.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				continue
			}

			name := filepath.Base(f.Name)
			ext := filepath.Ext(name)
			base := strings.TrimSuffix(name, ext)

			switch ext {
			case ".in":
				inputs[base] = string(content)
			case ".out", ".ans":
				outputs[base] = string(content)
			}
		}

		// Pair inputs and outputs
		var rank int32 = 1
		// Sort keys for deterministic ordering
		keys := make([]string, 0, len(inputs))
		for k := range inputs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			tc := &pb.TestCaseData{
				Rank:         rank,
				IsSample:     isSample,
				InputContent: inputs[key],
			}
			if out, ok := outputs[key]; ok {
				tc.OutputContent = out
			}
			testCases = append(testCases, tc)
			rank++
		}
	} else {
		// Handle individual file uploads
		inputFiles := r.MultipartForm.File["input_files"]
		outputFiles := r.MultipartForm.File["output_files"]

		for i, fh := range inputFiles {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			inputContent, err := io.ReadAll(f)
			if closeErr := f.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				continue
			}

			tc := &pb.TestCaseData{
				Rank:         int32(i + 1),
				IsSample:     isSample,
				InputContent: string(inputContent),
			}

			if i < len(outputFiles) {
				of, err := outputFiles[i].Open()
				if err == nil {
					outputContent, err := io.ReadAll(of)
					if closeErr := of.Close(); closeErr != nil && err == nil {
						err = closeErr
					}
					if err == nil {
						tc.OutputContent = string(outputContent)
					}
				}
			}

			testCases = append(testCases, tc)
		}
	}

	resp, err2 := h.client.BatchUploadTestCases(ctx, &pb.BatchUploadTestCasesRequest{
		ProblemId: problemID,
		TestCases: testCases,
	})
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
