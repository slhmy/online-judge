package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	// Server
	JudgehostID string `mapstructure:"judgehost_id"`

	// Redis
	RedisURL string `mapstructure:"redis_url"`

	// Orchestrator
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
}

func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("judgehost_id", "judgehost-01")
	v.SetDefault("redis_url", "localhost:6379")
	v.SetDefault("orchestrator_url", "http://localhost:8080")
	v.SetDefault("docker_host", "unix:///var/run/docker.sock")
	v.SetDefault("sandbox_workdir", "") // Empty means use default temp directory
	v.SetDefault("default_memory_limit", 524288) // 512 MB in KB
	v.SetDefault("default_time_limit", 10)       // 10 seconds
	v.SetDefault("max_processes", 50)
	v.SetDefault("compile_cache_ttl", 24)        // 24 hours
	v.SetDefault("compile_cache_enabled", true)  // Cache enabled by default

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