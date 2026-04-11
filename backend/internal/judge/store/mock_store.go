package store

import "context"

// JudgehostStoreInterface defines the interface for judgehost storage
type JudgehostStoreInterface interface {
	Register(ctx context.Context, id string, languages []string, maxConcurrent int32,
		memoryLimit int64, timeLimit float64, interactive, special bool, extra map[string]string) (string, error)
	UpdateHeartbeat(ctx context.Context, id string, status string, currentJobID string, activeJobs int32, completedJobIDs []string) error
	GetAvailableJudgehosts(ctx context.Context, languageID string, interactive, special bool) ([]*Judgehost, error)
	GetByID(ctx context.Context, id string) (*Judgehost, error)
	List(ctx context.Context, statusFilter string, page, pageSize int32) ([]*Judgehost, int32, error)
	Deregister(ctx context.Context, id string, force bool) (int32, error)
	SetCurrentJob(ctx context.Context, id string, jobID string) error
	IncrementActiveJobs(ctx context.Context, id string) error
	DecrementActiveJobs(ctx context.Context, id string) error
	GetPendingTasksCount(ctx context.Context, queueName string) (int32, error)
	PeekPendingTaskIDs(ctx context.Context, queueName string, limit int64) ([]string, error)
}