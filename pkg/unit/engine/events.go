package engine

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeStarted       = "engine.started"
	EventTypeStopped       = "engine.stopped"
	EventTypeError         = "engine.error"
	EventTypeHealthChanged = "engine.health_changed"
	EventTypeStartProgress = "engine.start_progress"
)

type StartedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStartedEvent(engine *Engine, processID string) *StartedEvent {
	return &StartedEvent{
		eventType: EventTypeStarted,
		domain:    "engine",
		payload: map[string]any{
			"engine_id":  engine.ID,
			"name":       engine.Name,
			"type":       engine.Type,
			"process_id": processID,
			"status":     engine.Status,
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

func NewStoppedEvent(engine *Engine, reason string) *StoppedEvent {
	return &StoppedEvent{
		eventType: EventTypeStopped,
		domain:    "engine",
		payload: map[string]any{
			"engine_id":  engine.ID,
			"name":       engine.Name,
			"type":       engine.Type,
			"status":     engine.Status,
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

type ErrorEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewErrorEvent(engine *Engine, errMsg string, errCode string) *ErrorEvent {
	return &ErrorEvent{
		eventType: EventTypeError,
		domain:    "engine",
		payload: map[string]any{
			"engine_id":  engine.ID,
			"name":       engine.Name,
			"type":       engine.Type,
			"status":     engine.Status,
			"error":      errMsg,
			"error_code": errCode,
			"timestamp":  time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ErrorEvent) Type() string          { return e.eventType }
func (e *ErrorEvent) Domain() string        { return e.domain }
func (e *ErrorEvent) Payload() any          { return e.payload }
func (e *ErrorEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ErrorEvent) CorrelationID() string { return e.correlationID }

type HealthChangedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewHealthChangedEvent(engine *Engine, oldStatus, newStatus EngineStatus, details map[string]any) *HealthChangedEvent {
	payload := map[string]any{
		"engine_id":  engine.ID,
		"name":       engine.Name,
		"type":       engine.Type,
		"old_status": string(oldStatus),
		"new_status": string(newStatus),
		"timestamp":  time.Now().Unix(),
	}
	for k, v := range details {
		payload[k] = v
	}

	return &HealthChangedEvent{
		eventType:     EventTypeHealthChanged,
		domain:        "engine",
		payload:       payload,
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *HealthChangedEvent) Type() string          { return e.eventType }
func (e *HealthChangedEvent) Domain() string        { return e.domain }
func (e *HealthChangedEvent) Payload() any          { return e.payload }
func (e *HealthChangedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *HealthChangedEvent) CorrelationID() string { return e.correlationID }

type StartProgressEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStartProgressEvent(serviceID, phase, message string, progress int) *StartProgressEvent {
	return &StartProgressEvent{
		eventType: EventTypeStartProgress,
		domain:    "engine",
		payload: map[string]any{
			"service_id": serviceID,
			"phase":      phase,
			"message":    message,
			"progress":   progress,
			"timestamp":  time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *StartProgressEvent) Type() string          { return e.eventType }
func (e *StartProgressEvent) Domain() string        { return e.domain }
func (e *StartProgressEvent) Payload() any          { return e.payload }
func (e *StartProgressEvent) Timestamp() time.Time  { return e.timestamp }
func (e *StartProgressEvent) CorrelationID() string { return e.correlationID }
