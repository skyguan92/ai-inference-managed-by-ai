package resource

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeAllocated       = "resource.allocated"
	EventTypeReleased        = "resource.released"
	EventTypePressureWarning = "resource.pressure_warning"
	EventTypePreemption      = "resource.preemption"
)

type AllocatedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewAllocatedEvent(slotID string, memory uint64) *AllocatedEvent {
	return &AllocatedEvent{
		eventType: EventTypeAllocated,
		domain:    "resource",
		payload: map[string]any{
			"slot_id": slotID,
			"memory":  memory,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *AllocatedEvent) Type() string          { return e.eventType }
func (e *AllocatedEvent) Domain() string        { return e.domain }
func (e *AllocatedEvent) Payload() any          { return e.payload }
func (e *AllocatedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *AllocatedEvent) CorrelationID() string { return e.correlationID }

type ReleasedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewReleasedEvent(slotID string) *ReleasedEvent {
	return &ReleasedEvent{
		eventType: EventTypeReleased,
		domain:    "resource",
		payload: map[string]any{
			"slot_id": slotID,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ReleasedEvent) Type() string          { return e.eventType }
func (e *ReleasedEvent) Domain() string        { return e.domain }
func (e *ReleasedEvent) Payload() any          { return e.payload }
func (e *ReleasedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ReleasedEvent) CorrelationID() string { return e.correlationID }

type PressureWarningEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewPressureWarningEvent(pressure PressureLevel, threshold float64) *PressureWarningEvent {
	return &PressureWarningEvent{
		eventType: EventTypePressureWarning,
		domain:    "resource",
		payload: map[string]any{
			"pressure":  string(pressure),
			"threshold": threshold,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *PressureWarningEvent) Type() string          { return e.eventType }
func (e *PressureWarningEvent) Domain() string        { return e.domain }
func (e *PressureWarningEvent) Payload() any          { return e.payload }
func (e *PressureWarningEvent) Timestamp() time.Time  { return e.timestamp }
func (e *PressureWarningEvent) CorrelationID() string { return e.correlationID }

type PreemptionEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewPreemptionEvent(slotID, reason string) *PreemptionEvent {
	return &PreemptionEvent{
		eventType: EventTypePreemption,
		domain:    "resource",
		payload: map[string]any{
			"slot_id": slotID,
			"reason":  reason,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *PreemptionEvent) Type() string          { return e.eventType }
func (e *PreemptionEvent) Domain() string        { return e.domain }
func (e *PreemptionEvent) Payload() any          { return e.payload }
func (e *PreemptionEvent) Timestamp() time.Time  { return e.timestamp }
func (e *PreemptionEvent) CorrelationID() string { return e.correlationID }
