package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestNewRequestMetrics(t *testing.T) {
	rm := NewRequestMetrics()
	if rm == nil {
		t.Fatal("NewRequestMetrics() returned nil")
	}
}

func TestRequestMetrics_InitialSnapshot(t *testing.T) {
	rm := NewRequestMetrics()
	snap := rm.Snapshot()

	if snap.TotalRequests != 0 {
		t.Errorf("expected TotalRequests=0, got %d", snap.TotalRequests)
	}
	if snap.TotalErrors != 0 {
		t.Errorf("expected TotalErrors=0, got %d", snap.TotalErrors)
	}
	if snap.AvgLatencyMs != 0 {
		t.Errorf("expected AvgLatencyMs=0, got %f", snap.AvgLatencyMs)
	}
	if snap.ErrorRate != 0 {
		t.Errorf("expected ErrorRate=0, got %f", snap.ErrorRate)
	}
}

func TestRequestMetrics_RecordSuccess(t *testing.T) {
	rm := NewRequestMetrics()
	rm.Record(10*time.Millisecond, false)
	rm.Record(20*time.Millisecond, false)

	snap := rm.Snapshot()

	if snap.TotalRequests != 2 {
		t.Errorf("expected TotalRequests=2, got %d", snap.TotalRequests)
	}
	if snap.TotalErrors != 0 {
		t.Errorf("expected TotalErrors=0, got %d", snap.TotalErrors)
	}
	// Average of 10ms and 20ms
	if snap.AvgLatencyMs != 15.0 {
		t.Errorf("expected AvgLatencyMs=15.0, got %f", snap.AvgLatencyMs)
	}
	if snap.ErrorRate != 0 {
		t.Errorf("expected ErrorRate=0, got %f", snap.ErrorRate)
	}
}

func TestRequestMetrics_RecordError(t *testing.T) {
	rm := NewRequestMetrics()
	rm.Record(10*time.Millisecond, false)
	rm.Record(20*time.Millisecond, true)

	snap := rm.Snapshot()

	if snap.TotalRequests != 2 {
		t.Errorf("expected TotalRequests=2, got %d", snap.TotalRequests)
	}
	if snap.TotalErrors != 1 {
		t.Errorf("expected TotalErrors=1, got %d", snap.TotalErrors)
	}
	if snap.ErrorRate != 0.5 {
		t.Errorf("expected ErrorRate=0.5, got %f", snap.ErrorRate)
	}
}

func TestRequestMetrics_AllErrors(t *testing.T) {
	rm := NewRequestMetrics()
	rm.Record(5*time.Millisecond, true)
	rm.Record(5*time.Millisecond, true)

	snap := rm.Snapshot()

	if snap.TotalRequests != 2 {
		t.Errorf("expected TotalRequests=2, got %d", snap.TotalRequests)
	}
	if snap.TotalErrors != 2 {
		t.Errorf("expected TotalErrors=2, got %d", snap.TotalErrors)
	}
	if snap.ErrorRate != 1.0 {
		t.Errorf("expected ErrorRate=1.0, got %f", snap.ErrorRate)
	}
}

func TestRequestMetrics_ConcurrentRecords(t *testing.T) {
	rm := NewRequestMetrics()
	n := 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			rm.Record(time.Millisecond, i%10 == 0)
		}(i)
	}
	wg.Wait()

	snap := rm.Snapshot()
	if snap.TotalRequests != int64(n) {
		t.Errorf("expected TotalRequests=%d, got %d", n, snap.TotalRequests)
	}
	if snap.TotalErrors != int64(n/10) {
		t.Errorf("expected TotalErrors=%d, got %d", n/10, snap.TotalErrors)
	}
}

func TestRequestSnapshot_Fields(t *testing.T) {
	snap := RequestSnapshot{
		TotalRequests: 10,
		TotalErrors:   2,
		AvgLatencyMs:  15.5,
		ErrorRate:     0.2,
	}

	if snap.TotalRequests != 10 {
		t.Errorf("TotalRequests mismatch")
	}
	if snap.TotalErrors != 2 {
		t.Errorf("TotalErrors mismatch")
	}
	if snap.AvgLatencyMs != 15.5 {
		t.Errorf("AvgLatencyMs mismatch")
	}
	if snap.ErrorRate != 0.2 {
		t.Errorf("ErrorRate mismatch")
	}
}
