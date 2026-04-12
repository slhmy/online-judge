package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

// ListProfiles retrieves users ordered by creation time desc.
func (s *UserStore) ListProfiles(ctx context.Context, page, pageSize int32) ([]*UserProfile, int32, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	var total int32
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM user_profiles`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.db.Query(ctx, `
		SELECT user_id, username, email, display_name, rating, solved_count, submission_count,
		       avatar_url, bio, country, role, created_at, updated_at
		FROM user_profiles
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	profiles := make([]*UserProfile, 0)
	for rows.Next() {
		var p UserProfile
		var uid pgtype.UUID
		var email, displayName, avatarURL, bio, country, role pgtype.Text
		var createdAt, updatedAt pgtype.Timestamp
		var rating, solvedCount, submissionCount pgtype.Int4

		if err := rows.Scan(&uid, &p.Username, &email, &displayName, &rating, &solvedCount, &submissionCount,
			&avatarURL, &bio, &country, &role, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}

		if uid.Valid {
			p.UserID = uuid.UUID(uid.Bytes).String()
		}
		if displayName.Valid {
			p.DisplayName = displayName.String
		}
		if email.Valid {
			p.Email = email.String
		}
		if rating.Valid {
			p.Rating = rating.Int32
		}
		if solvedCount.Valid {
			p.SolvedCount = solvedCount.Int32
		}
		if submissionCount.Valid {
			p.SubmissionCount = submissionCount.Int32
		}
		if avatarURL.Valid {
			p.AvatarURL = avatarURL.String
		}
		if bio.Valid {
			p.Bio = bio.String
		}
		if country.Valid {
			p.Country = country.String
		}
		if role.Valid {
			p.Role = role.String
		}
		if createdAt.Valid {
			p.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			p.UpdatedAt = updatedAt.Time
		}

		profiles = append(profiles, &p)
	}

	return profiles, total, nil
}

// UserProfile represents a user's profile from the database
type UserProfile struct {
	UserID          string
	Username        string
	DisplayName     string
	Rating          int32
	SolvedCount     int32
	SubmissionCount int32
	AvatarURL       string
	Bio             string
	Country         string
	Role            string
	Email           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// GetProfile retrieves a user profile by user_id
func (s *UserStore) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	var id, uid pgtype.UUID
	var email, displayName, avatarURL, bio, country, role pgtype.Text
	var createdAt, updatedAt pgtype.Timestamp
	var rating, solvedCount, submissionCount pgtype.Int4

	err = s.db.QueryRow(ctx, `
		SELECT id, user_id, username, email, display_name, rating, solved_count, submission_count,
		       avatar_url, bio, country, created_at, updated_at, role
		FROM user_profiles WHERE user_id = $1
	`, parsedUserID).Scan(&id, &uid, &profile.Username, &email, &displayName, &rating, &solvedCount,
		&submissionCount, &avatarURL, &bio, &country, &createdAt, &updatedAt, &role)
	if err != nil {
		return nil, err
	}

	if uid.Valid {
		profile.UserID = uuid.UUID(uid.Bytes).String()
	}
	if displayName.Valid {
		profile.DisplayName = displayName.String
	}
	if email.Valid {
		profile.Email = email.String
	}
	if rating.Valid {
		profile.Rating = rating.Int32
	}
	if solvedCount.Valid {
		profile.SolvedCount = solvedCount.Int32
	}
	if submissionCount.Valid {
		profile.SubmissionCount = submissionCount.Int32
	}
	if avatarURL.Valid {
		profile.AvatarURL = avatarURL.String
	}
	if bio.Valid {
		profile.Bio = bio.String
	}
	if country.Valid {
		profile.Country = country.String
	}
	if createdAt.Valid {
		profile.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time
	}
	if role.Valid {
		profile.Role = role.String
	}

	return &profile, nil
}

// UpdateProfile updates a user's profile
func (s *UserStore) UpdateProfile(ctx context.Context, userID string, displayName, avatarURL, bio, country string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		UPDATE user_profiles
		SET display_name = COALESCE(NULLIF($2, ''), display_name),
		    avatar_url = COALESCE(NULLIF($3, ''), avatar_url),
		    bio = COALESCE(NULLIF($4, ''), bio),
		    country = COALESCE(NULLIF($5, ''), country),
		    updated_at = NOW()
		WHERE user_id = $1
	`, parsedUserID, displayName, avatarURL, bio, country)
	return err
}

// UpdateRole updates a user's role.
func (s *UserStore) UpdateRole(ctx context.Context, userID, role string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		UPDATE user_profiles
		SET role = $2,
		    updated_at = NOW()
		WHERE user_id = $1
	`, parsedUserID, role)
	return err
}

// DeleteProfile removes a user profile and related data records.
func (s *UserStore) DeleteProfile(ctx context.Context, userID string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Remove judging details for this user's submissions first.
	if _, err := tx.Exec(ctx, `
		DELETE FROM judging_runs
		WHERE judging_id IN (
			SELECT j.id
			FROM judgings j
			JOIN submissions s ON s.id = j.submission_id
			WHERE s.user_id = $1
		)
	`, parsedUserID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM judgings
		WHERE submission_id IN (
			SELECT id FROM submissions WHERE user_id = $1
		)
	`, parsedUserID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM submissions WHERE user_id = $1`, parsedUserID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM contest_participants WHERE user_id = $1`, parsedUserID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM teams WHERE user_id = $1`, parsedUserID); err != nil {
		return err
	}

	res, err := tx.Exec(ctx, `DELETE FROM user_profiles WHERE user_id = $1`, parsedUserID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("profile not found")
	}

	return tx.Commit(ctx)
}

// UserStats represents computed user statistics
type UserStats struct {
	UserID            string
	SolvedCount       int32
	SubmissionCount   int32
	Rating            int32
	AcceptedCount     int32
	WrongAnswerCount  int32
	TimeLimitCount    int32
	MemoryLimitCount  int32
	RuntimeErrorCount int32
	CompileErrorCount int32
}

// GetStats retrieves user statistics by computing from submissions
func (s *UserStore) GetStats(ctx context.Context, userID string) (*UserStats, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	// Get profile stats first
	var profileRating, profileSolvedCount, profileSubmissionCount pgtype.Int4
	err = s.db.QueryRow(ctx, `
		SELECT rating, solved_count, submission_count
		FROM user_profiles WHERE user_id = $1
	`, parsedUserID).Scan(&profileRating, &profileSolvedCount, &profileSubmissionCount)
	if err != nil {
		return nil, err
	}

	// Compute verdict counts from submissions
	stats := &UserStats{
		UserID: userID,
	}
	if profileRating.Valid {
		stats.Rating = profileRating.Int32
	}
	if profileSolvedCount.Valid {
		stats.SolvedCount = profileSolvedCount.Int32
	}
	if profileSubmissionCount.Valid {
		stats.SubmissionCount = profileSubmissionCount.Int32
	}

	// Get verdict breakdown from judgings
	rows, err := s.db.Query(ctx, `
		SELECT j.verdict, COUNT(*) as count
		FROM submissions s
		JOIN judgings j ON s.id = j.submission_id
		WHERE s.user_id = $1 AND j.valid = true
		GROUP BY j.verdict
	`, parsedUserID)
	if err != nil {
		return stats, nil // Return profile stats even if submission breakdown fails
	}
	defer rows.Close()

	for rows.Next() {
		var verdict pgtype.Text
		var count pgtype.Int4
		if err := rows.Scan(&verdict, &count); err != nil {
			continue
		}
		if !verdict.Valid || !count.Valid {
			continue
		}

		switch verdict.String {
		case "correct":
			stats.AcceptedCount = count.Int32
		case "wrong-answer":
			stats.WrongAnswerCount = count.Int32
		case "timelimit", "time-limit":
			stats.TimeLimitCount = count.Int32
		case "memory-limit":
			stats.MemoryLimitCount = count.Int32
		case "run-error":
			stats.RuntimeErrorCount = count.Int32
		case "compiler-error":
			stats.CompileErrorCount = count.Int32
		}
	}

	return stats, nil
}

// UserSubmissionSummary represents a submission summary for listing
type UserSubmissionSummary struct {
	ID          string
	ProblemID   string
	ProblemName string
	LanguageID  string
	Verdict     string
	Runtime     float64
	Memory      int32
	SubmitTime  time.Time
	ContestID   string
	ContestName string
}

// ListSubmissions retrieves paginated submissions for a user
func (s *UserStore) ListSubmissions(ctx context.Context, userID string, verdictFilter, problemIDFilter string, page, pageSize int32) ([]*UserSubmissionSummary, int32, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT s.id, s.problem_id, p.name, s.language_id, j.verdict, j.max_runtime, j.max_memory,
		       s.submit_time, s.contest_id, c.name
		FROM submissions s
		LEFT JOIN problems p ON s.problem_id = p.id
		LEFT JOIN judgings j ON s.id = j.submission_id AND j.valid = true
		LEFT JOIN contests c ON s.contest_id = c.id
		WHERE s.user_id = $1
	`
	args := []interface{}{parsedUserID}
	argIdx := 2

	if verdictFilter != "" {
		query += " AND j.verdict = $" + string(rune('0'+argIdx))
		args = append(args, verdictFilter)
		argIdx++
	}

	if problemIDFilter != "" {
		parsedProblemID, err := uuid.Parse(problemIDFilter)
		if err == nil {
			query += " AND s.problem_id = $" + string(rune('0'+argIdx))
			args = append(args, parsedProblemID)
			argIdx++
		}
	}

	// Get total count
	var total int32
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS subq"
	err = s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	offset := (page - 1) * pageSize
	query += " ORDER BY s.submit_time DESC LIMIT $" + string(rune('0'+argIdx)) + " OFFSET $" + string(rune('0'+argIdx+1))
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var submissions []*UserSubmissionSummary
	for rows.Next() {
		var sub UserSubmissionSummary
		var subID, problemID pgtype.UUID
		var problemName, contestName pgtype.Text
		var contestID pgtype.UUID
		var submitTime pgtype.Timestamp
		var verdict pgtype.Text
		var maxRuntime pgtype.Numeric
		var maxMemory pgtype.Int4

		err := rows.Scan(&subID, &problemID, &problemName, &sub.LanguageID, &verdict, &maxRuntime, &maxMemory,
			&submitTime, &contestID, &contestName)
		if err != nil {
			return nil, 0, err
		}

		if subID.Valid {
			sub.ID = uuid.UUID(subID.Bytes).String()
		}
		if problemID.Valid {
			sub.ProblemID = uuid.UUID(problemID.Bytes).String()
		}
		if problemName.Valid {
			sub.ProblemName = problemName.String
		}
		if verdict.Valid {
			sub.Verdict = verdict.String
		}
		if maxRuntime.Valid {
			floatVal, _ := maxRuntime.Float64Value()
			sub.Runtime = floatVal.Float64
		}
		if maxMemory.Valid {
			sub.Memory = maxMemory.Int32
		}
		if submitTime.Valid {
			sub.SubmitTime = submitTime.Time
		}
		if contestID.Valid {
			sub.ContestID = uuid.UUID(contestID.Bytes).String()
		}
		if contestName.Valid {
			sub.ContestName = contestName.String
		}

		submissions = append(submissions, &sub)
	}

	return submissions, total, nil
}

// CreateProfile creates a new user profile
func (s *UserStore) CreateProfile(ctx context.Context, userID, username string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO user_profiles (user_id, username)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, parsedUserID, username)
	return err
}

// EnsureProfile creates a profile if it doesn't exist, or returns the existing one.
// Returns (profile, created, error).
func (s *UserStore) EnsureProfile(ctx context.Context, userID, email, username, role, avatarURL string) (*UserProfile, bool, error) {
	// Try to get existing profile first
	profile, err := s.GetProfile(ctx, userID)
	if err == nil {
		// Keep existing users in sync with admin assignment from auth flow.
		if strings.EqualFold(strings.TrimSpace(role), "admin") && !strings.EqualFold(strings.TrimSpace(profile.Role), "admin") {
			if updateErr := s.UpdateRole(ctx, userID, "admin"); updateErr == nil {
				profile.Role = "admin"
			}
		}

		// Keep email in sync with auth provider when available.
		if strings.TrimSpace(email) != "" && !strings.EqualFold(strings.TrimSpace(profile.Email), strings.TrimSpace(email)) {
			parsedUserID, parseErr := uuid.Parse(userID)
			if parseErr == nil {
				_, _ = s.db.Exec(ctx, `
					UPDATE user_profiles
					SET email = $2,
					    updated_at = NOW()
					WHERE user_id = $1
				`, parsedUserID, email)
			}
		}

		// Profile exists – update avatar if provided
		if avatarURL != "" && avatarURL != profile.AvatarURL {
			_ = s.UpdateProfile(ctx, userID, "", avatarURL, "", "")
			profile.AvatarURL = avatarURL
		}
		profile.Email = email
		return profile, false, nil
	}

	// Profile doesn't exist – create it
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, false, err
	}

	if username == "" {
		parts := strings.SplitN(email, "@", 2)
		username = parts[0]
	}
	if role == "" {
		role = "user"
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO user_profiles (user_id, username, email, role, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING
	`, parsedUserID, username, email, role, avatarURL)
	if err != nil {
		return nil, false, err
	}

	// Re-fetch to get normalized data
	profile, err = s.GetProfile(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	profile.Email = email
	return profile, true, nil
}
