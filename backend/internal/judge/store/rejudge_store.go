package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RejudgeStore struct {
	db *pgxpool.Pool
}

func NewRejudgeStore(db *pgxpool.Pool) *RejudgeStore {
	return &RejudgeStore{db: db}
}

// Rejudge represents a rejudging operation
type Rejudge struct {
	ID            string
	UserID        string
	ContestID     string
	ProblemID     string
	SubmissionIDs []string
	FromVerdict   string
	Status        string
	Reason        string
	AffectedCount int32
	CreatedAt     time.Time
	StartedAt     time.Time
	FinishedAt    time.Time
	AppliedAt     time.Time
	RevertedAt    time.Time
}

// RejudgeSubmission represents a submission in a rejudging operation
type RejudgeSubmission struct {
	RejudgingID       string
	SubmissionID      string
	OriginalJudgingID string
	NewJudgingID      string
	OriginalVerdict   string
	NewVerdict        string
	VerdictChanged    bool
	Status            string
}

// CreateRejudge creates a new rejudging operation
func (s *RejudgeStore) CreateRejudge(ctx context.Context, userID, contestID, problemID, fromVerdict, reason string, submissionIDs []string) (string, int32, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return "", 0, err
	}

	var contestIDArg interface{}
	if contestID != "" {
		parsedContestID, err := uuid.Parse(contestID)
		if err != nil {
			return "", 0, err
		}
		contestIDArg = parsedContestID
	}

	var problemIDArg interface{}
	if problemID != "" {
		parsedProblemID, err := uuid.Parse(problemID)
		if err != nil {
			return "", 0, err
		}
		problemIDArg = parsedProblemID
	}

	// Convert submission IDs to UUID array
	var submissionUUIDs []uuid.UUID
	for _, id := range submissionIDs {
		parsedID, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		submissionUUIDs = append(submissionUUIDs, parsedID)
	}

	var rejudgeID pgtype.UUID
	var affectedCount int32

	err = s.db.QueryRow(ctx, `
		INSERT INTO rejudgings (user_id, contest_id, problem_id, submission_ids, from_verdict, reason, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id, affected_count
	`, parsedUserID, contestIDArg, problemIDArg, submissionUUIDs, fromVerdict, reason).Scan(&rejudgeID, &affectedCount)
	if err != nil {
		return "", 0, err
	}

	if rejudgeID.Valid {
		return uuid.UUID(rejudgeID.Bytes).String(), affectedCount, nil
	}
	return "", 0, nil
}

// GetRejudge retrieves a rejudging operation by ID
func (s *RejudgeStore) GetRejudge(ctx context.Context, rejudgeID string) (*Rejudge, error) {
	parsedID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return nil, err
	}

	var r Rejudge
	var id, userID pgtype.UUID
	var contestID, problemID pgtype.UUID
	var submissionIDs []pgtype.UUID
	var createdAt, startedAt, finishedAt, appliedAt, revertedAt pgtype.Timestamp

	err = s.db.QueryRow(ctx, `
		SELECT id, user_id, contest_id, problem_id, submission_ids, from_verdict, status, reason,
		       affected_count, created_at, started_at, finished_at, applied_at, reverted_at
		FROM rejudgings WHERE id = $1
	`, parsedID).Scan(&id, &userID, &contestID, &problemID, &submissionIDs, &r.FromVerdict, &r.Status, &r.Reason,
		&r.AffectedCount, &createdAt, &startedAt, &finishedAt, &appliedAt, &revertedAt)
	if err != nil {
		return nil, err
	}

	if id.Valid {
		r.ID = uuid.UUID(id.Bytes).String()
	}
	if userID.Valid {
		r.UserID = uuid.UUID(userID.Bytes).String()
	}
	if contestID.Valid {
		r.ContestID = uuid.UUID(contestID.Bytes).String()
	}
	if problemID.Valid {
		r.ProblemID = uuid.UUID(problemID.Bytes).String()
	}
	for _, sid := range submissionIDs {
		if sid.Valid {
			r.SubmissionIDs = append(r.SubmissionIDs, uuid.UUID(sid.Bytes).String())
		}
	}
	if createdAt.Valid {
		r.CreatedAt = createdAt.Time
	}
	if startedAt.Valid {
		r.StartedAt = startedAt.Time
	}
	if finishedAt.Valid {
		r.FinishedAt = finishedAt.Time
	}
	if appliedAt.Valid {
		r.AppliedAt = appliedAt.Time
	}
	if revertedAt.Valid {
		r.RevertedAt = revertedAt.Time
	}

	return &r, nil
}

// ListRejudges lists rejudging operations with pagination
func (s *RejudgeStore) ListRejudges(ctx context.Context, contestID, problemID, status, userID string, page, pageSize int32) ([]*Rejudge, int32, error) {
	query := `
		SELECT id, user_id, contest_id, problem_id, submission_ids, from_verdict, status, reason,
		       affected_count, created_at, started_at, finished_at, applied_at, reverted_at
		FROM rejudgings WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if contestID != "" {
		parsedContestID, err := uuid.Parse(contestID)
		if err == nil {
			query += fmt.Sprintf(" AND contest_id = $%d", argIdx)
			args = append(args, parsedContestID)
			argIdx++
		}
	}
	if problemID != "" {
		parsedProblemID, err := uuid.Parse(problemID)
		if err == nil {
			query += fmt.Sprintf(" AND problem_id = $%d", argIdx)
			args = append(args, parsedProblemID)
			argIdx++
		}
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if userID != "" {
		parsedUserID, err := uuid.Parse(userID)
		if err == nil {
			query += fmt.Sprintf(" AND user_id = $%d", argIdx)
			args = append(args, parsedUserID)
			argIdx++
		}
	}

	// Get total count
	var total int32
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS subq"
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rejudges []*Rejudge
	for rows.Next() {
		var r Rejudge
		var id, userID pgtype.UUID
		var contestID, problemID pgtype.UUID
		var submissionIDs []pgtype.UUID
		var createdAt, startedAt, finishedAt, appliedAt, revertedAt pgtype.Timestamp

		err := rows.Scan(&id, &userID, &contestID, &problemID, &submissionIDs, &r.FromVerdict, &r.Status, &r.Reason,
			&r.AffectedCount, &createdAt, &startedAt, &finishedAt, &appliedAt, &revertedAt)
		if err != nil {
			return nil, 0, err
		}

		if id.Valid {
			r.ID = uuid.UUID(id.Bytes).String()
		}
		if userID.Valid {
			r.UserID = uuid.UUID(userID.Bytes).String()
		}
		if contestID.Valid {
			r.ContestID = uuid.UUID(contestID.Bytes).String()
		}
		if problemID.Valid {
			r.ProblemID = uuid.UUID(problemID.Bytes).String()
		}
		for _, sid := range submissionIDs {
			if sid.Valid {
				r.SubmissionIDs = append(r.SubmissionIDs, uuid.UUID(sid.Bytes).String())
			}
		}
		if createdAt.Valid {
			r.CreatedAt = createdAt.Time
		}
		if startedAt.Valid {
			r.StartedAt = startedAt.Time
		}
		if finishedAt.Valid {
			r.FinishedAt = finishedAt.Time
		}
		if appliedAt.Valid {
			r.AppliedAt = appliedAt.Time
		}
		if revertedAt.Valid {
			r.RevertedAt = revertedAt.Time
		}

		rejudges = append(rejudges, &r)
	}

	return rejudges, total, nil
}

// UpdateRejudgeStatus updates the status of a rejudging operation
func (s *RejudgeStore) UpdateRejudgeStatus(ctx context.Context, rejudgeID, status string) error {
	parsedID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return err
	}

	var timeField string
	switch status {
	case "judging":
		timeField = "started_at = NOW()"
	case "judged":
		timeField = "finished_at = NOW()"
	case "applied":
		timeField = "applied_at = NOW()"
	case "reverted":
		timeField = "reverted_at = NOW()"
	default:
		timeField = ""
	}

	query := "UPDATE rejudgings SET status = $1"
	if timeField != "" {
		query += ", " + timeField
	}
	query += " WHERE id = $2"

	_, err = s.db.Exec(ctx, query, status, parsedID)
	return err
}

// GetSubmissionsForRejudge gets submissions that match the rejudge filters
func (s *RejudgeStore) GetSubmissionsForRejudge(ctx context.Context, contestID, problemID, fromVerdict string, submissionIDs []string) ([]string, error) {
	// If specific submission IDs are provided, just return those
	if len(submissionIDs) > 0 {
		var validIDs []string
		for _, id := range submissionIDs {
			parsedID, err := uuid.Parse(id)
			if err != nil {
				continue
			}
			// Verify submission exists
			var exists bool
			err = s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM submissions WHERE id = $1)", parsedID).Scan(&exists)
			if err == nil && exists {
				validIDs = append(validIDs, id)
			}
		}
		return validIDs, nil
	}

	// Otherwise, find submissions matching filters
	query := `
		SELECT DISTINCT s.id
		FROM submissions s
		JOIN judgings j ON j.submission_id = s.id AND j.valid = true
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if contestID != "" {
		parsedContestID, err := uuid.Parse(contestID)
		if err != nil {
			return nil, err
		}
		query += fmt.Sprintf(" AND s.contest_id = $%d", argIdx)
		args = append(args, parsedContestID)
		argIdx++
	}
	if problemID != "" {
		parsedProblemID, err := uuid.Parse(problemID)
		if err != nil {
			return nil, err
		}
		query += fmt.Sprintf(" AND s.problem_id = $%d", argIdx)
		args = append(args, parsedProblemID)
		argIdx++
	}
	if fromVerdict != "" {
		query += fmt.Sprintf(" AND j.verdict = $%d", argIdx)
		args = append(args, fromVerdict)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id pgtype.UUID
		err := rows.Scan(&id)
		if err != nil {
			continue
		}
		if id.Valid {
			ids = append(ids, uuid.UUID(id.Bytes).String())
		}
	}

	return ids, nil
}

// CreateRejudgeSubmission creates a record for a submission in a rejudging operation
func (s *RejudgeStore) CreateRejudgeSubmission(ctx context.Context, rejudgeID, submissionID, originalJudgingID, originalVerdict string) error {
	parsedRejudgeID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return err
	}
	parsedSubmissionID, err := uuid.Parse(submissionID)
	if err != nil {
		return err
	}

	var originalJudgingIDArg interface{}
	if originalJudgingID != "" {
		parsedOriginalJudgingID, err := uuid.Parse(originalJudgingID)
		if err != nil {
			return err
		}
		originalJudgingIDArg = parsedOriginalJudgingID
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO rejudging_submissions (rejudging_id, submission_id, original_judging_id, original_verdict, status)
		VALUES ($1, $2, $3, $4, 'pending')
	`, parsedRejudgeID, parsedSubmissionID, originalJudgingIDArg, originalVerdict)
	return err
}

// GetRejudgeSubmissions retrieves submissions for a rejudging operation
func (s *RejudgeStore) GetRejudgeSubmissions(ctx context.Context, rejudgeID string, onlyChanged bool, status string, page, pageSize int32) ([]*RejudgeSubmission, int32, error) {
	parsedRejudgeID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT rejudging_id, submission_id, original_judging_id, new_judging_id,
		       original_verdict, new_verdict, verdict_changed, status
		FROM rejudging_submissions WHERE rejudging_id = $1
	`
	args := []interface{}{parsedRejudgeID}
	argIdx := 2

	if onlyChanged {
		query += fmt.Sprintf(" AND verdict_changed = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
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
	query += fmt.Sprintf(" ORDER BY submission_id LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var submissions []*RejudgeSubmission
	for rows.Next() {
		var rs RejudgeSubmission
		var rejudgingID, submissionID pgtype.UUID
		var originalJudgingID, newJudgingID pgtype.UUID

		err := rows.Scan(&rejudgingID, &submissionID, &originalJudgingID, &newJudgingID,
			&rs.OriginalVerdict, &rs.NewVerdict, &rs.VerdictChanged, &rs.Status)
		if err != nil {
			return nil, 0, err
		}

		if rejudgingID.Valid {
			rs.RejudgingID = uuid.UUID(rejudgingID.Bytes).String()
		}
		if submissionID.Valid {
			rs.SubmissionID = uuid.UUID(submissionID.Bytes).String()
		}
		if originalJudgingID.Valid {
			rs.OriginalJudgingID = uuid.UUID(originalJudgingID.Bytes).String()
		}
		if newJudgingID.Valid {
			rs.NewJudgingID = uuid.UUID(newJudgingID.Bytes).String()
		}

		submissions = append(submissions, &rs)
	}

	return submissions, total, nil
}

// UpdateRejudgeSubmission updates a submission's status after judging
func (s *RejudgeStore) UpdateRejudgeSubmission(ctx context.Context, rejudgeID, submissionID, newJudgingID, newVerdict string) error {
	parsedRejudgeID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return err
	}
	parsedSubmissionID, err := uuid.Parse(submissionID)
	if err != nil {
		return err
	}

	var newJudgingIDArg interface{}
	if newJudgingID != "" {
		parsedNewJudgingID, err := uuid.Parse(newJudgingID)
		if err != nil {
			return err
		}
		newJudgingIDArg = parsedNewJudgingID
	}

	_, err = s.db.Exec(ctx, `
		UPDATE rejudging_submissions
		SET new_judging_id = $3, new_verdict = $4, verdict_changed = (original_verdict != $4), status = 'done'
		WHERE rejudging_id = $1 AND submission_id = $2
	`, parsedRejudgeID, parsedSubmissionID, newJudgingIDArg, newVerdict)
	return err
}

// GetRejudgeProgress returns the progress of a rejudging operation
func (s *RejudgeStore) GetRejudgeProgress(ctx context.Context, rejudgeID string) (int32, int32, int32, error) {
	parsedID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return 0, 0, 0, err
	}

	var pendingCount, judgingCount, doneCount int32
	err = s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending'),
			COUNT(*) FILTER (WHERE status = 'judging'),
			COUNT(*) FILTER (WHERE status = 'done')
		FROM rejudging_submissions WHERE rejudging_id = $1
	`, parsedID).Scan(&pendingCount, &judgingCount, &doneCount)
	if err != nil {
		return 0, 0, 0, err
	}

	return pendingCount, judgingCount, doneCount, nil
}

// ApplyRejudge applies the rejudge results (makes new verdicts official)
func (s *RejudgeStore) ApplyRejudge(ctx context.Context, rejudgeID string) (int32, error) {
	parsedID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return 0, err
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Invalidate original judgings
	_, err = tx.Exec(ctx, `
		UPDATE judgings j
		SET valid = false
		FROM rejudging_submissions rs
		WHERE rs.rejudging_id = $1 AND j.id = rs.original_judging_id
	`, parsedID)
	if err != nil {
		return 0, err
	}

	// Mark new judgings as valid
	_, err = tx.Exec(ctx, `
		UPDATE judgings j
		SET valid = true
		FROM rejudging_submissions rs
		WHERE rs.rejudging_id = $1 AND j.id = rs.new_judging_id AND rs.new_judging_id IS NOT NULL
	`, parsedID)
	if err != nil {
		return 0, err
	}

	// Update rejudging status
	_, err = tx.Exec(ctx, `
		UPDATE rejudgings SET status = 'applied', applied_at = NOW() WHERE id = $1
	`, parsedID)
	if err != nil {
		return 0, err
	}

	// Count verdict changes
	var verdictChanges int32
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM rejudging_submissions WHERE rejudging_id = $1 AND verdict_changed = true
	`, parsedID).Scan(&verdictChanges)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return verdictChanges, nil
}

// RevertRejudge reverts the rejudge results (restores original verdicts)
func (s *RejudgeStore) RevertRejudge(ctx context.Context, rejudgeID string) (int32, error) {
	parsedID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return 0, err
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Invalidate new judgings
	_, err = tx.Exec(ctx, `
		UPDATE judgings j
		SET valid = false
		FROM rejudging_submissions rs
		WHERE rs.rejudging_id = $1 AND j.id = rs.new_judging_id AND rs.new_judging_id IS NOT NULL
	`, parsedID)
	if err != nil {
		return 0, err
	}

	// Restore original judgings
	_, err = tx.Exec(ctx, `
		UPDATE judgings j
		SET valid = true
		FROM rejudging_submissions rs
		WHERE rs.rejudging_id = $1 AND j.id = rs.original_judging_id
	`, parsedID)
	if err != nil {
		return 0, err
	}

	// Update rejudging status
	_, err = tx.Exec(ctx, `
		UPDATE rejudgings SET status = 'reverted', reverted_at = NOW() WHERE id = $1
	`, parsedID)
	if err != nil {
		return 0, err
	}

	// Count restored submissions
	var restoredCount int32
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM rejudging_submissions WHERE rejudging_id = $1 AND new_judging_id IS NOT NULL
	`, parsedID).Scan(&restoredCount)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return restoredCount, nil
}

// CancelRejudge cancels a pending rejudging operation
func (s *RejudgeStore) CancelRejudge(ctx context.Context, rejudgeID string) error {
	parsedID, err := uuid.Parse(rejudgeID)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Restore original judgings for pending submissions
	_, err = tx.Exec(ctx, `
		UPDATE judgings j
		SET valid = true
		FROM rejudging_submissions rs
		WHERE rs.rejudging_id = $1 AND j.id = rs.original_judging_id AND rs.status = 'pending'
	`, parsedID)
	if err != nil {
		return err
	}

	// Invalidate new judgings that were created during rejudge
	_, err = tx.Exec(ctx, `
		UPDATE judgings j
		SET valid = false
		FROM rejudging_submissions rs
		WHERE rs.rejudging_id = $1 AND j.id = rs.new_judging_id AND rs.new_judging_id IS NOT NULL
	`, parsedID)
	if err != nil {
		return err
	}

	// Update rejudging status
	_, err = tx.Exec(ctx, `
		UPDATE rejudgings SET status = 'cancelled' WHERE id = $1
	`, parsedID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RejudgeStoreInterface defines the interface for rejudge storage
type RejudgeStoreInterface interface {
	CreateRejudge(ctx context.Context, userID, contestID, problemID, fromVerdict, reason string, submissionIDs []string) (string, int32, error)
	GetRejudge(ctx context.Context, rejudgeID string) (*Rejudge, error)
	ListRejudges(ctx context.Context, contestID, problemID, status, userID string, page, pageSize int32) ([]*Rejudge, int32, error)
	UpdateRejudgeStatus(ctx context.Context, rejudgeID, status string) error
	GetSubmissionsForRejudge(ctx context.Context, contestID, problemID, fromVerdict string, submissionIDs []string) ([]string, error)
	CreateRejudgeSubmission(ctx context.Context, rejudgeID, submissionID, originalJudgingID, originalVerdict string) error
	GetRejudgeSubmissions(ctx context.Context, rejudgeID string, onlyChanged bool, status string, page, pageSize int32) ([]*RejudgeSubmission, int32, error)
	UpdateRejudgeSubmission(ctx context.Context, rejudgeID, submissionID, newJudgingID, newVerdict string) error
	GetRejudgeProgress(ctx context.Context, rejudgeID string) (int32, int32, int32, error)
	ApplyRejudge(ctx context.Context, rejudgeID string) (int32, error)
	RevertRejudge(ctx context.Context, rejudgeID string) (int32, error)
	CancelRejudge(ctx context.Context, rejudgeID string) error
}
