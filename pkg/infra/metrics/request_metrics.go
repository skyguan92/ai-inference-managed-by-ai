package metrics

import (
	"sync/atomic"
	"time"
)

// RequestMetrics tracks HTTP request counts, latency, and errors.
// It uses lock-free atomic counters for thread safety.
type RequestMetrics struct {
	totalRequests   atomic.Int64
	totalErrors     atomic.Int64
	totalLatencyMs  atomic.Int64
}

// NewRequestMetrics creates a new RequestMetrics instance.
func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{}
}

// Record records a completed request.
// latency is the request duration; isError indicates whether the request failed.
func (m *RequestMetrics) Record(latency time.Duration, isError bool) {
	m.totalRequests.Add(1)
	m.totalLatencyMs.Add(latency.Milliseconds())
	if isError {
		m.totalErrors.Add(1)
	}
}

// Snapshot returns a point-in-time snapshot of the counters.
func (m *RequestMetrics) Snapshot() RequestSnapshot {
	total := m.totalRequests.Load()
	errors := m.totalErrors.Load()
	latencyMs := m.totalLatencyMs.Load()

	var avgLatencyMs float64
	if total > 0 {
		avgLatencyMs = float64(latencyMs) / float64(total)
	}

	var errorRate float64
	if total > 0 {
		errorRate = float64(errors) / float64(total)
	}

	return RequestSnapshot{
		TotalRequests:  total,
		TotalErrors:    errors,
		AvgLatencyMs:   avgLatencyMs,
		ErrorRate:      errorRate,
	}
}

// RequestSnapshot is an immutable snapshot of request metrics at a point in time.
type RequestSnapshot struct {
	TotalRequests int64
	TotalErrors   int64
	AvgLatencyMs  float64
	ErrorRate     float64
}
