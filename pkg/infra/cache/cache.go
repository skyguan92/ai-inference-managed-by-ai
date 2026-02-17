package cache

import (
	"context"
	"sync"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (any, bool)
	Set(ctx context.Context, key string, value any, ttl time.Duration)
	Delete(ctx context.Context, key string)
	Clear(ctx context.Context)
	Size(ctx context.Context) int
}

type memCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	opts  *options
}

type cacheItem struct {
	value      any
	expiration time.Time
}

type options struct {
	defaultTTL time.Duration
	maxSize    int
}

type Option func(*options)

func WithTTL(ttl time.Duration) Option {
	return func(o *options) {
		o.defaultTTL = ttl
	}
}

func WithMaxSize(maxSize int) Option {
	return func(o *options) {
		o.maxSize = maxSize
	}
}

func New(opts ...Option) Cache {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return &memCache{
		items: make(map[string]cacheItem),
		opts:  o,
	}
}

func (c *memCache) Get(_ context.Context, key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		delete(c.items, key)
		return nil, false
	}

	return item.value, true
}

func (c *memCache) Set(_ context.Context, key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.opts.maxSize > 0 && len(c.items) >= c.opts.maxSize {
		c.evictOldest()
	}

	if ttl == 0 {
		ttl = c.opts.defaultTTL
	}

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	c.items[key] = cacheItem{
		value:      value,
		expiration: expiration,
	}
}

func (c *memCache) Delete(_ context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *memCache) Clear(_ context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheItem)
}

func (c *memCache) Size(_ context.Context) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *memCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	now := time.Now()

	for key, item := range c.items {
		if item.expiration.IsZero() || now.Before(item.expiration) {
			if oldestTime.IsZero() || item.expiration.Before(oldestTime) {
				oldestKey = key
				oldestTime = item.expiration
			}
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}
