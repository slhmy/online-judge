package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// CacheKeyPrefix is the prefix for all compilation cache keys in Redis
	CacheKeyPrefix = "compile:cache"
)

// CompileCache stores and retrieves compiled binaries using Redis
type CompileCache struct {
	redis   *redis.Client
	ttl     time.Duration
	enabled bool
}

// NewCompileCache creates a new compilation cache
func NewCompileCache(redisClient *redis.Client, ttl time.Duration, enabled bool) *CompileCache {
	return &CompileCache{
		redis:   redisClient,
		ttl:     ttl,
		enabled: enabled,
	}
}

// Get retrieves a cached compiled binary for the given source code
// Returns (binaryData, true) if found, (nil, false) if not found or cache disabled
func (c *CompileCache) Get(ctx context.Context, language, sourceCode string) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}

	key := c.buildKey(language, sourceCode)
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Not in cache
			return nil, false
		}
		// Redis error - log but don't block compilation
		fmt.Printf("Warning: compile cache get error: %v\n", err)
		return nil, false
	}

	return data, true
}

// Set stores a compiled binary in the cache
func (c *CompileCache) Set(ctx context.Context, language, sourceCode string, binary []byte) error {
	if !c.enabled {
		return nil
	}

	key := c.buildKey(language, sourceCode)
	err := c.redis.Set(ctx, key, binary, c.ttl).Err()
	if err != nil {
		fmt.Printf("Warning: compile cache set error: %v\n", err)
		return err
	}

	return nil
}

// buildKey constructs the Redis key for a source code + language combination
func (c *CompileCache) buildKey(language, sourceCode string) string {
	hash := computeHash(sourceCode)
	return fmt.Sprintf("%s:%s:%s", CacheKeyPrefix, language, hash)
}

// computeHash generates a SHA256 hash of the source code
func computeHash(source string) string {
	h := sha256.Sum256([]byte(source))
	return hex.EncodeToString(h[:])
}

// IsEnabled returns whether the cache is enabled
func (c *CompileCache) IsEnabled() bool {
	return c.enabled
}

// Clear removes all compilation cache entries (useful for testing/debugging)
func (c *CompileCache) Clear(ctx context.Context) error {
	pattern := fmt.Sprintf("%s:*", CacheKeyPrefix)
	keys, err := c.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	return c.redis.Del(ctx, keys...).Err()
}