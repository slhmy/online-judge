package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestCacheService(t *testing.T, config Config) (*Service, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewService(rdb, config)
	return service, mr
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 5*time.Minute, cfg.ProblemTTL)
	assert.Equal(t, 2*time.Minute, cfg.ContestTTL)
	assert.Equal(t, 10*time.Second, cfg.ScoreboardTTL)
}

func TestService_Get_Set(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set a value
	err := service.Set(ctx, "test-key", []byte("test-value"), 1*time.Minute)
	require.NoError(t, err)

	// Get the value
	data, err := service.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), data)
}

func TestService_Get_NotFound(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Get non-existent key
	data, err := service.Get(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_Get_CacheDisabled(t *testing.T) {
	config := Config{Enabled: false}
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set a value (should not be stored since cache is disabled)
	err := service.Set(ctx, "test-key", []byte("test-value"), 1*time.Minute)
	require.NoError(t, err)

	// Get should return nil since cache is disabled
	data, err := service.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_SetWithTags(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set value with tags
	err := service.Set(ctx, "problem:123", []byte("problem-data"), 1*time.Minute, "problem:123")
	require.NoError(t, err)

	// Verify tag exists
	tagKey := "cache:tag:problem:123"
	keys, err := service.redis.SMembers(ctx, tagKey).Result()
	require.NoError(t, err)
	assert.Contains(t, keys, "cache:problem:123")
}

func TestService_Delete(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set a value
	err := service.Set(ctx, "test-key", []byte("test-value"), 1*time.Minute)
	require.NoError(t, err)

	// Delete the value
	err = service.Delete(ctx, "test-key")
	require.NoError(t, err)

	// Get should return nil
	data, err := service.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_DeleteByTag(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set multiple values with same tag
	err := service.Set(ctx, "problem:123", []byte("data1"), 1*time.Minute, "problem:123")
	require.NoError(t, err)

	err = service.Set(ctx, "problem:123:details", []byte("data2"), 1*time.Minute, "problem:123")
	require.NoError(t, err)

	// Delete by tag
	err = service.DeleteByTag(ctx, "problem:123")
	require.NoError(t, err)

	// Both keys should be deleted
	data1, err := service.Get(ctx, "problem:123")
	require.NoError(t, err)
	assert.Nil(t, data1)

	data2, err := service.Get(ctx, "problem:123:details")
	require.NoError(t, err)
	assert.Nil(t, data2)
}

func TestService_TTLExpiration(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set a value with short TTL
	err := service.Set(ctx, "test-key", []byte("test-value"), 2*time.Second)
	require.NoError(t, err)

	// Get should return the value
	data, err := service.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), data)

	// Fast-forward time
	mr.FastForward(3 * time.Second)

	// Get should return nil after TTL expires
	data, err = service.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_FailOpen(t *testing.T) {
	config := DefaultConfig()

	// Create service with non-existent Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Non-existent Redis
	})
	service := NewService(rdb, config)

	ctx := context.Background()

	// Get should return nil (fail open) instead of error
	data, err := service.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Nil(t, data)

	// Set should return nil (fail open) instead of error
	err = service.Set(ctx, "test-key", []byte("test-value"), 1*time.Minute)
	require.NoError(t, err)

	// Delete should return nil (fail open)
	err = service.Delete(ctx, "test-key")
	require.NoError(t, err)

	// DeleteByTag should return nil (fail open)
	err = service.DeleteByTag(ctx, "test-tag")
	require.NoError(t, err)
}

func TestService_InvalidateProblemCache(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set a problem cache entry
	err := service.Set(ctx, "problem:123", []byte("problem-data"), 5*time.Minute, "problem:123")
	require.NoError(t, err)

	// Invalidate problem cache
	err = service.InvalidateProblemCache(ctx, "123")
	require.NoError(t, err)

	// Cache should be cleared
	data, err := service.Get(ctx, "problem:123")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_InvalidateContestCache(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set contest cache entries
	err := service.Set(ctx, "contest:456", []byte("contest-data"), 2*time.Minute, "contest:456")
	require.NoError(t, err)

	err = service.Set(ctx, "scoreboard:456", []byte("scoreboard-data"), 10*time.Second, "contest:456", "scoreboard:456")
	require.NoError(t, err)

	// Invalidate contest cache
	err = service.InvalidateContestCache(ctx, "456")
	require.NoError(t, err)

	// Contest cache should be cleared
	data, err := service.Get(ctx, "contest:456")
	require.NoError(t, err)
	assert.Nil(t, data)

	// Scoreboard cache should also be cleared (due to shared tag)
	data, err = service.Get(ctx, "scoreboard:456")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_InvalidateScoreboardCache(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set scoreboard cache entry
	err := service.Set(ctx, "scoreboard:456", []byte("scoreboard-data"), 10*time.Second, "contest:456", "scoreboard:456")
	require.NoError(t, err)

	// Invalidate scoreboard cache
	err = service.InvalidateScoreboardCache(ctx, "456")
	require.NoError(t, err)

	// Scoreboard cache should be cleared
	data, err := service.Get(ctx, "scoreboard:456")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_MultipleTags(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Set value with multiple tags
	err := service.Set(ctx, "scoreboard:456", []byte("scoreboard-data"), 10*time.Second, "contest:456", "scoreboard:456")
	require.NoError(t, err)

	// Verify both tags exist
	tagKey1 := "cache:tag:contest:456"
	keys1, err := service.redis.SMembers(ctx, tagKey1).Result()
	require.NoError(t, err)
	assert.Contains(t, keys1, "cache:scoreboard:456")

	tagKey2 := "cache:tag:scoreboard:456"
	keys2, err := service.redis.SMembers(ctx, tagKey2).Result()
	require.NoError(t, err)
	assert.Contains(t, keys2, "cache:scoreboard:456")

	// Delete by one tag should clear the cache
	err = service.DeleteByTag(ctx, "contest:456")
	require.NoError(t, err)

	data, err := service.Get(ctx, "scoreboard:456")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestService_DeleteByTag_EmptyTag(t *testing.T) {
	config := DefaultConfig()
	service, mr := setupTestCacheService(t, config)
	defer mr.Close()

	ctx := context.Background()

	// Delete by non-existent tag should not error
	err := service.DeleteByTag(ctx, "non-existent-tag")
	require.NoError(t, err)
}