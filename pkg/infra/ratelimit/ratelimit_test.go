package ratelimit

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		capacity int64
		wantRate float64
		wantCap  int64
	}{
		{
			name:     "valid parameters",
			rate:     10.0,
			capacity: 100,
			wantRate: 10.0,
			wantCap:  100,
		},
		{
			name:     "zero rate defaults to 1",
			rate:     0,
			capacity: 100,
			wantRate: 1.0,
			wantCap:  100,
		},
		{
			name:     "negative rate defaults to 1",
			rate:     -5.0,
			capacity: 100,
			wantRate: 1.0,
			wantCap:  100,
		},
		{
			name:     "zero capacity defaults to 1",
			rate:     10.0,
			capacity: 0,
			wantRate: 10.0,
			wantCap:  1,
		},
		{
			name:     "negative capacity defaults to 1",
			rate:     10.0,
			capacity: -5,
			wantRate: 10.0,
			wantCap:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := New(tt.rate, tt.capacity)
			tbl, ok := limiter.(*TokenBucketLimiter)
			if !ok {
				t.Fatalf("expected TokenBucketLimiter, got %T", limiter)
			}
			if tbl.rate != tt.wantRate {
				t.Errorf("rate = %v, want %v", tbl.rate, tt.wantRate)
			}
			if tbl.capacity != tt.wantCap {
				t.Errorf("capacity = %v, want %v", tbl.capacity, tt.wantCap)
			}
			if tbl.tokens == nil {
				t.Error("tokens map should not be nil")
			}
		})
	}
}

func TestAllow(t *testing.T) {
	t.Run("empty key returns error", func(t *testing.T) {
		limiter := New(10.0, 10)
		allowed, err := limiter.Allow("")
		if err == nil {
			t.Error("expected error for empty key, got nil")
		}
		if allowed {
			t.Error("expected allowed to be false for empty key")
		}
	})

	t.Run("first request allows", func(t *testing.T) {
		limiter := New(10.0, 10)
		allowed, err := limiter.Allow("key1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected first request to be allowed")
		}
	})

	t.Run("exhausted bucket denies", func(t *testing.T) {
		limiter := New(1.0, 2)
		limiter.Allow("key2")
		limiter.Allow("key2")
		allowed, err := limiter.Allow("key2")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if allowed {
			t.Error("expected third request to be denied when bucket exhausted")
		}
	})

	t.Run("different keys independent", func(t *testing.T) {
		limiter := New(1.0, 1)
		allowed1, _ := limiter.Allow("keyA")
		allowed2, _ := limiter.Allow("keyB")
		if !allowed1 || !allowed2 {
			t.Error("expected both different keys to be allowed")
		}
	})
}

func TestReset(t *testing.T) {
	t.Run("reset removes key", func(t *testing.T) {
		limiter := New(10.0, 10)
		limiter.Allow("key1")
		limiter.Allow("key1")
		limiter.Reset("key1")
		allowed, err := limiter.Allow("key1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected request to be allowed after reset")
		}
	})

	t.Run("reset non-existent key does not panic", func(t *testing.T) {
		limiter := New(10.0, 10)
		limiter.Reset("nonexistent")
	})

	t.Run("reset one key does not affect others", func(t *testing.T) {
		limiter := New(1.0, 1)
		limiter.Allow("key1")
		limiter.Allow("key2")
		limiter.Reset("key1")
		allowed, _ := limiter.Allow("key1")
		if !allowed {
			t.Error("expected key1 to be allowed after reset")
		}
		allowed, _ = limiter.Allow("key2")
		if allowed {
			t.Error("expected key2 to be denied (exhausted)")
		}
	})
}

func TestTokenRefill(t *testing.T) {
	t.Run("tokens refill over time", func(t *testing.T) {
		limiter := New(100.0, 10)
		for i := 0; i < 10; i++ {
			limiter.Allow("key1")
		}
		allowed, _ := limiter.Allow("key1")
		if allowed {
			t.Error("expected request to be denied when bucket exhausted")
		}
		time.Sleep(100 * time.Millisecond)
		allowed, _ = limiter.Allow("key1")
		if !allowed {
			t.Error("expected request to be allowed after refill")
		}
	})

	t.Run("tokens do not exceed capacity", func(t *testing.T) {
		limiter := New(1000.0, 5)
		limiter.Allow("key1")
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < 10; i++ {
			limiter.Allow("key1")
		}
		allowed, _ := limiter.Allow("key1")
		if allowed {
			t.Error("expected request to be denied")
		}
	})
}
