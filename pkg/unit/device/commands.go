package device

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
\t"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

// Domain errors are defined in errors.go

type DeviceProvider interface {
	Detect(ctx context.Context) ([]DeviceInfo, error)
	GetDevice(ctx context.Context, deviceID string) (*DeviceInfo, error)
	GetMetrics(ctx context.Context, deviceID string) (*DeviceMetrics, error)
	GetHealth(ctx context.Context, deviceID string) (*DeviceHealth, error)
	SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error
}

type DeviceMetrics struct {
	Utilization float64 `json:"utilization"`
	Temperature float64 `json:"temperature"`
	Power       float64 `json:"power"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
}

type DeviceHealth struct {
	Status string   `json:"status"`
	Issues []string `json:"issues,omitempty"`
}

type DetectCommand struct {
	provider DeviceProvider
	events   unit.EventPublisher
}

func NewDetectCommand(provider DeviceProvider) *DetectCommand {
	return &DetectCommand{provider: provider}
}

func NewDetectCommandWithEvents(provider DeviceProvider, events unit.EventPublisher) *DetectCommand {
	return &DetectCommand{provider: provider, events: events}
}

func (c *DetectCommand) Name() string {
	return "device.detect"
}

func (c *DetectCommand) Domain() string {
	return "device"
}

func (c *DetectCommand) Description() string {
	return "Detect hardware devices on the system"
}

func (c *DetectCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type:       "object",
		Properties: map[string]unit.Field{},
	}
}

func (c *DetectCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"devices": {
				Name: "devices",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
							"vendor":       {Name: "vendor", Schema: unit.Schema{Type: "string"}},
							"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
							"memory":       {Name: "memory", Schema: unit.Schema{Type: "number"}},
							"architecture": {Name: "architecture", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (c *DetectCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"devices": []map[string]any{{"id": "gpu-0", "name": "NVIDIA RTX 4090", "vendor": "NVIDIA", "type": "gpu", "memory": 24564}}},
			Description: "Detect all hardware devices",
		},
	}
}

func (c *DetectCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	devices, err := c.provider.Detect(ctx)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("detect devices: %w", err)
	}

	result := make([]map[string]any, len(devices))
	for i, d := range devices {
		result[i] = map[string]any{
			"id":           d.ID,
			"name":         d.Name,
			"vendor":       d.Vendor,
			"type":         d.Type,
			"memory":       d.Memory,
			"architecture": d.Architecture,
		}
	}

	output := map[string]any{"devices": result}
	ec.PublishCompleted(output)
	return output, nil
}

type SetPowerLimitCommand struct {
	provider DeviceProvider
	events   unit.EventPublisher
}

func NewSetPowerLimitCommand(provider DeviceProvider) *SetPowerLimitCommand {
	return &SetPowerLimitCommand{provider: provider}
}

func NewSetPowerLimitCommandWithEvents(provider DeviceProvider, events unit.EventPublisher) *SetPowerLimitCommand {
	return &SetPowerLimitCommand{provider: provider, events: events}
}

func (c *SetPowerLimitCommand) Name() string {
	return "device.set_power_limit"
}

func (c *SetPowerLimitCommand) Domain() string {
	return "device"
}

func (c *SetPowerLimitCommand) Description() string {
	return "Set power consumption limit for a device"
}

func (c *SetPowerLimitCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"device_id": {
				Name: "device_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Device identifier",
				},
			},
			"limit_watts": {
				Name: "limit_watts",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Power limit in watts",
					Min:         ptrs.Float64(0),
				},
			},
		},
		Required: []string{"device_id", "limit_watts"},
	}
}

func (c *SetPowerLimitCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *SetPowerLimitCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"device_id": "gpu-0", "limit_watts": 250.0},
			Output:      map[string]any{"success": true},
			Description: "Set power limit to 250 watts for gpu-0",
		},
	}
}

func (c *SetPowerLimitCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	deviceID, _ := inputMap["device_id"].(string)
	if deviceID == "" {
		err := ErrInvalidDeviceID
		ec.PublishFailed(err)
		return nil, err
	}

	limitWatts, ok := toFloat64(inputMap["limit_watts"])
	if !ok || limitWatts <= 0 {
		err := ErrInvalidPowerLimit
		ec.PublishFailed(err)
		return nil, err
	}

	if err := c.provider.SetPowerLimit(ctx, deviceID, limitWatts); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("set power limit for device %s: %w", deviceID, err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
