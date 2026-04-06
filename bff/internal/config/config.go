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

	// Auth
	IdentraGRPCHost string `mapstructure:"identra_grpc_host"`
	IdentraHTTPHost  string `mapstructure:"identra_http_host"`
	AdminEmail       string `mapstructure:"admin_email"`
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
	v.SetDefault("identra_grpc_host", "localhost:50051")
	v.SetDefault("identra_http_host", "localhost:8081")
	v.SetDefault("admin_email", "")

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