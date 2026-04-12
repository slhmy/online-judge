package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	contestService "github.com/slhmy/online-judge/backend/internal/contest/service"
	contestStore "github.com/slhmy/online-judge/backend/internal/contest/store"
	judgeService "github.com/slhmy/online-judge/backend/internal/judge/service"
	judgeStore "github.com/slhmy/online-judge/backend/internal/judge/store"
	notificationService "github.com/slhmy/online-judge/backend/internal/notification/service"
	"github.com/slhmy/online-judge/backend/internal/pkg/config"
	"github.com/slhmy/online-judge/backend/internal/pkg/middleware"
	"github.com/slhmy/online-judge/backend/internal/pkg/migration"
	problemService "github.com/slhmy/online-judge/backend/internal/problem/service"
	problemStore "github.com/slhmy/online-judge/backend/internal/problem/store"
	"github.com/slhmy/online-judge/backend/internal/queue"
	submissionService "github.com/slhmy/online-judge/backend/internal/submission/service"
	submissionStore "github.com/slhmy/online-judge/backend/internal/submission/store"
	userService "github.com/slhmy/online-judge/backend/internal/user/service"
	userStore "github.com/slhmy/online-judge/backend/internal/user/store"
	"github.com/slhmy/online-judge/backend/migrations"
	pbContest "github.com/slhmy/online-judge/gen/go/contest/v1"
	pbJudge "github.com/slhmy/online-judge/gen/go/judge/v1"
	pbNotification "github.com/slhmy/online-judge/gen/go/notification/v1"
	pbProblem "github.com/slhmy/online-judge/gen/go/problem/v1"
	pbSubmission "github.com/slhmy/online-judge/gen/go/submission/v1"
	pbUser "github.com/slhmy/online-judge/gen/go/user/v1"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// PostgreSQL connection (shared by most services)
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Run auto-migrations
	if err := migration.Run(context.Background(), dbpool, migrations.FS); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Redis connection (shared by most services)
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Asynq client (for submission and judge task queues)
	asynqClient := queue.NewAsynqClient(cfg.RedisURL)
	defer func() { _ = asynqClient.Close() }()

	// Legacy queue (for migration dual-write)
	legacyQueue := queue.NewLegacyQueue(rdb)

	// --- Initialize all services ---

	// Problem service
	pStore := problemStore.NewProblemStore(dbpool)
	execStore := problemStore.NewExecutableStore(dbpool)
	pService := problemService.NewProblemService(pStore, execStore, rdb)

	// Submission service
	sStore := submissionStore.NewSubmissionStore(dbpool)
	sService := submissionService.NewSubmissionService(
		sStore,
		rdb,
		asynqClient,
		legacyQueue,
		cfg.UseAsynqQueue,
		cfg.UseLegacyQueue,
	)

	// Contest service
	cStore := contestStore.NewContestStore(dbpool)
	cService := contestService.NewContestService(cStore, rdb)

	// User service
	uStore := userStore.NewUserStore(dbpool)
	uService := userService.NewUserService(uStore)

	// Notification service (Redis-only, no PostgreSQL)
	nService := notificationService.NewNotificationService(rdb)

	// Judge service (includes rejudge functionality)
	jStore := judgeStore.NewJudgehostStore(dbpool, rdb)
	rjStore := judgeStore.NewRejudgeStore(dbpool)
	jService := judgeService.NewJudgeService(jStore, asynqClient, rjStore, sStore, cStore, rdb)

	// --- Start unified gRPC server ---
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	jwtInterceptor, err := middleware.NewJWTInterceptor(cfg.IdentraJWKSURL, dbpool)
	if err != nil {
		log.Fatalf("Failed to initialize JWT interceptor: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(jwtInterceptor.Unary()))

	// Register all services on the same gRPC server
	pbProblem.RegisterProblemServiceServer(s, pService)
	pbSubmission.RegisterSubmissionServiceServer(s, sService)
	pbContest.RegisterContestServiceServer(s, cService)
	pbUser.RegisterUserServiceServer(s, uService)
	pbNotification.RegisterNotificationServiceServer(s, nService)
	pbJudge.RegisterJudgeServiceServer(s, jService)

	reflection.Register(s)

	log.Printf("Server listening on port %s (all services unified)", cfg.GRPCPort)
	log.Printf("  - Problem, Submission, Contest, User, Notification, Judge")
	log.Printf("  - Queue config: asynq=%v, legacy=%v", cfg.UseAsynqQueue, cfg.UseLegacyQueue)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
