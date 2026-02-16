package device

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeDetected      = "device.detected"
	EventTypeHealthChanged = "device.health_changed"
	EventTypeMetricsAlert  = "device.metrics_alert"
)

type DeviceInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Vendor       string   `json:"vendor"`
	Type         string   `json:"type"`
	Architecture string   `json:"architecture,omitempty"`
	Memory       uint64   `json:"memory,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type DetectedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewDetectedEvent(device DeviceInfo) *DetectedEvent {
	return &DetectedEvent{
		eventType:     EventTypeDetected,
		domain:        "device",
		payload:       map[string]any{"device": device},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *DetectedEvent) Type() string          { return e.eventType }
func (e *DetectedEvent) Domain() string        { return e.domain }
func (e *DetectedEvent) Payload() any          { return e.payload }
func (e *DetectedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *DetectedEvent) CorrelationID() string { return e.correlationID }

type HealthChangedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewHealthChangedEvent(deviceID, oldStatus, newStatus string) *HealthChangedEvent {
	return &HealthChangedEvent{
		eventType: EventTypeHealthChanged,
		domain:    "device",
		payload: map[string]any{
			"device_id":  deviceID,
			"old_status": oldStatus,
			"new_status": newStatus,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *HealthChangedEvent) Type() string          { return e.eventType }
func (e *HealthChangedEvent) Domain() string        { return e.domain }
func (e *HealthChangedEvent) Payload() any          { return e.payload }
func (e *HealthChangedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *HealthChangedEvent) CorrelationID() string { return e.correlationID }

type MetricsAlertEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewMetricsAlertEvent(deviceID, metric string, value, threshold float64) *MetricsAlertEvent {
	return &MetricsAlertEvent{
		eventType: EventTypeMetricsAlert,
		domain:    "device",
		payload: map[string]any{
			"device_id": deviceID,
			"metric":    metric,
			"value":     value,
			"threshold": threshold,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *MetricsAlertEvent) Type() string          { return e.eventType }
func (e *MetricsAlertEvent) Domain() string        { return e.domain }
func (e *MetricsAlertEvent) Payload() any          { return e.payload }
func (e *MetricsAlertEvent) Timestamp() time.Time  { return e.timestamp }
func (e *MetricsAlertEvent) CorrelationID() string { return e.correlationID }
