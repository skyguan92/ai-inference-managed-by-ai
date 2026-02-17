package generic

import (
	"context"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/hal"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider()
	if got := p.Name(); got != "generic" {
		t.Errorf("Provider.Name() = %v, want %v", got, "generic")
	}
}

func TestProvider_Vendor(t *testing.T) {
	p := NewProvider()
	if got := p.Vendor(); got != "Generic" {
		t.Errorf("Provider.Vendor() = %v, want %v", got, "Generic")
	}
}

func TestProvider_Available(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()
	if got := p.Available(ctx); !got {
		t.Errorf("Provider.Available() = %v, want %v", got, true)
	}
}

func TestProvider_Detect(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	infos, err := p.Detect(ctx)
	if err != nil {
		t.Fatalf("Provider.Detect() error = %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("Provider.Detect() returned %d infos, want 1", len(infos))
	}

	info := infos[0]
	if info.ID != "cpu-0" {
		t.Errorf("HardwareInfo.ID = %v, want %v", info.ID, "cpu-0")
	}
	if info.Type != hal.DeviceTypeCPU {
		t.Errorf("HardwareInfo.Type = %v, want %v", info.Type, hal.DeviceTypeCPU)
	}
	if info.DiscoveredAt.IsZero() {
		t.Error("HardwareInfo.DiscoveredAt is zero")
	}
}

func TestProvider_GetInfo(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	tests := []struct {
		name     string
		deviceID string
		wantErr  bool
	}{
		{
			name:     "valid device cpu-0",
			deviceID: "cpu-0",
			wantErr:  false,
		},
		{
			name:     "valid device cpu",
			deviceID: "cpu",
			wantErr:  false,
		},
		{
			name:     "invalid device",
			deviceID: "invalid-device",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := p.GetInfo(ctx, tt.deviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.GetInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if info == nil {
					t.Error("Provider.GetInfo() returned nil info")
					return
				}
				if info.ID != "cpu-0" {
					t.Errorf("HardwareInfo.ID = %v, want %v", info.ID, "cpu-0")
				}
				if info.Type != hal.DeviceTypeCPU {
					t.Errorf("HardwareInfo.Type = %v, want %v", info.Type, hal.DeviceTypeCPU)
				}
			}
		})
	}
}

func TestProvider_GetInfo_DeviceNotFound(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	_, err := p.GetInfo(ctx, "nonexistent")
	if err == nil {
		t.Error("Provider.GetInfo() expected error for nonexistent device, got nil")
	}
}

func TestProvider_GetMetrics(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	tests := []struct {
		name     string
		deviceID string
		wantErr  bool
	}{
		{
			name:     "valid device cpu-0",
			deviceID: "cpu-0",
			wantErr:  false,
		},
		{
			name:     "valid device cpu",
			deviceID: "cpu",
			wantErr:  false,
		},
		{
			name:     "invalid device",
			deviceID: "invalid-device",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := p.GetMetrics(ctx, tt.deviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if metrics == nil {
					t.Error("Provider.GetMetrics() returned nil metrics")
					return
				}
				if metrics.Timestamp.IsZero() {
					t.Error("HardwareMetrics.Timestamp is zero")
				}
			}
		})
	}
}

func TestProvider_GetMetrics_DeviceNotFound(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	_, err := p.GetMetrics(ctx, "nonexistent")
	if err == nil {
		t.Error("Provider.GetMetrics() expected error for nonexistent device, got nil")
	}
}

func TestProvider_GetHealth(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	tests := []struct {
		name     string
		deviceID string
		wantErr  bool
	}{
		{
			name:     "valid device cpu-0",
			deviceID: "cpu-0",
			wantErr:  false,
		},
		{
			name:     "valid device cpu",
			deviceID: "cpu",
			wantErr:  false,
		},
		{
			name:     "invalid device",
			deviceID: "invalid-device",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health, err := p.GetHealth(ctx, tt.deviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.GetHealth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if health == nil {
					t.Error("Provider.GetHealth() returned nil health")
					return
				}
				if health.Timestamp.IsZero() {
					t.Error("HardwareHealth.Timestamp is zero")
				}
				if health.Status == "" {
					t.Error("HardwareHealth.Status is empty")
				}
				if health.Details == nil {
					t.Error("HardwareHealth.Details is nil")
				}
			}
		})
	}
}

func TestProvider_GetHealth_DeviceNotFound(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	_, err := p.GetHealth(ctx, "nonexistent")
	if err == nil {
		t.Error("Provider.GetHealth() expected error for nonexistent device, got nil")
	}
}

func TestProvider_SetPowerLimit_NotSupported(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	err := p.SetPowerLimit(ctx, "cpu-0", 100.0)
	if err == nil {
		t.Error("Provider.SetPowerLimit() expected error, got nil")
		return
	}

	providerErr, ok := err.(*hal.ProviderError)
	if !ok {
		t.Errorf("Provider.SetPowerLimit() error type = %T, want *hal.ProviderError", err)
		return
	}
	if providerErr.Message != "operation not supported" {
		t.Errorf("ProviderError.Message = %v, want %v", providerErr.Message, "operation not supported")
	}
}
