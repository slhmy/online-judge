package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbJudge "github.com/slhmy/online-judge/gen/go/judge/v1"
	pbProblem "github.com/slhmy/online-judge/gen/go/problem/v1"
	pbSubmission "github.com/slhmy/online-judge/gen/go/submission/v1"
	"github.com/slhmy/online-judge/judge/internal/config"
	"github.com/slhmy/online-judge/judge/internal/worker"
)

// JudgehostClient handles communication with Judge Service via gRPC
type JudgehostClient struct {
	client pbJudge.JudgeServiceClient
	conn   *grpc.ClientConn
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

	// Create Judge Service gRPC client
	conn, err := grpc.NewClient(cfg.JudgeServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Judge Service: %v", err)
	}
	judgeClient := &JudgehostClient{
		client: pbJudge.NewJudgeServiceClient(conn),
		conn:   conn,
	}
	submissionClient := pbSubmission.NewSubmissionServiceClient(conn)
	problemClient := pbProblem.NewProblemServiceClient(conn)
	defer func() { _ = conn.Close() }()

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
		startAsynqWorker(cfg, rdb, ctx, queueName, judgeClient, completedJobsChan, submissionClient, problemClient)
	case "legacy":
		// Legacy mode only - no registration, no heartbeat
		startLegacyWorker(cfg, rdb, ctx, submissionClient, problemClient, judgeClient.client)
	case "both":
		// Both modes during migration
		startAsynqWorker(cfg, rdb, ctx, queueName, judgeClient, completedJobsChan, submissionClient, problemClient)
		startLegacyWorker(cfg, rdb, ctx, submissionClient, problemClient, judgeClient.client)
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
						log.Printf("Heartbeat acknowledged: pending=%d, assigned=%v", resp.PendingTasks, resp.AssignedTaskIds)
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
func startAsynqWorker(cfg *config.Config, rdb *redis.Client, ctx context.Context, queueName string, judgeClient *JudgehostClient, completedJobsChan chan<- string, submissionClient pbSubmission.SubmissionServiceClient, problemClient pbProblem.ProblemServiceClient) {
	// Create asynq handler
	handler := worker.NewAsynqHandler(cfg, rdb, submissionClient, problemClient, judgeClient.client)

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
func startLegacyWorker(cfg *config.Config, rdb *redis.Client, ctx context.Context, submissionClient pbSubmission.SubmissionServiceClient, problemClient pbProblem.ProblemServiceClient, judgeServiceClient pbJudge.JudgeServiceClient) {
	// Create legacy worker
	legacyWorker := worker.NewJudgeWorker(cfg.JudgehostID, cfg, rdb, submissionClient, problemClient, judgeServiceClient)

	log.Printf("Starting legacy worker (Redis sorted set queue)")

	// Run worker in goroutine
	go func() {
		if err := legacyWorker.Run(ctx); err != nil {
			log.Printf("Legacy worker error: %v", err)
		}
	}()
}

// Register registers the judgehost with Judge Service via gRPC
func (c *JudgehostClient) Register(ctx context.Context, judgehostID string) (string, error) {
	resp, err := c.client.RegisterJudgehost(ctx, &pbJudge.RegisterJudgehostRequest{
		JudgehostId: judgehostID,
		Capabilities: &pbJudge.JudgehostCapabilities{
			Languages:           []string{"cpp17", "cpp20", "python3", "go", "java", "rust"},
			MaxConcurrentJobs:   5,
			SupportsInteractive: true,
			SupportsSpecial:     true,
		},
	})
	if err != nil {
		return "", err
	}
	return resp.QueueName, nil
}

// Heartbeat sends heartbeat to Judge Service via gRPC
func (c *JudgehostClient) Heartbeat(ctx context.Context, judgehostID, status, currentJobID string, completedJobIDs []string) (*pbJudge.HeartbeatResponse, error) {
	return c.client.Heartbeat(ctx, &pbJudge.HeartbeatRequest{
		JudgehostId:     judgehostID,
		Status:          pbJudge.JudgehostStatus_JUDGEHOST_STATUS_IDLE,
		CurrentJobId:    currentJobID,
		ActiveJobs:      0,
		CompletedJobIds: completedJobIDs,
	})
}

// Deregister deregisters the judgehost from Judge Service via gRPC
func (c *JudgehostClient) Deregister(ctx context.Context, judgehostID string) error {
	_, err := c.client.DeregisterJudgehost(ctx, &pbJudge.DeregisterJudgehostRequest{
		JudgehostId: judgehostID,
	})
	return err
}

// asynqLogger implements asynq.Logger interface
type asynqLogger struct{}

func (l asynqLogger) Debug(args ...interface{}) { log.Println(args...) }
func (l asynqLogger) Info(args ...interface{})  { log.Println(args...) }
func (l asynqLogger) Warn(args ...interface{})  { log.Println(args...) }
func (l asynqLogger) Error(args ...interface{}) { log.Println(args...) }
func (l asynqLogger) Fatal(args ...interface{}) { log.Fatalln(args...) }
