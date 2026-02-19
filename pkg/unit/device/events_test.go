package device

import (
	"testing"
	"time"
)

func TestNewDetectedEvent(t *testing.T) {
	device := DeviceInfo{
		ID:           "gpu-0",
		Name:         "NVIDIA RTX 4090",
		Vendor:       "nvidia",
		Type:         "gpu",
		Architecture: "ada",
		Memory:       24 * 1024 * 1024 * 1024,
		Capabilities: []string{"cuda", "tensor"},
	}

	before := time.Now()
	evt := NewDetectedEvent(device)
	after := time.Now()

	if evt.Type() != EventTypeDetected {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeDetected)
	}
	if evt.Domain() != "device" {
		t.Errorf("Domain() = %q, want 'device'", evt.Domain())
	}
	if evt.Payload() == nil {
		t.Error("Payload() should not be nil")
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	ts := evt.Timestamp()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp() = %v, should be between %v and %v", ts, before, after)
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if _, exists := payload["device"]; !exists {
		t.Error("payload should contain 'device' key")
	}
}

func TestNewHealthChangedEvent(t *testing.T) {
	evt := NewHealthChangedEvent("gpu-0", "healthy", "degraded")

	if evt.Type() != EventTypeHealthChanged {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeHealthChanged)
	}
	if evt.Domain() != "device" {
		t.Errorf("Domain() = %q, want 'device'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	if evt.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["device_id"] != "gpu-0" {
		t.Errorf("payload[device_id] = %v, want 'gpu-0'", payload["device_id"])
	}
	if payload["old_status"] != "healthy" {
		t.Errorf("payload[old_status] = %v, want 'healthy'", payload["old_status"])
	}
	if payload["new_status"] != "degraded" {
		t.Errorf("payload[new_status] = %v, want 'degraded'", payload["new_status"])
	}
}

func TestNewMetricsAlertEvent(t *testing.T) {
	evt := NewMetricsAlertEvent("gpu-0", "temperature", 92.5, 85.0)

	if evt.Type() != EventTypeMetricsAlert {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeMetricsAlert)
	}
	if evt.Domain() != "device" {
		t.Errorf("Domain() = %q, want 'device'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["device_id"] != "gpu-0" {
		t.Errorf("payload[device_id] = %v, want 'gpu-0'", payload["device_id"])
	}
	if payload["metric"] != "temperature" {
		t.Errorf("payload[metric] = %v, want 'temperature'", payload["metric"])
	}
	if payload["value"] != 92.5 {
		t.Errorf("payload[value] = %v, want 92.5", payload["value"])
	}
	if payload["threshold"] != 85.0 {
		t.Errorf("payload[threshold] = %v, want 85.0", payload["threshold"])
	}
}

func TestDeviceEventTypeConstants(t *testing.T) {
	if EventTypeDetected != "device.detected" {
		t.Errorf("EventTypeDetected = %q, want 'device.detected'", EventTypeDetected)
	}
	if EventTypeHealthChanged != "device.health_changed" {
		t.Errorf("EventTypeHealthChanged = %q, want 'device.health_changed'", EventTypeHealthChanged)
	}
	if EventTypeMetricsAlert != "device.metrics_alert" {
		t.Errorf("EventTypeMetricsAlert = %q, want 'device.metrics_alert'", EventTypeMetricsAlert)
	}
}

func TestDeviceEvents_UniqueCorrelationIDs(t *testing.T) {
	device := DeviceInfo{ID: "gpu-0", Name: "Test GPU"}
	evt1 := NewDetectedEvent(device)
	evt2 := NewDetectedEvent(device)

	if evt1.CorrelationID() == evt2.CorrelationID() {
		t.Error("different events should have different correlation IDs")
	}
}
