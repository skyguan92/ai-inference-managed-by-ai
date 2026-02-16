package device

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type DeviceInfoResource struct {
	deviceID string
	provider DeviceProvider
}

func NewDeviceInfoResource(deviceID string, provider DeviceProvider) *DeviceInfoResource {
	return &DeviceInfoResource{
		deviceID: deviceID,
		provider: provider,
	}
}

func (r *DeviceInfoResource) URI() string {
	return fmt.Sprintf("asms://device/%s/info", r.deviceID)
}

func (r *DeviceInfoResource) Domain() string {
	return "device"
}

func (r *DeviceInfoResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Device information resource",
		Properties: map[string]unit.Field{
			"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
			"vendor":       {Name: "vendor", Schema: unit.Schema{Type: "string"}},
			"architecture": {Name: "architecture", Schema: unit.Schema{Type: "string"}},
			"capabilities": {Name: "capabilities", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"memory":       {Name: "memory", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *DeviceInfoResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	device, err := r.provider.GetDevice(ctx, r.deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device %s info: %w", r.deviceID, err)
	}

	return map[string]any{
		"id":           device.ID,
		"name":         device.Name,
		"vendor":       device.Vendor,
		"architecture": device.Architecture,
		"capabilities": device.Capabilities,
		"memory":       device.Memory,
	}, nil
}

func (r *DeviceInfoResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 1)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
					Error:     err,
				}
			}
		}
	}()

	return ch, nil
}

type DeviceMetricsResource struct {
	deviceID string
	provider DeviceProvider
	interval time.Duration
}

func NewDeviceMetricsResource(deviceID string, provider DeviceProvider) *DeviceMetricsResource {
	return &DeviceMetricsResource{
		deviceID: deviceID,
		provider: provider,
		interval: 5 * time.Second,
	}
}

func (r *DeviceMetricsResource) URI() string {
	return fmt.Sprintf("asms://device/%s/metrics", r.deviceID)
}

func (r *DeviceMetricsResource) Domain() string {
	return "device"
}

func (r *DeviceMetricsResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Real-time device metrics resource",
		Properties: map[string]unit.Field{
			"utilization":  {Name: "utilization", Schema: unit.Schema{Type: "number"}},
			"temperature":  {Name: "temperature", Schema: unit.Schema{Type: "number"}},
			"power":        {Name: "power", Schema: unit.Schema{Type: "number"}},
			"memory_used":  {Name: "memory_used", Schema: unit.Schema{Type: "number"}},
			"memory_total": {Name: "memory_total", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *DeviceMetricsResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	metrics, err := r.provider.GetMetrics(ctx, r.deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device %s metrics: %w", r.deviceID, err)
	}

	return map[string]any{
		"utilization":  metrics.Utilization,
		"temperature":  metrics.Temperature,
		"power":        metrics.Power,
		"memory_used":  metrics.MemoryUsed,
		"memory_total": metrics.MemoryTotal,
	}, nil
}

func (r *DeviceMetricsResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "update",
					Data:      data,
					Error:     err,
				}
			}
		}
	}()

	return ch, nil
}

type DeviceHealthResource struct {
	deviceID string
	provider DeviceProvider
	last     *DeviceHealth
	mu       sync.RWMutex
}

func NewDeviceHealthResource(deviceID string, provider DeviceProvider) *DeviceHealthResource {
	return &DeviceHealthResource{
		deviceID: deviceID,
		provider: provider,
	}
}

func (r *DeviceHealthResource) URI() string {
	return fmt.Sprintf("asms://device/%s/health", r.deviceID)
}

func (r *DeviceHealthResource) Domain() string {
	return "device"
}

func (r *DeviceHealthResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Device health status resource",
		Properties: map[string]unit.Field{
			"status": {Name: "status", Schema: unit.Schema{Type: "string", Enum: []any{"healthy", "warning", "critical", "unknown"}}},
			"issues": {Name: "issues", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
		},
	}
}

func (r *DeviceHealthResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	health, err := r.provider.GetHealth(ctx, r.deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device %s health: %w", r.deviceID, err)
	}

	r.mu.Lock()
	r.last = health
	r.mu.Unlock()

	return map[string]any{
		"status": health.Status,
		"issues": health.Issues,
	}, nil
}

func (r *DeviceHealthResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				if err != nil {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "error",
						Error:     err,
					}
					continue
				}

				r.mu.RLock()
				lastStatus := ""
				if r.last != nil {
					lastStatus = r.last.Status
				}
				r.mu.RUnlock()

				dataMap, ok := data.(map[string]any)
				if ok && lastStatus != "" {
					newStatus, _ := dataMap["status"].(string)
					if newStatus != lastStatus {
						ch <- unit.ResourceUpdate{
							URI:       r.URI(),
							Timestamp: time.Now(),
							Operation: "health_changed",
							Data:      data,
						}
					}
				} else {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "refresh",
						Data:      data,
					}
				}
			}
		}
	}()

	return ch, nil
}

func ParseDeviceResourceURI(uri string) (resourceType, deviceID string, ok bool) {
	if !strings.HasPrefix(uri, "asms://device/") {
		return "", "", false
	}

	parts := strings.Split(strings.TrimPrefix(uri, "asms://device/"), "/")
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[1], parts[0], true
}
