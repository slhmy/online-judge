package service

import (
	"context"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/user/v1"
	"github.com/online-judge/backend/internal/user/store"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
	store *store.UserStore
}

func NewUserService(store *store.UserStore) *UserService {
	return &UserService{
		store: store,
	}
}

func (s *UserService) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
	profile, err := s.store.GetProfile(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.GetUserProfileResponse{
		Profile: &pb.UserProfile{
			UserId:         profile.UserID,
			Username:       profile.Username,
			DisplayName:    profile.DisplayName,
			Rating:         profile.Rating,
			SolvedCount:    profile.SolvedCount,
			SubmissionCount: profile.SubmissionCount,
			AvatarUrl:      profile.AvatarURL,
			Bio:            profile.Bio,
			Country:        profile.Country,
			CreatedAt:      profile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      profile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

func (s *UserService) UpdateUserProfile(ctx context.Context, req *pb.UpdateUserProfileRequest) (*pb.UpdateUserProfileResponse, error) {
	// TODO: Validate user_id from context matches req.UserId for authorization

	err := s.store.UpdateProfile(ctx, req.UserId, req.DisplayName, req.AvatarUrl, req.Bio, req.Country)
	if err != nil {
		return nil, err
	}

	// Fetch updated profile
	profile, err := s.store.GetProfile(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateUserProfileResponse{
		Profile: &pb.UserProfile{
			UserId:         profile.UserID,
			Username:       profile.Username,
			DisplayName:    profile.DisplayName,
			Rating:         profile.Rating,
			SolvedCount:    profile.SolvedCount,
			SubmissionCount: profile.SubmissionCount,
			AvatarUrl:      profile.AvatarURL,
			Bio:            profile.Bio,
			Country:        profile.Country,
			CreatedAt:      profile.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      profile.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

func (s *UserService) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error) {
	stats, err := s.store.GetStats(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// Calculate acceptance rate
	acceptanceRate := 0.0
	if stats.SubmissionCount > 0 {
		acceptanceRate = float64(stats.AcceptedCount) / float64(stats.SubmissionCount)
	}

	return &pb.GetUserStatsResponse{
		Stats: &pb.UserStats{
			UserId:           stats.UserID,
			SolvedCount:      stats.SolvedCount,
			SubmissionCount:  stats.SubmissionCount,
			Rating:           stats.Rating,
			AcceptedCount:    stats.AcceptedCount,
			WrongAnswerCount: stats.WrongAnswerCount,
			TimeLimitCount:   stats.TimeLimitCount,
			MemoryLimitCount: stats.MemoryLimitCount,
			RuntimeErrorCount: stats.RuntimeErrorCount,
			CompileErrorCount: stats.CompileErrorCount,
			AcceptanceRate:   acceptanceRate,
		},
	}, nil
}

func (s *UserService) ListUserSubmissions(ctx context.Context, req *pb.ListUserSubmissionsRequest) (*pb.ListUserSubmissionsResponse, error) {
	page := req.GetPagination().GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPagination().GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	submissions, total, err := s.store.ListSubmissions(ctx, req.UserId, req.Verdict, req.ProblemId, page, pageSize)
	if err != nil {
		return nil, err
	}

	var pbSubmissions []*pb.UserSubmissionSummary
	for _, sub := range submissions {
		pbSubmissions = append(pbSubmissions, &pb.UserSubmissionSummary{
			Id:          sub.ID,
			ProblemId:   sub.ProblemID,
			ProblemName: sub.ProblemName,
			LanguageId:  sub.LanguageID,
			Verdict:     sub.Verdict,
			Runtime:     sub.Runtime,
			Memory:      sub.Memory,
			SubmitTime:  sub.SubmitTime.Format("2006-01-02T15:04:05Z07:00"),
			ContestId:   sub.ContestID,
			ContestName: sub.ContestName,
		})
	}

	return &pb.ListUserSubmissionsResponse{
		Submissions: pbSubmissions,
		Pagination: &commonv1.PaginatedResponse{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	}, nil
}