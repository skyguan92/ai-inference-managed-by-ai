package metrics

import (
	"context"
	"time"
)

type Collector interface {
	Collect(ctx context.Context) (Metrics, error)
}

type Metrics struct {
	CPU       float64
	Memory    MemoryMetrics
	Disk      DiskMetrics
	Network   NetworkMetrics
	Timestamp time.Time
}

type MemoryMetrics struct {
	Used      uint64
	Total     uint64
	Available uint64
	Percent   float64
}

type DiskMetrics struct {
	Used    uint64
	Total   uint64
	Free    uint64
	Percent float64
}

type NetworkMetrics struct {
	BytesSent   uint64
	BytesRecv   uint64
	PacketsSent uint64
	PacketsRecv uint64
}
