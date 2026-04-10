package service

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"

	commonv1 "github.com/online-judge/gen/go/common/v1"
	pb "github.com/online-judge/gen/go/contest/v1"
	"github.com/online-judge/backend/internal/contest/store"
	"github.com/online-judge/backend/internal/pkg/middleware"
)

var ErrUnauthorized = errors.New("unauthorized")

type ContestService struct {
	pb.UnimplementedContestServiceServer
	store store.ContestStoreInterface
	redis *redis.Client
}

func NewContestService(s store.ContestStoreInterface, redis *redis.Client) *ContestService {
	return &ContestService{
		store: s,
		redis: redis,
	}
}

func (s *ContestService) ListContests(ctx context.Context, req *pb.ListContestsRequest) (*pb.ListContestsResponse, error) {
	// Default pagination
	page := req.GetPagination().GetPage()
	pageSize := req.GetPagination().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	contests, total, err := s.store.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.ListContestsResponse{
		Contests: contests,
		Pagination: &commonv1.PaginatedResponse{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	}, nil
}

func (s *ContestService) GetContest(ctx context.Context, req *pb.GetContestRequest) (*pb.GetContestResponse, error) {
	contest, err := s.store.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Get contest problems
	problems, err := s.store.GetContestProblems(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.GetContestResponse{
		Contest:  contest,
		Problems: problems,
	}, nil
}

func (s *ContestService) CreateContest(ctx context.Context, req *pb.CreateContestRequest) (*pb.CreateContestResponse, error) {
	id, err := s.store.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.CreateContestResponse{Id: id}, nil
}

func (s *ContestService) RegisterContest(ctx context.Context, req *pb.RegisterContestRequest) (*pb.RegisterContestResponse, error) {
	// Get user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	teamID, err := s.store.Register(ctx, req.GetContestId(), userID, req.GetTeamName(), req.GetAffiliation())
	if err != nil {
		return nil, err
	}

	return &pb.RegisterContestResponse{TeamId: teamID}, nil
}

func (s *ContestService) GetContestProblems(ctx context.Context, req *pb.GetContestProblemsRequest) (*pb.GetContestProblemsResponse, error) {
	problems, err := s.store.GetContestProblems(ctx, req.GetContestId())
	if err != nil {
		return nil, err
	}

	return &pb.GetContestProblemsResponse{Problems: problems}, nil
}

func (s *ContestService) GetScoreboard(ctx context.Context, req *pb.GetScoreboardRequest) (*pb.GetScoreboardResponse, error) {
	entries, contestTime, isFrozen, err := s.store.GetScoreboard(ctx, req.GetContestId(), req.GetShowFrozen())
	if err != nil {
		return nil, err
	}

	return &pb.GetScoreboardResponse{
		Entries:     entries,
		ContestTime: contestTime,
		IsFrozen:    isFrozen,
	}, nil
}
