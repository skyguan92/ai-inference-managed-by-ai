package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type mockDeviceCommand struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockDeviceCommand) Name() string              { return m.name }
func (m *mockDeviceCommand) Domain() string            { return "device" }
func (m *mockDeviceCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockDeviceCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockDeviceCommand) Description() string       { return "" }
func (m *mockDeviceCommand) Examples() []unit.Example  { return nil }
func (m *mockDeviceCommand) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}

type mockDeviceQuery struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockDeviceQuery) Name() string              { return m.name }
func (m *mockDeviceQuery) Domain() string            { return "device" }
func (m *mockDeviceQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockDeviceQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockDeviceQuery) Description() string       { return "" }
func (m *mockDeviceQuery) Examples() []unit.Example  { return nil }
func (m *mockDeviceQuery) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}

func TestDeviceService_DetectAll_Success(t *testing.T) {
	registry := unit.NewRegistry()

	detectCmd := &mockDeviceCommand{
		name: "device.detect",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"devices": []map[string]any{
					{
						"id":           "gpu-0",
						"name":         "NVIDIA RTX 4090",
						"vendor":       "NVIDIA",
						"type":         "gpu",
						"architecture": "Ada Lovelace",
						"memory":       uint64(24 * 1024 * 1024 * 1024),
						"capabilities": []string{"cuda", "tensor"},
					},
					{
						"id":           "gpu-1",
						"name":         "NVIDIA RTX 3090",
						"vendor":       "NVIDIA",
						"type":         "gpu",
						"architecture": "Ampere",
						"memory":       uint64(24 * 1024 * 1024 * 1024),
						"capabilities": []string{"cuda", "tensor"},
					},
				},
			}, nil
		},
	}
	_ = registry.RegisterCommand(detectCmd)

	svc := NewDeviceService(registry)
	result, err := svc.DetectAll(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result.Devices) != 2 {
		t.Errorf("expected 2 devices, got: %d", len(result.Devices))
	}
	if result.Devices[0].Name != "NVIDIA RTX 4090" {
		t.Errorf("expected first device 'NVIDIA RTX 4090', got: %s", result.Devices[0].Name)
	}
}

func TestDeviceService_DetectAll_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewDeviceService(registry)

	_, err := svc.DetectAll(context.Background())
	if err == nil {
		t.Fatal("expected error when command not found")
	}
}

func TestDeviceService_DetectAll_ExecuteError(t *testing.T) {
	registry := unit.NewRegistry()

	detectCmd := &mockDeviceCommand{
		name: "device.detect",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("detection failed")
		},
	}
	_ = registry.RegisterCommand(detectCmd)

	svc := NewDeviceService(registry)
	_, err := svc.DetectAll(context.Background())

	if err == nil {
		t.Fatal("expected error when detection fails")
	}
}

func TestDeviceService_GetDeviceInfo_Success(t *testing.T) {
	registry := unit.NewRegistry()

	infoQuery := &mockDeviceQuery{
		name: "device.info",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"id":           "gpu-0",
				"name":         "NVIDIA RTX 4090",
				"vendor":       "NVIDIA",
				"architecture": "Ada Lovelace",
				"memory":       uint64(24 * 1024 * 1024 * 1024),
				"capabilities": []string{"cuda", "tensor", "fp8"},
			}, nil
		},
	}
	_ = registry.RegisterQuery(infoQuery)

	svc := NewDeviceService(registry)
	info, err := svc.GetDeviceInfo(context.Background(), "gpu-0")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if info.Name != "NVIDIA RTX 4090" {
		t.Errorf("expected name 'NVIDIA RTX 4090', got: %s", info.Name)
	}
	if info.Memory != 24*1024*1024*1024 {
		t.Errorf("expected memory 24GB, got: %d", info.Memory)
	}
}

func TestDeviceService_GetDeviceInfo_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewDeviceService(registry)

	_, err := svc.GetDeviceInfo(context.Background(), "gpu-0")
	if err == nil {
		t.Fatal("expected error when query not found")
	}
}

func TestDeviceService_GetMetricsWithHistory_Success(t *testing.T) {
	registry := unit.NewRegistry()

	metricsQuery := &mockDeviceQuery{
		name: "device.metrics",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"utilization":  75.5,
				"temperature":  65.0,
				"power":        250.0,
				"memory_used":  uint64(12 * 1024 * 1024 * 1024),
				"memory_total": uint64(24 * 1024 * 1024 * 1024),
				"history": []any{
					map[string]any{
						"timestamp":   int64(1700000000),
						"utilization": 70.0,
						"temperature": 60.0,
						"power":       200.0,
					},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(metricsQuery)

	svc := NewDeviceService(registry)
	metrics, err := svc.GetMetricsWithHistory(context.Background(), "gpu-0", true)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if metrics.Utilization != 75.5 {
		t.Errorf("expected utilization 75.5, got: %f", metrics.Utilization)
	}
	if len(metrics.History) != 1 {
		t.Errorf("expected 1 history item, got: %d", len(metrics.History))
	}
}

func TestDeviceService_GetMetricsWithHistory_NoHistory(t *testing.T) {
	registry := unit.NewRegistry()

	metricsQuery := &mockDeviceQuery{
		name: "device.metrics",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"utilization":  50.0,
				"temperature":  55.0,
				"power":        150.0,
				"memory_used":  uint64(8 * 1024 * 1024 * 1024),
				"memory_total": uint64(24 * 1024 * 1024 * 1024),
			}, nil
		},
	}
	_ = registry.RegisterQuery(metricsQuery)

	svc := NewDeviceService(registry)
	metrics, err := svc.GetMetricsWithHistory(context.Background(), "gpu-0", false)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(metrics.History) != 0 {
		t.Errorf("expected 0 history items, got: %d", len(metrics.History))
	}
}

func TestDeviceService_CheckHealth_Success(t *testing.T) {
	registry := unit.NewRegistry()

	healthQuery := &mockDeviceQuery{
		name: "device.health",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"status": "healthy",
				"issues": []string{},
			}, nil
		},
	}
	_ = registry.RegisterQuery(healthQuery)

	svc := NewDeviceService(registry)
	health, err := svc.CheckHealth(context.Background(), "gpu-0")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected status 'healthy', got: %s", health.Status)
	}
}

func TestDeviceService_CheckHealth_WithIssues(t *testing.T) {
	registry := unit.NewRegistry()

	healthQuery := &mockDeviceQuery{
		name: "device.health",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"status": "warning",
				"issues": []string{"high_temperature", "elevated_power"},
			}, nil
		},
	}
	_ = registry.RegisterQuery(healthQuery)

	svc := NewDeviceService(registry)
	health, err := svc.CheckHealth(context.Background(), "gpu-0")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "warning" {
		t.Errorf("expected status 'warning', got: %s", health.Status)
	}
	if len(health.Issues) != 2 {
		t.Errorf("expected 2 issues, got: %d", len(health.Issues))
	}
}

func TestDeviceService_SetPowerLimit_Success(t *testing.T) {
	registry := unit.NewRegistry()

	powerCmd := &mockDeviceCommand{
		name: "device.set_power_limit",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			if inputMap["limit_watts"].(float64) != 300.0 {
				t.Errorf("expected limit_watts 300.0, got: %v", inputMap["limit_watts"])
			}
			return map[string]any{"success": true}, nil
		},
	}
	_ = registry.RegisterCommand(powerCmd)

	svc := NewDeviceService(registry)
	result, err := svc.SetPowerLimit(context.Background(), "gpu-0", 300.0)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestDeviceService_SetPowerLimit_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewDeviceService(registry)

	_, err := svc.SetPowerLimit(context.Background(), "gpu-0", 300.0)
	if err == nil {
		t.Fatal("expected error when command not found")
	}
}

func TestDeviceService_GetAllDevicesHealth_Success(t *testing.T) {
	registry := unit.NewRegistry()

	healthQuery := &mockDeviceQuery{
		name: "device.health",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"devices": []any{
					map[string]any{
						"device_id": "gpu-0",
						"status":    "healthy",
						"issues":    []string{},
					},
					map[string]any{
						"device_id": "gpu-1",
						"status":    "warning",
						"issues":    []string{"high_temperature"},
					},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(healthQuery)

	svc := NewDeviceService(registry)
	result, err := svc.GetAllDevicesHealth(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result.Devices) != 2 {
		t.Errorf("expected 2 devices, got: %d", len(result.Devices))
	}
	if result.Overall != "warning" {
		t.Errorf("expected overall 'warning', got: %s", result.Overall)
	}
}

func TestDeviceService_GetAllDevicesHealth_CriticalOverall(t *testing.T) {
	registry := unit.NewRegistry()

	healthQuery := &mockDeviceQuery{
		name: "device.health",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"devices": []any{
					map[string]any{
						"device_id": "gpu-0",
						"status":    "healthy",
						"issues":    []string{},
					},
					map[string]any{
						"device_id": "gpu-1",
						"status":    "critical",
						"issues":    []string{"hardware_failure"},
					},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(healthQuery)

	svc := NewDeviceService(registry)
	result, err := svc.GetAllDevicesHealth(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Overall != "critical" {
		t.Errorf("expected overall 'critical', got: %s", result.Overall)
	}
}

func TestDeviceService_GetAllDevicesHealth_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewDeviceService(registry)

	_, err := svc.GetAllDevicesHealth(context.Background())
	if err == nil {
		t.Fatal("expected error when query not found")
	}
}
