package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds cache TTL configuration
type Config struct {
	Enabled       bool          `mapstructure:"cache_enabled"`
	ProblemTTL    time.Duration `mapstructure:"cache_problem_ttl"`    // 5 minutes
	ContestTTL    time.Duration `mapstructure:"cache_contest_ttl"`    // 2 minutes
	ScoreboardTTL time.Duration `mapstructure:"cache_scoreboard_ttl"` // 10 seconds
}

// DefaultConfig returns sensible defaults for cache configuration
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		ProblemTTL:    5 * time.Minute,
		ContestTTL:    2 * time.Minute,
		ScoreboardTTL: 10 * time.Second,
	}
}

// Service implements Redis-based caching with tag support
type Service struct {
	redis  *redis.Client
	config Config
}

// NewService creates a new cache service
func NewService(redisClient *redis.Client, config Config) *Service {
	return &Service{
		redis:  redisClient,
		config: config,
	}
}

// keyPrefix is the prefix for all cache keys
const keyPrefix = "cache"

// tagPrefix is the prefix for tag sets (Redis sets containing cache keys)
const tagPrefix = "cache:tag"

// buildKey creates a cache key with prefix
func buildKey(key string) string {
	return fmt.Sprintf("%s:%s", keyPrefix, key)
}

// buildTagKey creates a tag key for storing associated cache keys
func buildTagKey(tag string) string {
	return fmt.Sprintf("%s:%s", tagPrefix, tag)
}

// Get retrieves cached data. Returns nil if not found.
// Implements fail-open: returns nil on Redis errors (allows request to proceed)
func (s *Service) Get(ctx context.Context, key string) ([]byte, error) {
	if !s.config.Enabled {
		return nil, nil
	}

	// Check if Redis is available
	if err := s.redis.Ping(ctx).Err(); err != nil {
		// Fail open - return nil to allow request to proceed
		return nil, nil
	}

	fullKey := buildKey(key)
	data, err := s.redis.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		// Key not found
		return nil, nil
	}
	if err != nil {
		// Other error - fail open
		return nil, nil
	}

	return data, nil
}

// Set stores data with TTL and associates tags for targeted invalidation.
// Tags are stored as Redis sets containing the cache keys.
func (s *Service) Set(ctx context.Context, key string, value []byte, ttl time.Duration, tags ...string) error {
	if !s.config.Enabled {
		return nil
	}

	// Check if Redis is available
	if err := s.redis.Ping(ctx).Err(); err != nil {
		// Fail open - don't cache but don't fail the request
		return nil
	}

	fullKey := buildKey(key)

	// Use pipeline for atomic operations
	pipe := s.redis.Pipeline()

	// Set the cache value
	pipe.Set(ctx, fullKey, value, ttl)

	// Associate cache key with each tag
	for _, tag := range tags {
		tagKey := buildTagKey(tag)
		pipe.SAdd(ctx, tagKey, fullKey)
		// Set tag TTL slightly longer than the cache entry
		// This ensures tags are cleaned up after cache entries expire
		pipe.Expire(ctx, tagKey, ttl+30*time.Second)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Delete removes a specific cache key
func (s *Service) Delete(ctx context.Context, key string) error {
	if !s.config.Enabled {
		return nil
	}

	// Check if Redis is available
	if err := s.redis.Ping(ctx).Err(); err != nil {
		return nil
	}

	fullKey := buildKey(key)
	return s.redis.Del(ctx, fullKey).Err()
}

// DeleteByTag removes all cache keys associated with a tag
func (s *Service) DeleteByTag(ctx context.Context, tag string) error {
	if !s.config.Enabled {
		return nil
	}

	// Check if Redis is available
	if err := s.redis.Ping(ctx).Err(); err != nil {
		return nil
	}

	tagKey := buildTagKey(tag)

	// Get all cache keys associated with this tag
	keys, err := s.redis.SMembers(ctx, tagKey).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		// No keys to delete
		return nil
	}

	// Use pipeline for atomic deletion
	pipe := s.redis.Pipeline()

	// Delete all cache keys
	pipe.Del(ctx, keys...)

	// Delete the tag set itself
	pipe.Del(ctx, tagKey)

	_, err = pipe.Exec(ctx)
	return err
}

// InvalidateProblemCache invalidates all cache entries for a problem
func (s *Service) InvalidateProblemCache(ctx context.Context, problemID string) error {
	return s.DeleteByTag(ctx, fmt.Sprintf("problem:%s", problemID))
}

// InvalidateContestCache invalidates all cache entries for a contest
func (s *Service) InvalidateContestCache(ctx context.Context, contestID string) error {
	return s.DeleteByTag(ctx, fmt.Sprintf("contest:%s", contestID))
}

// InvalidateScoreboardCache invalidates scoreboard cache for a contest
func (s *Service) InvalidateScoreboardCache(ctx context.Context, contestID string) error {
	// Delete both by tag and direct key for faster invalidation
	pipe := s.redis.Pipeline()

	// Delete by tag
	tagKey := buildTagKey(fmt.Sprintf("scoreboard:%s", contestID))
	keys, _ := s.redis.SMembers(ctx, tagKey).Result()
	if len(keys) > 0 {
		pipe.Del(ctx, keys...)
	}
	pipe.Del(ctx, tagKey)

	// Also delete direct key for immediate effect
	pipe.Del(ctx, buildKey(fmt.Sprintf("scoreboard:%s", contestID)))

	_, err := pipe.Exec(ctx)
	return err
}

// GetConfig returns the cache configuration
func (s *Service) GetConfig() Config {
	return s.config
}

// IsEnabled returns whether caching is enabled
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}