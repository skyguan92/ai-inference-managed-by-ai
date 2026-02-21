package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/ratelimit"
)

// TestRateLimitAllows verifies that requests within the limit pass through.
func TestRateLimitAllows(t *testing.T) {
	// capacity=5 means 5 requests are allowed before hitting the limit.
	limiter := ratelimit.New(1.0, 5)
	handler := RateLimit(limiter)(okHandler)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

// TestRateLimitExceeded verifies that once the bucket is empty, further requests get 429.
func TestRateLimitExceeded(t *testing.T) {
	// capacity=2 means only 2 requests before limit is hit.
	limiter := ratelimit.New(0.001, 2) // very slow refill
	handler := RateLimit(limiter)(okHandler)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Third request should be rate limited.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "RATE_LIMITED") {
		t.Errorf("expected RATE_LIMITED in body, got: %s", body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429")
	}
}

// TestRateLimitPerClientIsolation verifies different clients have independent buckets.
func TestRateLimitPerClientIsolation(t *testing.T) {
	limiter := ratelimit.New(0.001, 1) // 1 request per client
	handler := RateLimit(limiter)(okHandler)

	// Client A exhausts its bucket.
	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "192.168.1.1:1111"
	recA := httptest.NewRecorder()
	handler.ServeHTTP(recA, reqA)

	reqA2 := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA2.RemoteAddr = "192.168.1.1:1111"
	recA2 := httptest.NewRecorder()
	handler.ServeHTTP(recA2, reqA2)

	if recA2.Code != http.StatusTooManyRequests {
		t.Errorf("client A should be rate limited, got %d", recA2.Code)
	}

	// Client B should still be allowed.
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "192.168.1.2:2222"
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)

	if recB.Code != http.StatusOK {
		t.Errorf("client B should not be rate limited, got %d", recB.Code)
	}
}

// TestRateLimitEmptyRemoteAddr verifies requests with empty RemoteAddr use "unknown" key.
func TestRateLimitEmptyRemoteAddr(t *testing.T) {
	limiter := ratelimit.New(1.0, 10)
	handler := RateLimit(limiter)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "" // empty
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// TestRateLimitResponseContentType verifies the 429 response has JSON content type.
func TestRateLimitResponseContentType(t *testing.T) {
	limiter := ratelimit.New(0.001, 1) // 1 request allowed
	handler := RateLimit(limiter)(okHandler)

	addr := "10.10.10.10:9999"

	// Exhaust the single token.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = addr
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Second request should get 429 with JSON content type.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = addr
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec2.Code)
	}
	ct := rec2.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content type, got %q", ct)
	}
}
