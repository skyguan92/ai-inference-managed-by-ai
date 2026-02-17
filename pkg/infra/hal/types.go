package hal

import "time"

const (
	VendorNVIDIA  = "NVIDIA"
	VendorAMD     = "AMD"
	VendorIntel   = "Intel"
	VendorApple   = "Apple"
	VendorUnknown = "Unknown"
)

const (
	DeviceTypeGPU         = "gpu"
	DeviceTypeCPU         = "cpu"
	DeviceTypeNPU         = "npu"
	DeviceTypeAccelerator = "accelerator"
)

const (
	HealthStatusHealthy  = "healthy"
	HealthStatusWarning  = "warning"
	HealthStatusCritical = "critical"
	HealthStatusUnknown  = "unknown"
)

type HardwareInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Vendor       string    `json:"vendor"`
	Type         string    `json:"type"`
	Architecture string    `json:"architecture,omitempty"`
	Memory       uint64    `json:"memory,omitempty"`
	Driver       string    `json:"driver,omitempty"`
	Firmware     string    `json:"firmware,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

type HardwareMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	Utilization float64   `json:"utilization"`
	Temperature float64   `json:"temperature"`
	Power       float64   `json:"power"`
	PowerLimit  float64   `json:"power_limit,omitempty"`
	MemoryUsed  uint64    `json:"memory_used"`
	MemoryTotal uint64    `json:"memory_total"`
	ClockCore   uint64    `json:"clock_core,omitempty"`
	ClockMemory uint64    `json:"clock_memory,omitempty"`
}

type HardwareHealth struct {
	Timestamp time.Time      `json:"timestamp"`
	Status    string         `json:"status"`
	Issues    []string       `json:"issues,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}
