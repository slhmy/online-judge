package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/online-judge/gen/go/problem/v1"
)

type ProblemStore struct {
	db *pgxpool.Pool
}

func NewProblemStore(db *pgxpool.Pool) *ProblemStore {
	return &ProblemStore{db: db}
}

func (s *ProblemStore) List(ctx context.Context, req *pb.ListProblemsRequest) ([]*pb.ProblemSummary, int32, error) {
	query := `
		SELECT id, external_id, name, difficulty, time_limit, memory_limit, points, allow_submit
		FROM problems
		WHERE is_published = true
	`
	args := []interface{}{}
	argIdx := 1

	if req.GetDifficulty() != "" {
		query += " AND difficulty = $" + string(rune('0'+argIdx))
		args = append(args, req.GetDifficulty())
		argIdx++
	}

	if req.GetSearch() != "" {
		query += " AND name ILIKE $" + string(rune('0'+argIdx))
		args = append(args, "%"+req.GetSearch()+"%")
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
	query += " ORDER BY external_id LIMIT $" + string(rune('0'+argIdx)) + " OFFSET $" + string(rune('0'+argIdx+1))
	args = append(args, req.GetPagination().GetPageSize(), offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var problems []*pb.ProblemSummary
	for rows.Next() {
		var p pb.ProblemSummary
		var problemID pgtype.UUID
		err := rows.Scan(
			&problemID, &p.ExternalId, &p.Name, &p.Difficulty,
			&p.TimeLimit, &p.MemoryLimit, &p.Points, &p.AllowSubmit,
		)
		if err != nil {
			return nil, 0, err
		}
		if problemID.Valid {
			p.Id = uuid.UUID(problemID.Bytes).String()
		}
		problems = append(problems, &p)
	}

	return problems, total, nil
}

func (s *ProblemStore) GetByID(ctx context.Context, id string) (*pb.Problem, error) {
	// Parse the string ID to UUID for query parameter
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, external_id, name, time_limit, memory_limit, output_limit, process_limit,
		       special_run_id, special_compare_id, special_compare_args,
		       problem_statement_path, difficulty, color, points,
		       allow_submit, allow_judge, is_published, author_id, created_at, updated_at
		FROM problems
		WHERE id = $1
	`

	var p pb.Problem
	var problemID pgtype.UUID
	var specialRunID, specialCompareID, authorID pgtype.UUID
	var specialCompareArgs, problemStatementPath, color pgtype.Text
	var createdAt, updatedAt pgtype.Timestamp

	err = s.db.QueryRow(ctx, query, parsedID).Scan(
		&problemID, &p.ExternalId, &p.Name, &p.TimeLimit, &p.MemoryLimit, &p.OutputLimit, &p.ProcessLimit,
		&specialRunID, &specialCompareID, &specialCompareArgs,
		&problemStatementPath, &p.Difficulty, &color, &p.Points,
		&p.AllowSubmit, &p.AllowJudge, &p.IsPublished, &authorID, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if problemID.Valid {
		p.Id = uuid.UUID(problemID.Bytes).String()
	}
	if specialRunID.Valid {
		p.SpecialRunId = uuid.UUID(specialRunID.Bytes).String()
	}
	if specialCompareID.Valid {
		p.SpecialCompareId = uuid.UUID(specialCompareID.Bytes).String()
	}
	if specialCompareArgs.Valid {
		p.SpecialCompareArgs = specialCompareArgs.String
	}
	if problemStatementPath.Valid {
		p.ProblemStatementPath = problemStatementPath.String
	}
	if color.Valid {
		p.Color = color.String
	}
	if authorID.Valid {
		p.AuthorId = uuid.UUID(authorID.Bytes).String()
	}
	if createdAt.Valid {
		p.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if updatedAt.Valid {
		p.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &p, nil
}

func (s *ProblemStore) Create(ctx context.Context, req *pb.CreateProblemRequest) (string, error) {
	query := `
		INSERT INTO problems (external_id, name, time_limit, memory_limit, output_limit, difficulty, points, is_published)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING id
	`

	var problemID pgtype.UUID
	err := s.db.QueryRow(ctx, query,
		req.GetExternalId(), req.GetName(), req.GetTimeLimit(), req.GetMemoryLimit(),
		req.GetOutputLimit(), req.GetDifficulty(), req.GetPoints(),
	).Scan(&problemID)
	if err != nil {
		return "", err
	}

	if problemID.Valid {
		return uuid.UUID(problemID.Bytes).String(), nil
	}
	return "", nil
}

func (s *ProblemStore) Update(ctx context.Context, id string, req *pb.UpdateProblemRequest) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	query := `
		UPDATE problems
		SET name = $2, time_limit = $3, memory_limit = $4, is_published = $5, allow_submit = $6, updated_at = NOW()
		WHERE id = $1
	`

	_, err = s.db.Exec(ctx, query,
		parsedID, req.GetName(), req.GetTimeLimit(), req.GetMemoryLimit(),
		req.GetIsPublished(), req.GetAllowSubmit(),
	)
	return err
}

func (s *ProblemStore) Delete(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, "DELETE FROM problems WHERE id = $1", parsedID)
	return err
}

func (s *ProblemStore) ListTestCases(ctx context.Context, problemID string, samplesOnly bool) ([]*pb.TestCase, error) {
	parsedID, err := uuid.Parse(problemID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, problem_id, rank, is_sample, input_path, output_path, description, is_interactive, input_content, output_content
		FROM test_cases
		WHERE problem_id = $1
	`
	args := []interface{}{parsedID}

	if samplesOnly {
		query += " AND is_sample = true"
	}

	query += " ORDER BY rank"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []*pb.TestCase
	for rows.Next() {
		var tc pb.TestCase
		var tcID, tcProblemID pgtype.UUID
		var description, inputContent, outputContent pgtype.Text
		err := rows.Scan(
			&tcID, &tcProblemID, &tc.Rank, &tc.IsSample,
			&tc.InputPath, &tc.OutputPath, &description, &tc.IsInteractive,
			&inputContent, &outputContent,
		)
		if err != nil {
			return nil, err
		}
		if tcID.Valid {
			tc.Id = uuid.UUID(tcID.Bytes).String()
		}
		if tcProblemID.Valid {
			tc.ProblemId = uuid.UUID(tcProblemID.Bytes).String()
		}
		if description.Valid {
			tc.Description = description.String
		}
		if inputContent.Valid {
			tc.InputContent = inputContent.String
		}
		if outputContent.Valid {
			tc.OutputContent = outputContent.String
		}
		testCases = append(testCases, &tc)
	}

	return testCases, nil
}

func (s *ProblemStore) CreateTestCase(ctx context.Context, req *pb.CreateTestCaseRequest) (string, string, string, error) {
	parsedID, err := uuid.Parse(req.GetProblemId())
	if err != nil {
		return "", "", "", err
	}

	query := `
		INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, input_path, output_path
	`

	var tcID pgtype.UUID
	var inputPath, outputPath pgtype.Text
	err = s.db.QueryRow(ctx, query,
		parsedID, req.GetRank(), req.GetIsSample(),
		"problems/"+req.GetProblemId()+"/input/test.txt",
		"problems/"+req.GetProblemId()+"/output/test.txt",
		req.GetDescription(),
	).Scan(&tcID, &inputPath, &outputPath)
	if err != nil {
		return "", "", "", err
	}

	var idStr, inputPathStr, outputPathStr string
	if tcID.Valid {
		idStr = uuid.UUID(tcID.Bytes).String()
	}
	if inputPath.Valid {
		inputPathStr = inputPath.String
	}
	if outputPath.Valid {
		outputPathStr = outputPath.String
	}

	return idStr, inputPathStr, outputPathStr, nil
}

func (s *ProblemStore) ListLanguages(ctx context.Context) ([]*pb.Language, error) {
	query := `
		SELECT id, external_id, name, time_factor, extensions, allow_submit, allow_judge,
		       compile_command, run_command, version
		FROM languages
		WHERE allow_submit = true
		ORDER BY name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var languages []*pb.Language
	for rows.Next() {
		var lang pb.Language
		var extensions [][]byte
		var compileCommand, runCommand, version pgtype.Text
		err := rows.Scan(
			&lang.Id, &lang.ExternalId, &lang.Name, &lang.TimeFactor,
			&extensions, &lang.AllowSubmit, &lang.AllowJudge,
			&compileCommand, &runCommand, &version,
		)
		if err != nil {
			return nil, err
		}

		// Convert extensions array to string slice
		for _, ext := range extensions {
			lang.Extensions = append(lang.Extensions, string(ext))
		}

		if compileCommand.Valid {
			lang.CompileCommand = compileCommand.String
		}
		if runCommand.Valid {
			lang.RunCommand = runCommand.String
		}
		if version.Valid {
			lang.Version = version.String
		}

		languages = append(languages, &lang)
	}

	return languages, nil
}

func (s *ProblemStore) UpdateTestCase(ctx context.Context, id string, req *pb.UpdateTestCaseRequest) (*pb.TestCase, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE test_cases
		SET rank = $2, is_sample = $3, description = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, problem_id, rank, is_sample, input_path, output_path, description, is_interactive, input_content, output_content
	`

	var tc pb.TestCase
	var tcID, tcProblemID pgtype.UUID
	var description, inputContent, outputContent pgtype.Text
	err = s.db.QueryRow(ctx, query,
		parsedID, req.GetRank(), req.GetIsSample(), req.GetDescription(),
	).Scan(
		&tcID, &tcProblemID, &tc.Rank, &tc.IsSample,
		&tc.InputPath, &tc.OutputPath, &description, &tc.IsInteractive,
		&inputContent, &outputContent,
	)
	if err != nil {
		return nil, err
	}

	if tcID.Valid {
		tc.Id = uuid.UUID(tcID.Bytes).String()
	}
	if tcProblemID.Valid {
		tc.ProblemId = uuid.UUID(tcProblemID.Bytes).String()
	}
	if description.Valid {
		tc.Description = description.String
	}
	if inputContent.Valid {
		tc.InputContent = inputContent.String
	}
	if outputContent.Valid {
		tc.OutputContent = outputContent.String
	}

	return &tc, nil
}

func (s *ProblemStore) DeleteTestCase(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, "DELETE FROM test_cases WHERE id = $1", parsedID)
	return err
}

func (s *ProblemStore) BatchCreateTestCases(ctx context.Context, req *pb.BatchUploadTestCasesRequest) ([]*pb.TestCase, error) {
	parsedProblemID, err := uuid.Parse(req.GetProblemId())
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO test_cases (problem_id, rank, is_sample, input_content, output_content, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, problem_id, rank, is_sample, input_path, output_path, description, is_interactive, input_content, output_content
	`

	var testCases []*pb.TestCase
	for _, tcData := range req.GetTestCases() {
		var tc pb.TestCase
		var tcID, tcProblemID pgtype.UUID
		var description, inputContent, outputContent pgtype.Text

		err := s.db.QueryRow(ctx, query,
			parsedProblemID, tcData.GetRank(), tcData.GetIsSample(),
			tcData.GetInputContent(), tcData.GetOutputContent(), tcData.GetDescription(),
		).Scan(
			&tcID, &tcProblemID, &tc.Rank, &tc.IsSample,
			&tc.InputPath, &tc.OutputPath, &description, &tc.IsInteractive,
			&inputContent, &outputContent,
		)
		if err != nil {
			return nil, err
		}

		if tcID.Valid {
			tc.Id = uuid.UUID(tcID.Bytes).String()
		}
		if tcProblemID.Valid {
			tc.ProblemId = uuid.UUID(tcProblemID.Bytes).String()
		}
		if description.Valid {
			tc.Description = description.String
		}
		if inputContent.Valid {
			tc.InputContent = inputContent.String
		}
		if outputContent.Valid {
			tc.OutputContent = outputContent.String
		}

		testCases = append(testCases, &tc)
	}

	return testCases, nil
}

func (s *ProblemStore) GetProblemStatement(ctx context.Context, problemID string, language string) (*pb.ProblemStatement, error) {
	parsedID, err := uuid.Parse(problemID)
	if err != nil {
		return nil, err
	}

	if language == "" {
		language = "en"
	}

	query := `
		SELECT id, problem_id, language, format, title, content, created_at, updated_at
		FROM problem_statements
		WHERE problem_id = $1 AND language = $2
	`

	var stmt pb.ProblemStatement
	var stmtID, stmtProblemID pgtype.UUID
	var createdAt, updatedAt pgtype.Timestamp

	err = s.db.QueryRow(ctx, query, parsedID, language).Scan(
		&stmtID, &stmtProblemID, &stmt.Language, &stmt.Format, &stmt.Title, &stmt.Content,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if stmtID.Valid {
		stmt.Id = uuid.UUID(stmtID.Bytes).String()
	}
	if stmtProblemID.Valid {
		stmt.ProblemId = uuid.UUID(stmtProblemID.Bytes).String()
	}
	if createdAt.Valid {
		stmt.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if updatedAt.Valid {
		stmt.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &stmt, nil
}

func (s *ProblemStore) SetProblemStatement(ctx context.Context, req *pb.SetProblemStatementRequest) (*pb.ProblemStatement, error) {
	parsedID, err := uuid.Parse(req.GetProblemId())
	if err != nil {
		return nil, err
	}

	language := req.GetLanguage()
	if language == "" {
		language = "en"
	}

	format := req.GetFormat()
	if format == "" {
		format = "markdown"
	}

	query := `
		INSERT INTO problem_statements (problem_id, language, format, title, content)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (problem_id, language) DO UPDATE SET
			format = EXCLUDED.format,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			updated_at = NOW()
		RETURNING id, problem_id, language, format, title, content, created_at, updated_at
	`

	var stmt pb.ProblemStatement
	var stmtID, stmtProblemID pgtype.UUID
	var createdAt, updatedAt pgtype.Timestamp

	err = s.db.QueryRow(ctx, query,
		parsedID, language, format, req.GetTitle(), req.GetContent(),
	).Scan(
		&stmtID, &stmtProblemID, &stmt.Language, &stmt.Format, &stmt.Title, &stmt.Content,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if stmtID.Valid {
		stmt.Id = uuid.UUID(stmtID.Bytes).String()
	}
	if stmtProblemID.Valid {
		stmt.ProblemId = uuid.UUID(stmtProblemID.Bytes).String()
	}
	if createdAt.Valid {
		stmt.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if updatedAt.Valid {
		stmt.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &stmt, nil
}
