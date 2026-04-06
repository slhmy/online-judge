package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// General rate limits
	RequestsPerMinute int `mapstructure:"rate_limit_requests_per_minute"`
	BurstSize         int `mapstructure:"rate_limit_burst_size"`

	// Submission-specific limits (stricter to prevent spam)
	SubmissionRequestsPerMinute int `mapstructure:"rate_limit_submission_requests_per_minute"`
	SubmissionBurstSize         int `mapstructure:"rate_limit_submission_burst_size"`

	// Per-IP limits (for unauthenticated users)
	IPRequestsPerMinute int `mapstructure:"rate_limit_ip_requests_per_minute"`
	IPBurstSize         int `mapstructure:"rate_limit_ip_burst_size"`

	// Whether rate limiting is enabled
	Enabled bool `mapstructure:"rate_limit_enabled"`
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute:           60,
		BurstSize:                   10,
		SubmissionRequestsPerMinute: 5,
		SubmissionBurstSize:          2,
		IPRequestsPerMinute:         30,
		IPBurstSize:                 5,
		Enabled:                     true,
	}
}

// RateLimiter implements distributed rate limiting using Redis
type RateLimiter struct {
	redis  *redis.Client
	config RateLimitConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

// rateLimitResponse is the JSON response for rate limited requests
type rateLimitResponse struct {
	Error       string `json:"error"`
	RetryAfter  int    `json:"retry_after"`
	Limit       int    `json:"limit"`
	Remaining   int    `json:"remaining"`
	ResetIn     int    `json:"reset_in"`
}

// RateLimitMiddleware returns a middleware that implements distributed rate limiting
// It supports both per-user (authenticated) and per-IP (unauthenticated) rate limiting
func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if rate limiting is disabled
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// Check if Redis is available
		if err := rl.redis.Ping(ctx).Err(); err != nil {
			// If Redis is unavailable, allow the request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		// Determine the key and limits based on authentication status
		userID := GetUserID(ctx)
		clientIP := getClientIP(r)

		var key string
		var limit, burst int

		if userID != "" {
			// Authenticated user - use user-based rate limiting
			key = fmt.Sprintf("ratelimit:user:%s", userID)
			limit = rl.config.RequestsPerMinute
			burst = rl.config.BurstSize
		} else {
			// Unauthenticated - use IP-based rate limiting
			key = fmt.Sprintf("ratelimit:ip:%s", clientIP)
			limit = rl.config.IPRequestsPerMinute
			burst = rl.config.IPBurstSize
		}

		allowed, remaining, resetIn, err := rl.checkRateLimit(ctx, key, limit, burst, time.Minute)
		if err != nil {
			// On error, fail open
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(resetIn))

		if !allowed {
			rl.writeRateLimitResponse(w, limit, remaining, resetIn)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SubmissionRateLimitMiddleware returns a stricter rate limit middleware for submission endpoints
func (rl *RateLimiter) SubmissionRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if rate limiting is disabled
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// Check if Redis is available
		if err := rl.redis.Ping(ctx).Err(); err != nil {
			// If Redis is unavailable, allow the request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		// Use stricter limits for submissions
		userID := GetUserID(ctx)
		clientIP := getClientIP(r)

		var key string
		var limit, burst int

		if userID != "" {
			// Authenticated user - use submission-specific rate limiting
			key = fmt.Sprintf("ratelimit:submission:user:%s", userID)
			limit = rl.config.SubmissionRequestsPerMinute
			burst = rl.config.SubmissionBurstSize
		} else {
			// Unauthenticated - use IP-based rate limiting (very strict)
			key = fmt.Sprintf("ratelimit:submission:ip:%s", clientIP)
			limit = rl.config.SubmissionRequestsPerMinute
			burst = rl.config.SubmissionBurstSize
		}

		allowed, remaining, resetIn, err := rl.checkRateLimit(ctx, key, limit, burst, time.Minute)
		if err != nil {
			// On error, fail open
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(resetIn))

		if !allowed {
			rl.writeRateLimitResponse(w, limit, remaining, resetIn)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// checkRateLimit implements a sliding window rate limit algorithm using Redis
// Returns: (allowed, remaining, resetInSeconds, error)
func (rl *RateLimiter) checkRateLimit(ctx context.Context, key string, limit, burst int, window time.Duration) (bool, int, int, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	windowStartUnix := windowStart.Unix()

	// Use Redis transaction to implement sliding window
	pipe := rl.redis.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStartUnix))

	// Count current entries in the window
	countCmd := pipe.ZCard(ctx, key)

	// Execute the pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, 0, err
	}

	count, err := countCmd.Result()
	if err != nil {
		return false, 0, 0, err
	}

	remaining := int(int64(limit) - count)
	if remaining < 0 {
		remaining = 0
	}

	// Check if at or above limit (including burst allowance)
	if count >= int64(limit+burst) {
		// Calculate time until oldest entry expires
		oldestCmd := rl.redis.ZRangeWithScores(ctx, key, 0, 0)
		oldest, err := oldestCmd.Result()
		if err == nil && len(oldest) > 0 {
			oldestTime := int64(oldest[0].Score)
			resetIn := int(oldestTime + int64(window.Seconds()) - now.Unix())
			if resetIn < 1 {
				resetIn = 1
			}
			return false, 0, resetIn, nil
		}
		return false, 0, int(window.Seconds()), nil
	}

	// Add current request timestamp
	member := fmt.Sprintf("%d:%d", now.UnixNano(), count)
	rl.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: member,
	})

	// Set expiration on the key
	rl.redis.Expire(ctx, key, window+time.Second)

	return true, remaining, int(window.Seconds()), nil
}

// TokenBucketRateLimit implements a token bucket algorithm using Redis
// This is more accurate for burst handling
func (rl *RateLimiter) TokenBucketRateLimit(ctx context.Context, key string, rate float64, burst int) (bool, int, error) {
	now := float64(time.Now().UnixNano()) / 1e9

	// Get current bucket state
	result, err := rl.redis.HMGet(ctx, key, "tokens", "last_update").Result()
	if err != nil && err != redis.Nil {
		return false, 0, err
	}

	var tokens float64
	var lastUpdate float64

	if result[0] != nil {
		tokens, _ = strconv.ParseFloat(result[0].(string), 64)
	} else {
		tokens = float64(burst)
	}

	if result[1] != nil {
		lastUpdate, _ = strconv.ParseFloat(result[1].(string), 64)
	} else {
		lastUpdate = now
	}

	// Calculate new token count
	elapsed := now - lastUpdate
	tokens = math.Min(tokens+elapsed*rate, float64(burst))

	allowed := tokens >= 1.0
	remaining := int(tokens)

	if allowed {
		tokens -= 1.0
		remaining = int(tokens)
	}

	// Update bucket state
	rl.redis.HMSet(ctx, key, map[string]interface{}{
		"tokens":      tokens,
		"last_update": now,
	})
	rl.redis.Expire(ctx, key, time.Duration(int(1/rate))*time.Second*2)

	return allowed, remaining, nil
}

// writeRateLimitResponse writes a rate limit exceeded response
func (rl *RateLimiter) writeRateLimitResponse(w http.ResponseWriter, limit, remaining, resetIn int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(resetIn))
	w.WriteHeader(http.StatusTooManyRequests)

	response := rateLimitResponse{
		Error:      "rate limit exceeded",
		RetryAfter: resetIn,
		Limit:      limit,
		Remaining:  remaining,
		ResetIn:     resetIn,
	}

	json.NewEncoder(w).Encode(response)
}

// getClientIP extracts the client IP from the request
// It checks common proxy headers first
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain multiple IPs, first is the original client
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (used by some proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}