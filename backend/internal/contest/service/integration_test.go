package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/online-judge/gen/go/common/v1"
	pb "github.com/online-judge/gen/go/contest/v1"
	"github.com/online-judge/backend/internal/contest/store"
	"github.com/online-judge/backend/internal/pkg/middleware"
)

func TestContestService_Integration_CRUDFlow(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	// Step 1: Create contest
	createResp, err := service.CreateContest(ctx, &pb.CreateContestRequest{
		ExternalId: "ICPC-2024",
		Name:       "ICPC Regional 2024",
		ShortName:  "ICPC24",
		StartTime:  "2024-03-01T10:00:00Z",
		EndTime:    "2024-03-01T15:00:00Z",
		FreezeTime: "2024-03-01T14:00:00Z",
		Public:     true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.Id)
	contestID := createResp.Id

	// Step 2: Get contest
	getResp, err := service.GetContest(ctx, &pb.GetContestRequest{Id: contestID})
	require.NoError(t, err)
	assert.Equal(t, "ICPC Regional 2024", getResp.Contest.Name)
	assert.Equal(t, "ICPC24", getResp.Contest.ShortName)

	// Step 3: Add contest problems
	mockStore.ContestProblems[contestID] = []*pb.ContestProblem{
		{
			ProblemId:   "prob-1",
			ShortName:   "A",
			Rank:        1,
			Points:      100,
			AllowSubmit: true,
		},
		{
			ProblemId:   "prob-2",
			ShortName:   "B",
			Rank:        2,
			Points:      200,
			AllowSubmit: true,
		},
	}

	// Step 4: Get contest problems
	problemsResp, err := service.GetContestProblems(ctx, &pb.GetContestProblemsRequest{ContestId: contestID})
	require.NoError(t, err)
	assert.Len(t, problemsResp.Problems, 2)

	// Step 5: List contests
	listResp, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Contests), 1)
}

func TestContestService_Integration_Registration(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Create a contest
	_, err := service.CreateContest(ctx, &pb.CreateContestRequest{
		Name:      "Test Contest",
		ShortName: "TC",
		Public:    true,
	})
	require.NoError(t, err)

	// Register for contest
	registerResp, err := service.RegisterContest(ctx, &pb.RegisterContestRequest{
		ContestId:   "contest-1",
		TeamName:    "Team Alpha",
		Affiliation: "University A",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, registerResp.TeamId)

	// Register without team name
	registerResp2, err := service.RegisterContest(ctx, &pb.RegisterContestRequest{
		ContestId:   "contest-1",
		Affiliation: "University B",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, registerResp2.TeamId)
}

func TestContestService_Integration_Scoreboard(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	// Setup contest and teams
	mockStore.Contests["contest-1"] = &pb.Contest{
		Id:   "contest-1",
		Name: "Test Contest",
	}
	mockStore.Teams["team-1"] = &pb.Team{
		Id:          "team-1",
		Name:        "Team Alpha",
		DisplayName: "Team Alpha",
		Affiliation: "University A",
		Points:      3,
		TotalTime:   300,
	}
	mockStore.Teams["team-2"] = &pb.Team{
		Id:          "team-2",
		Name:        "Team Beta",
		DisplayName: "Team Beta",
		Affiliation: "University B",
		Points:      2,
		TotalTime:   250,
	}
	mockStore.Teams["team-3"] = &pb.Team{
		Id:          "team-3",
		Name:        "Team Gamma",
		DisplayName: "Team Gamma",
		Affiliation: "University C",
		Points:      3,
		TotalTime:   350,
	}

	// Get scoreboard
	scoreboardResp, err := service.GetScoreboard(ctx, &pb.GetScoreboardRequest{
		ContestId: "contest-1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, scoreboardResp.Entries)
	assert.Equal(t, "01:00:00", scoreboardResp.ContestTime)
	assert.False(t, scoreboardResp.IsFrozen)

	// Verify teams are in correct order
	assert.GreaterOrEqual(t, len(scoreboardResp.Entries), 2)
}

func TestContestService_Integration_PublicPrivateContests(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	// Create public and private contests
	mockStore.Contests["public-1"] = &pb.Contest{
		Id:     "public-1",
		Name:   "Public Contest 1",
		Public: true,
	}
	mockStore.Contests["public-2"] = &pb.Contest{
		Id:     "public-2",
		Name:   "Public Contest 2",
		Public: true,
	}
	mockStore.Contests["private-1"] = &pb.Contest{
		Id:     "private-1",
		Name:   "Private Contest",
		Public: false,
	}

	// List all contests
	allResp, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, allResp.Contests, 3)

	// List only public contests
	publicResp, err := service.ListContests(ctx, &pb.ListContestsRequest{
		PublicOnly: true,
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, publicResp.Contests, 2)
	for _, c := range publicResp.Contests {
		assert.True(t, c.Public)
	}
}

func TestContestService_Integration_Pagination(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	// Create many contests
	for i := 0; i < 25; i++ {
		id := string(rune('a' + i))
		mockStore.Contests[id] = &pb.Contest{
			Id:     id,
			Name:   "Contest",
			Public: true,
		}
	}

	// Page 1
	page1, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, page1.Contests, 10)
	assert.Equal(t, int32(25), page1.Pagination.Total)

	// Page 2
	page2, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 2, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, page2.Contests, 10)

	// Page 3
	page3, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 3, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, page3.Contests, 5)

	// Beyond total
	page4, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 4, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Len(t, page4.Contests, 0)
}

func TestContestService_Integration_ErrorHandling(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	t.Run("GetNonExistentContest", func(t *testing.T) {
		_, err := service.GetContest(ctx, &pb.GetContestRequest{Id: "non-existent"})
		require.Error(t, err)
	})

	t.Run("RegisterUnauthorized", func(t *testing.T) {
		// No user_id in context
		_, err := service.RegisterContest(context.Background(), &pb.RegisterContestRequest{
			ContestId: "contest-1",
		})
		require.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("ListStoreError", func(t *testing.T) {
		errorStore := store.NewMockContestStore()
		errorStore.ListError = assert.AnError
		errorService := NewContestService(errorStore, nil)

		_, err := errorService.ListContests(ctx, &pb.ListContestsRequest{
			Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
		})
		require.Error(t, err)
	})

	t.Run("GetStoreError", func(t *testing.T) {
		errorStore := store.NewMockContestStore()
		errorStore.GetError = assert.AnError
		errorService := NewContestService(errorStore, nil)

		_, err := errorService.GetContest(ctx, &pb.GetContestRequest{Id: "contest-1"})
		require.Error(t, err)
	})

	t.Run("CreateStoreError", func(t *testing.T) {
		errorStore := store.NewMockContestStore()
		errorStore.CreateError = assert.AnError
		errorService := NewContestService(errorStore, nil)

		_, err := errorService.CreateContest(ctx, &pb.CreateContestRequest{Name: "Test"})
		require.Error(t, err)
	})

	t.Run("RegisterStoreError", func(t *testing.T) {
		errorStore := store.NewMockContestStore()
		errorStore.RegisterError = assert.AnError
		errorService := NewContestService(errorStore, nil)
		ctxWithUser := middleware.ContextWithUserID(context.Background(), "user-1")

		_, err := errorService.RegisterContest(ctxWithUser, &pb.RegisterContestRequest{
			ContestId: "contest-1",
		})
		require.Error(t, err)
	})
}

func TestContestService_Integration_DefaultPagination(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	// Create contests
	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		mockStore.Contests[id] = &pb.Contest{
			Id:     id,
			Name:   "Contest",
			Public: true,
		}
	}

	// Request with no pagination (should use defaults)
	resp, err := service.ListContests(ctx, &pb.ListContestsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Contests, 5)
	assert.Equal(t, int32(1), resp.Pagination.Page)
	assert.Equal(t, int32(20), resp.Pagination.PageSize) // default
}

func TestContestService_Integration_ContestWithProblems(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := context.Background()

	// Setup contest with problems
	mockStore.Contests["contest-1"] = &pb.Contest{
		Id:        "contest-1",
		Name:      "Algorithm Contest",
		ShortName: "AC",
		Public:    true,
	}
	mockStore.ContestProblems["contest-1"] = []*pb.ContestProblem{
		{
			ProblemId:   "prob-1",
			ShortName:   "A",
			Rank:        1,
			Color:       "red",
			Points:      100,
			AllowSubmit: true,
		},
		{
			ProblemId:   "prob-2",
			ShortName:   "B",
			Rank:        2,
			Color:       "blue",
			Points:      200,
			AllowSubmit: true,
		},
		{
			ProblemId:   "prob-3",
			ShortName:   "C",
			Rank:        3,
			Color:       "green",
			Points:      300,
			AllowSubmit: false,
		},
	}

	// Get contest with problems
	resp, err := service.GetContest(ctx, &pb.GetContestRequest{Id: "contest-1"})
	require.NoError(t, err)
	assert.Equal(t, "contest-1", resp.Contest.Id)
	assert.Len(t, resp.Problems, 3)

	// Get problems separately
	problemsResp, err := service.GetContestProblems(ctx, &pb.GetContestProblemsRequest{ContestId: "contest-1"})
	require.NoError(t, err)
	assert.Len(t, problemsResp.Problems, 3)

	// Verify problem order
	assert.Equal(t, "A", problemsResp.Problems[0].ShortName)
	assert.Equal(t, "B", problemsResp.Problems[1].ShortName)
	assert.Equal(t, "C", problemsResp.Problems[2].ShortName)
}

// Full integration test
func TestContestService_Integration_FullFlow(t *testing.T) {
	mockStore := store.NewMockContestStore()
	service := NewContestService(mockStore, nil)
	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Step 1: Create contest
	createResp, err := service.CreateContest(ctx, &pb.CreateContestRequest{
		ExternalId: "FINAL-2024",
		Name:       "Final Contest 2024",
		ShortName:  "FINAL24",
		StartTime:  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		EndTime:    time.Now().Add(30 * time.Hour).Format(time.RFC3339),
		Public:     true,
	})
	require.NoError(t, err)
	contestID := createResp.Id

	// Step 2: Get contest
	getResp, err := service.GetContest(ctx, &pb.GetContestRequest{Id: contestID})
	require.NoError(t, err)
	assert.Equal(t, "Final Contest 2024", getResp.Contest.Name)

	// Step 3: Add problems
	mockStore.ContestProblems[contestID] = []*pb.ContestProblem{
		{ProblemId: "prob-1", ShortName: "A", Points: 100, AllowSubmit: true},
		{ProblemId: "prob-2", ShortName: "B", Points: 200, AllowSubmit: true},
	}

	// Step 4: Register teams
	_, err = service.RegisterContest(ctx, &pb.RegisterContestRequest{
		ContestId:   contestID,
		TeamName:    "Team Alpha",
		Affiliation: "University A",
	})
	require.NoError(t, err)

	_, err = service.RegisterContest(ctx, &pb.RegisterContestRequest{
		ContestId:   contestID,
		TeamName:    "Team Beta",
		Affiliation: "University B",
	})
	require.NoError(t, err)

	// Step 5: Update scores
	mockStore.Teams["team-1"] = &pb.Team{
		Id:          "team-1",
		DisplayName: "Team Alpha",
		Points:      2,
		TotalTime:   200,
	}
	mockStore.Teams["team-2"] = &pb.Team{
		Id:          "team-2",
		DisplayName: "Team Beta",
		Points:      1,
		TotalTime:   150,
	}

	// Step 6: Get scoreboard
	scoreboard, err := service.GetScoreboard(ctx, &pb.GetScoreboardRequest{ContestId: contestID})
	require.NoError(t, err)
	assert.NotEmpty(t, scoreboard.Entries)

	// Step 7: List contests
	listResp, err := service.ListContests(ctx, &pb.ListContestsRequest{
		Pagination: &commonv1.Pagination{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Contests), 1)
}
