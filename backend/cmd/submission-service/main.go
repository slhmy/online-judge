package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/online-judge/backend/gen/go/submission/v1"
	"github.com/online-judge/backend/internal/pkg/config"
	"github.com/online-judge/backend/internal/queue"
	"github.com/online-judge/backend/internal/submission/service"
	"github.com/online-judge/backend/internal/submission/store"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// PostgreSQL
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Redis (for cache + pub/sub)
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Asynq client (for task queue)
	asynqClient := queue.NewAsynqClient(cfg.RedisURL)
	defer asynqClient.Close()

	// Legacy queue (for migration dual-write)
	legacyQueue := queue.NewLegacyQueue(rdb)

	// Create service with queue configuration
	submissionStore := store.NewSubmissionStore(dbpool)
	submissionService := service.NewSubmissionService(
		submissionStore,
		rdb,
		asynqClient,
		legacyQueue,
		cfg.UseAsynqQueue,
		cfg.UseLegacyQueue,
	)

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterSubmissionServiceServer(s, submissionService)
	reflection.Register(s)

	log.Printf("Submission Service listening on port %s (asynq=%v, legacy=%v)", cfg.GRPCPort, cfg.UseAsynqQueue, cfg.UseLegacyQueue)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
