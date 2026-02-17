package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

type Limiter interface {
	Allow(key string) (bool, error)
	Reset(key string)
}

type TokenBucketLimiter struct {
	rate     float64
	capacity int64
	mu       sync.Mutex
	tokens   map[string]*bucket
}

type bucket struct {
	tokens     int64
	lastUpdate time.Time
}

func New(rate float64, capacity int64) Limiter {
	if rate <= 0 {
		rate = 1.0
	}
	if capacity <= 0 {
		capacity = 1
	}
	return &TokenBucketLimiter{
		rate:     rate,
		capacity: capacity,
		tokens:   make(map[string]*bucket),
	}
}

func (l *TokenBucketLimiter) Allow(key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("key cannot be empty")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	b, exists := l.tokens[key]
	now := time.Now()

	if !exists {
		l.tokens[key] = &bucket{
			tokens:     l.capacity - 1,
			lastUpdate: now,
		}
		return true, nil
	}

	elapsed := now.Sub(b.lastUpdate).Seconds()
	refill := int64(elapsed * l.rate)
	b.tokens = min(b.tokens+refill, l.capacity)
	b.lastUpdate = now

	if b.tokens > 0 {
		b.tokens--
		return true, nil
	}

	return false, nil
}

func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.tokens, key)
}
