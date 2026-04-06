package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"

	"github.com/online-judge/judge/internal/config"
	"github.com/online-judge/judge/internal/queue"
	"github.com/online-judge/judge/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Redis connection (for queue + cache + pub/sub)
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Connected to Redis at %s", cfg.RedisURL)

	// Create judge queue client
	judgeQueue := queue.NewJudgeQueue(rdb, cfg.OrchestratorURL)

	// Create and start worker
	w := worker.NewJudgeWorker(cfg.JudgehostID, cfg, judgeQueue, rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down judge daemon...")
		cancel()
	}()

	log.Printf("Starting judge daemon: %s (orchestrator: %s)", cfg.JudgehostID, cfg.OrchestratorURL)
	if err := w.Run(ctx); err != nil {
		log.Fatalf("Judge worker error: %v", err)
	}
	log.Println("Judge daemon stopped")
}