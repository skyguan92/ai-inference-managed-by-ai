package service

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type DeviceService struct {
	registry *unit.Registry
}

func NewDeviceService(registry *unit.Registry) *DeviceService {
	return &DeviceService{registry: registry}
}

type DetectAllResult struct {
	Devices []DeviceInfo `json:"devices"`
}

type DeviceInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Vendor       string   `json:"vendor"`
	Type         string   `json:"type"`
	Architecture string   `json:"architecture,omitempty"`
	Memory       uint64   `json:"memory,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func (s *DeviceService) DetectAll(ctx context.Context) (*DetectAllResult, error) {
	cmd := s.registry.GetCommand("device.detect")
	if cmd == nil {
		return nil, fmt.Errorf("device.detect command not found")
	}

	output, err := cmd.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("detect all devices: %w", err)
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected output type from device.detect")
	}

	devicesRaw, ok := outputMap["devices"].([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected devices format from device.detect")
	}

	devices := make([]DeviceInfo, len(devicesRaw))
	for i, d := range devicesRaw {
		devices[i] = mapToDeviceInfo(d)
	}

	return &DetectAllResult{Devices: devices}, nil
}

type DeviceFullInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Vendor       string   `json:"vendor"`
	Architecture string   `json:"architecture,omitempty"`
	Memory       uint64   `json:"memory,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func (s *DeviceService) GetDeviceInfo(ctx context.Context, deviceID string) (*DeviceFullInfo, error) {
	query := s.registry.GetQuery("device.info")
	if query == nil {
		return nil, fmt.Errorf("device.info query not found")
	}

	output, err := query.Execute(ctx, map[string]any{"device_id": deviceID})
	if err != nil {
		return nil, fmt.Errorf("get device info for %s: %w", deviceID, err)
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected output type from device.info")
	}

	return &DeviceFullInfo{
		ID:           getString(outputMap, "id"),
		Name:         getString(outputMap, "name"),
		Vendor:       getString(outputMap, "vendor"),
		Architecture: getString(outputMap, "architecture"),
		Memory:       getUint64(outputMap, "memory"),
		Capabilities: getStringSlice(outputMap, "capabilities"),
	}, nil
}

type MetricsResult struct {
	Utilization float64             `json:"utilization"`
	Temperature float64             `json:"temperature"`
	Power       float64             `json:"power"`
	MemoryUsed  uint64              `json:"memory_used"`
	MemoryTotal uint64              `json:"memory_total"`
	History     []MetricHistoryItem `json:"history,omitempty"`
}

type MetricHistoryItem struct {
	Timestamp   int64   `json:"timestamp"`
	Utilization float64 `json:"utilization"`
	Temperature float64 `json:"temperature"`
	Power       float64 `json:"power"`
}

func (s *DeviceService) GetMetricsWithHistory(ctx context.Context, deviceID string, history bool) (*MetricsResult, error) {
	query := s.registry.GetQuery("device.metrics")
	if query == nil {
		return nil, fmt.Errorf("device.metrics query not found")
	}

	input := map[string]any{
		"device_id": deviceID,
		"history":   history,
	}

	output, err := query.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get metrics for device %s: %w", deviceID, err)
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected output type from device.metrics")
	}

	result := &MetricsResult{
		Utilization: getFloat64(outputMap, "utilization"),
		Temperature: getFloat64(outputMap, "temperature"),
		Power:       getFloat64(outputMap, "power"),
		MemoryUsed:  getUint64(outputMap, "memory_used"),
		MemoryTotal: getUint64(outputMap, "memory_total"),
	}

	if history {
		if histRaw, ok := outputMap["history"].([]any); ok {
			result.History = make([]MetricHistoryItem, len(histRaw))
			for i, h := range histRaw {
				if hMap, ok := h.(map[string]any); ok {
					result.History[i] = MetricHistoryItem{
						Timestamp:   getInt64(hMap, "timestamp"),
						Utilization: getFloat64(hMap, "utilization"),
						Temperature: getFloat64(hMap, "temperature"),
						Power:       getFloat64(hMap, "power"),
					}
				}
			}
		}
	}

	return result, nil
}

type HealthResult struct {
	Status string   `json:"status"`
	Issues []string `json:"issues,omitempty"`
}

func (s *DeviceService) CheckHealth(ctx context.Context, deviceID string) (*HealthResult, error) {
	query := s.registry.GetQuery("device.health")
	if query == nil {
		return nil, fmt.Errorf("device.health query not found")
	}

	output, err := query.Execute(ctx, map[string]any{"device_id": deviceID})
	if err != nil {
		return nil, fmt.Errorf("check health for device %s: %w", deviceID, err)
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected output type from device.health")
	}

	return &HealthResult{
		Status: getString(outputMap, "status"),
		Issues: getStringSlice(outputMap, "issues"),
	}, nil
}

type SetPowerLimitResult struct {
	Success bool `json:"success"`
}

func (s *DeviceService) SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) (*SetPowerLimitResult, error) {
	cmd := s.registry.GetCommand("device.set_power_limit")
	if cmd == nil {
		return nil, fmt.Errorf("device.set_power_limit command not found")
	}

	input := map[string]any{
		"device_id":   deviceID,
		"limit_watts": limitWatts,
	}

	output, err := cmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("set power limit for device %s: %w", deviceID, err)
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected output type from device.set_power_limit")
	}

	return &SetPowerLimitResult{
		Success: getBool(outputMap, "success"),
	}, nil
}

type AllDevicesHealthResult struct {
	Devices []DeviceHealthStatus `json:"devices"`
	Overall string               `json:"overall"`
}

type DeviceHealthStatus struct {
	DeviceID string   `json:"device_id"`
	Status   string   `json:"status"`
	Issues   []string `json:"issues,omitempty"`
}

func (s *DeviceService) GetAllDevicesHealth(ctx context.Context) (*AllDevicesHealthResult, error) {
	query := s.registry.GetQuery("device.health")
	if query == nil {
		return nil, fmt.Errorf("device.health query not found")
	}

	output, err := query.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("get all devices health: %w", err)
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected output type from device.health")
	}

	devicesRaw, ok := outputMap["devices"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected devices format from device.health")
	}

	devices := make([]DeviceHealthStatus, len(devicesRaw))
	overallStatus := "healthy"

	for i, d := range devicesRaw {
		if dMap, ok := d.(map[string]any); ok {
			status := getString(dMap, "status")
			devices[i] = DeviceHealthStatus{
				DeviceID: getString(dMap, "device_id"),
				Status:   status,
				Issues:   getStringSlice(dMap, "issues"),
			}

			if status == "critical" {
				overallStatus = "critical"
			} else if status == "warning" && overallStatus != "critical" {
				overallStatus = "warning"
			}
		}
	}

	return &AllDevicesHealthResult{
		Devices: devices,
		Overall: overallStatus,
	}, nil
}

func mapToDeviceInfo(m map[string]any) DeviceInfo {
	return DeviceInfo{
		ID:           getString(m, "id"),
		Name:         getString(m, "name"),
		Vendor:       getString(m, "vendor"),
		Type:         getString(m, "type"),
		Architecture: getString(m, "architecture"),
		Memory:       getUint64(m, "memory"),
		Capabilities: getStringSlice(m, "capabilities"),
	}
}
