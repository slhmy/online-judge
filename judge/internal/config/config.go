package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Config struct {
	// Server
	JudgehostID string `mapstructure:"judgehost_id"`

	// Redis
	RedisURL string `mapstructure:"redis_url"`

	// Judge Service (gRPC/HTTP endpoint for judgehost registration)
	JudgeServiceURL string `mapstructure:"judge_service_url"`

	// Orchestrator (BFF URL for fetching submission/problem data)
	OrchestratorURL string `mapstructure:"orchestrator_url"`

	// Docker
	DockerHost string `mapstructure:"docker_host"`

	// Sandbox
	SandboxWorkDir string `mapstructure:"sandbox_workdir"`

	// Resource Limits
	DefaultMemoryLimit int64 `mapstructure:"default_memory_limit"`
	DefaultTimeLimit    int64 `mapstructure:"default_time_limit"`
	MaxProcesses        int   `mapstructure:"max_processes"`

	// Compilation Cache
	CompileCacheTTL     int  `mapstructure:"compile_cache_ttl"`     // hours
	CompileCacheEnabled bool `mapstructure:"compile_cache_enabled"`

	// Heartbeat
	HeartbeatInterval int `mapstructure:"heartbeat_interval"` // seconds
}

func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("judgehost_id", "") // Empty means auto-generate
	v.SetDefault("redis_url", "localhost:6379")
	v.SetDefault("judge_service_url", "http://localhost:9090")
	v.SetDefault("orchestrator_url", "http://localhost:8080")
	v.SetDefault("docker_host", "unix:///var/run/docker.sock")
	v.SetDefault("sandbox_workdir", "") // Empty means use default temp directory
	v.SetDefault("default_memory_limit", 524288) // 512 MB in KB
	v.SetDefault("default_time_limit", 10)       // 10 seconds
	v.SetDefault("max_processes", 50)
	v.SetDefault("compile_cache_ttl", 24)        // 24 hours
	v.SetDefault("compile_cache_enabled", true)  // Cache enabled by default
	v.SetDefault("heartbeat_interval", 10)       // 10 seconds

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

	// Auto-generate JudgehostID if not configured
	if cfg.JudgehostID == "" {
		cfg.JudgehostID = generateJudgehostID()
	}

	return &cfg, nil
}

// generateJudgehostID generates a unique judgehost ID
func generateJudgehostID() string {
	// Try to use hostname as base
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Generate UUID suffix for uniqueness
	uuidSuffix := uuid.New().String()[:8]

	return fmt.Sprintf("judgehost-%s-%s", strings.ToLower(hostname), uuidSuffix)
}