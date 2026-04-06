package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/online-judge/backend/gen/go/contest/v1"
)

type ContestStore struct {
	db *pgxpool.Pool
}

func NewContestStore(db *pgxpool.Pool) *ContestStore {
	return &ContestStore{db: db}
}

// List returns contests with pagination
func (s *ContestStore) List(ctx context.Context, req *pb.ListContestsRequest) ([]*pb.Contest, int32, error) {
	query := `
		SELECT id, external_id, name, short_name, start_time, end_time, freeze_time, unfreeze_time, public, created_at
		FROM contests
	`
	args := []interface{}{}
	argIdx := 1

	if req.GetPublicOnly() {
		query += " WHERE public = $" + fmt.Sprintf("%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	// Get total count
	var total int32
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS subq"
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	offset := (req.GetPagination().GetPage() - 1) * req.GetPagination().GetPageSize()
	query += " ORDER BY start_time DESC LIMIT $" + fmt.Sprintf("%d", argIdx) + " OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, req.GetPagination().GetPageSize(), offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contests []*pb.Contest
	for rows.Next() {
		var c pb.Contest
		var contestID pgtype.UUID
		var externalID pgtype.Text
		var shortName pgtype.Text
		var startTime, endTime pgtype.Timestamp
		var freezeTime, unfreezeTime pgtype.Timestamp
		var createdAt pgtype.Timestamp

		err := rows.Scan(
			&contestID, &externalID, &c.Name, &shortName,
			&startTime, &endTime, &freezeTime, &unfreezeTime,
			&c.Public, &createdAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if contestID.Valid {
			c.Id = uuid.UUID(contestID.Bytes).String()
		}
		if externalID.Valid {
			c.ExternalId = externalID.String
		}
		if shortName.Valid {
			c.ShortName = shortName.String
		}
		if startTime.Valid {
			c.StartTime = startTime.Time.Format(time.RFC3339)
		}
		if endTime.Valid {
			c.EndTime = endTime.Time.Format(time.RFC3339)
		}
		if freezeTime.Valid {
			c.FreezeTime = freezeTime.Time.Format(time.RFC3339)
		}
		if unfreezeTime.Valid {
			c.UnfreezeTime = unfreezeTime.Time.Format(time.RFC3339)
		}
		if createdAt.Valid {
			c.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}

		contests = append(contests, &c)
	}

	return contests, total, nil
}

// GetByID returns a contest by ID
func (s *ContestStore) GetByID(ctx context.Context, id string) (*pb.Contest, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, external_id, name, short_name, start_time, end_time, freeze_time, unfreeze_time, public, created_at
		FROM contests
		WHERE id = $1
	`

	var c pb.Contest
	var contestID pgtype.UUID
	var externalID pgtype.Text
	var shortName pgtype.Text
	var startTime, endTime pgtype.Timestamp
	var freezeTime, unfreezeTime pgtype.Timestamp
	var createdAt pgtype.Timestamp

	err = s.db.QueryRow(ctx, query, parsedID).Scan(
		&contestID, &externalID, &c.Name, &shortName,
		&startTime, &endTime, &freezeTime, &unfreezeTime,
		&c.Public, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	if contestID.Valid {
		c.Id = uuid.UUID(contestID.Bytes).String()
	}
	if externalID.Valid {
		c.ExternalId = externalID.String
	}
	if shortName.Valid {
		c.ShortName = shortName.String
	}
	if startTime.Valid {
		c.StartTime = startTime.Time.Format(time.RFC3339)
	}
	if endTime.Valid {
		c.EndTime = endTime.Time.Format(time.RFC3339)
	}
	if freezeTime.Valid {
		c.FreezeTime = freezeTime.Time.Format(time.RFC3339)
	}
	if unfreezeTime.Valid {
		c.UnfreezeTime = unfreezeTime.Time.Format(time.RFC3339)
	}
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}

	return &c, nil
}

// Create creates a new contest
func (s *ContestStore) Create(ctx context.Context, req *pb.CreateContestRequest) (string, error) {
	query := `
		INSERT INTO contests (external_id, name, short_name, start_time, end_time, freeze_time, public)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var contestID pgtype.UUID

	// Parse timestamps
	var startTime, endTime, freezeTime interface{}
	if req.GetStartTime() != "" {
		t, err := time.Parse(time.RFC3339, req.GetStartTime())
		if err != nil {
			return "", fmt.Errorf("invalid start_time: %w", err)
		}
		startTime = t
	}
	if req.GetEndTime() != "" {
		t, err := time.Parse(time.RFC3339, req.GetEndTime())
		if err != nil {
			return "", fmt.Errorf("invalid end_time: %w", err)
		}
		endTime = t
	}
	if req.GetFreezeTime() != "" {
		t, err := time.Parse(time.RFC3339, req.GetFreezeTime())
		if err != nil {
			return "", fmt.Errorf("invalid freeze_time: %w", err)
		}
		freezeTime = t
	}

	var externalID interface{}
	if req.GetExternalId() != "" {
		externalID = req.GetExternalId()
	}

	err := s.db.QueryRow(ctx, query,
		externalID, req.GetName(), req.GetShortName(),
		startTime, endTime, freezeTime, req.GetPublic(),
	).Scan(&contestID)
	if err != nil {
		return "", err
	}

	if contestID.Valid {
		return uuid.UUID(contestID.Bytes).String(), nil
	}
	return "", nil
}

// GetContestProblems returns problems for a contest
func (s *ContestStore) GetContestProblems(ctx context.Context, contestID string) ([]*pb.ContestProblem, error) {
	parsedID, err := uuid.Parse(contestID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT problem_id, short_name, rank, color, points, allow_submit
		FROM contest_problems
		WHERE contest_id = $1
		ORDER BY rank
	`

	rows, err := s.db.Query(ctx, query, parsedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []*pb.ContestProblem
	for rows.Next() {
		var p pb.ContestProblem
		var problemID pgtype.UUID
		var color pgtype.Text

		err := rows.Scan(
			&problemID, &p.ShortName, &p.Rank, &color, &p.Points, &p.AllowSubmit,
		)
		if err != nil {
			return nil, err
		}

		if problemID.Valid {
			p.ProblemId = uuid.UUID(problemID.Bytes).String()
		}
		if color.Valid {
			p.Color = color.String
		}

		problems = append(problems, &p)
	}

	return problems, nil
}

// Register registers a user for a contest
func (s *ContestStore) Register(ctx context.Context, contestID, userID, teamName, affiliation string) (string, error) {
	parsedContestID, err := uuid.Parse(contestID)
	if err != nil {
		return "", err
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// Create or get team
	var teamID pgtype.UUID
	query := `
		INSERT INTO teams (user_id, name, display_name, affiliation, contest_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var displayName interface{}
	if teamName != "" {
		displayName = teamName
	}
	var affiliationVal interface{}
	if affiliation != "" {
		affiliationVal = affiliation
	}

	err = tx.QueryRow(ctx, query, parsedUserID, teamName, displayName, affiliationVal, parsedContestID).Scan(&teamID)
	if err != nil {
		return "", err
	}

	// Add to contest_participants
	_, err = tx.Exec(ctx, `
		INSERT INTO contest_participants (contest_id, user_id)
		VALUES ($1, $2)
	`, parsedContestID, parsedUserID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	if teamID.Valid {
		return uuid.UUID(teamID.Bytes).String(), nil
	}
	return "", nil
}

// GetScoreboard returns the scoreboard for a contest
func (s *ContestStore) GetScoreboard(ctx context.Context, contestID string, showFrozen bool) ([]*pb.ScoreboardEntry, string, bool, error) {
	parsedID, err := uuid.Parse(contestID)
	if err != nil {
		return nil, "", false, err
	}

	// Get contest info for freeze time
	var freezeTime pgtype.Timestamp
	var startTime pgtype.Timestamp
	err = s.db.QueryRow(ctx, `
		SELECT start_time, freeze_time FROM contests WHERE id = $1
	`, parsedID).Scan(&startTime, &freezeTime)
	if err != nil {
		return nil, "", false, err
	}

	// Check if currently frozen
	isFrozen := false
	if freezeTime.Valid && startTime.Valid {
		now := time.Now()
		if now.After(freezeTime.Time) {
			isFrozen = true
		}
	}

	// Calculate contest time
	contestTime := ""
	if startTime.Valid {
		elapsed := time.Since(startTime.Time)
		contestTime = formatDuration(elapsed)
	}

	// Get teams with their scores
	query := `
		SELECT t.id, t.name, t.display_name, t.affiliation, t.points, t.total_time
		FROM teams t
		WHERE t.contest_id = $1
		ORDER BY t.points DESC, t.total_time ASC
	`

	rows, err := s.db.Query(ctx, query, parsedID)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var entries []*pb.ScoreboardEntry
	rank := int32(1)
	for rows.Next() {
		var entry pb.ScoreboardEntry
		var teamID pgtype.UUID
		var displayName, affiliation pgtype.Text

		err := rows.Scan(
			&teamID, &entry.TeamName, &displayName, &affiliation, &entry.NumSolved, &entry.TotalTime,
		)
		if err != nil {
			return nil, "", false, err
		}

		if teamID.Valid {
			entry.TeamId = uuid.UUID(teamID.Bytes).String()
		}
		if displayName.Valid {
			entry.TeamName = displayName.String
		}
		if affiliation.Valid {
			entry.Affiliation = affiliation.String
		}
		entry.Rank = rank
		rank++

		// Get problem scores for this team
		problemScores, err := s.getProblemScores(ctx, contestID, entry.TeamId, showFrozen)
		if err != nil {
			return nil, "", false, err
		}
		entry.Problems = problemScores

		entries = append(entries, &entry)
	}

	return entries, contestTime, isFrozen, nil
}

// getProblemScores returns problem scores for a team
func (s *ContestStore) getProblemScores(ctx context.Context, contestID, teamID string, showFrozen bool) ([]*pb.ProblemScore, error) {
	parsedContestID, err := uuid.Parse(contestID)
	if err != nil {
		return nil, err
	}
	parsedTeamID, err := uuid.Parse(teamID)
	if err != nil {
		return nil, err
	}

	// Get all contest problems
	problems, err := s.GetContestProblems(ctx, contestID)
	if err != nil {
		return nil, err
	}

	var scores []*pb.ProblemScore
	for _, p := range problems {
		score := &pb.ProblemScore{
			ProblemShortName: p.ShortName,
		}

		// Count submissions for this problem
		parsedProblemID, err := uuid.Parse(p.ProblemId)
		if err != nil {
			scores = append(scores, score)
			continue
		}

		// Get correct submission count and time
		var correctCount int32
		var correctTime pgtype.Timestamp
		err = s.db.QueryRow(ctx, `
			SELECT COUNT(*), MAX(s.submit_time)
			FROM submissions s
			JOIN judgings j ON j.submission_id = s.id
			WHERE s.team_id = $1 AND s.problem_id = $2 AND s.contest_id = $3 AND j.verdict = 'correct' AND j.valid = true
		`, parsedTeamID, parsedProblemID, parsedContestID).Scan(&correctCount, &correctTime)
		if err == nil && correctCount > 0 {
			score.NumCorrect = 1
			if correctTime.Valid {
				// Calculate time in minutes from contest start
				var contestStart pgtype.Timestamp
				s.db.QueryRow(ctx, `SELECT start_time FROM contests WHERE id = $1`, parsedContestID).Scan(&contestStart)
				if contestStart.Valid {
					score.Time = int32(correctTime.Time.Sub(contestStart.Time).Minutes())
				}
			}
		}

		// Count pending (wrong) submissions
		var wrongCount int32
		err = s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM submissions s
			JOIN judgings j ON j.submission_id = s.id
			WHERE s.team_id = $1 AND s.problem_id = $2 AND s.contest_id = $3
			  AND j.verdict != 'correct' AND j.valid = true
		`, parsedTeamID, parsedProblemID, parsedContestID).Scan(&wrongCount)
		if err == nil {
			score.NumPending = wrongCount
		}

		scores = append(scores, score)
	}

	return scores, nil
}

// formatDuration formats duration in HH:MM:SS format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// RefreshScoreboard recalculates all team scores for a contest
func (s *ContestStore) RefreshScoreboard(ctx context.Context, contestID string) error {
	parsedContestID, err := uuid.Parse(contestID)
	if err != nil {
		return err
	}

	// Get all teams for this contest
	rows, err := s.db.Query(ctx, `
		SELECT id FROM teams WHERE contest_id = $1
	`, parsedContestID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var teamIDs []uuid.UUID
	for rows.Next() {
		var teamID pgtype.UUID
		if err := rows.Scan(&teamID); err != nil {
			continue
		}
		if teamID.Valid {
			teamIDs = append(teamIDs, uuid.UUID(teamID.Bytes))
		}
	}

	// Update each team's score
	for _, teamID := range teamIDs {
		if err := s.UpdateTeamScore(ctx, contestID, teamID.String()); err != nil {
			// Log but continue
			fmt.Printf("Failed to update team score for team %s: %v\n", teamID.String(), err)
		}
	}

	return nil
}

// UpdateTeamScore recalculates a team's points and total time
func (s *ContestStore) UpdateTeamScore(ctx context.Context, contestID, teamID string) error {
	parsedContestID, err := uuid.Parse(contestID)
	if err != nil {
		return err
	}
	parsedTeamID, err := uuid.Parse(teamID)
	if err != nil {
		return err
	}

	// Get contest start time
	var contestStart pgtype.Timestamp
	err = s.db.QueryRow(ctx, `SELECT start_time FROM contests WHERE id = $1`, parsedContestID).Scan(&contestStart)
	if err != nil {
		return err
	}

	// Get contest problems
	problems, err := s.GetContestProblems(ctx, contestID)
	if err != nil {
		return err
	}

	var totalPoints int32
	var totalTime int32

	// Calculate score for each problem (ICPC style)
	for _, p := range problems {
		parsedProblemID, err := uuid.Parse(p.ProblemId)
		if err != nil {
			continue
		}

		// Find first correct submission for this problem
		var correctTime pgtype.Timestamp
		var correctExists bool
		err = s.db.QueryRow(ctx, `
			SELECT s.submit_time, EXISTS(SELECT 1)
			FROM submissions s
			JOIN judgings j ON j.submission_id = s.id
			WHERE s.team_id = $1 AND s.problem_id = $2 AND s.contest_id = $3
			  AND j.verdict = 'correct' AND j.valid = true
			ORDER BY s.submit_time ASC
			LIMIT 1
		`, parsedTeamID, parsedProblemID, parsedContestID).Scan(&correctTime, &correctExists)
		if err != nil || !correctExists {
			continue
		}

		totalPoints++

		// Count wrong submissions before the correct one
		var wrongCount int32
		if correctTime.Valid {
			err = s.db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM submissions s
				JOIN judgings j ON j.submission_id = s.id
				WHERE s.team_id = $1 AND s.problem_id = $2 AND s.contest_id = $3
				  AND j.verdict != 'correct' AND j.valid = true
				  AND s.submit_time < $4
			`, parsedTeamID, parsedProblemID, parsedContestID, correctTime.Time).Scan(&wrongCount)
			if err != nil {
				wrongCount = 0
			}

			// Calculate penalty time (correct time + 20 * wrong count)
			timeFromStart := int32(correctTime.Time.Sub(contestStart.Time).Minutes())
			totalTime += timeFromStart + (wrongCount * 20)
		}
	}

	// Update team record
	_, err = s.db.Exec(ctx, `
		UPDATE teams SET points = $1, total_time = $2 WHERE id = $3
	`, totalPoints, totalTime, parsedTeamID)

	return err
}