package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
)

// mockIdentraClient is a test double for identraClientInterface.
type mockIdentraClient struct {
	loginErr  error
	loginResp *pb.LoginByPasswordResponse
}

func (m *mockIdentraClient) LoginByPassword(_ context.Context, _, _ string) (*pb.LoginByPasswordResponse, error) {
	return m.loginResp, m.loginErr
}

func (m *mockIdentraClient) RefreshToken(_ context.Context, _ string) (*pb.RefreshTokenResponse, error) {
	return nil, nil
}

func (m *mockIdentraClient) GetCurrentUser(_ context.Context, _ string) (*pb.GetCurrentUserLoginInfoResponse, error) {
	return nil, nil
}

// newTestAuthHandler creates a minimal AuthHandler wired to a miniredis instance.
// db and identraDB are intentionally nil – tests that only exercise the lockout
// logic never reach database calls.
func newTestAuthHandler(t *testing.T, mockClient identraClientInterface) (*AuthHandler, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	h := &AuthHandler{
		identraClient: mockClient,
		redis:         rdb,
	}

	return h, mr
}

// --- isLoginLocked / recordFailedLogin / clearFailedLogin unit tests ---

func TestRecordAndCheckLoginLock(t *testing.T) {
	mc := &mockIdentraClient{}
	h, mr := newTestAuthHandler(t, mc)
	defer mr.Close()

	ctx := context.Background()
	email := "user@example.com"

	// Initially not locked
	locked, _ := h.isLoginLocked(ctx, email)
	if locked {
		t.Fatal("Expected account not to be locked initially")
	}

	// Record failures up to the threshold
	for i := 0; i < loginMaxFailedAttempts; i++ {
		h.recordFailedLogin(ctx, email)
	}

	// Should now be locked
	locked, retryAfter := h.isLoginLocked(ctx, email)
	if !locked {
		t.Fatal("Expected account to be locked after max failed attempts")
	}
	if retryAfter <= 0 {
		t.Errorf("Expected retryAfter > 0, got %d", retryAfter)
	}
}

func TestClearFailedLoginUnlocks(t *testing.T) {
	mc := &mockIdentraClient{}
	h, mr := newTestAuthHandler(t, mc)
	defer mr.Close()

	ctx := context.Background()
	email := "user@example.com"

	// Lock the account
	for i := 0; i < loginMaxFailedAttempts; i++ {
		h.recordFailedLogin(ctx, email)
	}

	locked, _ := h.isLoginLocked(ctx, email)
	if !locked {
		t.Fatal("Expected account to be locked")
	}

	// Clear and verify
	h.clearFailedLogin(ctx, email)

	locked, _ = h.isLoginLocked(ctx, email)
	if locked {
		t.Fatal("Expected account to be unlocked after clearFailedLogin")
	}
}

func TestLockoutExpiresAfterTTL(t *testing.T) {
	mc := &mockIdentraClient{}
	h, mr := newTestAuthHandler(t, mc)
	defer mr.Close()

	ctx := context.Background()
	email := "user@example.com"

	for i := 0; i < loginMaxFailedAttempts; i++ {
		h.recordFailedLogin(ctx, email)
	}

	locked, _ := h.isLoginLocked(ctx, email)
	if !locked {
		t.Fatal("Expected account to be locked")
	}

	// Fast-forward past the lockout window
	mr.FastForward(loginLockoutDuration + time.Second)

	locked, _ = h.isLoginLocked(ctx, email)
	if locked {
		t.Fatal("Expected lockout to expire after TTL")
	}
}

func TestLoginReturnsTooManyAttemptsWhenLocked(t *testing.T) {
	mc := &mockIdentraClient{loginErr: errors.New("wrong password")}
	h, mr := newTestAuthHandler(t, mc)
	defer mr.Close()

	ctx := context.Background()
	email := "locked@example.com"

	// Pre-populate the lock
	for i := 0; i < loginMaxFailedAttempts; i++ {
		h.recordFailedLogin(ctx, email)
	}

	body := `{"email":"locked@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}

	var resp AuthErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.ErrorCode != AuthErrorCodeTooManyAttempts {
		t.Errorf("Expected error code %s, got %s", AuthErrorCodeTooManyAttempts, resp.ErrorCode)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

func TestLoginIncrementsCounterOnFailure(t *testing.T) {
	mc := &mockIdentraClient{loginErr: errors.New("wrong password")}
	h, mr := newTestAuthHandler(t, mc)
	defer mr.Close()

	ctx := context.Background()
	email := "fail@example.com"

	for i := 0; i < loginMaxFailedAttempts-1; i++ {
		body := `{"email":"fail@example.com","password":"wrong"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Attempt %d: Expected 401, got %d", i+1, rec.Code)
		}
	}

	// Account should not be locked yet
	locked, _ := h.isLoginLocked(ctx, email)
	if locked {
		t.Fatal("Account should not be locked before reaching max attempts")
	}

	// One more failure should trigger the lock
	body := `{"email":"fail@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Final failure: Expected 401, got %d", rec.Code)
	}

	locked, _ = h.isLoginLocked(ctx, email)
	if !locked {
		t.Fatal("Account should be locked after max failed attempts")
	}
}
