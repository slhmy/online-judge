package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	pb "github.com/online-judge/gen/go/contest/v1"
)

// ContestStoreInterface defines the interface for ContestStore
type ContestStoreInterface interface {
	List(ctx context.Context, req *pb.ListContestsRequest) ([]*pb.Contest, int32, error)
	GetByID(ctx context.Context, id string) (*pb.Contest, error)
	Create(ctx context.Context, req *pb.CreateContestRequest) (string, error)
	GetContestProblems(ctx context.Context, contestID string) ([]*pb.ContestProblem, error)
	Register(ctx context.Context, contestID, userID, teamName, affiliation string) (string, error)
	GetScoreboard(ctx context.Context, contestID string, showFrozen bool) ([]*pb.ScoreboardEntry, string, bool, error)
	RefreshScoreboard(ctx context.Context, contestID string) error
	UpdateTeamScore(ctx context.Context, contestID, teamID string) error
}

// MockContestStore is a mock implementation of ContestStoreInterface for testing
type MockContestStore struct {
	Contests        map[string]*pb.Contest
	ContestProblems map[string][]*pb.ContestProblem
	Teams           map[string]*pb.Team
	CreateError     error
	GetError        error
	ListError       error
	RegisterError   error
}

func NewMockContestStore() *MockContestStore {
	return &MockContestStore{
		Contests:        make(map[string]*pb.Contest),
		ContestProblems: make(map[string][]*pb.ContestProblem),
		Teams:           make(map[string]*pb.Team),
	}
}

func (m *MockContestStore) List(ctx context.Context, req *pb.ListContestsRequest) ([]*pb.Contest, int32, error) {
	if m.ListError != nil {
		return nil, 0, m.ListError
	}

	var contests []*pb.Contest
	for _, c := range m.Contests {
		if !req.GetPublicOnly() || c.Public {
			contests = append(contests, c)
		}
	}

	pageSize := req.GetPagination().GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	page := req.GetPagination().GetPage()
	if page <= 0 {
		page = 1
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > int32(len(contests)) {
		end = int32(len(contests))
	}
	if start > int32(len(contests)) {
		return []*pb.Contest{}, int32(len(contests)), nil
	}

	return contests[start:end], int32(len(contests)), nil
}

func (m *MockContestStore) GetByID(ctx context.Context, id string) (*pb.Contest, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}

	c, ok := m.Contests[id]
	if !ok {
		return nil, errors.New("contest not found")
	}
	return c, nil
}

func (m *MockContestStore) Create(ctx context.Context, req *pb.CreateContestRequest) (string, error) {
	if m.CreateError != nil {
		return "", m.CreateError
	}

	id := uuid.New().String()
	m.Contests[id] = &pb.Contest{
		Id:         id,
		ExternalId: req.GetExternalId(),
		Name:       req.GetName(),
		ShortName:  req.GetShortName(),
		StartTime:  req.GetStartTime(),
		EndTime:    req.GetEndTime(),
		FreezeTime: req.GetFreezeTime(),
		Public:     req.GetPublic(),
	}
	return id, nil
}

func (m *MockContestStore) GetContestProblems(ctx context.Context, contestID string) ([]*pb.ContestProblem, error) {
	problems, ok := m.ContestProblems[contestID]
	if !ok {
		return []*pb.ContestProblem{}, nil
	}
	return problems, nil
}

func (m *MockContestStore) Register(ctx context.Context, contestID, userID, teamName, affiliation string) (string, error) {
	if m.RegisterError != nil {
		return "", m.RegisterError
	}

	teamID := uuid.New().String()
	m.Teams[teamID] = &pb.Team{
		Id:          teamID,
		UserId:      userID,
		Name:        teamName,
		DisplayName: teamName,
		Affiliation: affiliation,
	}
	return teamID, nil
}

func (m *MockContestStore) GetScoreboard(ctx context.Context, contestID string, showFrozen bool) ([]*pb.ScoreboardEntry, string, bool, error) {
	var entries []*pb.ScoreboardEntry
	rank := int32(1)
	for _, team := range m.Teams {
		entries = append(entries, &pb.ScoreboardEntry{
			Rank:        rank,
			TeamId:      team.Id,
			TeamName:    team.DisplayName,
			Affiliation: team.Affiliation,
			NumSolved:   team.Points,
			TotalTime:   team.TotalTime,
		})
		rank++
	}
	return entries, "01:00:00", false, nil
}

// RefreshScoreboard refreshes the scoreboard for a contest
func (m *MockContestStore) RefreshScoreboard(ctx context.Context, contestID string) error {
	return nil
}

// UpdateTeamScore updates a team's score
func (m *MockContestStore) UpdateTeamScore(ctx context.Context, contestID, teamID string) error {
	return nil
}
