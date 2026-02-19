package metrics

import (
	"context"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// memoryStatusEx matches the MEMORYSTATUSEX Windows structure.
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

var (
	modkernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes   = modkernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = modkernel32.NewProc("GlobalMemoryStatusEx")
)

type systemCollector struct{}

func NewCollector() Collector {
	return &systemCollector{}
}

func (c *systemCollector) Collect(ctx context.Context) (Metrics, error) {
	var metrics Metrics

	cpu, err := c.collectCPU()
	if err != nil {
		return Metrics{}, err
	}
	metrics.CPU = cpu

	mem, err := c.collectMemory()
	if err != nil {
		return Metrics{}, err
	}
	metrics.Memory = mem

	disk, err := c.collectDisk()
	if err != nil {
		return Metrics{}, err
	}
	metrics.Disk = disk

	metrics.Network = NetworkMetrics{}
	metrics.Timestamp = time.Now()

	return metrics, nil
}

func (c *systemCollector) getSystemTimes() (idle, kernel, user uint64, err error) {
	var idleTime, kernelTime, userTime windows.Filetime
	r1, _, callErr := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if r1 == 0 {
		return 0, 0, 0, callErr
	}
	idle = uint64(idleTime.HighDateTime)<<32 | uint64(idleTime.LowDateTime)
	kernel = uint64(kernelTime.HighDateTime)<<32 | uint64(kernelTime.LowDateTime)
	user = uint64(userTime.HighDateTime)<<32 | uint64(userTime.LowDateTime)
	return
}

// collectCPU takes two snapshots 200ms apart to compute current CPU usage.
// A single snapshot gives cumulative averages since boot, not current load.
func (c *systemCollector) collectCPU() (float64, error) {
	idle1, kernel1, user1, err := c.getSystemTimes()
	if err != nil {
		return 0, err
	}

	time.Sleep(200 * time.Millisecond)

	idle2, kernel2, user2, err := c.getSystemTimes()
	if err != nil {
		return 0, err
	}

	deltaIdle := idle2 - idle1
	// kernel time includes idle time; total non-idle = (kernel+user) - idle
	deltaTotal := (kernel2 + user2) - (kernel1 + user1)
	if deltaTotal == 0 {
		return 0, nil
	}

	percent := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
	return percent, nil
}

func (c *systemCollector) collectMemory() (MemoryMetrics, error) {
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))
	r1, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if r1 == 0 {
		return MemoryMetrics{}, err
	}

	total := memStatus.ullTotalPhys
	available := memStatus.ullAvailPhys
	used := total - available
	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return MemoryMetrics{
		Used:      used,
		Total:     total,
		Available: available,
		Percent:   percent,
	}, nil
}

func (c *systemCollector) collectDisk() (DiskMetrics, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return DiskMetrics{}, err
	}
	// Use the volume root of the current working directory (e.g., "C:\").
	volRoot := filepath.VolumeName(cwd) + `\`

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	wd, err := windows.UTF16PtrFromString(volRoot)
	if err != nil {
		return DiskMetrics{}, err
	}
	err = windows.GetDiskFreeSpaceEx(wd, &freeBytesAvailable, &totalBytes, &totalFreeBytes)
	if err != nil {
		return DiskMetrics{}, err
	}

	used := totalBytes - totalFreeBytes
	var percent float64
	if totalBytes > 0 {
		percent = float64(used) / float64(totalBytes) * 100
	}

	return DiskMetrics{
		Used:    used,
		Total:   totalBytes,
		Free:    totalFreeBytes,
		Percent: percent,
	}, nil
}
