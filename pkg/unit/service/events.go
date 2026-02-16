package service

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeCreated = "service.created"
	EventTypeScaled  = "service.scaled"
	EventTypeFailed  = "service.failed"
	EventTypeStarted = "service.started"
	EventTypeStopped = "service.stopped"
)

type CreatedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewCreatedEvent(service *ModelService) *CreatedEvent {
	return &CreatedEvent{
		eventType: EventTypeCreated,
		domain:    "service",
		payload: map[string]any{
			"service_id":     service.ID,
			"model_id":       service.ModelID,
			"status":         service.Status,
			"replicas":       service.Replicas,
			"resource_class": service.ResourceClass,
			"created_at":     time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *CreatedEvent) Type() string          { return e.eventType }
func (e *CreatedEvent) Domain() string        { return e.domain }
func (e *CreatedEvent) Payload() any          { return e.payload }
func (e *CreatedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *CreatedEvent) CorrelationID() string { return e.correlationID }

type ScaledEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewScaledEvent(service *ModelService, oldReplicas, newReplicas int) *ScaledEvent {
	return &ScaledEvent{
		eventType: EventTypeScaled,
		domain:    "service",
		payload: map[string]any{
			"service_id":   service.ID,
			"model_id":     service.ModelID,
			"old_replicas": oldReplicas,
			"new_replicas": newReplicas,
			"timestamp":    time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ScaledEvent) Type() string          { return e.eventType }
func (e *ScaledEvent) Domain() string        { return e.domain }
func (e *ScaledEvent) Payload() any          { return e.payload }
func (e *ScaledEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ScaledEvent) CorrelationID() string { return e.correlationID }

type FailedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewFailedEvent(service *ModelService, errMsg string, errCode string) *FailedEvent {
	return &FailedEvent{
		eventType: EventTypeFailed,
		domain:    "service",
		payload: map[string]any{
			"service_id": service.ID,
			"model_id":   service.ModelID,
			"status":     service.Status,
			"error":      errMsg,
			"error_code": errCode,
			"timestamp":  time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *FailedEvent) Type() string          { return e.eventType }
func (e *FailedEvent) Domain() string        { return e.domain }
func (e *FailedEvent) Payload() any          { return e.payload }
func (e *FailedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *FailedEvent) CorrelationID() string { return e.correlationID }

type StartedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStartedEvent(service *ModelService) *StartedEvent {
	return &StartedEvent{
		eventType: EventTypeStarted,
		domain:    "service",
		payload: map[string]any{
			"service_id": service.ID,
			"model_id":   service.ModelID,
			"status":     service.Status,
			"replicas":   service.Replicas,
			"endpoints":  service.Endpoints,
			"started_at": time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *StartedEvent) Type() string          { return e.eventType }
func (e *StartedEvent) Domain() string        { return e.domain }
func (e *StartedEvent) Payload() any          { return e.payload }
func (e *StartedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *StartedEvent) CorrelationID() string { return e.correlationID }

type StoppedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStoppedEvent(service *ModelService, reason string) *StoppedEvent {
	return &StoppedEvent{
		eventType: EventTypeStopped,
		domain:    "service",
		payload: map[string]any{
			"service_id": service.ID,
			"model_id":   service.ModelID,
			"status":     service.Status,
			"reason":     reason,
			"stopped_at": time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *StoppedEvent) Type() string          { return e.eventType }
func (e *StoppedEvent) Domain() string        { return e.domain }
func (e *StoppedEvent) Payload() any          { return e.payload }
func (e *StoppedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *StoppedEvent) CorrelationID() string { return e.correlationID }
