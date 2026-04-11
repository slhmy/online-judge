package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/slhmy/online-judge/judge/internal/config"
	"github.com/slhmy/online-judge/judge/internal/queue"
	"github.com/slhmy/online-judge/judge/internal/worker"
)

// JudgehostClient handles communication with Judge Service
type JudgehostClient struct {
	judgeServiceURL string
	httpClient      *http.Client
}

// RegisterResponse is the response from RegisterJudgehost API
type RegisterResponse struct {
	JudgehostID string `json:"judgehost_id"`
	QueueName   string `json:"queue_name"`
	Status      string `json:"status"`
}

// HeartbeatRequest is the request body for heartbeat API
type HeartbeatRequest struct {
	JudgehostID     string   `json:"judgehost_id"`
	Status          string   `json:"status"`
	CurrentJobID    string   `json:"current_job_id,omitempty"`
	ActiveJobs      int      `json:"active_jobs"`
	CompletedJobIDs []string `json:"completed_job_ids,omitempty"`
}

// HeartbeatResponse is the response from heartbeat API
type HeartbeatResponse struct {
	Acknowledged    bool     `json:"acknowledged"`
	PendingTasks    int      `json:"pending_tasks"`
	AssignedTaskIDs []string `json:"assigned_task_ids,omitempty"`
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Judge daemon starting with ID: %s, queue_mode: %s", cfg.JudgehostID, cfg.QueueMode)

	// Redis connection (for asynq + cache + legacy queue)
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Connected to Redis at %s", cfg.RedisURL)

	// Create Judge Service client
	judgeClient := &JudgehostClient{
		judgeServiceURL: cfg.JudgeServiceURL,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}

	// Register judgehost (only for asynq mode)
	var queueName string
	if cfg.QueueMode != "legacy" {
		queueName, err = judgeClient.Register(ctx, cfg.JudgehostID)
		if err != nil {
			log.Fatalf("Failed to register judgehost: %v", err)
		}
		log.Printf("Registered judgehost %s, assigned queue: %s", cfg.JudgehostID, queueName)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Track completed jobs for heartbeat
	completedJobs := make([]string, 0)
	completedJobsChan := make(chan string, 100)

	// Start workers based on queue mode
	switch cfg.QueueMode {
	case "asynq":
		// Asynq mode only
		startAsynqWorker(cfg, rdb, ctx, queueName, judgeClient, completedJobsChan)
	case "legacy":
		// Legacy mode only - no registration, no heartbeat
		startLegacyWorker(cfg, rdb, ctx)
	case "both":
		// Both modes during migration
		startAsynqWorker(cfg, rdb, ctx, queueName, judgeClient, completedJobsChan)
		startLegacyWorker(cfg, rdb, ctx)
	default:
		log.Fatalf("Unknown queue mode: %s", cfg.QueueMode)
	}

	// Start heartbeat goroutine (only for asynq mode)
	if cfg.QueueMode != "legacy" {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.HeartbeatInterval) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case jobID := <-completedJobsChan:
					completedJobs = append(completedJobs, jobID)
					// Keep only recent completed jobs (last 100)
					if len(completedJobs) > 100 {
						completedJobs = completedJobs[len(completedJobs)-100:]
					}
				case <-ticker.C:
					// Send heartbeat
					// Get current job from asynq handler
					var currentJobID string
					var status string
					// Note: We'll need to track this from the worker
					status = "idle"

					resp, err := judgeClient.Heartbeat(ctx, cfg.JudgehostID, status, currentJobID, completedJobs)
					if err != nil {
						log.Printf("Heartbeat failed: %v", err)
					} else {
						log.Printf("Heartbeat acknowledged: pending=%d, assigned=%v", resp.PendingTasks, resp.AssignedTaskIDs)
						// Clear completed jobs after successful heartbeat
						completedJobs = make([]string, 0)
					}
				}
			}
		}()
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down judge daemon...")

	// Graceful shutdown
	cancel()

	// Deregister judgehost (only for asynq mode)
	if cfg.QueueMode != "legacy" {
		if err := judgeClient.Deregister(ctx, cfg.JudgehostID); err != nil {
			log.Printf("Failed to deregister judgehost: %v", err)
		}
	}

	log.Println("Judge daemon stopped")
}

// startAsynqWorker starts the asynq-based worker
func startAsynqWorker(cfg *config.Config, rdb *redis.Client, ctx context.Context, queueName string, judgeClient *JudgehostClient, completedJobsChan chan<- string) {
	// Create asynq handler
	handler := worker.NewAsynqHandler(cfg, rdb)

	// Create asynq server (listen on dedicated queue)
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisURL},
		asynq.Config{
			// Only process tasks from this judgehost's queue
			Queues: map[string]int{
				queueName: 10, // High priority for own queue
				"default": 1,  // Low priority for fallback
			},
			// Error handling
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("Task %s failed: %v", task.Type(), err)
			}),
			// Logging
			Logger: asynqLogger{},
		},
	)

	log.Printf("Starting asynq server, listening on queue: %s", queueName)
	if err := server.Start(handler); err != nil {
		log.Fatalf("Failed to start asynq server: %v", err)
	}

	// Handle shutdown in goroutine
	go func() {
		<-ctx.Done()
		server.Stop()
	}()
}

// startLegacyWorker starts the legacy Redis sorted set worker
func startLegacyWorker(cfg *config.Config, rdb *redis.Client, ctx context.Context) {
	// Create legacy queue client
	judgeQueue := queue.NewJudgeQueue(rdb, cfg.OrchestratorURL)

	// Create legacy worker
	legacyWorker := worker.NewJudgeWorker(cfg.JudgehostID, cfg, judgeQueue, rdb)

	log.Printf("Starting legacy worker (Redis sorted set queue)")

	// Run worker in goroutine
	go func() {
		if err := legacyWorker.Run(ctx); err != nil {
			log.Printf("Legacy worker error: %v", err)
		}
	}()
}

// Register registers the judgehost with Judge Service
func (c *JudgehostClient) Register(ctx context.Context, judgehostID string) (string, error) {
	url := fmt.Sprintf("%s/internal/judge/judgehosts", c.judgeServiceURL)

	body := map[string]interface{}{
		"judgehost_id": judgehostID,
		"capabilities": map[string]interface{}{
			"languages":            []string{"cpp17", "cpp20", "python3", "go", "java", "rust"},
			"max_concurrent_jobs":  5,
			"supports_interactive": true,
			"supports_special":     true,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to register: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := json.Marshal(resp.Body)
		return "", fmt.Errorf("register failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode register response: %w", err)
	}

	return result.QueueName, nil
}

// Heartbeat sends heartbeat to Judge Service
func (c *JudgehostClient) Heartbeat(ctx context.Context, judgehostID, status, currentJobID string, completedJobIDs []string) (*HeartbeatResponse, error) {
	url := fmt.Sprintf("%s/internal/judge/judgehosts/%s/heartbeat", c.judgeServiceURL, judgehostID)

	body := HeartbeatRequest{
		JudgehostID:     judgehostID,
		Status:          status,
		CurrentJobID:    currentJobID,
		ActiveJobs:      0, // Will be updated later
		CompletedJobIDs: completedJobIDs,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := json.Marshal(resp.Body)
		return nil, fmt.Errorf("heartbeat failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode heartbeat response: %w", err)
	}

	return &result, nil
}

// Deregister deregisters the judgehost from Judge Service
func (c *JudgehostClient) Deregister(ctx context.Context, judgehostID string) error {
	url := fmt.Sprintf("%s/internal/judge/judgehosts/%s", c.judgeServiceURL, judgehostID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deregister failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deregister failed: status %d", resp.StatusCode)
	}

	return nil
}

// asynqLogger implements asynq.Logger interface
type asynqLogger struct{}

func (l asynqLogger) Debug(args ...interface{}) { log.Println(args...) }
func (l asynqLogger) Info(args ...interface{})  { log.Println(args...) }
func (l asynqLogger) Warn(args ...interface{})  { log.Println(args...) }
func (l asynqLogger) Error(args ...interface{}) { log.Println(args...) }
func (l asynqLogger) Fatal(args ...interface{}) { log.Fatalln(args...) }
