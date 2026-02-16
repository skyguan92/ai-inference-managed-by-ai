package device

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type InfoQuery struct {
	provider DeviceProvider
}

func NewInfoQuery(provider DeviceProvider) *InfoQuery {
	return &InfoQuery{provider: provider}
}

func (q *InfoQuery) Name() string {
	return "device.info"
}

func (q *InfoQuery) Domain() string {
	return "device"
}

func (q *InfoQuery) Description() string {
	return "Get detailed information about a device or all devices"
}

func (q *InfoQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"device_id": {
				Name: "device_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Device identifier (optional, returns all if not specified)",
				},
			},
		},
		Optional: []string{"device_id"},
	}
}

func (q *InfoQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
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

func (q *InfoQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"device_id": "gpu-0"},
			Output:      map[string]any{"id": "gpu-0", "name": "NVIDIA RTX 4090", "vendor": "NVIDIA", "architecture": "Ada Lovelace", "capabilities": []string{"cuda", "tensor"}, "memory": 24564},
			Description: "Get info for a specific device",
		},
		{
			Input:       map[string]any{},
			Output:      map[string]any{"devices": []map[string]any{{"id": "gpu-0", "name": "NVIDIA RTX 4090"}}},
			Description: "Get info for all devices when device_id is not provided",
		},
	}
}

func (q *InfoQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)
	deviceID, _ := inputMap["device_id"].(string)

	if deviceID != "" {
		device, err := q.provider.GetDevice(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("get device %s: %w", deviceID, err)
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

	devices, err := q.provider.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect devices: %w", err)
	}

	result := make([]map[string]any, len(devices))
	for i, d := range devices {
		result[i] = map[string]any{
			"id":           d.ID,
			"name":         d.Name,
			"vendor":       d.Vendor,
			"architecture": d.Architecture,
			"capabilities": d.Capabilities,
			"memory":       d.Memory,
		}
	}

	return map[string]any{"devices": result}, nil
}

type MetricsQuery struct {
	provider DeviceProvider
}

func NewMetricsQuery(provider DeviceProvider) *MetricsQuery {
	return &MetricsQuery{provider: provider}
}

func (q *MetricsQuery) Name() string {
	return "device.metrics"
}

func (q *MetricsQuery) Domain() string {
	return "device"
}

func (q *MetricsQuery) Description() string {
	return "Get real-time metrics for a device"
}

func (q *MetricsQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"device_id": {
				Name: "device_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Device identifier (optional, returns first device metrics if not specified)",
				},
			},
			"history": {
				Name: "history",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Include historical data",
				},
			},
		},
		Optional: []string{"device_id", "history"},
	}
}

func (q *MetricsQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"utilization":  {Name: "utilization", Schema: unit.Schema{Type: "number", Description: "GPU utilization percentage (0-100)"}},
			"temperature":  {Name: "temperature", Schema: unit.Schema{Type: "number", Description: "Temperature in Celsius"}},
			"power":        {Name: "power", Schema: unit.Schema{Type: "number", Description: "Power consumption in watts"}},
			"memory_used":  {Name: "memory_used", Schema: unit.Schema{Type: "number", Description: "Used memory in bytes"}},
			"memory_total": {Name: "memory_total", Schema: unit.Schema{Type: "number", Description: "Total memory in bytes"}},
		},
	}
}

func (q *MetricsQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"device_id": "gpu-0"},
			Output:      map[string]any{"utilization": 75.5, "temperature": 65.0, "power": 200.0, "memory_used": 16384000000, "memory_total": 24564000000},
			Description: "Get real-time metrics for a specific device",
		},
	}
}

func (q *MetricsQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)
	deviceID, _ := inputMap["device_id"].(string)

	if deviceID == "" {
		devices, err := q.provider.Detect(ctx)
		if err != nil {
			return nil, fmt.Errorf("detect devices: %w", err)
		}
		if len(devices) == 0 {
			return nil, ErrDeviceNotFound
		}
		deviceID = devices[0].ID
	}

	metrics, err := q.provider.GetMetrics(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get metrics for device %s: %w", deviceID, err)
	}

	return map[string]any{
		"utilization":  metrics.Utilization,
		"temperature":  metrics.Temperature,
		"power":        metrics.Power,
		"memory_used":  metrics.MemoryUsed,
		"memory_total": metrics.MemoryTotal,
	}, nil
}

type HealthQuery struct {
	provider DeviceProvider
}

func NewHealthQuery(provider DeviceProvider) *HealthQuery {
	return &HealthQuery{provider: provider}
}

func (q *HealthQuery) Name() string {
	return "device.health"
}

func (q *HealthQuery) Domain() string {
	return "device"
}

func (q *HealthQuery) Description() string {
	return "Check health status of a device"
}

func (q *HealthQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"device_id": {
				Name: "device_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Device identifier (optional, checks all devices if not specified)",
				},
			},
		},
		Optional: []string{"device_id"},
	}
}

func (q *HealthQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"status": {Name: "status", Schema: unit.Schema{Type: "string", Enum: []any{"healthy", "warning", "critical", "unknown"}}},
			"issues": {Name: "issues", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
		},
	}
}

func (q *HealthQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"device_id": "gpu-0"},
			Output:      map[string]any{"status": "healthy", "issues": []string{}},
			Description: "Health check for a healthy device",
		},
		{
			Input:       map[string]any{"device_id": "gpu-1"},
			Output:      map[string]any{"status": "warning", "issues": []string{"High temperature detected"}},
			Description: "Health check for a device with warnings",
		},
	}
}

func (q *HealthQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)
	deviceID, _ := inputMap["device_id"].(string)

	if deviceID != "" {
		health, err := q.provider.GetHealth(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("get health for device %s: %w", deviceID, err)
		}
		return map[string]any{
			"status": health.Status,
			"issues": health.Issues,
		}, nil
	}

	devices, err := q.provider.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect devices: %w", err)
	}

	results := make([]map[string]any, len(devices))
	for i, d := range devices {
		health, err := q.provider.GetHealth(ctx, d.ID)
		if err != nil {
			results[i] = map[string]any{
				"device_id": d.ID,
				"status":    "unknown",
				"issues":    []string{err.Error()},
			}
			continue
		}
		results[i] = map[string]any{
			"device_id": d.ID,
			"status":    health.Status,
			"issues":    health.Issues,
		}
	}

	return map[string]any{"devices": results}, nil
}
