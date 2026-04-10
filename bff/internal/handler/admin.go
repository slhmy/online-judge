package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	commonv1 "github.com/online-judge/gen/go/common/v1"
	pb "github.com/online-judge/gen/go/judge/v1"
	"github.com/online-judge/bff/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type AdminHandler struct {
	db             *sql.DB
	rejudgeService RejudgeClient
}

// RejudgeClient is the subset of the generated JudgeServiceClient used for rejudge operations.
// The generated pb.JudgeServiceClient satisfies this interface.
type RejudgeClient interface {
	CreateRejudge(ctx context.Context, in *pb.CreateRejudgeRequest, opts ...grpc.CallOption) (*pb.CreateRejudgeResponse, error)
	GetRejudge(ctx context.Context, in *pb.GetRejudgeRequest, opts ...grpc.CallOption) (*pb.GetRejudgeResponse, error)
	ListRejudges(ctx context.Context, in *pb.ListRejudgesRequest, opts ...grpc.CallOption) (*pb.ListRejudgesResponse, error)
	CancelRejudge(ctx context.Context, in *pb.CancelRejudgeRequest, opts ...grpc.CallOption) (*pb.CancelRejudgeResponse, error)
	ApplyRejudge(ctx context.Context, in *pb.ApplyRejudgeRequest, opts ...grpc.CallOption) (*pb.ApplyRejudgeResponse, error)
	RevertRejudge(ctx context.Context, in *pb.RevertRejudgeRequest, opts ...grpc.CallOption) (*pb.RevertRejudgeResponse, error)
	GetRejudgeSubmissions(ctx context.Context, in *pb.GetRejudgeSubmissionsRequest, opts ...grpc.CallOption) (*pb.GetRejudgeSubmissionsResponse, error)
}

func NewAdminHandler(databaseURL string, rejudgeService RejudgeClient) *AdminHandler {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		panic(err)
	}

	return &AdminHandler{
		db:             db,
		rejudgeService: rejudgeService,
	}
}

// ListUsers returns all users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT user_id, username, role, rating, solved_count, submission_count, created_at
		FROM user_profiles
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	type User struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		Role            string `json:"role"`
		Rating          int    `json:"rating"`
		SolvedCount     int    `json:"solved_count"`
		SubmissionCount int    `json:"submission_count"`
		CreatedAt       string `json:"created_at"`
	}

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Rating, &u.SolvedCount, &u.SubmissionCount, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
	})
}

// UpdateUserRole updates a user's role
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Role != "user" && req.Role != "admin" {
		http.Error(w, `{"error": "invalid role"}`, http.StatusBadRequest)
		return
	}

	// Check if current user is admin
	currentRole := middleware.GetUserRole(r.Context())
	if currentRole != "admin" {
		http.Error(w, `{"error": "forbidden"}`, http.StatusForbidden)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		UPDATE user_profiles SET role = $1, updated_at = NOW() WHERE user_id = $2
	`, req.Role, userID)
	if err != nil {
		http.Error(w, `{"error": "failed to update role"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// protoJSON is the shared protojson marshaler for writing gRPC responses as JSON.
var protoJSON = protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}

// CreateRejudge creates a new rejudging operation
func (h *AdminHandler) CreateRejudge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubmissionIDs []string `json:"submission_ids"`
		ContestID     string   `json:"contest_id"`
		ProblemID     string   `json:"problem_id"`
		FromVerdict   string   `json:"from_verdict"`
		Reason        string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error": "invalid request"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.rejudgeService.CreateRejudge(r.Context(), &pb.CreateRejudgeRequest{
		SubmissionIds: body.SubmissionIDs,
		ContestId:     body.ContestID,
		ProblemId:     body.ProblemID,
		FromVerdict:   body.FromVerdict,
		Reason:        body.Reason,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// GetRejudge retrieves a rejudging operation
func (h *AdminHandler) GetRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.GetRejudge(r.Context(), &pb.GetRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// ListRejudges lists rejudging operations
func (h *AdminHandler) ListRejudges(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	req := &pb.ListRejudgesRequest{
		ContestId: query.Get("contest_id"),
		ProblemId: query.Get("problem_id"),
		UserId:    query.Get("user_id"),
	}

	if statusStr := query.Get("status"); statusStr != "" {
		if v, ok := pb.RejudgeStatus_value[statusStr]; ok {
			req.Status = pb.RejudgeStatus(v)
		}
	}

	pagination := &commonv1.Pagination{}
	if page := query.Get("page"); page != "" {
		if v, err := strconv.ParseInt(page, 10, 32); err == nil {
			pagination.Page = int32(v)
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if v, err := strconv.ParseInt(pageSize, 10, 32); err == nil {
			pagination.PageSize = int32(v)
		}
	}
	req.Pagination = pagination

	resp, err := h.rejudgeService.ListRejudges(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// CancelRejudge cancels a pending rejudging operation
func (h *AdminHandler) CancelRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.CancelRejudge(r.Context(), &pb.CancelRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// ApplyRejudge applies the rejudge results
func (h *AdminHandler) ApplyRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.ApplyRejudge(r.Context(), &pb.ApplyRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// RevertRejudge reverts the rejudge results
func (h *AdminHandler) RevertRejudge(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "id")

	resp, err := h.rejudgeService.RevertRejudge(r.Context(), &pb.RevertRejudgeRequest{Id: rejudgeID})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// GetRejudgeSubmissions retrieves submissions for a rejudging operation
func (h *AdminHandler) GetRejudgeSubmissions(w http.ResponseWriter, r *http.Request) {
	rejudgeID := chi.URLParam(r, "rejudge_id")
	query := r.URL.Query()

	req := &pb.GetRejudgeSubmissionsRequest{
		RejudgeId:   rejudgeID,
		OnlyChanged: query.Get("only_changed") == "true",
	}

	if statusStr := query.Get("status"); statusStr != "" {
		if v, ok := pb.RejudgeSubmissionStatus_value[statusStr]; ok {
			req.Status = pb.RejudgeSubmissionStatus(v)
		}
	}

	pagination := &commonv1.Pagination{}
	if page := query.Get("page"); page != "" {
		if v, err := strconv.ParseInt(page, 10, 32); err == nil {
			pagination.Page = int32(v)
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if v, err := strconv.ParseInt(pageSize, 10, 32); err == nil {
			pagination.PageSize = int32(v)
		}
	}
	req.Pagination = pagination

	resp, err := h.rejudgeService.GetRejudgeSubmissions(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	data, _ := protoJSON.Marshal(resp)
	_, _ = w.Write(data)
}

// ListTestCases returns all test cases for a problem
func (h *AdminHandler) ListTestCases(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemId")

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, problem_id, rank, is_sample, input_path, output_path, description, is_interactive, input_content, output_content
		FROM test_cases
		WHERE problem_id = $1
		ORDER BY rank
	`, problemID)
	if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	type TestCase struct {
		ID            string `json:"id"`
		ProblemID     string `json:"problem_id"`
		Rank          int    `json:"rank"`
		IsSample      bool   `json:"is_sample"`
		InputPath     string `json:"input_path"`
		OutputPath    string `json:"output_path"`
		InputContent  string `json:"input_content"`
		OutputContent string `json:"output_content"`
		Description   string `json:"description"`
		IsInteractive bool   `json:"is_interactive"`
	}

	var testCases []TestCase
	for rows.Next() {
		var tc TestCase
		var inputContent, outputContent, description sql.NullString
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Rank, &tc.IsSample, &tc.InputPath, &tc.OutputPath, &description, &tc.IsInteractive, &inputContent, &outputContent); err != nil {
			continue
		}
		if inputContent.Valid {
			tc.InputContent = inputContent.String
		}
		if outputContent.Valid {
			tc.OutputContent = outputContent.String
		}
		if description.Valid {
			tc.Description = description.String
		}
		testCases = append(testCases, tc)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"test_cases": testCases,
	})
}

// CreateTestCase creates a new test case
func (h *AdminHandler) CreateTestCase(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemId")

	var req struct {
		Rank          int    `json:"rank"`
		IsSample      bool   `json:"is_sample"`
		InputContent  string `json:"input_content"`
		OutputContent string `json:"output_content"`
		Description   string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var id string
	err := h.db.QueryRowContext(r.Context(), `
		INSERT INTO test_cases (problem_id, rank, is_sample, input_content, output_content, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, problemID, req.Rank, req.IsSample, req.InputContent, req.OutputContent, req.Description).Scan(&id)
	if err != nil {
		http.Error(w, `{"error": "failed to create test case"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// UpdateTestCase updates a test case
func (h *AdminHandler) UpdateTestCase(w http.ResponseWriter, r *http.Request) {
	testCaseID := chi.URLParam(r, "id")

	var req struct {
		Rank        int    `json:"rank"`
		IsSample    bool   `json:"is_sample"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		UPDATE test_cases
		SET rank = $2, is_sample = $3, description = $4, updated_at = NOW()
		WHERE id = $1
	`, testCaseID, req.Rank, req.IsSample, req.Description)
	if err != nil {
		http.Error(w, `{"error": "failed to update test case"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteTestCase deletes a test case
func (h *AdminHandler) DeleteTestCase(w http.ResponseWriter, r *http.Request) {
	testCaseID := chi.URLParam(r, "id")

	_, err := h.db.ExecContext(r.Context(), "DELETE FROM test_cases WHERE id = $1", testCaseID)
	if err != nil {
		http.Error(w, `{"error": "failed to delete test case"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BatchUploadTestCases handles batch upload of test cases (ZIP or separate files)
func (h *AdminHandler) BatchUploadTestCases(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemId")

	// Parse multipart form
	err := r.ParseMultipartForm(50 << 20) // 50MB max
	if err != nil {
		http.Error(w, `{"error": "failed to parse form"}`, http.StatusBadRequest)
		return
	}

	var testCases []struct {
		Rank          int
		IsSample      bool
		InputContent  string
		OutputContent string
	}

	// Check if ZIP file is uploaded
	zipFile, _, err := r.FormFile("zip_file")
	if err == nil {
		defer func() { _ = zipFile.Close() }()

		// Read ZIP file
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, zipFile); err != nil {
			http.Error(w, `{"error": "failed to read zip file"}`, http.StatusBadRequest)
			return
		}

		// Extract test cases from ZIP
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			http.Error(w, `{"error": "failed to parse zip file"}`, http.StatusBadRequest)
			return
		}

		// Find all .in and .out files
		inputFiles := make(map[int][]byte)
		outputFiles := make(map[int][]byte)

		for _, file := range zipReader.File {
			name := file.Name
			ext := strings.ToLower(filepath.Ext(name))
			baseName := strings.TrimSuffix(name, ext)

			// Extract rank number from filename (e.g., "1.in" -> 1)
			rank, err := strconv.Atoi(baseName)
			if err != nil {
				continue // Skip files with non-numeric names
			}

			reader, err := file.Open()
			if err != nil {
				continue
			}

			content, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				continue
			}

			switch ext {
			case ".in":
				inputFiles[rank] = content
			case ".out":
				outputFiles[rank] = content
			}
		}

		// Match input/output pairs
		for rank, input := range inputFiles {
			if output, ok := outputFiles[rank]; ok {
				testCases = append(testCases, struct {
					Rank          int
					IsSample      bool
					InputContent  string
					OutputContent string
				}{
					Rank:          rank,
					InputContent:  string(input),
					OutputContent: string(output),
				})
			}
		}

		// Sort by rank
		sort.Slice(testCases, func(i, j int) bool {
			return testCases[i].Rank < testCases[j].Rank
		})
	} else {
		// Handle separate file uploads
		inputFiles := r.MultipartForm.File["input_files"]
		outputFiles := r.MultipartForm.File["output_files"]

		// Parse input files
		inputContents := make(map[int]string)
		for _, fileHeader := range inputFiles {
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				continue
			}

			// Extract rank from filename
			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			baseName := strings.TrimSuffix(fileHeader.Filename, ext)
			rank, err := strconv.Atoi(baseName)
			if err != nil {
				continue
			}
			inputContents[rank] = string(content)
		}

		// Parse output files
		outputContents := make(map[int]string)
		for _, fileHeader := range outputFiles {
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				continue
			}

			// Extract rank from filename
			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			baseName := strings.TrimSuffix(fileHeader.Filename, ext)
			rank, err := strconv.Atoi(baseName)
			if err != nil {
				continue
			}
			outputContents[rank] = string(content)
		}

		// Match input/output pairs
		for rank, input := range inputContents {
			if output, ok := outputContents[rank]; ok {
				testCases = append(testCases, struct {
					Rank          int
					IsSample      bool
					InputContent  string
					OutputContent string
				}{
					Rank:          rank,
					InputContent:  input,
					OutputContent: output,
				})
			}
		}

		// Sort by rank
		sort.Slice(testCases, func(i, j int) bool {
			return testCases[i].Rank < testCases[j].Rank
		})
	}

	// Get is_sample checkbox values
	isSampleStr := r.FormValue("is_sample")
	defaultIsSample := isSampleStr == "true"

	// Get custom is_sample values per rank
	for i := range testCases {
		customIsSample := r.FormValue("is_sample_" + strconv.Itoa(testCases[i].Rank))
		switch customIsSample {
		case "true":
			testCases[i].IsSample = true
		case "false":
			testCases[i].IsSample = false
		default:
			testCases[i].IsSample = defaultIsSample
		}
	}

	// Insert test cases into database
	var createdIDs []string
	for _, tc := range testCases {
		var id string
		err := h.db.QueryRowContext(r.Context(), `
			INSERT INTO test_cases (problem_id, rank, is_sample, input_content, output_content)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, problemID, tc.Rank, tc.IsSample, tc.InputContent, tc.OutputContent).Scan(&id)
		if err != nil {
			http.Error(w, `{"error": "failed to insert test case"}`, http.StatusInternalServerError)
			return
		}
		createdIDs = append(createdIDs, id)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ids":        createdIDs,
		"count":      len(createdIDs),
		"test_cases": testCases,
	})
}

// ToggleTestCaseSample toggles the is_sample flag for a test case
func (h *AdminHandler) ToggleTestCaseSample(w http.ResponseWriter, r *http.Request) {
	testCaseID := chi.URLParam(r, "id")

	// Get current is_sample value
	var currentIsSample bool
	err := h.db.QueryRowContext(r.Context(), "SELECT is_sample FROM test_cases WHERE id = $1", testCaseID).Scan(&currentIsSample)
	if err != nil {
		http.Error(w, `{"error": "test case not found"}`, http.StatusNotFound)
		return
	}

	// Toggle the value
	_, err = h.db.ExecContext(r.Context(), "UPDATE test_cases SET is_sample = $2 WHERE id = $1", testCaseID, !currentIsSample)
	if err != nil {
		http.Error(w, `{"error": "failed to update test case"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{
		"is_sample": !currentIsSample,
	})
}

// RegisterRoutes registers admin routes
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/users", h.ListUsers)
		r.Put("/users/{id}/role", h.UpdateUserRole)

		// Rejudge routes
		r.Route("/rejudges", func(r chi.Router) {
			r.Get("/", h.ListRejudges)
			r.Post("/", h.CreateRejudge)
			r.Get("/{id}", h.GetRejudge)
			r.Delete("/{id}", h.CancelRejudge)
			r.Post("/{id}/apply", h.ApplyRejudge)
			r.Post("/{id}/revert", h.RevertRejudge)
			r.Get("/{rejudge_id}/submissions", h.GetRejudgeSubmissions)
		})
	})

	// Test case admin routes
	r.Route("/problems/{problemId}/testcases", func(r chi.Router) {
		r.Get("/", h.ListTestCases)
		r.Post("/", h.CreateTestCase)
		r.Post("/batch", h.BatchUploadTestCases)
	})
	r.Put("/testcases/{id}", h.UpdateTestCase)
	r.Delete("/testcases/{id}", h.DeleteTestCase)
	r.Put("/testcases/{id}/toggle-sample", h.ToggleTestCaseSample)
}
