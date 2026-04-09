package queue

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const (
	// TaskTypeJudge is the task type for judge jobs
	TaskTypeJudge = "task:type_judge"
)

// JudgeJobPayload is the payload for judge tasks
type JudgeJobPayload struct {
	SubmissionID string `json:"submission_id"`
	ProblemID    string `json:"problem_id"`
	Language     string `json:"language"`
	Priority     int    `json:"priority"`
}

// AsynqClient wraps asynq client for task enqueueing
type AsynqClient struct {
	client *asynq.Client
}

// NewAsynqClient creates a new asynq client
func NewAsynqClient(redisAddr string) *AsynqClient {
	return &AsynqClient{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

// EnqueueJudgeTask enqueues a judge task with given options
func (c *AsynqClient) EnqueueJudgeTask(payload *JudgeJobPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TaskTypeJudge, data)

	info, err := c.client.Enqueue(
		task,
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Minute),
		asynq.Queue("default"),
	)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// Close closes the asynq client
func (c *AsynqClient) Close() error {
	return c.client.Close()
}