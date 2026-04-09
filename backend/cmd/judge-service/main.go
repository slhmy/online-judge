package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/online-judge/backend/gen/go/judge/v1"
	"github.com/online-judge/backend/internal/judge/service"
	"github.com/online-judge/backend/internal/judge/store"
	"github.com/online-judge/backend/internal/pkg/config"
	"github.com/online-judge/backend/internal/queue"
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

	// Redis (for cache + heartbeat status)
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Asynq client (for task queue)
	asynqClient := queue.NewAsynqClient(cfg.RedisURL)
	defer asynqClient.Close()

	// Create service
	judgehostStore := store.NewJudgehostStore(dbpool, rdb)
	judgeService := service.NewJudgeService(judgehostStore, asynqClient)

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterJudgeServiceServer(s, judgeService)
	reflection.Register(s)

	log.Printf("Judge Service listening on port %s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}