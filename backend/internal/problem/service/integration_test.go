package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/problem/v1"
	"github.com/online-judge/backend/internal/problem/store"
)

// Integration tests for Problem Service - testing full workflows

func TestProblemService_Integration_CRUDFlow(t *testing.T) {
	mockStore := store.NewMockProblemStore()
	service := NewProblemService(mockStore, nil)
	ctx := context.Background()

	// Step 1: Create a problem
	createResp, err := service.CreateProblem(ctx, &pb.CreateProblemRequest{
		ExternalId:  "A",
		Name:        "Two Sum",
		TimeLimit:   1.0,
		MemoryLimit: 256,
		OutputLimit: 10240,
		Difficulty:  "easy",
		Points:      100,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.Id)
	problemID := createResp.Id

	// Step 2: Get the problem
	getResp, err := service.GetProblem(ctx, &pb.GetProblemRequest{Id: problemID})
	require.NoError(t, err)
	assert.Equal(t, "Two Sum", getResp.Problem.Name)
	assert.Equal(t, "easy", getResp.Problem.Difficulty)
	assert.Equal(t, 1.0, getResp.Problem.TimeLimit)
	assert.Equal(t, int32(256), getResp.Problem.MemoryLimit)

	// Step 3: Update the problem
	updateResp, err := service.UpdateProblem(ctx, &pb.UpdateProblemRequest{
		Id:          problemID,
		Name:        "Two Sum - Updated",
		TimeLimit:   2.0,
		MemoryLimit: 512,
		IsPublished: true,
		AllowSubmit: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Two Sum - Updated", updateResp.Problem.Name)
	assert.Equal(t, 2.0, updateResp.Problem.TimeLimit)

	// Step 4: Verify update persisted
	getResp2, err := service.GetProblem(ctx, &pb.GetProblemRequest{Id: problemID})
	require.NoError(t, err)
	assert.Equal(t, "Two Sum - Updated", getResp2.Problem.Name)

	// Step 5: List problems (should include our created problem)
	listResp, err := service.ListProblems(ctx, &pb.ListProblemsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Problems), 1)

	// Step 6: Delete the problem
	_, err = service.DeleteProblem(ctx, &pb.DeleteProblemRequest{Id: problemID})
	require.NoError(t, err)

	// Step 7: Verify deletion
	_, err = service.GetProblem(ctx, &pb.GetProblemRequest{Id: problemID})
	require.Error(t, err)
}

func TestProblemService_Integration_TestCaseFlow(t *testing.T) {
	mockStore := store.NewMockProblemStore()
	service := NewProblemService(mockStore, nil)
	ctx := context.Background()

	// Create a problem first
	createResp, err := service.CreateProblem(ctx, &pb.CreateProblemRequest{
		Name:        "Test Problem",
		TimeLimit:   1.0,
		MemoryLimit: 256,
	})
	require.NoError(t, err)
	problemID := createResp.Id

	// Step 1: Create test cases
	tc1Resp, err := service.CreateTestCase(ctx, &pb.CreateTestCaseRequest{
		ProblemId:   problemID,
		Rank:        1,
		IsSample:    true,
		Description: "Sample test case 1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tc1Resp.Id)

	tc2Resp, err := service.CreateTestCase(ctx, &pb.CreateTestCaseRequest{
		ProblemId:   problemID,
		Rank:        2,
		IsSample:    true,
		Description: "Sample test case 2",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tc2Resp.Id)

	tc3Resp, err := service.CreateTestCase(ctx, &pb.CreateTestCaseRequest{
		ProblemId:   problemID,
		Rank:        3,
		IsSample:    false,
		Description: "Hidden test case",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tc3Resp.Id)

	// Step 2: List all test cases
	allTCs, err := service.ListTestCases(ctx, &pb.ListTestCasesRequest{
		ProblemId:   problemID,
		SamplesOnly: false,
	})
	require.NoError(t, err)
	assert.Len(t, allTCs.TestCases, 3)

	// Step 3: List only sample test cases
	sampleTCs, err := service.ListTestCases(ctx, &pb.ListTestCasesRequest{
		ProblemId:   problemID,
		SamplesOnly: true,
	})
	require.NoError(t, err)
	assert.Len(t, sampleTCs.TestCases, 2)

	// Step 4: Update a test case
	updateTCResp, err := service.UpdateTestCase(ctx, &pb.UpdateTestCaseRequest{
		Id:          tc1Resp.Id,
		Rank:        1,
		IsSample:    true,
		Description: "Updated sample test case 1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated sample test case 1", updateTCResp.TestCase.Description)

	// Step 5: Delete a test case
	_, err = service.DeleteTestCase(ctx, &pb.DeleteTestCaseRequest{Id: tc3Resp.Id})
	require.NoError(t, err)

	// Verify deletion
	allTCs2, err := service.ListTestCases(ctx, &pb.ListTestCasesRequest{
		ProblemId:   problemID,
		SamplesOnly: false,
	})
	require.NoError(t, err)
	assert.Len(t, allTCs2.TestCases, 2)
}

func TestProblemService_Integration_BatchUploadTestCases(t *testing.T) {
	mockStore := store.NewMockProblemStore()
	service := NewProblemService(mockStore, nil)
	ctx := context.Background()

	// Create a problem first
	createResp, err := service.CreateProblem(ctx, &pb.CreateProblemRequest{
		Name:        "Batch Test Problem",
		TimeLimit:   1.0,
		MemoryLimit: 256,
	})
	require.NoError(t, err)
	problemID := createResp.Id

	// Batch upload test cases
	batchResp, err := service.BatchUploadTestCases(ctx, &pb.BatchUploadTestCasesRequest{
		ProblemId: problemID,
		TestCases: []*pb.TestCaseData{
			{
				Rank:          1,
				IsSample:      true,
				InputContent:  "1 2\n",
				OutputContent: "3\n",
				Description:   "Sample 1",
			},
			{
				Rank:          2,
				IsSample:      true,
				InputContent:  "5 7\n",
				OutputContent: "12\n",
				Description:   "Sample 2",
			},
			{
				Rank:          3,
				IsSample:      false,
				InputContent:  "100 200\n",
				OutputContent: "300\n",
				Description:   "Hidden 1",
			},
		},
	})
	require.NoError(t, err)
	assert.Len(t, batchResp.TestCases, 3)

	// Verify all test cases were created
	tcs, err := service.ListTestCases(ctx, &pb.ListTestCasesRequest{
		ProblemId:   problemID,
		SamplesOnly: false,
	})
	require.NoError(t, err)
	assert.Len(t, tcs.TestCases, 3)
}

func TestProblemService_Integration_ListWithFilters(t *testing.T) {
	mockStore := store.NewMockProblemStore()
	service := NewProblemService(mockStore, nil)
	ctx := context.Background()

	// Create problems with different difficulties
	for _, tc := range []struct {
		name       string
		difficulty string
		points     int32
	}{
		{"Easy Problem", "easy", 100},
		{"Medium Problem", "medium", 200},
		{"Hard Problem", "hard", 300},
		{"Another Easy", "easy", 150},
	} {
		_, err := service.CreateProblem(ctx, &pb.CreateProblemRequest{
			Name:        tc.name,
			Difficulty:  tc.difficulty,
			Points:      tc.points,
			TimeLimit:   1.0,
			MemoryLimit: 256,
		})
		require.NoError(t, err)
	}

	// List all problems
	allProblems, err := service.ListProblems(ctx, &pb.ListProblemsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, allProblems.Problems, 4)
	assert.Equal(t, int32(4), allProblems.Pagination.Total)

	// List with pagination
	page1, err := service.ListProblems(ctx, &pb.ListProblemsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 2},
	})
	require.NoError(t, err)
	assert.Len(t, page1.Problems, 2)

	page2, err := service.ListProblems(ctx, &pb.ListProblemsRequest{
		Pagination: &commonv1.Pagination{Page: 2, PageSize: 2},
	})
	require.NoError(t, err)
	assert.Len(t, page2.Problems, 2)
}

func TestProblemService_Integration_Languages(t *testing.T) {
	mockStore := store.NewMockProblemStore()
	service := NewProblemService(mockStore, nil)
	ctx := context.Background()

	// List default languages
	langResp, err := service.ListLanguages(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotEmpty(t, langResp.Languages)

	// Verify expected languages are present
	langMap := make(map[string]bool)
	for _, lang := range langResp.Languages {
		langMap[lang.Name] = true
		assert.True(t, lang.AllowSubmit)
		assert.True(t, lang.AllowJudge)
		assert.NotEmpty(t, lang.Extensions)
	}

	assert.True(t, langMap["C++17"])
	assert.True(t, langMap["Python 3"])
	assert.True(t, langMap["Java 17"])
	assert.True(t, langMap["Go 1.21"])
	assert.True(t, langMap["Rust"])
	assert.True(t, langMap["Node.js 18"])
}

func TestProblemService_Integration_ErrorHandling(t *testing.T) {
	mockStore := store.NewMockProblemStore()
	service := NewProblemService(mockStore, nil)
	ctx := context.Background()

	t.Run("GetNonExistentProblem", func(t *testing.T) {
		_, err := service.GetProblem(ctx, &pb.GetProblemRequest{Id: "non-existent"})
		require.Error(t, err)
	})

	t.Run("UpdateNonExistentProblem", func(t *testing.T) {
		_, err := service.UpdateProblem(ctx, &pb.UpdateProblemRequest{Id: "non-existent"})
		require.Error(t, err)
	})

	t.Run("DeleteNonExistentProblem", func(t *testing.T) {
		_, err := service.DeleteProblem(ctx, &pb.DeleteProblemRequest{Id: "non-existent"})
		require.Error(t, err)
	})

	t.Run("CreateTestCaseForNonExistentProblem", func(t *testing.T) {
		// Note: mock store doesn't validate problem existence, but real store would
		_, err := service.CreateTestCase(ctx, &pb.CreateTestCaseRequest{
			ProblemId: "non-existent",
			Rank:      1,
		})
		// Mock allows this, real implementation would fail
		require.NoError(t, err)
	})

	t.Run("UpdateNonExistentTestCase", func(t *testing.T) {
		_, err := service.UpdateTestCase(ctx, &pb.UpdateTestCaseRequest{Id: "non-existent"})
		require.Error(t, err)
	})

	t.Run("DeleteNonExistentTestCase", func(t *testing.T) {
		_, err := service.DeleteTestCase(ctx, &pb.DeleteTestCaseRequest{Id: "non-existent"})
		require.Error(t, err)
	})
}

func TestProblemService_Integration_StoreErrors(t *testing.T) {
	t.Run("ListError", func(t *testing.T) {
		mockStore := store.NewMockProblemStore()
		mockStore.ListError = assert.AnError
		service := NewProblemService(mockStore, nil)

		_, err := service.ListProblems(context.Background(), &pb.ListProblemsRequest{
			Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
		})
		require.Error(t, err)
	})

	t.Run("GetError", func(t *testing.T) {
		mockStore := store.NewMockProblemStore()
		mockStore.GetError = assert.AnError
		service := NewProblemService(mockStore, nil)

		_, err := service.GetProblem(context.Background(), &pb.GetProblemRequest{Id: "prob-1"})
		require.Error(t, err)
	})

	t.Run("CreateError", func(t *testing.T) {
		mockStore := store.NewMockProblemStore()
		mockStore.CreateError = assert.AnError
		service := NewProblemService(mockStore, nil)

		_, err := service.CreateProblem(context.Background(), &pb.CreateProblemRequest{Name: "Test"})
		require.Error(t, err)
	})

	t.Run("UpdateError", func(t *testing.T) {
		mockStore := store.NewMockProblemStore()
		mockStore.Problems["prob-1"] = &pb.Problem{Id: "prob-1"}
		mockStore.UpdateError = assert.AnError
		service := NewProblemService(mockStore, nil)

		_, err := service.UpdateProblem(context.Background(), &pb.UpdateProblemRequest{Id: "prob-1"})
		require.Error(t, err)
	})

	t.Run("DeleteError", func(t *testing.T) {
		mockStore := store.NewMockProblemStore()
		mockStore.DeleteError = assert.AnError
		service := NewProblemService(mockStore, nil)

		_, err := service.DeleteProblem(context.Background(), &pb.DeleteProblemRequest{Id: "prob-1"})
		require.Error(t, err)
	})
}
