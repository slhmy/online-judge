package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/slhmy/online-judge/bff/internal/cache"
	"github.com/slhmy/online-judge/bff/internal/config"
	"github.com/slhmy/online-judge/bff/internal/handler"
	"github.com/slhmy/online-judge/bff/internal/middleware"
	"github.com/slhmy/online-judge/bff/internal/sse"
	pbContest "github.com/slhmy/online-judge/gen/go/contest/v1"
	pbJudge "github.com/slhmy/online-judge/gen/go/judge/v1"
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

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	// Create cache service
	cacheConfig := cache.Config{
		Enabled:       cfg.CacheEnabled,
		ProblemTTL:    time.Duration(cfg.CacheProblemTTL) * time.Second,
		ContestTTL:    time.Duration(cfg.CacheContestTTL) * time.Second,
		ScoreboardTTL: time.Duration(cfg.CacheScoreboardTTL) * time.Second,
	}
	cacheService := cache.NewService(rdb, cacheConfig)

	// gRPC clients
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	//nolint:staticcheck // grpc.Dial is deprecated but will be supported throughout 1.x
	backendConn, err := grpc.Dial(cfg.BackendServiceAddr, opts...)
	if err != nil {
		log.Fatalf("Failed to connect to backend service: %v", err)
	}
	defer func() { _ = backendConn.Close() }()

	// Create gRPC clients
	problemClient := pbProblem.NewProblemServiceClient(backendConn)
	submissionClient := pbSubmission.NewSubmissionServiceClient(backendConn)
	contestClient := pbContest.NewContestServiceClient(backendConn)
	userClient := pbUser.NewUserServiceClient(backendConn)
	judgeClient := pbJudge.NewJudgeServiceClient(backendConn)

	// Create SSE hub (manages real-time updates via Server-Sent Events)
	sseHub := sse.NewHub(rdb)

	// Create auth middleware
	authMiddleware := middleware.NewAuth("", rdb, cfg.IdentraGRPCHost)

	// Create rate limiter
	rateLimitConfig := middleware.RateLimitConfig{
		Enabled:                     cfg.RateLimitEnabled,
		RequestsPerMinute:           cfg.RateLimitRequestsPerMinute,
		BurstSize:                   cfg.RateLimitBurstSize,
		SubmissionRequestsPerMinute: cfg.RateLimitSubmissionRequestsPerMin,
		SubmissionBurstSize:         cfg.RateLimitSubmissionBurstSize,
		IPRequestsPerMinute:         cfg.RateLimitIPRequestsPerMinute,
		IPBurstSize:                 cfg.RateLimitIPBurstSize,
	}
	rateLimiter := middleware.NewRateLimiter(rdb, rateLimitConfig)

	// Create handlers
	problemHandler := handler.NewProblemHandler(problemClient, cacheService)
	submissionHandler := handler.NewSubmissionHandler(submissionClient)
	contestHandler := handler.NewContestHandler(contestClient, cacheService)
	userHandler := handler.NewUserHandler(userClient)
	authHandler := handler.NewAuthHandler(
		cfg.IdentraGRPCHost,
		userClient,
		cfg.AdminEmail,
		cfg.OAuthRedirectURL,
		rdb,
	)
	adminHandler := handler.NewAdminHandler(judgeClient, userClient)
	sseHandler := handler.NewSSEHandler(sseHub)
	internalHandler := handler.NewInternalHandler(submissionClient, problemClient, rdb, cacheService, cfg.DatabaseURL)
	testRunHandler := handler.NewTestRunHandler(problemClient)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	// General rate limiting middleware (per-user and per-IP)
	r.Use(rateLimiter.RateLimitMiddleware)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		authHandler.RegisterRoutes(r)

		// Problems (public)
		r.Get("/problems", problemHandler.ListProblems)
		r.Get("/problems/{id}", problemHandler.GetProblem)
		r.Get("/problems/{id}/statement", problemHandler.GetProblemStatement)
		r.Get("/languages", problemHandler.ListLanguages)

		// Problem admin routes (protected)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.Post("/problems", problemHandler.CreateProblem)
			r.Put("/problems/{id}", problemHandler.UpdateProblem)
			r.Delete("/problems/{id}", problemHandler.DeleteProblem)
			r.Put("/problems/{id}/statement", problemHandler.SetProblemStatement)

			// Test case management
			r.Get("/problems/{id}/testcases", problemHandler.ListTestCases)
			r.Post("/problems/{id}/testcases", problemHandler.CreateTestCase)
			r.Post("/problems/{id}/testcases/batch", problemHandler.BatchUploadTestCases)
			r.Put("/testcases/{id}", problemHandler.UpdateTestCase)
			r.Delete("/testcases/{id}", problemHandler.DeleteTestCase)
			r.Put("/testcases/{id}/toggle-sample", problemHandler.ToggleTestCaseSample)
		})

		// Submissions (mixed - some public, some protected)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.With(rateLimiter.SubmissionRateLimitMiddleware).Post("/submissions", submissionHandler.Create)
		})
		r.Get("/submissions", submissionHandler.List)
		r.Get("/submissions/{id}", submissionHandler.Get)
		r.Get("/submissions/{id}/judging", submissionHandler.GetJudging)
		r.Get("/submissions/{id}/runs", submissionHandler.GetRuns)

		// Contests (public)
		r.Get("/contests", contestHandler.List)
		r.Get("/contests/{id}", contestHandler.Get)
		r.Get("/contests/{id}/problems", contestHandler.GetProblems)
		r.Get("/contests/{id}/scoreboard", contestHandler.GetScoreboard)
		r.Post("/contests/{id}/register", contestHandler.Register)

		// Users (public profile stats, protected update)
		r.Get("/users/{id}/profile", userHandler.GetProfile)
		r.Get("/users/{id}/stats", userHandler.GetStats)
		r.Get("/users/{id}/submissions", userHandler.GetSubmissions)

		// User profile update (protected)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.Put("/users/{id}/profile", userHandler.UpdateProfile)
		})

		// Admin routes (protected)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			adminHandler.RegisterRoutes(r)
			// Rejudge submission (admin only)
			r.Post("/submissions/{id}/rejudge", submissionHandler.Rejudge)
		})

		// Test runs (public - sample test case execution)
		testRunHandler.RegisterRoutes(r)
	})

	// SSE endpoint for real-time submission updates
	sseHandler.RegisterRoutes(r)

	// Internal API routes (for judge daemon)
	internalHandler.RegisterRoutes(r)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down BFF...")
		sseHub.Stop()
		os.Exit(0)
	}()

	log.Printf("BFF listening on port %s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, r); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
