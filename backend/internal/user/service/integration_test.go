package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/user/v1"
	"github.com/online-judge/backend/internal/user/store"
)

func TestUserService_GetUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockUserStore)
		request *pb.GetUserProfileRequest
		want    func(t *testing.T, resp *pb.GetUserProfileResponse, err error)
	}{
		{
			name: "get existing user profile",
			setup: func(m *store.MockUserStore) {
				m.Profiles["user-1"] = &store.UserProfile{
					UserID:         "user-1",
					Username:       "testuser",
					DisplayName:    "Test User",
					Rating:         1500,
					SolvedCount:    50,
					SubmissionCount: 100,
					AvatarURL:      "https://example.com/avatar.png",
					Bio:            "Hello world",
					Country:        "US",
					CreatedAt:      time.Now().Add(-24 * time.Hour),
					UpdatedAt:      time.Now(),
				}
			},
			request: &pb.GetUserProfileRequest{UserId: "user-1"},
			want: func(t *testing.T, resp *pb.GetUserProfileResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "user-1", resp.Profile.UserId)
				assert.Equal(t, "testuser", resp.Profile.Username)
				assert.Equal(t, "Test User", resp.Profile.DisplayName)
				assert.Equal(t, int32(1500), resp.Profile.Rating)
				assert.Equal(t, int32(50), resp.Profile.SolvedCount)
				assert.Equal(t, int32(100), resp.Profile.SubmissionCount)
			},
		},
		{
			name:    "get non-existent user profile",
			setup:   func(m *store.MockUserStore) {},
			request: &pb.GetUserProfileRequest{UserId: "non-existent"},
			want: func(t *testing.T, resp *pb.GetUserProfileResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockUserStore) {
				m.GetProfileError = assert.AnError
			},
			request: &pb.GetUserProfileRequest{UserId: "user-1"},
			want: func(t *testing.T, resp *pb.GetUserProfileResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockUserStore()
			tt.setup(mockStore)

			service := NewUserService(mockStore)
			resp, err := service.GetUserProfile(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestUserService_UpdateUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockUserStore)
		request *pb.UpdateUserProfileRequest
		want    func(t *testing.T, resp *pb.UpdateUserProfileResponse, err error)
	}{
		{
			name: "update display name and bio",
			setup: func(m *store.MockUserStore) {
				m.Profiles["user-1"] = &store.UserProfile{
					UserID:      "user-1",
					Username:    "testuser",
					DisplayName: "Old Name",
					Bio:         "Old bio",
					CreatedAt:   time.Now().Add(-24 * time.Hour),
					UpdatedAt:   time.Now().Add(-24 * time.Hour),
				}
			},
			request: &pb.UpdateUserProfileRequest{
				UserId:      "user-1",
				DisplayName: "New Name",
				Bio:         "New bio",
			},
			want: func(t *testing.T, resp *pb.UpdateUserProfileResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "New Name", resp.Profile.DisplayName)
				assert.Equal(t, "New bio", resp.Profile.Bio)
			},
		},
		{
			name: "update avatar and country",
			setup: func(m *store.MockUserStore) {
				m.Profiles["user-1"] = &store.UserProfile{
					UserID:      "user-1",
					Username:    "testuser",
					DisplayName: "Test User",
					CreatedAt:   time.Now().Add(-24 * time.Hour),
					UpdatedAt:   time.Now().Add(-24 * time.Hour),
				}
			},
			request: &pb.UpdateUserProfileRequest{
				UserId:     "user-1",
				AvatarUrl:  "https://example.com/new-avatar.png",
				Country:    "UK",
			},
			want: func(t *testing.T, resp *pb.UpdateUserProfileResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "https://example.com/new-avatar.png", resp.Profile.AvatarUrl)
				assert.Equal(t, "UK", resp.Profile.Country)
			},
		},
		{
			name:    "update non-existent profile",
			setup:   func(m *store.MockUserStore) {},
			request: &pb.UpdateUserProfileRequest{UserId: "non-existent", DisplayName: "Name"},
			want: func(t *testing.T, resp *pb.UpdateUserProfileResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockUserStore) {
				m.Profiles["user-1"] = &store.UserProfile{UserID: "user-1"}
				m.UpdateError = assert.AnError
			},
			request: &pb.UpdateUserProfileRequest{UserId: "user-1"},
			want: func(t *testing.T, resp *pb.UpdateUserProfileResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockUserStore()
			tt.setup(mockStore)

			service := NewUserService(mockStore)
			resp, err := service.UpdateUserProfile(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestUserService_GetUserStats(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockUserStore)
		request *pb.GetUserStatsRequest
		want    func(t *testing.T, resp *pb.GetUserStatsResponse, err error)
	}{
		{
			name: "get user stats with submissions",
			setup: func(m *store.MockUserStore) {
				m.Stats["user-1"] = &store.UserStats{
					UserID:           "user-1",
					SolvedCount:      50,
					SubmissionCount:  100,
					Rating:           1650,
					AcceptedCount:    50,
					WrongAnswerCount: 20,
					TimeLimitCount:   15,
					MemoryLimitCount: 5,
					RuntimeErrorCount: 8,
					CompileErrorCount: 2,
				}
			},
			request: &pb.GetUserStatsRequest{UserId: "user-1"},
			want: func(t *testing.T, resp *pb.GetUserStatsResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "user-1", resp.Stats.UserId)
				assert.Equal(t, int32(50), resp.Stats.SolvedCount)
				assert.Equal(t, int32(100), resp.Stats.SubmissionCount)
				assert.Equal(t, int32(1650), resp.Stats.Rating)
				assert.Equal(t, 0.5, resp.Stats.AcceptanceRate)
			},
		},
		{
			name: "get user stats with zero submissions",
			setup: func(m *store.MockUserStore) {
				m.Stats["user-2"] = &store.UserStats{
					UserID:          "user-2",
					SolvedCount:     0,
					SubmissionCount: 0,
					Rating:          1000,
				}
			},
			request: &pb.GetUserStatsRequest{UserId: "user-2"},
			want: func(t *testing.T, resp *pb.GetUserStatsResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, int32(0), resp.Stats.SubmissionCount)
				assert.Equal(t, 0.0, resp.Stats.AcceptanceRate)
			},
		},
		{
			name:    "get non-existent user stats",
			setup:   func(m *store.MockUserStore) {},
			request: &pb.GetUserStatsRequest{UserId: "non-existent"},
			want: func(t *testing.T, resp *pb.GetUserStatsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockUserStore) {
				m.GetStatsError = assert.AnError
			},
			request: &pb.GetUserStatsRequest{UserId: "user-1"},
			want: func(t *testing.T, resp *pb.GetUserStatsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockUserStore()
			tt.setup(mockStore)

			service := NewUserService(mockStore)
			resp, err := service.GetUserStats(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestUserService_ListUserSubmissions(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockUserStore)
		request *pb.ListUserSubmissionsRequest
		want    func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error)
	}{
		{
			name: "list user submissions",
			setup: func(m *store.MockUserStore) {
				m.Submissions["user-1"] = []*store.UserSubmissionSummary{
					{
						ID:          "sub-1",
						ProblemID:   "prob-1",
						ProblemName: "Problem A",
						LanguageID:  "cpp",
						Verdict:     "correct",
						Runtime:     0.5,
						Memory:      1024,
						SubmitTime:  time.Now(),
					},
					{
						ID:          "sub-2",
						ProblemID:   "prob-2",
						ProblemName: "Problem B",
						LanguageID:  "python",
						Verdict:     "wrong-answer",
						Runtime:     1.0,
						Memory:      2048,
						SubmitTime:  time.Now().Add(-time.Hour),
					},
				}
			},
			request: &pb.ListUserSubmissionsRequest{
				UserId: "user-1",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 2)
				assert.Equal(t, int32(2), resp.Pagination.Total)
			},
		},
		{
			name: "list submissions with verdict filter",
			setup: func(m *store.MockUserStore) {
				m.Submissions["user-1"] = []*store.UserSubmissionSummary{
					{
						ID:      "sub-1",
						Verdict: "correct",
					},
					{
						ID:      "sub-2",
						Verdict: "wrong-answer",
					},
				}
			},
			request: &pb.ListUserSubmissionsRequest{
				UserId:  "user-1",
				Verdict: "correct",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 1)
				assert.Equal(t, "correct", resp.Submissions[0].Verdict)
			},
		},
		{
			name: "list submissions with problem filter",
			setup: func(m *store.MockUserStore) {
				m.Submissions["user-1"] = []*store.UserSubmissionSummary{
					{
						ID:        "sub-1",
						ProblemID: "prob-1",
					},
					{
						ID:        "sub-2",
						ProblemID: "prob-2",
					},
				}
			},
			request: &pb.ListUserSubmissionsRequest{
				UserId:    "user-1",
				ProblemId: "prob-1",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 1)
				assert.Equal(t, "prob-1", resp.Submissions[0].ProblemId)
			},
		},
		{
			name: "list submissions with pagination",
			setup: func(m *store.MockUserStore) {
				var subs []*store.UserSubmissionSummary
				for i := 0; i < 25; i++ {
					subs = append(subs, &store.UserSubmissionSummary{
						ID:      string(rune('0' + i)),
						Verdict: "correct",
					})
				}
				m.Submissions["user-1"] = subs
			},
			request: &pb.ListUserSubmissionsRequest{
				UserId: "user-1",
				Pagination: &commonv1.Pagination{
					Page:     2,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 10)
				assert.Equal(t, int32(25), resp.Pagination.Total)
			},
		},
		{
			name: "list submissions for user with no submissions",
			setup: func(m *store.MockUserStore) {
				m.Submissions["user-1"] = []*store.UserSubmissionSummary{}
			},
			request: &pb.ListUserSubmissionsRequest{
				UserId: "user-1",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 0)
				assert.Equal(t, int32(0), resp.Pagination.Total)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockUserStore) {
				m.ListError = assert.AnError
			},
			request: &pb.ListUserSubmissionsRequest{
				UserId: "user-1",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListUserSubmissionsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockUserStore()
			tt.setup(mockStore)

			service := NewUserService(mockStore)
			resp, err := service.ListUserSubmissions(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

// Integration test: Full user profile flow
func TestUserService_Integration_ProfileFlow(t *testing.T) {
	mockStore := store.NewMockUserStore()
	service := NewUserService(mockStore)

	userID := "user-integration-test"

	// Step 1: Create profile
	err := mockStore.CreateProfile(context.Background(), userID, "integrationuser")
	require.NoError(t, err)

	// Step 2: Get profile
	getResp, err := service.GetUserProfile(context.Background(), &pb.GetUserProfileRequest{UserId: userID})
	require.NoError(t, err)
	assert.Equal(t, "integrationuser", getResp.Profile.Username)

	// Step 3: Update profile
	updateResp, err := service.UpdateUserProfile(context.Background(), &pb.UpdateUserProfileRequest{
		UserId:      userID,
		DisplayName: "Integration Test User",
		Bio:         "This is a test user",
		Country:     "US",
	})
	require.NoError(t, err)
	assert.Equal(t, "Integration Test User", updateResp.Profile.DisplayName)
	assert.Equal(t, "This is a test user", updateResp.Profile.Bio)
	assert.Equal(t, "US", updateResp.Profile.Country)

	// Step 4: Verify update persisted
	getResp2, err := service.GetUserProfile(context.Background(), &pb.GetUserProfileRequest{UserId: userID})
	require.NoError(t, err)
	assert.Equal(t, "Integration Test User", getResp2.Profile.DisplayName)

	// Step 5: Set stats
	mockStore.Stats[userID] = &store.UserStats{
		UserID:          userID,
		SolvedCount:     10,
		SubmissionCount: 20,
		Rating:          1200,
		AcceptedCount:   10,
	}

	// Step 6: Get stats
	statsResp, err := service.GetUserStats(context.Background(), &pb.GetUserStatsRequest{UserId: userID})
	require.NoError(t, err)
	assert.Equal(t, int32(10), statsResp.Stats.SolvedCount)
	assert.Equal(t, 0.5, statsResp.Stats.AcceptanceRate)
}