package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/contest/v1"
	"github.com/online-judge/backend/internal/contest/store"
)

func TestContestService_ListContests(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockContestStore)
		request *pb.ListContestsRequest
		want    func(t *testing.T, resp *pb.ListContestsResponse, err error)
	}{
		{
			name: "list all contests",
			setup: func(m *store.MockContestStore) {
				m.Contests["contest-1"] = &pb.Contest{
					Id:        "contest-1",
					Name:      "Contest 1",
					ShortName: "C1",
					Public:    true,
				}
				m.Contests["contest-2"] = &pb.Contest{
					Id:        "contest-2",
					Name:      "Contest 2",
					ShortName: "C2",
					Public:    false,
				}
			},
			request: &pb.ListContestsRequest{
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListContestsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Contests, 2)
				assert.Equal(t, int32(2), resp.Pagination.Total)
			},
		},
		{
			name: "list only public contests",
			setup: func(m *store.MockContestStore) {
				m.Contests["contest-1"] = &pb.Contest{
					Id:     "contest-1",
					Name:   "Public Contest",
					Public: true,
				}
				m.Contests["contest-2"] = &pb.Contest{
					Id:     "contest-2",
					Name:   "Private Contest",
					Public: false,
				}
			},
			request: &pb.ListContestsRequest{
				PublicOnly: true,
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListContestsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Contests, 1)
				assert.Equal(t, "contest-1", resp.Contests[0].Id)
			},
		},
		{
			name: "list contests with pagination",
			setup: func(m *store.MockContestStore) {
				for i := 1; i <= 25; i++ {
					m.Contests[string(rune('0'+i))] = &pb.Contest{
						Id:     string(rune('0' + i)),
						Name:   "Contest",
						Public: true,
					}
				}
			},
			request: &pb.ListContestsRequest{
				Pagination: &commonv1.Pagination{
					Page:     2,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListContestsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Contests, 10)
				assert.Equal(t, int32(25), resp.Pagination.Total)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockContestStore) {
				m.ListError = assert.AnError
			},
			request: &pb.ListContestsRequest{
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListContestsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockContestStore()
			tt.setup(mockStore)

			service := NewContestService(mockStore, nil)
			resp, err := service.ListContests(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestContestService_GetContest(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockContestStore)
		request *pb.GetContestRequest
		want    func(t *testing.T, resp *pb.GetContestResponse, err error)
	}{
		{
			name: "get existing contest",
			setup: func(m *store.MockContestStore) {
				m.Contests["contest-1"] = &pb.Contest{
					Id:        "contest-1",
					Name:      "Test Contest",
					ShortName: "TC",
					Public:    true,
				}
				m.ContestProblems["contest-1"] = []*pb.ContestProblem{
					{
						ProblemId:  "prob-1",
						ShortName:  "A",
						Rank:       1,
						Points:     100,
						AllowSubmit: true,
					},
				}
			},
			request: &pb.GetContestRequest{Id: "contest-1"},
			want: func(t *testing.T, resp *pb.GetContestResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "contest-1", resp.Contest.Id)
				assert.Equal(t, "Test Contest", resp.Contest.Name)
				assert.Len(t, resp.Problems, 1)
			},
		},
		{
			name:    "get non-existent contest",
			setup:   func(m *store.MockContestStore) {},
			request: &pb.GetContestRequest{Id: "non-existent"},
			want: func(t *testing.T, resp *pb.GetContestResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockContestStore) {
				m.GetError = assert.AnError
			},
			request: &pb.GetContestRequest{Id: "contest-1"},
			want: func(t *testing.T, resp *pb.GetContestResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockContestStore()
			tt.setup(mockStore)

			service := NewContestService(mockStore, nil)
			resp, err := service.GetContest(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestContestService_CreateContest(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockContestStore)
		request *pb.CreateContestRequest
		want    func(t *testing.T, resp *pb.CreateContestResponse, err error)
	}{
		{
			name:  "create contest successfully",
			setup: func(m *store.MockContestStore) {},
			request: &pb.CreateContestRequest{
				ExternalId: "ICPC-2024",
				Name:       "ICPC Regional 2024",
				ShortName:  "ICPC24",
				StartTime:  "2024-03-01T10:00:00Z",
				EndTime:    "2024-03-01T15:00:00Z",
				FreezeTime: "2024-03-01T14:00:00Z",
				Public:     true,
			},
			want: func(t *testing.T, resp *pb.CreateContestResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Id)
			},
		},
		{
			name: "create private contest",
			setup: func(m *store.MockContestStore) {},
			request: &pb.CreateContestRequest{
				Name:      "Private Contest",
				ShortName: "PRIV",
				Public:    false,
			},
			want: func(t *testing.T, resp *pb.CreateContestResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Id)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockContestStore) {
				m.CreateError = assert.AnError
			},
			request: &pb.CreateContestRequest{
				Name: "Contest",
			},
			want: func(t *testing.T, resp *pb.CreateContestResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockContestStore()
			tt.setup(mockStore)

			service := NewContestService(mockStore, nil)
			resp, err := service.CreateContest(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestContestService_RegisterContest(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockContestStore)
		request *pb.RegisterContestRequest
		want    func(t *testing.T, resp *pb.RegisterContestResponse, err error)
	}{
		{
			name:  "register successfully",
			setup: func(m *store.MockContestStore) {},
			request: &pb.RegisterContestRequest{
				ContestId:   "contest-1",
				TeamName:    "Team Alpha",
				Affiliation: "University A",
			},
			want: func(t *testing.T, resp *pb.RegisterContestResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.TeamId)
			},
		},
		{
			name: "register without team name",
			setup: func(m *store.MockContestStore) {},
			request: &pb.RegisterContestRequest{
				ContestId:   "contest-1",
				Affiliation: "University B",
			},
			want: func(t *testing.T, resp *pb.RegisterContestResponse, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockContestStore) {
				m.RegisterError = assert.AnError
			},
			request: &pb.RegisterContestRequest{
				ContestId: "contest-1",
				TeamName:  "Team Beta",
			},
			want: func(t *testing.T, resp *pb.RegisterContestResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockContestStore()
			tt.setup(mockStore)

			service := NewContestService(mockStore, nil)

			// Create context with user ID for authorization
			ctx := context.WithValue(context.Background(), "user_id", "user-1")

			resp, err := service.RegisterContest(ctx, tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestContestService_GetContestProblems(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockContestStore)
		request *pb.GetContestProblemsRequest
		want    func(t *testing.T, resp *pb.GetContestProblemsResponse, err error)
	}{
		{
			name: "get contest problems",
			setup: func(m *store.MockContestStore) {
				m.ContestProblems["contest-1"] = []*pb.ContestProblem{
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
				}
			},
			request: &pb.GetContestProblemsRequest{ContestId: "contest-1"},
			want: func(t *testing.T, resp *pb.GetContestProblemsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Problems, 2)
				assert.Equal(t, "A", resp.Problems[0].ShortName)
				assert.Equal(t, "B", resp.Problems[1].ShortName)
			},
		},
		{
			name:    "no problems for contest",
			setup:   func(m *store.MockContestStore) {},
			request: &pb.GetContestProblemsRequest{ContestId: "contest-empty"},
			want: func(t *testing.T, resp *pb.GetContestProblemsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Problems, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockContestStore()
			tt.setup(mockStore)

			service := NewContestService(mockStore, nil)
			resp, err := service.GetContestProblems(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestContestService_GetScoreboard(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockContestStore)
		request *pb.GetScoreboardRequest
		want    func(t *testing.T, resp *pb.GetScoreboardResponse, err error)
	}{
		{
			name: "get scoreboard with teams",
			setup: func(m *store.MockContestStore) {
				m.Teams["team-1"] = &pb.Team{
					Id:          "team-1",
					Name:        "Team Alpha",
					DisplayName: "Team Alpha",
					Affiliation: "University A",
					Points:      2,
					TotalTime:   150,
				}
				m.Teams["team-2"] = &pb.Team{
					Id:          "team-2",
					Name:        "Team Beta",
					DisplayName: "Team Beta",
					Affiliation: "University B",
					Points:      1,
					TotalTime:   60,
				}
				m.Contests["contest-1"] = &pb.Contest{
					Id: "contest-1",
				}
			},
			request: &pb.GetScoreboardRequest{ContestId: "contest-1"},
			want: func(t *testing.T, resp *pb.GetScoreboardResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Entries)
				assert.Equal(t, "01:00:00", resp.ContestTime)
				assert.False(t, resp.IsFrozen)
			},
		},
		{
			name:    "empty scoreboard",
			setup:   func(m *store.MockContestStore) {},
			request: &pb.GetScoreboardRequest{ContestId: "contest-1"},
			want: func(t *testing.T, resp *pb.GetScoreboardResponse, err error) {
				require.NoError(t, err)
				assert.Empty(t, resp.Entries)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockContestStore()
			tt.setup(mockStore)

			service := NewContestService(mockStore, nil)
			resp, err := service.GetScoreboard(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}