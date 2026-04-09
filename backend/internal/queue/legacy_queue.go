package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// LegacyQueueKey is the Redis key for the legacy judge queue
const LegacyQueueKey = "judge:queue"

// LegacyQueue manages the legacy Redis sorted set queue for migration
type LegacyQueue struct {
	redis *redis.Client
}

// NewLegacyQueue creates a new legacy queue client
func NewLegacyQueue(redis *redis.Client) *LegacyQueue {
	return &LegacyQueue{
		redis: redis,
	}
}

// EnqueueJudgeJob adds a judge job to the legacy Redis sorted set queue
// This is used for dual-write during migration period
func (q *LegacyQueue) EnqueueJudgeJob(ctx context.Context, payload *JudgeJobPayload) error {
	job := &LegacyJudgeJob{
		SubmissionID: payload.SubmissionID,
		ProblemID:    payload.ProblemID,
		Language:     payload.Language,
		Priority:     payload.Priority,
		SubmitTime:   time.Now(),
	}

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Use Redis sorted set for priority queue
	// Lower score = higher priority
	// Priority is inverted so lower values are processed first
	// Submit time breaks ties (earlier = lower score = higher priority)
	score := float64(job.Priority*1000000) - float64(job.SubmitTime.Unix())

	return q.redis.ZAdd(ctx, LegacyQueueKey, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

// LegacyJudgeJob represents a judging job in the legacy queue format
type LegacyJudgeJob struct {
	SubmissionID string    `json:"submission_id"`
	ProblemID    string    `json:"problem_id"`
	Language     string    `json:"language"`
	Priority     int       `json:"priority"`
	SubmitTime   time.Time `json:"submit_time"`
	RejudgeID    string    `json:"rejudge_id,omitempty"`
}

// QueueLength returns the number of jobs in the legacy queue
func (q *LegacyQueue) QueueLength(ctx context.Context) (int64, error) {
	return q.redis.ZCard(ctx, LegacyQueueKey).Result()
}