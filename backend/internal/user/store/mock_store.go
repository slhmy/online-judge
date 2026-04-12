package store

import (
	"context"
	"errors"
	"time"
)

// UserStoreInterface defines the interface for UserStore
type UserStoreInterface interface {
	ListProfiles(ctx context.Context, page, pageSize int32) ([]*UserProfile, int32, error)
	GetProfile(ctx context.Context, userID string) (*UserProfile, error)
	UpdateProfile(ctx context.Context, userID string, displayName, avatarURL, bio, country string) error
	UpdateRole(ctx context.Context, userID, role string) error
	DeleteProfile(ctx context.Context, userID string) error
	GetStats(ctx context.Context, userID string) (*UserStats, error)
	ListSubmissions(ctx context.Context, userID string, verdictFilter, problemIDFilter string, page, pageSize int32) ([]*UserSubmissionSummary, int32, error)
	CreateProfile(ctx context.Context, userID, username string) error
	EnsureProfile(ctx context.Context, userID, email, username, role, avatarURL string) (*UserProfile, bool, error)
}

// MockUserStore is a mock implementation of UserStoreInterface for testing
type MockUserStore struct {
	Profiles        map[string]*UserProfile
	Stats           map[string]*UserStats
	Submissions     map[string][]*UserSubmissionSummary
	GetProfileError error
	UpdateError     error
	GetStatsError   error
	ListError       error
	CreateError     error
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		Profiles:    make(map[string]*UserProfile),
		Stats:       make(map[string]*UserStats),
		Submissions: make(map[string][]*UserSubmissionSummary),
	}
}

func (m *MockUserStore) ListProfiles(ctx context.Context, page, pageSize int32) ([]*UserProfile, int32, error) {
	if m.ListError != nil {
		return nil, 0, m.ListError
	}

	profiles := make([]*UserProfile, 0, len(m.Profiles))
	for _, p := range m.Profiles {
		profiles = append(profiles, p)
	}

	total := int32(len(profiles))
	if pageSize <= 0 {
		return profiles, total, nil
	}

	start := (page - 1) * pageSize
	if start < 0 || start >= total {
		return []*UserProfile{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return profiles[start:end], total, nil
}

func (m *MockUserStore) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	if m.GetProfileError != nil {
		return nil, m.GetProfileError
	}

	profile, ok := m.Profiles[userID]
	if !ok {
		return nil, errors.New("profile not found")
	}
	return profile, nil
}

func (m *MockUserStore) UpdateProfile(ctx context.Context, userID string, displayName, avatarURL, bio, country string) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}

	profile, ok := m.Profiles[userID]
	if !ok {
		return errors.New("profile not found")
	}

	if displayName != "" {
		profile.DisplayName = displayName
	}
	if avatarURL != "" {
		profile.AvatarURL = avatarURL
	}
	if bio != "" {
		profile.Bio = bio
	}
	if country != "" {
		profile.Country = country
	}
	profile.UpdatedAt = time.Now()

	return nil
}

func (m *MockUserStore) UpdateRole(ctx context.Context, userID, role string) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}

	profile, ok := m.Profiles[userID]
	if !ok {
		return errors.New("profile not found")
	}

	profile.Role = role
	profile.UpdatedAt = time.Now()
	return nil
}

func (m *MockUserStore) DeleteProfile(ctx context.Context, userID string) error {
	if m.UpdateError != nil {
		return m.UpdateError
	}

	if _, ok := m.Profiles[userID]; !ok {
		return errors.New("profile not found")
	}

	delete(m.Profiles, userID)
	delete(m.Stats, userID)
	delete(m.Submissions, userID)
	return nil
}

func (m *MockUserStore) GetStats(ctx context.Context, userID string) (*UserStats, error) {
	if m.GetStatsError != nil {
		return nil, m.GetStatsError
	}

	stats, ok := m.Stats[userID]
	if !ok {
		return nil, errors.New("stats not found")
	}
	return stats, nil
}

func (m *MockUserStore) ListSubmissions(ctx context.Context, userID string, verdictFilter, problemIDFilter string, page, pageSize int32) ([]*UserSubmissionSummary, int32, error) {
	if m.ListError != nil {
		return nil, 0, m.ListError
	}

	subs, ok := m.Submissions[userID]
	if !ok {
		return []*UserSubmissionSummary{}, 0, nil
	}

	// Apply filters
	var filtered []*UserSubmissionSummary
	for _, sub := range subs {
		if verdictFilter != "" && sub.Verdict != verdictFilter {
			continue
		}
		if problemIDFilter != "" && sub.ProblemID != problemIDFilter {
			continue
		}
		filtered = append(filtered, sub)
	}

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > int32(len(filtered)) {
		end = int32(len(filtered))
	}
	if start > int32(len(filtered)) {
		return []*UserSubmissionSummary{}, int32(len(filtered)), nil
	}

	return filtered[start:end], int32(len(filtered)), nil
}

func (m *MockUserStore) CreateProfile(ctx context.Context, userID, username string) error {
	if m.CreateError != nil {
		return m.CreateError
	}

	if _, exists := m.Profiles[userID]; exists {
		return nil // Already exists, do nothing
	}

	m.Profiles[userID] = &UserProfile{
		UserID:    userID,
		Username:  username,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.Stats[userID] = &UserStats{
		UserID: userID,
	}

	m.Submissions[userID] = []*UserSubmissionSummary{}

	return nil
}

func (m *MockUserStore) EnsureProfile(ctx context.Context, userID, email, username, role, avatarURL string) (*UserProfile, bool, error) {
	if m.CreateError != nil {
		return nil, false, m.CreateError
	}

	if profile, exists := m.Profiles[userID]; exists {
		profile.Email = email
		return profile, false, nil
	}

	if role == "" {
		role = "user"
	}

	profile := &UserProfile{
		UserID:    userID,
		Username:  username,
		Role:      role,
		AvatarURL: avatarURL,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.Profiles[userID] = profile
	m.Stats[userID] = &UserStats{UserID: userID}
	m.Submissions[userID] = []*UserSubmissionSummary{}

	return profile, true, nil
}

// Ensure MockUserStore implements UserStoreInterface
var _ UserStoreInterface = (*MockUserStore)(nil)
