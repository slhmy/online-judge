package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	// Server
	GRPCPort string `mapstructure:"grpc_port"`
	HTTPPort string `mapstructure:"http_port"`

	// Redis
	RedisURL string `mapstructure:"redis_url"`

	// Database
	DatabaseURL string `mapstructure:"database_url"`

	// Backend Services
	ProblemServiceAddr      string `mapstructure:"problem_service_addr"`
	SubmissionServiceAddr   string `mapstructure:"submission_service_addr"`
	ContestServiceAddr      string `mapstructure:"contest_service_addr"`
	NotificationServiceAddr string `mapstructure:"notification_service_addr"`
	UserServiceAddr         string `mapstructure:"user_service_addr"`

	// Auth
	IdentraGRPCHost string `mapstructure:"identra_grpc_host"`
	IdentraHTTPHost  string `mapstructure:"identra_http_host"`
	AdminEmail       string `mapstructure:"admin_email"`

	// Rate Limiting
	RateLimitEnabled                   bool `mapstructure:"rate_limit_enabled"`
	RateLimitRequestsPerMinute         int  `mapstructure:"rate_limit_requests_per_minute"`
	RateLimitBurstSize                 int  `mapstructure:"rate_limit_burst_size"`
	RateLimitSubmissionRequestsPerMin  int  `mapstructure:"rate_limit_submission_requests_per_minute"`
	RateLimitSubmissionBurstSize       int  `mapstructure:"rate_limit_submission_burst_size"`
	RateLimitIPRequestsPerMinute       int  `mapstructure:"rate_limit_ip_requests_per_minute"`
	RateLimitIPBurstSize               int  `mapstructure:"rate_limit_ip_burst_size"`
}

func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("grpc_port", "8002")
	v.SetDefault("http_port", "8080")
	v.SetDefault("redis_url", "localhost:6379")
	v.SetDefault("database_url", "postgres://oj:oj@localhost:5432/oj?sslmode=disable")
	v.SetDefault("problem_service_addr", "localhost:8002")
	v.SetDefault("submission_service_addr", "localhost:8003")
	v.SetDefault("contest_service_addr", "localhost:8004")
	v.SetDefault("notification_service_addr", "localhost:8005")
	v.SetDefault("user_service_addr", "localhost:8006")
	v.SetDefault("identra_grpc_host", "localhost:50051")
	v.SetDefault("identra_http_host", "localhost:8081")
	v.SetDefault("admin_email", "")

	// Rate Limiting defaults
	v.SetDefault("rate_limit_enabled", true)
	v.SetDefault("rate_limit_requests_per_minute", 60)
	v.SetDefault("rate_limit_burst_size", 10)
	v.SetDefault("rate_limit_submission_requests_per_minute", 5)
	v.SetDefault("rate_limit_submission_burst_size", 2)
	v.SetDefault("rate_limit_ip_requests_per_minute", 30)
	v.SetDefault("rate_limit_ip_burst_size", 5)

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