package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionStore struct {
	db *pgxpool.Pool
}

func NewSubmissionStore(db *pgxpool.Pool) *SubmissionStore {
	return &SubmissionStore{db: db}
}

func (s *SubmissionStore) Create(ctx context.Context, userID, problemID, contestID, languageID, sourceCode string) (string, error) {
	var id pgtype.UUID

	// Parse UUIDs
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}

	parsedProblemID, err := uuid.Parse(problemID)
	if err != nil {
		return "", err
	}

	// Handle empty contest_id
	var contestIDArg interface{}
	if contestID == "" {
		contestIDArg = nil
	} else {
		parsedContestID, err := uuid.Parse(contestID)
		if err != nil {
			return "", err
		}
		contestIDArg = parsedContestID
	}

	err = s.db.QueryRow(ctx, `
		INSERT INTO submissions (user_id, problem_id, contest_id, language_id, source_code, source_path, submit_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, parsedUserID, parsedProblemID, contestIDArg, languageID, sourceCode, "stored-in-db", time.Now()).Scan(&id)
	if err != nil {
		return "", err
	}
	if id.Valid {
		return uuid.UUID(id.Bytes).String(), nil
	}
	return "", nil
}

type Submission struct {
	ID         string
	UserID     string
	ProblemID  string
	ContestID  string
	LanguageID string
	SourceCode string
	SubmitTime time.Time
}

func (s *SubmissionStore) GetByID(ctx context.Context, id string) (*Submission, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	var sub Submission
	var subID pgtype.UUID
	var userID pgtype.UUID
	var problemID pgtype.UUID
	var contestID pgtype.UUID
	var submitTime pgtype.Timestamp

	err = s.db.QueryRow(ctx, `
		SELECT id, user_id, problem_id, contest_id, language_id, source_code, submit_time
		FROM submissions WHERE id = $1
	`, parsedID).Scan(&subID, &userID, &problemID, &contestID, &sub.LanguageID, &sub.SourceCode, &submitTime)
	if err != nil {
		return nil, err
	}

	if subID.Valid {
		sub.ID = uuid.UUID(subID.Bytes).String()
	}
	if userID.Valid {
		sub.UserID = uuid.UUID(userID.Bytes).String()
	}
	if problemID.Valid {
		sub.ProblemID = uuid.UUID(problemID.Bytes).String()
	}
	if contestID.Valid {
		sub.ContestID = uuid.UUID(contestID.Bytes).String()
	}
	if submitTime.Valid {
		sub.SubmitTime = submitTime.Time
	}

	return &sub, nil
}

type SubmissionSummary struct {
	ID          string
	UserID      string
	Username    string
	ProblemID   string
	ProblemName string
	LanguageID  string
	SubmitTime  time.Time
}

func (s *SubmissionStore) List(ctx context.Context, userID, problemID, contestID string, page, pageSize int32) ([]*SubmissionSummary, int32, error) {
	query := `
		SELECT s.id, s.user_id, COALESCE(up.username, ''), s.problem_id, p.name, s.language_id, s.submit_time
		FROM submissions s
		LEFT JOIN problems p ON s.problem_id = p.id
		LEFT JOIN user_profiles up ON s.user_id = up.user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if userID != "" {
		query += " AND user_id = $" + string(rune('0'+argIdx))
		args = append(args, userID)
		argIdx++
	}
	if problemID != "" {
		query += " AND problem_id = $" + string(rune('0'+argIdx))
		args = append(args, problemID)
		argIdx++
	}
	if contestID != "" {
		query += " AND contest_id = $" + string(rune('0'+argIdx))
		args = append(args, contestID)
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
	offset := (page - 1) * pageSize
	query += " ORDER BY submit_time DESC LIMIT $" + string(rune('0'+argIdx)) + " OFFSET $" + string(rune('0'+argIdx+1))
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var submissions []*SubmissionSummary
	for rows.Next() {
		var sub SubmissionSummary
		var subID, uID, pID pgtype.UUID
		var username pgtype.Text
		var problemName pgtype.Text
		var submitTime pgtype.Timestamp

		err := rows.Scan(&subID, &uID, &username, &pID, &problemName, &sub.LanguageID, &submitTime)
		if err != nil {
			return nil, 0, err
		}

		if subID.Valid {
			sub.ID = uuid.UUID(subID.Bytes).String()
		}
		if uID.Valid {
			sub.UserID = uuid.UUID(uID.Bytes).String()
		}
		if username.Valid {
			sub.Username = username.String
		}
		if pID.Valid {
			sub.ProblemID = uuid.UUID(pID.Bytes).String()
		}
		if problemName.Valid {
			sub.ProblemName = problemName.String
		}
		if submitTime.Valid {
			sub.SubmitTime = submitTime.Time
		}

		submissions = append(submissions, &sub)
	}

	return submissions, total, nil
}

func (s *SubmissionStore) CreateJudging(ctx context.Context, submissionID, judgehostID string) (string, error) {
	var id pgtype.UUID
	err := s.db.QueryRow(ctx, `
		INSERT INTO judgings (submission_id, judgehost_id)
		VALUES ($1, $2)
		RETURNING id
	`, submissionID, judgehostID).Scan(&id)
	if err != nil {
		return "", err
	}
	if id.Valid {
		return uuid.UUID(id.Bytes).String(), nil
	}
	return "", nil
}

func (s *SubmissionStore) UpdateJudging(ctx context.Context, judgingID string, verdict string, maxRuntime float64, maxMemory int64, compileSuccess bool) error {
	_, err := s.db.Exec(ctx, `
		UPDATE judgings
		SET verdict = $2, max_runtime = $3, max_memory = $4, compile_success = $5, end_time = NOW()
		WHERE id = $1
	`, judgingID, verdict, maxRuntime, maxMemory, compileSuccess)
	return err
}

func (s *SubmissionStore) CreateJudgingRun(ctx context.Context, judgingID, testCaseID string, rank int, verdict string, runtime, wallTime float64, memory int64) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO judging_runs (judging_id, test_case_id, rank, verdict, runtime, wall_time, memory)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, judgingID, testCaseID, rank, verdict, runtime, wallTime, memory)
	return err
}

// Judging represents a judging result
type Judging struct {
	ID             string
	SubmissionID   string
	JudgehostID    string
	StartTime      time.Time
	EndTime        time.Time
	MaxRuntime     float64
	MaxMemory      int32
	Verdict        string
	CompileSuccess bool
	Valid          bool
	Verified       bool
	VerifiedBy     string
	Score          float64
}

// GetJudging retrieves the judging result for a submission
// If judgingID is empty, returns the latest (most recent) valid judging
func (s *SubmissionStore) GetJudging(ctx context.Context, submissionID string) (*Judging, error) {
	parsedSubmissionID, err := uuid.Parse(submissionID)
	if err != nil {
		return nil, err
	}

	var j Judging
	var id, subID pgtype.UUID
	var startTime, endTime pgtype.Timestamp
	var maxRuntime pgtype.Numeric
	var maxMemory pgtype.Int4
	var score pgtype.Numeric
	var verifiedBy pgtype.Text

	err = s.db.QueryRow(ctx, `
		SELECT id, submission_id, judgehost_id, start_time, end_time, max_runtime, max_memory, verdict,
		       compile_success, valid, verified, verified_by, score
		FROM judgings
		WHERE submission_id = $1 AND valid = true
		ORDER BY created_at DESC
		LIMIT 1
	`, parsedSubmissionID).Scan(&id, &subID, &j.JudgehostID, &startTime, &endTime, &maxRuntime, &maxMemory, &j.Verdict,
		&j.CompileSuccess, &j.Valid, &j.Verified, &verifiedBy, &score)
	if err != nil {
		return nil, err
	}

	if id.Valid {
		j.ID = uuid.UUID(id.Bytes).String()
	}
	if subID.Valid {
		j.SubmissionID = uuid.UUID(subID.Bytes).String()
	}
	if startTime.Valid {
		j.StartTime = startTime.Time
	}
	if endTime.Valid {
		j.EndTime = endTime.Time
	}
	if maxRuntime.Valid {
		floatVal, _ := maxRuntime.Float64Value()
		j.MaxRuntime = floatVal.Float64
	}
	if maxMemory.Valid {
		j.MaxMemory = maxMemory.Int32
	}
	if verifiedBy.Valid {
		j.VerifiedBy = verifiedBy.String
	}
	if score.Valid {
		floatVal, _ := score.Float64Value()
		j.Score = floatVal.Float64
	}

	return &j, nil
}

// GetJudgingByID retrieves a specific judging by ID
func (s *SubmissionStore) GetJudgingByID(ctx context.Context, judgingID string) (*Judging, error) {
	parsedJudgingID, err := uuid.Parse(judgingID)
	if err != nil {
		return nil, err
	}

	var j Judging
	var id, subID pgtype.UUID
	var startTime, endTime pgtype.Timestamp
	var maxRuntime pgtype.Numeric
	var maxMemory pgtype.Int4
	var score pgtype.Numeric
	var verifiedBy pgtype.Text

	err = s.db.QueryRow(ctx, `
		SELECT id, submission_id, judgehost_id, start_time, end_time, max_runtime, max_memory, verdict,
		       compile_success, valid, verified, verified_by, score
		FROM judgings
		WHERE id = $1
	`, parsedJudgingID).Scan(&id, &subID, &j.JudgehostID, &startTime, &endTime, &maxRuntime, &maxMemory, &j.Verdict,
		&j.CompileSuccess, &j.Valid, &j.Verified, &verifiedBy, &score)
	if err != nil {
		return nil, err
	}

	if id.Valid {
		j.ID = uuid.UUID(id.Bytes).String()
	}
	if subID.Valid {
		j.SubmissionID = uuid.UUID(subID.Bytes).String()
	}
	if startTime.Valid {
		j.StartTime = startTime.Time
	}
	if endTime.Valid {
		j.EndTime = endTime.Time
	}
	if maxRuntime.Valid {
		floatVal, _ := maxRuntime.Float64Value()
		j.MaxRuntime = floatVal.Float64
	}
	if maxMemory.Valid {
		j.MaxMemory = maxMemory.Int32
	}
	if verifiedBy.Valid {
		j.VerifiedBy = verifiedBy.String
	}
	if score.Valid {
		floatVal, _ := score.Float64Value()
		j.Score = floatVal.Float64
	}

	return &j, nil
}

// JudgingRun represents an individual test case run
type JudgingRun struct {
	ID              string
	JudgingID       string
	TestCaseID      string
	Rank            int32
	Runtime         float64
	WallTime        float64
	Memory          int32
	Verdict         string
	OutputRunPath   string
	OutputDiffPath  string
	OutputErrorPath string
}

// GetJudgingRuns retrieves all test case runs for a judging
func (s *SubmissionStore) GetJudgingRuns(ctx context.Context, judgingID string) ([]*JudgingRun, error) {
	parsedJudgingID, err := uuid.Parse(judgingID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, judging_id, test_case_id, rank, runtime, wall_time, memory, verdict,
		       output_run_path, output_diff_path, output_error_path
		FROM judging_runs
		WHERE judging_id = $1
		ORDER BY rank ASC
	`, parsedJudgingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*JudgingRun
	for rows.Next() {
		var run JudgingRun
		var id, judgingIDUUID, testCaseID pgtype.UUID
		var runtime, wallTime pgtype.Numeric
		var memory pgtype.Int4
		var outputRunPath, outputDiffPath, outputErrorPath pgtype.Text

		err := rows.Scan(&id, &judgingIDUUID, &testCaseID, &run.Rank, &runtime, &wallTime, &memory, &run.Verdict,
			&outputRunPath, &outputDiffPath, &outputErrorPath)
		if err != nil {
			return nil, err
		}

		if id.Valid {
			run.ID = uuid.UUID(id.Bytes).String()
		}
		if judgingIDUUID.Valid {
			run.JudgingID = uuid.UUID(judgingIDUUID.Bytes).String()
		}
		if testCaseID.Valid {
			run.TestCaseID = uuid.UUID(testCaseID.Bytes).String()
		}
		if runtime.Valid {
			floatVal, _ := runtime.Float64Value()
			run.Runtime = floatVal.Float64
		}
		if wallTime.Valid {
			floatVal, _ := wallTime.Float64Value()
			run.WallTime = floatVal.Float64
		}
		if memory.Valid {
			run.Memory = memory.Int32
		}
		if outputRunPath.Valid {
			run.OutputRunPath = outputRunPath.String
		}
		if outputDiffPath.Valid {
			run.OutputDiffPath = outputDiffPath.String
		}
		if outputErrorPath.Valid {
			run.OutputErrorPath = outputErrorPath.String
		}

		runs = append(runs, &run)
	}

	return runs, nil
}

// UpdateSubmissionStatus updates the status of a submission
func (s *SubmissionStore) UpdateSubmissionStatus(ctx context.Context, submissionID string, status string) error {
	parsedID, err := uuid.Parse(submissionID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		UPDATE submissions SET status = $1 WHERE id = $2
	`, status, parsedID)
	return err
}

// InvalidateJudging marks a judging as invalid (for rejudges)
func (s *SubmissionStore) InvalidateJudging(ctx context.Context, judgingID string) error {
	parsedID, err := uuid.Parse(judgingID)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
		UPDATE judgings SET valid = false WHERE id = $1
	`, parsedID)
	return err
}
