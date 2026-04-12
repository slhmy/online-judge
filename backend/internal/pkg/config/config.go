package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	// Server
	GRPCPort string `mapstructure:"grpc_port"`
	HTTPPort string `mapstructure:"http_port"`

	// Database
	DatabaseURL string `mapstructure:"database_url"`

	// Redis
	RedisURL string `mapstructure:"redis_url"`

	// Auth
	IdentraJWKSURL string `mapstructure:"identra_jwks_url"`

	// Queue Migration
	// USE_ASYNQ_QUEUE: true = use asynq only, false = use legacy only
	// When both are enabled, dual-write is performed
	UseAsynqQueue bool `mapstructure:"use_asynq_queue"`
	// UseLegacyQueue: enable legacy Redis sorted set queue (for dual-write during migration)
	UseLegacyQueue bool `mapstructure:"use_legacy_queue"`
}

func Load() (*Config, error) {
	return LoadWithPrefix("")
}

func LoadWithPrefix(prefix string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("grpc_port", "8002")
	v.SetDefault("http_port", "8080")
	v.SetDefault("database_url", "postgres://postgres:postgres@localhost:5432/oj?sslmode=disable")
	v.SetDefault("redis_url", "localhost:6379")
	v.SetDefault("identra_jwks_url", "http://localhost:8081/.well-known/jwks.json")
	// Queue migration defaults: during migration, enable both queues
	v.SetDefault("use_asynq_queue", true)   // Primary queue (asynq)
	v.SetDefault("use_legacy_queue", false) // Legacy queue (disabled by default after migration)

	// Bind environment variables
	v.SetEnvPrefix(prefix)
	v.AutomaticEnv()

	// Read config file if exists
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")

	// Ignore error if config file doesn't exist
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
