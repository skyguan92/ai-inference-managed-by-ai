package metrics

import (
	"context"
	"sync"
	"time"
)

// CachedCollector wraps a Collector and refreshes metrics in the background
// on a fixed interval. Reads are always non-blocking (return last cached value).
type CachedCollector struct {
	inner    Collector
	interval time.Duration
	mu       sync.RWMutex
	last     Metrics
	lastErr  error
	stop     chan struct{}
}

func NewCachedCollector(c Collector, interval time.Duration) *CachedCollector {
	return &CachedCollector{
		inner:    c,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start begins the background refresh goroutine.
// The first collection happens immediately (blocking), subsequent ones are on the ticker.
func (cc *CachedCollector) Start(ctx context.Context) {
	// do initial collection
	m, err := cc.inner.Collect(ctx)
	cc.mu.Lock()
	cc.last, cc.lastErr = m, err
	cc.mu.Unlock()

	go func() {
		ticker := time.NewTicker(cc.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m, err := cc.inner.Collect(context.Background())
				cc.mu.Lock()
				cc.last, cc.lastErr = m, err
				cc.mu.Unlock()
			case <-cc.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop halts the background refresh goroutine.
func (cc *CachedCollector) Stop() {
	close(cc.stop)
}

// Collect returns the last cached metrics immediately (non-blocking).
func (cc *CachedCollector) Collect(ctx context.Context) (Metrics, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.last, cc.lastErr
}
