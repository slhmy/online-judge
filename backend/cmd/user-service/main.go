package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/online-judge/backend/gen/go/user/v1"
	"github.com/online-judge/backend/internal/pkg/config"
	"github.com/online-judge/backend/internal/user/service"
	"github.com/online-judge/backend/internal/user/store"
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

	// Create service
	userStore := store.NewUserStore(dbpool)
	userService := service.NewUserService(userStore)

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, userService)
	reflection.Register(s)

	log.Printf("User Service listening on port %s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
