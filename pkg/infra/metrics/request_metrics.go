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
	return RequestSnapshot{
		TotalRequests:    m.totalRequests.Load(),
		TotalErrors:      m.totalErrors.Load(),
		TotalLatencyMs:   m.totalLatencyMs.Load(),
	}
}

// RequestSnapshot is an immutable snapshot of request metrics at a point in time.
// Expose raw counters so callers (e.g. Prometheus) can compute rates and averages.
type RequestSnapshot struct {
	TotalRequests  int64
	TotalErrors    int64
	TotalLatencyMs int64
}
