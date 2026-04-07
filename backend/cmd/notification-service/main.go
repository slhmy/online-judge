package main

import (
	"log"
	"net"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/online-judge/backend/gen/go/notification/v1"
	"github.com/online-judge/backend/internal/notification/service"
	"github.com/online-judge/backend/internal/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	notificationService := service.NewNotificationService(rdb)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterNotificationServiceServer(s, notificationService)
	reflection.Register(s)

	log.Printf("Notification Service listening on port %s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
