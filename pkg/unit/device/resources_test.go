package device

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestDeviceInfoResource_URI(t *testing.T) {
	r := NewDeviceInfoResource("gpu-0", nil)
	expected := "asms://device/gpu-0/info"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestDeviceInfoResource_Domain(t *testing.T) {
	r := NewDeviceInfoResource("gpu-0", nil)
	if r.Domain() != "device" {
		t.Errorf("expected domain 'device', got '%s'", r.Domain())
	}
}

func TestDeviceInfoResource_Schema(t *testing.T) {
	r := NewDeviceInfoResource("gpu-0", nil)
	schema := r.Schema()
	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["id"]; !ok {
		t.Error("expected 'id' property in schema")
	}
}

func TestDeviceInfoResource_Get(t *testing.T) {
	tests := []struct {
		name     string
		provider DeviceProvider
		deviceID string
		wantErr  bool
	}{
		{
			name: "successful get",
			provider: &mockProvider{
				devices: []DeviceInfo{
					{ID: "gpu-0", Name: "RTX 4090", Vendor: "NVIDIA", Architecture: "Ada Lovelace", Memory: 24564},
				},
			},
			deviceID: "gpu-0",
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			deviceID: "gpu-0",
			wantErr:  true,
		},
		{
			name: "device not found",
			provider: &mockProvider{
				err: ErrDeviceNotFound,
			},
			deviceID: "gpu-0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewDeviceInfoResource(tt.deviceID, tt.provider)
			result, err := r.Get(context.Background())

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

			if resultMap["id"] != tt.deviceID {
				t.Errorf("expected id=%s, got %v", tt.deviceID, resultMap["id"])
			}
		})
	}
}

func TestDeviceMetricsResource_URI(t *testing.T) {
	r := NewDeviceMetricsResource("gpu-0", nil)
	expected := "asms://device/gpu-0/metrics"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestDeviceMetricsResource_Get(t *testing.T) {
	tests := []struct {
		name     string
		provider DeviceProvider
		deviceID string
		wantErr  bool
	}{
		{
			name: "successful get",
			provider: &mockProvider{
				metrics: &DeviceMetrics{
					Utilization: 75.5,
					Temperature: 65.0,
					Power:       200.0,
					MemoryUsed:  16384000000,
					MemoryTotal: 24564000000,
				},
			},
			deviceID: "gpu-0",
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			deviceID: "gpu-0",
			wantErr:  true,
		},
		{
			name: "provider error",
			provider: &mockProvider{
				err: errors.New("metrics error"),
			},
			deviceID: "gpu-0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewDeviceMetricsResource(tt.deviceID, tt.provider)
			result, err := r.Get(context.Background())

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

			if _, ok := resultMap["utilization"]; !ok {
				t.Error("expected 'utilization' field")
			}
		})
	}
}

func TestDeviceHealthResource_URI(t *testing.T) {
	r := NewDeviceHealthResource("gpu-0", nil)
	expected := "asms://device/gpu-0/health"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestDeviceHealthResource_Get(t *testing.T) {
	tests := []struct {
		name       string
		provider   DeviceProvider
		deviceID   string
		wantErr    bool
		wantStatus string
	}{
		{
			name: "healthy device",
			provider: &mockProvider{
				health: &DeviceHealth{Status: "healthy", Issues: []string{}},
			},
			deviceID:   "gpu-0",
			wantErr:    false,
			wantStatus: "healthy",
		},
		{
			name: "warning device",
			provider: &mockProvider{
				health: &DeviceHealth{Status: "warning", Issues: []string{"High temp"}},
			},
			deviceID:   "gpu-0",
			wantErr:    false,
			wantStatus: "warning",
		},
		{
			name:     "nil provider",
			provider: nil,
			deviceID: "gpu-0",
			wantErr:  true,
		},
		{
			name: "provider error",
			provider: &mockProvider{
				err: errors.New("health error"),
			},
			deviceID: "gpu-0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewDeviceHealthResource(tt.deviceID, tt.provider)
			result, err := r.Get(context.Background())

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

			if status, ok := resultMap["status"].(string); ok {
				if status != tt.wantStatus {
					t.Errorf("expected status=%s, got %s", tt.wantStatus, status)
				}
			} else {
				t.Error("expected 'status' to be string")
			}
		})
	}
}

func TestDeviceInfoResource_Watch(t *testing.T) {
	provider := &mockProvider{
		devices: []DeviceInfo{
			{ID: "gpu-0", Name: "RTX 4090", Vendor: "NVIDIA"},
		},
	}
	r := NewDeviceInfoResource("gpu-0", provider)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok {
			if update.URI != r.URI() {
				t.Errorf("expected URI=%s, got %s", r.URI(), update.URI)
			}
		}
	case <-ctx.Done():
	}
}

func TestDeviceHealthResource_Watch(t *testing.T) {
	provider := &mockProvider{
		health: &DeviceHealth{Status: "healthy", Issues: []string{}},
	}
	r := NewDeviceHealthResource("gpu-0", provider)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	receivedUpdate := false
	select {
	case update, ok := <-ch:
		if ok {
			receivedUpdate = true
			if update.URI != r.URI() {
				t.Errorf("expected URI=%s, got %s", r.URI(), update.URI)
			}
		}
	case <-ctx.Done():
	}

	if !receivedUpdate {
		t.Log("No update received before context timeout (expected behavior for short test)")
	}
}

func TestParseDeviceResourceURI(t *testing.T) {
	tests := []struct {
		uri          string
		wantType     string
		wantDeviceID string
		wantOK       bool
	}{
		{"asms://device/gpu-0/info", "info", "gpu-0", true},
		{"asms://device/gpu-0/metrics", "metrics", "gpu-0", true},
		{"asms://device/gpu-0/health", "health", "gpu-0", true},
		{"asms://device/gpu-1/info", "info", "gpu-1", true},
		{"asms://model/gpu-0/info", "", "", false},
		{"asms://device/gpu-0", "", "", false},
		{"asms://device/gpu-0/info/extra", "", "", false},
		{"invalid-uri", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			resourceType, deviceID, ok := ParseDeviceResourceURI(tt.uri)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if resourceType != tt.wantType {
				t.Errorf("expected type=%s, got %s", tt.wantType, resourceType)
			}
			if deviceID != tt.wantDeviceID {
				t.Errorf("expected deviceID=%s, got %s", tt.wantDeviceID, deviceID)
			}
		})
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewDeviceInfoResource("gpu-0", nil)
	var _ unit.Resource = NewDeviceMetricsResource("gpu-0", nil)
	var _ unit.Resource = NewDeviceHealthResource("gpu-0", nil)
}
