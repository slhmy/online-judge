package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	judgehostKeyPrefix = "judgehost:"
	judgehostListKey   = "judgehosts:list"
)

// JudgehostStore manages judgehost data in PostgreSQL and Redis
type JudgehostStore struct {
	db   *pgxpool.Pool
	redis *redis.Client
}

// NewJudgehostStore creates a new judgehost store
func NewJudgehostStore(db *pgxpool.Pool, redis *redis.Client) *JudgehostStore {
	return &JudgehostStore{
		db:    db,
		redis: redis,
	}
}

// Judgehost represents a registered judgehost
type Judgehost struct {
	ID             string
	QueueName      string
	Languages      []string
	MaxConcurrent  int32
	MemoryLimit    int64
	TimeLimit      float64
	Interactive    bool
	Special        bool
	Extra          map[string]string
	Status         string
	CurrentJobID   string
	LastPing       time.Time
	ActiveJobs     int32
	CompletedJobs  int32
	RegisteredAt   time.Time
}

// Register inserts a new judgehost and returns the assigned queue name
func (s *JudgehostStore) Register(ctx context.Context, id string, languages []string, maxConcurrent int32,
	memoryLimit int64, timeLimit float64, interactive, special bool, extra map[string]string) (string, error) {
	// Generate queue name based on judgehost ID
	queueName := fmt.Sprintf("judgehost-%s", id)

	// Insert into PostgreSQL
	var registeredAt pgtype.Timestamp
	err := s.db.QueryRow(ctx, `
		INSERT INTO judgehosts (id, queue_name, languages, max_concurrent, memory_limit, time_limit,
		                       supports_interactive, supports_special, extra, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'idle')
		RETURNING registered_at
	`, id, queueName, languages, maxConcurrent, memoryLimit, timeLimit, interactive, special, extra).Scan(&registeredAt)
	if err != nil {
		return "", fmt.Errorf("failed to register judgehost: %w", err)
	}

	// Cache in Redis for fast lookup
	judgehostKey := judgehostKeyPrefix + id
	s.redis.HSet(ctx, judgehostKey, map[string]interface{}{
		"id":                id,
		"queue_name":        queueName,
		"languages":         fmt.Sprintf("%v", languages),
		"max_concurrent":    maxConcurrent,
		"status":            "idle",
		"last_ping":         time.Now().Format(time.RFC3339),
		"active_jobs":       0,
		"supports_interactive": interactive,
		"supports_special":  special,
	})
	s.redis.Expire(ctx, judgehostKey, 24*time.Hour)

	// Add to active judgehosts list
	s.redis.SAdd(ctx, judgehostListKey, id)

	return queueName, nil
}

// UpdateHeartbeat updates judgehost status and last_ping time
func (s *JudgehostStore) UpdateHeartbeat(ctx context.Context, id string, status string, currentJobID string,
	activeJobs int32, completedJobIDs []string) error {
	// Update Redis cache (fast path)
	judgehostKey := judgehostKeyPrefix + id
	s.redis.HSet(ctx, judgehostKey, map[string]interface{}{
		"status":       status,
		"current_job_id": currentJobID,
		"last_ping":    time.Now().Format(time.RFC3339),
		"active_jobs":  activeJobs,
	})
	s.redis.Expire(ctx, judgehostKey, 24*time.Hour)

	// Update PostgreSQL status (async, for persistence)
	go func() {
		_, err := s.db.Exec(context.Background(), `
			UPDATE judgehosts SET status = $1, last_ping = NOW(), active_jobs = $2
			WHERE id = $3
		`, status, activeJobs, id)
		if err != nil {
			// Log error but don't fail heartbeat
			fmt.Printf("failed to update judgehost status in db: %v\n", err)
		}

		// Update completed_jobs count
		if len(completedJobIDs) > 0 {
			_, err = s.db.Exec(context.Background(), `
				UPDATE judgehosts SET completed_jobs = completed_jobs + $1 WHERE id = $2
			`, len(completedJobIDs), id)
			if err != nil {
				fmt.Printf("failed to update completed_jobs: %v\n", err)
			}
		}
	}()

	return nil
}

// GetAvailableJudgehosts retrieves all idle and recently active judgehosts
func (s *JudgehostStore) GetAvailableJudgehosts(ctx context.Context, languageID string, interactive, special bool) ([]*Judgehost, error) {
	// Query PostgreSQL for eligible judgehosts
	rows, err := s.db.Query(ctx, `
		SELECT id, queue_name, languages, max_concurrent, memory_limit, time_limit,
		       supports_interactive, supports_special, status, last_ping, active_jobs
		FROM judgehosts
		WHERE status = 'idle'
		  AND active_jobs < max_concurrent
		  AND ($1 = '' OR languages @> ARRAY[$1])
		  AND (NOT $2 OR supports_interactive = true)
		  AND (NOT $3 OR supports_special = true)
		  AND last_ping > NOW() - INTERVAL '5 minutes'
		ORDER BY last_ping DESC
	`, languageID, interactive, special)
	if err != nil {
		return nil, fmt.Errorf("failed to query available judgehosts: %w", err)
	}
	defer rows.Close()

	var judgehosts []*Judgehost
	for rows.Next() {
		var j Judgehost
		var lastPing pgtype.Timestamp
		var languages []string

		err := rows.Scan(&j.ID, &j.QueueName, &languages, &j.MaxConcurrent, &j.MemoryLimit, &j.TimeLimit,
			&j.Interactive, &j.Special, &j.Status, &lastPing, &j.ActiveJobs)
		if err != nil {
			return nil, fmt.Errorf("failed to scan judgehost: %w", err)
		}

		j.Languages = languages
		if lastPing.Valid {
			j.LastPing = lastPing.Time
		}
		judgehosts = append(judgehosts, &j)
	}

	return judgehosts, nil
}

// GetByID retrieves a judgehost by ID
func (s *JudgehostStore) GetByID(ctx context.Context, id string) (*Judgehost, error) {
	var j Judgehost
	var registeredAt, lastPing pgtype.Timestamp
	var languages []string

	err := s.db.QueryRow(ctx, `
		SELECT id, queue_name, languages, max_concurrent, memory_limit, time_limit,
		       supports_interactive, supports_special, status, current_job_id,
		       last_ping, active_jobs, completed_jobs, registered_at
		FROM judgehosts WHERE id = $1
	`, id).Scan(&j.ID, &j.QueueName, &languages, &j.MaxConcurrent, &j.MemoryLimit, &j.TimeLimit,
		&j.Interactive, &j.Special, &j.Status, &j.CurrentJobID, &lastPing, &j.ActiveJobs,
		&j.CompletedJobs, &registeredAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get judgehost: %w", err)
	}

	j.Languages = languages
	if registeredAt.Valid {
		j.RegisteredAt = registeredAt.Time
	}
	if lastPing.Valid {
		j.LastPing = lastPing.Time
	}

	return &j, nil
}

// List retrieves all judgehosts with optional status filter
func (s *JudgehostStore) List(ctx context.Context, statusFilter string, page, pageSize int32) ([]*Judgehost, int32, error) {
	query := `
		SELECT id, queue_name, languages, max_concurrent, memory_limit, time_limit,
		       supports_interactive, supports_special, status, current_job_id,
		       last_ping, active_jobs, completed_jobs, registered_at
		FROM judgehosts
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if statusFilter != "" && statusFilter != "unspecified" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, statusFilter)
		argIdx++
	}

	// Get total count
	var total int32
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS subq"
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count judgehosts: %w", err)
	}

	// Add pagination
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY registered_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list judgehosts: %w", err)
	}
	defer rows.Close()

	var judgehosts []*Judgehost
	for rows.Next() {
		var j Judgehost
		var registeredAt, lastPing pgtype.Timestamp
		var languages []string

		err := rows.Scan(&j.ID, &j.QueueName, &languages, &j.MaxConcurrent, &j.MemoryLimit, &j.TimeLimit,
			&j.Interactive, &j.Special, &j.Status, &j.CurrentJobID, &lastPing, &j.ActiveJobs,
			&j.CompletedJobs, &registeredAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan judgehost: %w", err)
		}

		j.Languages = languages
		if registeredAt.Valid {
			j.RegisteredAt = registeredAt.Time
		}
		if lastPing.Valid {
			j.LastPing = lastPing.Time
		}
		judgehosts = append(judgehosts, &j)
	}

	return judgehosts, total, nil
}

// Deregister removes a judgehost
func (s *JudgehostStore) Deregister(ctx context.Context, id string, force bool) (int32, error) {
	// Check if judgehost is busy
	var status string
	var activeJobs int32
	err := s.db.QueryRow(ctx, `
		SELECT status, active_jobs FROM judgehosts WHERE id = $1
	`, id).Scan(&status, &activeJobs)
	if err != nil {
		return 0, fmt.Errorf("failed to check judgehost status: %w", err)
	}

	if status == "busy" && activeJobs > 0 && !force {
		return 0, fmt.Errorf("judgehost is busy with %d active jobs, use force=true to deregister", activeJobs)
	}

	// Get count of pending tasks to reassign
	var pendingTasks int32
	// This would need to query the task queue - placeholder for now

	// Remove from PostgreSQL
	_, err = s.db.Exec(ctx, `DELETE FROM judgehosts WHERE id = $1`, id)
	if err != nil {
		return 0, fmt.Errorf("failed to deregister judgehost: %w", err)
	}

	// Remove from Redis
	judgehostKey := judgehostKeyPrefix + id
	s.redis.Del(ctx, judgehostKey)
	s.redis.SRem(ctx, judgehostListKey, id)

	return pendingTasks, nil
}

// SetCurrentJob sets the current job for a judgehost
func (s *JudgehostStore) SetCurrentJob(ctx context.Context, id string, jobID string) error {
	judgehostKey := judgehostKeyPrefix + id
	return s.redis.HSet(ctx, judgehostKey, "current_job_id", jobID).Err()
}

// IncrementActiveJobs increments the active job count
func (s *JudgehostStore) IncrementActiveJobs(ctx context.Context, id string) error {
	judgehostKey := judgehostKeyPrefix + id
	return s.redis.HIncrBy(ctx, judgehostKey, "active_jobs", 1).Err()
}

// DecrementActiveJobs decrements the active job count
func (s *JudgehostStore) DecrementActiveJobs(ctx context.Context, id string) error {
	judgehostKey := judgehostKeyPrefix + id
	return s.redis.HIncrBy(ctx, judgehostKey, "active_jobs", -1).Err()
}

// GetPendingTasksCount gets the count of pending tasks in a judgehost's queue
func (s *JudgehostStore) GetPendingTasksCount(ctx context.Context, queueName string) (int32, error) {
	// This would query Asynq queue stats - placeholder for now
	// Would use: asynqInspector.GetTaskInfo(queueName, ...)
	return 0, nil
}