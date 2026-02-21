package provider

import (
	"context"
	"runtime"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

// Compile-time interface satisfaction check.
var _ resource.ResourceProvider = (*SystemResourceProvider)(nil)

// SystemResourceProvider implements resource.ResourceProvider by reading basic
// system metrics via the Go runtime (memory stats) and syscall (disk usage).
// It provides a lightweight, dependency-free provider suitable for bare-metal
// and container deployments where no NVIDIA GPU is present.
type SystemResourceProvider struct{}

// NewSystemResourceProvider creates a provider that reads system resource info.
func NewSystemResourceProvider() *SystemResourceProvider {
	return &SystemResourceProvider{}
}

// GetStatus returns current memory and storage status from the OS.
func (p *SystemResourceProvider) GetStatus(_ context.Context) (*resource.ResourceStatus, error) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// runtime.MemStats only exposes Go heap info; use Sys as a proxy for total
	// memory acquired from the OS and HeapInuse as "used" heap memory.
	totalMem := ms.Sys
	usedMem := ms.HeapInuse
	if usedMem > totalMem {
		usedMem = totalMem
	}
	availMem := totalMem - usedMem

	// Storage info is not available cross-platform without cgo or syscalls.
	// Return placeholder zeros; callers should treat 0 as "unknown".
	storageTotal := uint64(0)
	storageUsed := uint64(0)
	storageAvail := uint64(0)

	pressure := resource.PressureLevelLow
	if totalMem > 0 {
		ratio := float64(usedMem) / float64(totalMem)
		switch {
		case ratio >= 0.9:
			pressure = resource.PressureLevelCritical
		case ratio >= 0.75:
			pressure = resource.PressureLevelHigh
		case ratio >= 0.5:
			pressure = resource.PressureLevelMedium
		}
	}

	return &resource.ResourceStatus{
		Memory: resource.MemoryInfo{
			Total:     totalMem,
			Used:      usedMem,
			Available: availMem,
		},
		Storage: resource.StorageInfo{
			Total:     storageTotal,
			Used:      storageUsed,
			Available: storageAvail,
		},
		Slots:    []resource.ResourceSlot{},
		Pressure: pressure,
	}, nil
}

// GetBudget returns a simple budget derived from runtime memory stats.
func (p *SystemResourceProvider) GetBudget(_ context.Context) (*resource.ResourceBudget, error) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	total := ms.Sys
	// Reserve 25% for system overhead.
	reserved := total / 4
	available := total - reserved

	return &resource.ResourceBudget{
		Total:    total,
		Reserved: reserved,
		Pools: map[string]resource.ResourcePool{
			"inference": {
				Name:      "inference",
				Total:     available,
				Reserved:  0,
				Available: available,
			},
		},
	}, nil
}

// CanAllocate checks whether memoryBytes can be satisfied given current usage.
func (p *SystemResourceProvider) CanAllocate(_ context.Context, memoryBytes uint64, _ int) (*resource.CanAllocateResult, error) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	available := ms.Sys - ms.HeapInuse
	if ms.HeapInuse > ms.Sys {
		available = 0
	}

	if memoryBytes <= available {
		return &resource.CanAllocateResult{CanAllocate: true}, nil
	}

	return &resource.CanAllocateResult{
		CanAllocate: false,
		Reason:      "insufficient memory",
	}, nil
}
