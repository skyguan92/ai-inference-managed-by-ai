package device

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type mockProvider struct {
	devices    []DeviceInfo
	health     *DeviceHealth
	metrics    *DeviceMetrics
	err        error
	powerLimit float64
	powerSetID string
}

func (m *mockProvider) Detect(ctx context.Context) ([]DeviceInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.devices, nil
}

func (m *mockProvider) GetDevice(ctx context.Context, deviceID string) (*DeviceInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, d := range m.devices {
		if d.ID == deviceID {
			return &d, nil
		}
	}
	return nil, ErrDeviceNotFound
}

func (m *mockProvider) GetMetrics(ctx context.Context, deviceID string) (*DeviceMetrics, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metrics, nil
}

func (m *mockProvider) GetHealth(ctx context.Context, deviceID string) (*DeviceHealth, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.health, nil
}

func (m *mockProvider) SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error {
	if m.err != nil {
		return m.err
	}
	m.powerSetID = deviceID
	m.powerLimit = limitWatts
	return nil
}

func TestDetectCommand_Name(t *testing.T) {
	cmd := NewDetectCommand(nil)
	if cmd.Name() != "device.detect" {
		t.Errorf("expected name 'device.detect', got '%s'", cmd.Name())
	}
}

func TestDetectCommand_Domain(t *testing.T) {
	cmd := NewDetectCommand(nil)
	if cmd.Domain() != "device" {
		t.Errorf("expected domain 'device', got '%s'", cmd.Domain())
	}
}

func TestDetectCommand_Schemas(t *testing.T) {
	cmd := NewDetectCommand(nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
	if _, ok := outputSchema.Properties["devices"]; !ok {
		t.Error("expected 'devices' property in output schema")
	}
}

func TestDetectCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		provider    DeviceProvider
		input       any
		wantErr     bool
		wantDevices int
	}{
		{
			name: "successful detection",
			provider: &mockProvider{
				devices: []DeviceInfo{
					{ID: "gpu-0", Name: "RTX 4090", Vendor: "NVIDIA", Type: "gpu", Memory: 24564},
					{ID: "gpu-1", Name: "RTX 4080", Vendor: "NVIDIA", Type: "gpu", Memory: 16384},
				},
			},
			input:       map[string]any{},
			wantErr:     false,
			wantDevices: 2,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name: "provider error",
			provider: &mockProvider{
				err: errors.New("detection failed"),
			},
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name: "empty devices",
			provider: &mockProvider{
				devices: []DeviceInfo{},
			},
			input:       map[string]any{},
			wantErr:     false,
			wantDevices: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewDetectCommand(tt.provider)
			result, err := cmd.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			devices, ok := resultMap["devices"].([]map[string]any)
			if !ok {
				t.Error("expected 'devices' to be []map[string]any")
				return
			}

			if len(devices) != tt.wantDevices {
				t.Errorf("expected %d devices, got %d", tt.wantDevices, len(devices))
			}
		})
	}
}

func TestSetPowerLimitCommand_Name(t *testing.T) {
	cmd := NewSetPowerLimitCommand(nil)
	if cmd.Name() != "device.set_power_limit" {
		t.Errorf("expected name 'device.set_power_limit', got '%s'", cmd.Name())
	}
}

func TestSetPowerLimitCommand_Domain(t *testing.T) {
	cmd := NewSetPowerLimitCommand(nil)
	if cmd.Domain() != "device" {
		t.Errorf("expected domain 'device', got '%s'", cmd.Domain())
	}
}

func TestSetPowerLimitCommand_Schemas(t *testing.T) {
	cmd := NewSetPowerLimitCommand(nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestSetPowerLimitCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		provider    DeviceProvider
		input       any
		wantErr     bool
		wantSuccess bool
		checkPower  bool
		expectedPwr float64
	}{
		{
			name:     "successful set power limit",
			provider: &mockProvider{},
			input: map[string]any{
				"device_id":   "gpu-0",
				"limit_watts": 250.0,
			},
			wantErr:     false,
			wantSuccess: true,
			checkPower:  true,
			expectedPwr: 250.0,
		},
		{
			name:     "nil provider",
			provider: nil,
			input: map[string]any{
				"device_id":   "gpu-0",
				"limit_watts": 250.0,
			},
			wantErr: true,
		},
		{
			name:     "missing device_id",
			provider: &mockProvider{},
			input: map[string]any{
				"limit_watts": 250.0,
			},
			wantErr: true,
		},
		{
			name:     "missing limit_watts",
			provider: &mockProvider{},
			input: map[string]any{
				"device_id": "gpu-0",
			},
			wantErr: true,
		},
		{
			name:     "negative limit",
			provider: &mockProvider{},
			input: map[string]any{
				"device_id":   "gpu-0",
				"limit_watts": -100.0,
			},
			wantErr: true,
		},
		{
			name:     "zero limit",
			provider: &mockProvider{},
			input: map[string]any{
				"device_id":   "gpu-0",
				"limit_watts": 0.0,
			},
			wantErr: true,
		},
		{
			name: "provider error",
			provider: &mockProvider{
				err: errors.New("power limit failed"),
			},
			input: map[string]any{
				"device_id":   "gpu-0",
				"limit_watts": 250.0,
			},
			wantErr: true,
		},
		{
			name:     "invalid input type",
			provider: &mockProvider{},
			input:    "invalid",
			wantErr:  true,
		},
		{
			name:     "integer limit_watts",
			provider: &mockProvider{},
			input: map[string]any{
				"device_id":   "gpu-0",
				"limit_watts": 300,
			},
			wantErr:     false,
			wantSuccess: true,
			checkPower:  true,
			expectedPwr: 300.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSetPowerLimitCommand(tt.provider)
			result, err := cmd.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			success, ok := resultMap["success"].(bool)
			if !ok {
				t.Error("expected 'success' to be bool")
				return
			}

			if success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v", tt.wantSuccess, success)
			}

			if tt.checkPower {
				if mp, ok := tt.provider.(*mockProvider); ok {
					if mp.powerLimit != tt.expectedPwr {
						t.Errorf("expected power limit %f, got %f", tt.expectedPwr, mp.powerLimit)
					}
					if mp.powerSetID != "gpu-0" {
						t.Errorf("expected power set for 'gpu-0', got '%s'", mp.powerSetID)
					}
				}
			}
		})
	}
}

func TestDetectCommand_Description(t *testing.T) {
	cmd := NewDetectCommand(nil)
	if cmd.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestDetectCommand_Examples(t *testing.T) {
	cmd := NewDetectCommand(nil)
	examples := cmd.Examples()
	if len(examples) == 0 {
		t.Error("expected at least one example")
	}
}

func TestSetPowerLimitCommand_Description(t *testing.T) {
	cmd := NewSetPowerLimitCommand(nil)
	if cmd.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestSetPowerLimitCommand_Examples(t *testing.T) {
	cmd := NewSetPowerLimitCommand(nil)
	examples := cmd.Examples()
	if len(examples) == 0 {
		t.Error("expected at least one example")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewDetectCommand(nil)
	var _ unit.Command = NewSetPowerLimitCommand(nil)
}
