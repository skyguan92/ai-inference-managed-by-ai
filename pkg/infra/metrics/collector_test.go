package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector()
	if collector == nil {
		t.Error("NewCollector() returned nil")
	}

	_, ok := collector.(*systemCollector)
	if !ok {
		t.Error("NewCollector() did not return *systemCollector")
	}
}

func TestCollector_Collect(t *testing.T) {
	collector := NewCollector()

	metrics, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if metrics.CPU < 0 || metrics.CPU > 100 {
		t.Errorf("CPU percent out of range: %v", metrics.CPU)
	}

	if metrics.Memory.Total == 0 {
		t.Error("Memory total should not be zero")
	}

	if metrics.Memory.Used > metrics.Memory.Total {
		t.Errorf("Memory used (%d) should not exceed total (%d)", metrics.Memory.Used, metrics.Memory.Total)
	}

	if metrics.Memory.Percent < 0 || metrics.Memory.Percent > 100 {
		t.Errorf("Memory percent out of range: %v", metrics.Memory.Percent)
	}

	if metrics.Disk.Total == 0 {
		t.Error("Disk total should not be zero")
	}

	if metrics.Disk.Used > metrics.Disk.Total {
		t.Errorf("Disk used (%d) should not exceed total (%d)", metrics.Disk.Used, metrics.Disk.Total)
	}

	if metrics.Disk.Percent < 0 || metrics.Disk.Percent > 100 {
		t.Errorf("Disk percent out of range: %v", metrics.Disk.Percent)
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestMetrics_Structure(t *testing.T) {
	metrics := Metrics{
		CPU:       50.5,
		Memory:    MemoryMetrics{Used: 1000, Total: 2000, Available: 1000, Percent: 50.0},
		Disk:      DiskMetrics{Used: 500, Total: 1000, Free: 500, Percent: 50.0},
		Network:   NetworkMetrics{BytesSent: 100, BytesRecv: 200, PacketsSent: 10, PacketsRecv: 20},
		Timestamp: time.Now(),
	}

	if metrics.CPU != 50.5 {
		t.Errorf("CPU mismatch: got %v", metrics.CPU)
	}

	if metrics.Memory.Used != 1000 {
		t.Errorf("Memory.Used mismatch: got %v", metrics.Memory.Used)
	}

	if metrics.Memory.Percent != 50.0 {
		t.Errorf("Memory.Percent mismatch: got %v", metrics.Memory.Percent)
	}

	if metrics.Disk.Used != 500 {
		t.Errorf("Disk.Used mismatch: got %v", metrics.Disk.Used)
	}

	if metrics.Network.BytesSent != 100 {
		t.Errorf("Network.BytesSent mismatch: got %v", metrics.Network.BytesSent)
	}

	if metrics.Network.PacketsRecv != 20 {
		t.Errorf("Network.PacketsRecv mismatch: got %v", metrics.Network.PacketsRecv)
	}
}

func TestMemoryMetrics_PercentCalculation(t *testing.T) {
	mem := MemoryMetrics{
		Used:      8 * 1024 * 1024 * 1024,
		Total:     16 * 1024 * 1024 * 1024,
		Available: 8 * 1024 * 1024 * 1024,
		Percent:   50.0,
	}

	expected := float64(8*1024*1024*1024) / float64(16*1024*1024*1024) * 100
	if mem.Percent != expected {
		t.Errorf("Expected percent %.2f, got %.2f", expected, mem.Percent)
	}
}

func TestDiskMetrics_PercentCalculation(t *testing.T) {
	disk := DiskMetrics{
		Used:    500 * 1024 * 1024 * 1024,
		Total:   1000 * 1024 * 1024 * 1024,
		Free:    500 * 1024 * 1024 * 1024,
		Percent: 50.0,
	}

	expected := float64(500*1024*1024*1024) / float64(1000*1024*1024*1024) * 100
	if disk.Percent != expected {
		t.Errorf("Expected percent %.2f, got %.2f", expected, disk.Percent)
	}
}
