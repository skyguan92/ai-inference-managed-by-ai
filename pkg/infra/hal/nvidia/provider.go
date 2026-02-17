package nvidia

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/hal"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/device"
)

type smiInterface interface {
	Available(ctx context.Context) bool
	Query(ctx context.Context) (*smiOutput, error)
	QueryDevice(ctx context.Context, deviceID string) (*smiOutput, error)
	SetPowerLimit(ctx context.Context, deviceID string, limitWatts uint) error
	ResetDevice(ctx context.Context, deviceID string) error
}

type Provider struct {
	smi       smiInterface
	cache     map[string]*hal.HardwareInfo
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
	cacheTime time.Time
}

type Option func(*Provider)

func WithSMIPath(path string) Option {
	return func(p *Provider) {
		p.smi = NewSMI(path)
	}
}

func withSMI(smi smiInterface) Option {
	return func(p *Provider) {
		p.smi = smi
	}
}

func WithCacheTTL(ttl time.Duration) Option {
	return func(p *Provider) {
		p.cacheTTL = ttl
	}
}

func NewProvider(opts ...Option) *Provider {
	p := &Provider{
		smi:      NewSMI(""),
		cache:    make(map[string]*hal.HardwareInfo),
		cacheTTL: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) Name() string {
	return "nvidia"
}

func (p *Provider) Vendor() string {
	return hal.VendorNVIDIA
}

func (p *Provider) Available(ctx context.Context) bool {
	return p.smi.Available(ctx)
}

func (p *Provider) Detect(ctx context.Context) ([]device.DeviceInfo, error) {
	output, err := p.smi.Query(ctx)
	if err != nil {
		return nil, err
	}

	devices := make([]device.DeviceInfo, 0, len(output.GPUs))
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	for i, gpu := range output.GPUs {
		info := &hal.HardwareInfo{
			ID:           fmt.Sprintf("nvidia-%d", i),
			Name:         gpu.ProductName,
			Vendor:       hal.VendorNVIDIA,
			Type:         hal.DeviceTypeGPU,
			Architecture: gpu.ProductBrand,
			Memory:       parseMemory(gpu.FBMemoryUsage.Total),
			DiscoveredAt: time.Now(),
		}

		p.cache[info.ID] = info

		devices = append(devices, device.DeviceInfo{
			ID:           info.ID,
			Name:         info.Name,
			Vendor:       info.Vendor,
			Type:         info.Type,
			Architecture: info.Architecture,
			Memory:       info.Memory,
		})
	}

	p.cacheTime = time.Now()
	return devices, nil
}

func (p *Provider) GetDevice(ctx context.Context, deviceID string) (*device.DeviceInfo, error) {
	if info := p.getCachedInfo(deviceID); info != nil {
		return &device.DeviceInfo{
			ID:           info.ID,
			Name:         info.Name,
			Vendor:       info.Vendor,
			Type:         info.Type,
			Architecture: info.Architecture,
			Memory:       info.Memory,
		}, nil
	}

	output, err := p.smi.QueryDevice(ctx, p.toSMIID(deviceID))
	if err != nil {
		return nil, err
	}

	if len(output.GPUs) == 0 {
		return nil, hal.ErrDeviceNotFound
	}

	gpu := output.GPUs[0]
	info := &hal.HardwareInfo{
		ID:           deviceID,
		Name:         gpu.ProductName,
		Vendor:       hal.VendorNVIDIA,
		Type:         hal.DeviceTypeGPU,
		Architecture: gpu.ProductBrand,
		Memory:       parseMemory(gpu.FBMemoryUsage.Total),
		DiscoveredAt: time.Now(),
	}

	p.cacheMu.Lock()
	p.cache[deviceID] = info
	p.cacheMu.Unlock()

	return &device.DeviceInfo{
		ID:           info.ID,
		Name:         info.Name,
		Vendor:       info.Vendor,
		Type:         info.Type,
		Architecture: info.Architecture,
		Memory:       info.Memory,
	}, nil
}

func (p *Provider) GetMetrics(ctx context.Context, deviceID string) (*device.DeviceMetrics, error) {
	output, err := p.smi.QueryDevice(ctx, p.toSMIID(deviceID))
	if err != nil {
		return nil, err
	}

	if len(output.GPUs) == 0 {
		return nil, hal.ErrDeviceNotFound
	}

	gpu := output.GPUs[0]
	return &device.DeviceMetrics{
		Utilization: parsePercentage(gpu.Utilization.GPUUtil),
		Temperature: parseTemperature(gpu.Temperature.GPUTemp),
		Power:       parsePower(gpu.PowerReadings.PowerDraw),
		MemoryUsed:  parseMemory(gpu.FBMemoryUsage.Used),
		MemoryTotal: parseMemory(gpu.FBMemoryUsage.Total),
	}, nil
}

func (p *Provider) GetHealth(ctx context.Context, deviceID string) (*device.DeviceHealth, error) {
	output, err := p.smi.QueryDevice(ctx, p.toSMIID(deviceID))
	if err != nil {
		return nil, err
	}

	if len(output.GPUs) == 0 {
		return nil, hal.ErrDeviceNotFound
	}

	gpu := output.GPUs[0]

	health := &device.DeviceHealth{
		Status: hal.HealthStatusHealthy,
		Issues: []string{},
	}

	if strings.HasPrefix(gpu.Performance, "P") {
		perfLevel := gpu.Performance[1:]
		if perfLevel != "" {
			if level, err := strconv.Atoi(perfLevel); err == nil && level >= 8 {
				health.Status = hal.HealthStatusWarning
				health.Issues = append(health.Issues, "GPU in low performance state")
			}
		}
	}

	temp := parseTemperature(gpu.Temperature.GPUTemp)
	if temp > 90 {
		health.Status = hal.HealthStatusCritical
		health.Issues = append(health.Issues, fmt.Sprintf("High temperature: %.0f°C", temp))
	} else if temp > 80 {
		if health.Status != hal.HealthStatusCritical {
			health.Status = hal.HealthStatusWarning
		}
		health.Issues = append(health.Issues, fmt.Sprintf("Elevated temperature: %.0f°C", temp))
	}

	memUsed := parseMemory(gpu.FBMemoryUsage.Used)
	memTotal := parseMemory(gpu.FBMemoryUsage.Total)
	if memTotal > 0 {
		memPercent := float64(memUsed) / float64(memTotal) * 100
		if memPercent > 95 {
			if health.Status != hal.HealthStatusCritical {
				health.Status = hal.HealthStatusWarning
			}
			health.Issues = append(health.Issues, "High memory usage")
		}
	}

	if len(health.Issues) == 0 {
		health.Issues = nil
	}

	return health, nil
}

func (p *Provider) SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error {
	if limitWatts <= 0 {
		return hal.ErrNotSupported.WithCause(fmt.Errorf("invalid power limit: %.2f", limitWatts))
	}
	return p.smi.SetPowerLimit(ctx, p.toSMIID(deviceID), uint(limitWatts))
}

func (p *Provider) getCachedInfo(deviceID string) *hal.HardwareInfo {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	if time.Since(p.cacheTime) > p.cacheTTL {
		return nil
	}

	return p.cache[deviceID]
}

func (p *Provider) toSMIID(deviceID string) string {
	if idx := p.extractIndex(deviceID); idx >= 0 {
		return fmt.Sprintf("%d", idx)
	}
	return deviceID
}

func (p *Provider) extractIndex(deviceID string) int {
	var idx int
	if _, err := fmt.Sscanf(deviceID, "nvidia-%d", &idx); err == nil {
		return idx
	}
	if _, err := fmt.Sscanf(deviceID, "gpu-%d", &idx); err == nil {
		return idx
	}
	if _, err := fmt.Sscanf(deviceID, "%d", &idx); err == nil {
		return idx
	}
	return -1
}
