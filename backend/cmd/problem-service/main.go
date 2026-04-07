package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/online-judge/backend/gen/go/problem/v1"
	"github.com/online-judge/backend/internal/pkg/config"
	"github.com/online-judge/backend/internal/problem/service"
	"github.com/online-judge/backend/internal/problem/store"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// PostgreSQL connection
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Create store and service
	problemStore := store.NewProblemStore(dbpool)
	problemService := service.NewProblemService(problemStore, rdb)

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProblemServiceServer(s, problemService)
	reflection.Register(s)

	log.Printf("Problem Service listening on port %s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
