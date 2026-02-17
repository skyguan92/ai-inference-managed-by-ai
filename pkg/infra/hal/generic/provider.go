package generic

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/hal"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "generic"
}

func (p *Provider) Vendor() string {
	return "Generic"
}

func (p *Provider) Available(ctx context.Context) bool {
	return true
}

func (p *Provider) Detect(ctx context.Context) ([]hal.HardwareInfo, error) {
	info, err := p.getCPUInfo()
	if err != nil {
		return nil, err
	}
	return []hal.HardwareInfo{*info}, nil
}

func (p *Provider) GetInfo(ctx context.Context, deviceID string) (*hal.HardwareInfo, error) {
	if deviceID != "cpu-0" && deviceID != "cpu" {
		return nil, hal.ErrDeviceNotFound.WithCause(fmt.Errorf("device %s not found", deviceID))
	}
	return p.getCPUInfo()
}

func (p *Provider) GetMetrics(ctx context.Context, deviceID string) (*hal.HardwareMetrics, error) {
	if deviceID != "cpu-0" && deviceID != "cpu" {
		return nil, hal.ErrDeviceNotFound.WithCause(fmt.Errorf("device %s not found", deviceID))
	}
	return p.getMetrics()
}

func (p *Provider) GetHealth(ctx context.Context, deviceID string) (*hal.HardwareHealth, error) {
	if deviceID != "cpu-0" && deviceID != "cpu" {
		return nil, hal.ErrDeviceNotFound.WithCause(fmt.Errorf("device %s not found", deviceID))
	}
	return p.getHealth()
}

func (p *Provider) SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error {
	return hal.ErrNotSupported.WithCause(fmt.Errorf("power management not supported on generic provider"))
}

func (p *Provider) getCPUInfo() (*hal.HardwareInfo, error) {
	info := &hal.HardwareInfo{
		ID:           "cpu-0",
		Name:         "CPU",
		Vendor:       "Generic",
		Type:         hal.DeviceTypeCPU,
		DiscoveredAt: time.Now(),
	}

	if cpuInfo, err := readCPUInfo(); err == nil {
		info.Name = cpuInfo.ModelName
		info.Architecture = cpuInfo.Architecture
		info.Vendor = cpuInfo.Vendor
	}

	if memInfo, err := readMemInfo(); err == nil {
		info.Memory = memInfo.Total
	}

	return info, nil
}

func (p *Provider) getMetrics() (*hal.HardwareMetrics, error) {
	metrics := &hal.HardwareMetrics{
		Timestamp:   time.Now(),
		Utilization: float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 100,
	}

	if memInfo, err := readMemInfo(); err == nil {
		metrics.MemoryTotal = memInfo.Total
		metrics.MemoryUsed = memInfo.Used
	}

	return metrics, nil
}

func (p *Provider) getHealth() (*hal.HardwareHealth, error) {
	health := &hal.HardwareHealth{
		Timestamp: time.Now(),
		Status:    hal.HealthStatusHealthy,
		Details:   make(map[string]any),
	}

	memInfo, err := readMemInfo()
	if err == nil {
		health.Details["memory_total"] = memInfo.Total
		health.Details["memory_available"] = memInfo.Available

		if memInfo.Total > 0 {
			usedPercent := float64(memInfo.Total-memInfo.Available) / float64(memInfo.Total) * 100
			if usedPercent > 95 {
				health.Status = hal.HealthStatusCritical
				health.Issues = append(health.Issues, fmt.Sprintf("Critical memory usage: %.1f%%", usedPercent))
			} else if usedPercent > 85 {
				health.Status = hal.HealthStatusWarning
				health.Issues = append(health.Issues, fmt.Sprintf("High memory usage: %.1f%%", usedPercent))
			}
		}
	}

	health.Details["cpu_cores"] = runtime.NumCPU()

	return health, nil
}

type CPUInfo struct {
	ModelName    string
	Architecture string
	Vendor       string
	Cores        int
}

func readCPUInfo() (*CPUInfo, error) {
	info := &CPUInfo{
		Cores: runtime.NumCPU(),
	}

	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return info, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "model name":
			if info.ModelName == "" {
				info.ModelName = value
			}
		case "vendor_id":
			if info.Vendor == "" {
				info.Vendor = value
			}
		case "cpu cores":
			if cores, err := strconv.Atoi(value); err == nil {
				info.Cores = cores
			}
		}
	}

	return info, nil
}

type MemInfo struct {
	Total     uint64
	Free      uint64
	Available uint64
	Used      uint64
}

func readMemInfo() (*MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &MemInfo{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.TrimSuffix(value, " kB")
		valueKB, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			continue
		}
		valueBytes := valueKB * 1024

		switch key {
		case "MemTotal":
			info.Total = valueBytes
		case "MemFree":
			info.Free = valueBytes
		case "MemAvailable":
			info.Available = valueBytes
		}
	}

	if info.Available == 0 {
		info.Available = info.Free
	}
	info.Used = info.Total - info.Available

	return info, nil
}
