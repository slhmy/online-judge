package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbProblem "github.com/online-judge/backend/gen/go/problem/v1"
	pbSubmission "github.com/online-judge/backend/gen/go/submission/v1"
	pbContest "github.com/online-judge/backend/gen/go/contest/v1"
	"github.com/online-judge/bff/internal/config"
	"github.com/online-judge/bff/internal/handler"
	"github.com/online-judge/bff/internal/middleware"
	"github.com/online-judge/bff/internal/sse"
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

	// gRPC clients
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	problemConn, err := grpc.Dial(cfg.ProblemServiceAddr, opts...)
	if err != nil {
		log.Fatalf("Failed to connect to problem service: %v", err)
	}
	defer problemConn.Close()

	submissionConn, err := grpc.Dial(cfg.SubmissionServiceAddr, opts...)
	if err != nil {
		log.Fatalf("Failed to connect to submission service: %v", err)
	}
	defer submissionConn.Close()

	contestConn, err := grpc.Dial(cfg.ContestServiceAddr, opts...)
	if err != nil {
		log.Fatalf("Failed to connect to contest service: %v", err)
	}
	defer contestConn.Close()

	// Create gRPC clients
	problemClient := pbProblem.NewProblemServiceClient(problemConn)
	submissionClient := pbSubmission.NewSubmissionServiceClient(submissionConn)
	contestClient := pbContest.NewContestServiceClient(contestConn)

	// Create SSE hub (manages real-time updates via Server-Sent Events)
	sseHub := sse.NewHub(rdb)

	// Create auth middleware
	authMiddleware := middleware.NewAuth("")

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
	problemHandler := handler.NewProblemHandler(problemClient)
	submissionHandler := handler.NewSubmissionHandler(submissionClient)
	contestHandler := handler.NewContestHandler(contestClient)
	authHandler := handler.NewAuthHandler(cfg.IdentraGRPCHost, cfg.IdentraHTTPHost, cfg.DatabaseURL, cfg.AdminEmail)
	adminHandler := handler.NewAdminHandler(cfg.DatabaseURL)
	sseHandler := handler.NewSSEHandler(sseHub)
	internalHandler := handler.NewInternalHandler(submissionClient, problemClient, rdb)

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
		w.Write([]byte("OK"))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		authHandler.RegisterRoutes(r)

		// Problems (public)
		r.Get("/problems", problemHandler.ListProblems)
		r.Get("/problems/{id}", problemHandler.GetProblem)
		r.Get("/languages", problemHandler.ListLanguages)

			// Problem admin routes (protected)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireAuth)
				r.Post("/problems", problemHandler.CreateProblem)
				r.Put("/problems/{id}", problemHandler.UpdateProblem)
				r.Delete("/problems/{id}", problemHandler.DeleteProblem)
			})

		// Submissions (mixed - some public, some protected)
		r.With(rateLimiter.SubmissionRateLimitMiddleware).Post("/submissions", submissionHandler.Create)
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

		// Admin routes (protected)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			adminHandler.RegisterRoutes(r)
			// Rejudge submission (admin only)
			r.Post("/submissions/{id}/rejudge", submissionHandler.Rejudge)
		})
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