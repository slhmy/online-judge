package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestRateLimiter(t *testing.T, config RateLimitConfig) (*RateLimiter, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	rl := NewRateLimiter(rdb, config)
	return rl, mr
}

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	if cfg.RequestsPerMinute != 60 {
		t.Errorf("Expected RequestsPerMinute=60, got %d", cfg.RequestsPerMinute)
	}
	if cfg.BurstSize != 10 {
		t.Errorf("Expected BurstSize=10, got %d", cfg.BurstSize)
	}
	if cfg.SubmissionRequestsPerMinute != 5 {
		t.Errorf("Expected SubmissionRequestsPerMinute=5, got %d", cfg.SubmissionRequestsPerMinute)
	}
	if cfg.SubmissionBurstSize != 2 {
		t.Errorf("Expected SubmissionBurstSize=2, got %d", cfg.SubmissionBurstSize)
	}
	if cfg.IPRequestsPerMinute != 30 {
		t.Errorf("Expected IPRequestsPerMinute=30, got %d", cfg.IPRequestsPerMinute)
	}
	if cfg.IPBurstSize != 5 {
		t.Errorf("Expected IPBurstSize=5, got %d", cfg.IPBurstSize)
	}
	if !cfg.Enabled {
		t.Error("Expected Enabled=true")
	}
}

func TestRateLimitMiddleware_Disabled(t *testing.T) {
	config := RateLimitConfig{Enabled: false}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_IPBased(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 3,
		IPBurstSize:         1, // Total allowed = limit + burst = 4
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Make multiple requests from same IP
	// With limit=3 and burst=1, we can make 4 requests before being rate limited
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()

		rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)

		if i < 4 {
			if rec.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i, rec.Code)
			}
		} else {
			// 5th request should be rate limited
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status 429, got %d", i, rec.Code)
			}
		}
	}
}

func TestRateLimitMiddleware_UserBased(t *testing.T) {
	config := RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 3,
		BurstSize:         1, // Total allowed = limit + burst = 4
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create request with user context
	// With limit=3 and burst=1, we can make 4 requests before being rate limited
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), UserIDKey, "user-123")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)

		if i < 4 {
			if rec.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i, rec.Code)
			}
		} else {
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status 429, got %d", i, rec.Code)
			}
		}
	}
}

func TestRateLimitMiddleware_XForwardedFor(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 1,
		IPBurstSize:         0,
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Request with X-Forwarded-For header
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	rec1 := httptest.NewRecorder()

	rl.RateLimitMiddleware(handler).ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("First request: Expected status 200, got %d", rec1.Code)
	}

	// Second request with same IP in X-Forwarded-For should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, proxy-ip")
	rec2 := httptest.NewRecorder()

	rl.RateLimitMiddleware(handler).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: Expected status 429, got %d", rec2.Code)
	}

	// Request with different IP should pass
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.Header.Set("X-Forwarded-For", "10.0.0.2, proxy-ip")
	rec3 := httptest.NewRecorder()

	rl.RateLimitMiddleware(handler).ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("Different IP request: Expected status 200, got %d", rec3.Code)
	}
}

func TestSubmissionRateLimitMiddleware(t *testing.T) {
	config := RateLimitConfig{
		Enabled:                     true,
		SubmissionRequestsPerMinute: 2,
		SubmissionBurstSize:         0,
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test authenticated user
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/submissions", nil)
		ctx := context.WithValue(req.Context(), UserIDKey, "user-456")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		rl.SubmissionRateLimitMiddleware(handler).ServeHTTP(rec, req)

		if i < 2 {
			if rec.Code != http.StatusOK {
				t.Errorf("Submission %d: Expected status 200, got %d", i, rec.Code)
			}
		} else {
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("Submission %d: Expected status 429, got %d", i, rec.Code)
			}
		}
	}
}

func TestRateLimitHeaders(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 5,
		IPBurstSize:         2,
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check rate limit headers
	if rec.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("Expected X-RateLimit-Limit=5, got %s", rec.Header().Get("X-RateLimit-Limit"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining to be set")
	}
}

func TestRateLimitErrorResponse(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 1,
		IPBurstSize:         0, // Total allowed = 1
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request passes
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec1.Code)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rec2.Code)
	}

	// Check response body
	var resp rateLimitResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if resp.Error != "rate limit exceeded" {
		t.Errorf("Expected error message 'rate limit exceeded', got %s", resp.Error)
	}
	if resp.Limit != 1 {
		t.Errorf("Expected Limit=1, got %d", resp.Limit)
	}
	// RetryAfter should be at least 1 second (time until oldest entry expires)
	if resp.RetryAfter < 1 {
		t.Errorf("Expected RetryAfter >= 1, got %d", resp.RetryAfter)
	}

	// Check Retry-After header
	if rec2.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

func TestRateLimitMiddleware_RedirectUnavailable(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 1,
		IPBurstSize:         0,
	}

	// Create rate limiter with non-existent Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Non-existent Redis
	})
	rl := NewRateLimiter(rdb, config)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)

	// Should fail open when Redis is unavailable
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 (fail open), got %d", rec.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:1234",
			want:       "192.168.1.1:1234",
		},
		{
			name:       "X-Forwarded-For single",
			remoteAddr: "proxy:1234",
			xff:        "10.0.0.1",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "proxy:1234",
			xff:        "10.0.0.1, 192.168.1.1, 172.16.0.1",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For with spaces",
			remoteAddr: "proxy:1234",
			xff:        " 10.0.0.1 , 192.168.1.1 ",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "proxy:1234",
			xri:        "10.0.0.2",
			want:       "10.0.0.2",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr: "proxy:1234",
			xff:        "10.0.0.1",
			xri:        "10.0.0.2",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			got := getClientIP(req)
			if !strings.Contains(got, tt.want) {
				t.Errorf("getClientIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDifferentUsersIndependent(t *testing.T) {
	config := RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 1,
		BurstSize:         0,
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// User 1 makes request
	req1 := httptest.NewRequest("GET", "/test", nil)
	ctx1 := context.WithValue(req1.Context(), UserIDKey, "user-1")
	req1 = req1.WithContext(ctx1)
	rec1 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("User 1 first request: Expected status 200, got %d", rec1.Code)
	}

	// User 1 second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	ctx2 := context.WithValue(req2.Context(), UserIDKey, "user-1")
	req2 = req2.WithContext(ctx2)
	rec2 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("User 1 second request: Expected status 429, got %d", rec2.Code)
	}

	// User 2 should be independent and not rate limited
	req3 := httptest.NewRequest("GET", "/test", nil)
	ctx3 := context.WithValue(req3.Context(), UserIDKey, "user-2")
	req3 = req3.WithContext(ctx3)
	rec3 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("User 2 request: Expected status 200, got %d", rec3.Code)
	}
}

func TestDifferentIPsIndependent(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 1,
		IPBurstSize:         0,
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// IP 1 makes request
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("IP 1 first request: Expected status 200, got %d", rec1.Code)
	}

	// IP 1 second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1 second request: Expected status 429, got %d", rec2.Code)
	}

	// IP 2 should be independent and not rate limited
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.2:1234"
	rec3 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("IP 2 request: Expected status 200, got %d", rec3.Code)
	}
}

func TestRateLimitReset(t *testing.T) {
	config := RateLimitConfig{
		Enabled:             true,
		IPRequestsPerMinute: 2,
		IPBurstSize:         0,
	}
	rl, mr := setupTestRateLimiter(t, config)
	defer mr.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Exhaust rate limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)
	}

	// Should be rate limited now
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rec.Code)
	}

	// Fast-forward time in miniredis
	mr.FastForward(61 * time.Second)

	// Should be allowed again
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	rl.RateLimitMiddleware(handler).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("After reset: Expected status 200, got %d", rec2.Code)
	}
}