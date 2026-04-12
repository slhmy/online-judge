package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	// Server
	HTTPPort string `mapstructure:"http_port"`

	// Redis
	RedisURL string `mapstructure:"redis_url"`

	// Database
	DatabaseURL string `mapstructure:"database_url"`

	// Backend Services (unified gRPC entrypoint)
	BackendServiceAddr string `mapstructure:"backend_service_addr"`

	// Auth
	IdentraGRPCHost string `mapstructure:"identra_grpc_host"`
	AdminEmail      string `mapstructure:"admin_email"`

	// Rate Limiting
	RateLimitEnabled                  bool `mapstructure:"rate_limit_enabled"`
	RateLimitRequestsPerMinute        int  `mapstructure:"rate_limit_requests_per_minute"`
	RateLimitBurstSize                int  `mapstructure:"rate_limit_burst_size"`
	RateLimitSubmissionRequestsPerMin int  `mapstructure:"rate_limit_submission_requests_per_minute"`
	RateLimitSubmissionBurstSize      int  `mapstructure:"rate_limit_submission_burst_size"`
	RateLimitIPRequestsPerMinute      int  `mapstructure:"rate_limit_ip_requests_per_minute"`
	RateLimitIPBurstSize              int  `mapstructure:"rate_limit_ip_burst_size"`

	// Caching
	CacheEnabled       bool `mapstructure:"cache_enabled"`
	CacheProblemTTL    int  `mapstructure:"cache_problem_ttl"`    // seconds
	CacheContestTTL    int  `mapstructure:"cache_contest_ttl"`    // seconds
	CacheScoreboardTTL int  `mapstructure:"cache_scoreboard_ttl"` // seconds

	// OAuth Configuration
	OAuthRedirectURL string `mapstructure:"oauth_redirect_url"`
}

func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("http_port", "8080")
	v.SetDefault("redis_url", "localhost:6379")
	v.SetDefault("database_url", "postgres://oj:oj@localhost:5432/oj?sslmode=disable")
	v.SetDefault("backend_service_addr", "localhost:8002")
	v.SetDefault("identra_grpc_host", "localhost:50051")
	v.SetDefault("admin_email", "")

	// Rate Limiting defaults
	v.SetDefault("rate_limit_enabled", true)
	v.SetDefault("rate_limit_requests_per_minute", 60)
	v.SetDefault("rate_limit_burst_size", 10)
	v.SetDefault("rate_limit_submission_requests_per_minute", 5)
	v.SetDefault("rate_limit_submission_burst_size", 2)
	v.SetDefault("rate_limit_ip_requests_per_minute", 30)
	v.SetDefault("rate_limit_ip_burst_size", 5)

	// Caching defaults
	v.SetDefault("cache_enabled", true)
	v.SetDefault("cache_problem_ttl", 300)   // 5 minutes
	v.SetDefault("cache_contest_ttl", 120)   // 2 minutes
	v.SetDefault("cache_scoreboard_ttl", 10) // 10 seconds

	// OAuth defaults
	v.SetDefault("oauth_redirect_url", "http://localhost:3000/auth/callback")

	v.AutomaticEnv()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")

	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
