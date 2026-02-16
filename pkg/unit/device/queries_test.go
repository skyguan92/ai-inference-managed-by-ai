package device

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestInfoQuery_Name(t *testing.T) {
	q := NewInfoQuery(nil)
	if q.Name() != "device.info" {
		t.Errorf("expected name 'device.info', got '%s'", q.Name())
	}
}

func TestInfoQuery_Domain(t *testing.T) {
	q := NewInfoQuery(nil)
	if q.Domain() != "device" {
		t.Errorf("expected domain 'device', got '%s'", q.Domain())
	}
}

func TestInfoQuery_Schemas(t *testing.T) {
	q := NewInfoQuery(nil)

	inputSchema := q.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}

	outputSchema := q.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestInfoQuery_Execute(t *testing.T) {
	tests := []struct {
		name       string
		provider   DeviceProvider
		input      any
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "get specific device",
			provider: &mockProvider{
				devices: []DeviceInfo{
					{ID: "gpu-0", Name: "RTX 4090", Vendor: "NVIDIA", Architecture: "Ada Lovelace", Memory: 24564, Capabilities: []string{"cuda", "tensor"}},
				},
			},
			input:      map[string]any{"device_id": "gpu-0"},
			wantErr:    false,
			checkField: "name",
			checkValue: "RTX 4090",
		},
		{
			name: "get all devices",
			provider: &mockProvider{
				devices: []DeviceInfo{
					{ID: "gpu-0", Name: "RTX 4090", Vendor: "NVIDIA"},
					{ID: "gpu-1", Name: "RTX 4080", Vendor: "NVIDIA"},
				},
			},
			input:      map[string]any{},
			wantErr:    false,
			checkField: "devices",
			checkValue: nil,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"device_id": "gpu-0"},
			wantErr:  true,
		},
		{
			name: "device not found",
			provider: &mockProvider{
				devices: []DeviceInfo{},
				err:     ErrDeviceNotFound,
			},
			input:   map[string]any{"device_id": "gpu-0"},
			wantErr: true,
		},
		{
			name: "provider error on detect",
			provider: &mockProvider{
				err: errors.New("detection error"),
			},
			input:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewInfoQuery(tt.provider)
			result, err := q.Execute(context.Background(), tt.input)

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

			if tt.checkValue != nil {
				if val, exists := resultMap[tt.checkField]; exists {
					if val != tt.checkValue {
						t.Errorf("expected %s=%v, got %v", tt.checkField, tt.checkValue, val)
					}
				} else {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}
		})
	}
}

func TestMetricsQuery_Name(t *testing.T) {
	q := NewMetricsQuery(nil)
	if q.Name() != "device.metrics" {
		t.Errorf("expected name 'device.metrics', got '%s'", q.Name())
	}
}

func TestMetricsQuery_Domain(t *testing.T) {
	q := NewMetricsQuery(nil)
	if q.Domain() != "device" {
		t.Errorf("expected domain 'device', got '%s'", q.Domain())
	}
}

func TestMetricsQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider DeviceProvider
		input    any
		wantErr  bool
		checkVal float64
	}{
		{
			name: "get metrics for specific device",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				metrics: &DeviceMetrics{
					Utilization: 75.5,
					Temperature: 65.0,
					Power:       200.0,
					MemoryUsed:  16384000000,
					MemoryTotal: 24564000000,
				},
			},
			input:    map[string]any{"device_id": "gpu-0"},
			wantErr:  false,
			checkVal: 75.5,
		},
		{
			name: "get metrics without device_id",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				metrics: &DeviceMetrics{
					Utilization: 50.0,
					Temperature: 60.0,
					Power:       150.0,
				},
			},
			input:    map[string]any{},
			wantErr:  false,
			checkVal: 50.0,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"device_id": "gpu-0"},
			wantErr:  true,
		},
		{
			name: "no devices found",
			provider: &mockProvider{
				devices: []DeviceInfo{},
			},
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name: "provider error",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				err:     errors.New("metrics error"),
			},
			input:   map[string]any{"device_id": "gpu-0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewMetricsQuery(tt.provider)
			result, err := q.Execute(context.Background(), tt.input)

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

			if util, ok := resultMap["utilization"].(float64); ok {
				if util != tt.checkVal {
					t.Errorf("expected utilization=%f, got %f", tt.checkVal, util)
				}
			} else {
				t.Error("expected 'utilization' to be float64")
			}
		})
	}
}

func TestHealthQuery_Name(t *testing.T) {
	q := NewHealthQuery(nil)
	if q.Name() != "device.health" {
		t.Errorf("expected name 'device.health', got '%s'", q.Name())
	}
}

func TestHealthQuery_Domain(t *testing.T) {
	q := NewHealthQuery(nil)
	if q.Domain() != "device" {
		t.Errorf("expected domain 'device', got '%s'", q.Domain())
	}
}

func TestHealthQuery_Execute(t *testing.T) {
	tests := []struct {
		name        string
		provider    DeviceProvider
		input       any
		wantErr     bool
		checkStatus string
	}{
		{
			name: "healthy device",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				health:  &DeviceHealth{Status: "healthy", Issues: []string{}},
			},
			input:       map[string]any{"device_id": "gpu-0"},
			wantErr:     false,
			checkStatus: "healthy",
		},
		{
			name: "warning device",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				health:  &DeviceHealth{Status: "warning", Issues: []string{"High temperature"}},
			},
			input:       map[string]any{"device_id": "gpu-0"},
			wantErr:     false,
			checkStatus: "warning",
		},
		{
			name: "critical device",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				health:  &DeviceHealth{Status: "critical", Issues: []string{"Memory error", "Overheating"}},
			},
			input:       map[string]any{"device_id": "gpu-0"},
			wantErr:     false,
			checkStatus: "critical",
		},
		{
			name: "check all devices",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}, {ID: "gpu-1"}},
				health:  &DeviceHealth{Status: "healthy", Issues: []string{}},
			},
			input:       map[string]any{},
			wantErr:     false,
			checkStatus: "",
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"device_id": "gpu-0"},
			wantErr:  true,
		},
		{
			name: "provider error",
			provider: &mockProvider{
				devices: []DeviceInfo{{ID: "gpu-0"}},
				err:     errors.New("health check error"),
			},
			input:   map[string]any{"device_id": "gpu-0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewHealthQuery(tt.provider)
			result, err := q.Execute(context.Background(), tt.input)

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

			if tt.checkStatus != "" {
				status, ok := resultMap["status"].(string)
				if !ok {
					t.Error("expected 'status' to be string")
					return
				}
				if status != tt.checkStatus {
					t.Errorf("expected status=%s, got %s", tt.checkStatus, status)
				}
			}
		})
	}
}

func TestInfoQuery_Description(t *testing.T) {
	q := NewInfoQuery(nil)
	if q.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestInfoQuery_Examples(t *testing.T) {
	q := NewInfoQuery(nil)
	if len(q.Examples()) == 0 {
		t.Error("expected at least one example")
	}
}

func TestMetricsQuery_Description(t *testing.T) {
	q := NewMetricsQuery(nil)
	if q.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestMetricsQuery_Examples(t *testing.T) {
	q := NewMetricsQuery(nil)
	if len(q.Examples()) == 0 {
		t.Error("expected at least one example")
	}
}

func TestHealthQuery_Description(t *testing.T) {
	q := NewHealthQuery(nil)
	if q.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestHealthQuery_Examples(t *testing.T) {
	q := NewHealthQuery(nil)
	if len(q.Examples()) == 0 {
		t.Error("expected at least one example")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewInfoQuery(nil)
	var _ unit.Query = NewMetricsQuery(nil)
	var _ unit.Query = NewHealthQuery(nil)
}
