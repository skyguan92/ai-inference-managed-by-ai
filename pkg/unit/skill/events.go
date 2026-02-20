package skill

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeAdded    = "skill.added"
	EventTypeRemoved  = "skill.removed"
	EventTypeEnabled  = "skill.enabled"
	EventTypeDisabled = "skill.disabled"
)

type AddedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewAddedEvent(skill *Skill) *AddedEvent {
	return &AddedEvent{
		eventType:     EventTypeAdded,
		domain:        "skill",
		payload:       map[string]any{"skill_id": skill.ID, "name": skill.Name, "source": skill.Source},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *AddedEvent) Type() string          { return e.eventType }
func (e *AddedEvent) Domain() string        { return e.domain }
func (e *AddedEvent) Payload() any          { return e.payload }
func (e *AddedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *AddedEvent) CorrelationID() string { return e.correlationID }

type RemovedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRemovedEvent(skillID string) *RemovedEvent {
	return &RemovedEvent{
		eventType:     EventTypeRemoved,
		domain:        "skill",
		payload:       map[string]any{"skill_id": skillID},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RemovedEvent) Type() string          { return e.eventType }
func (e *RemovedEvent) Domain() string        { return e.domain }
func (e *RemovedEvent) Payload() any          { return e.payload }
func (e *RemovedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RemovedEvent) CorrelationID() string { return e.correlationID }

type EnabledEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewEnabledEvent(skillID string) *EnabledEvent {
	return &EnabledEvent{
		eventType:     EventTypeEnabled,
		domain:        "skill",
		payload:       map[string]any{"skill_id": skillID},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *EnabledEvent) Type() string          { return e.eventType }
func (e *EnabledEvent) Domain() string        { return e.domain }
func (e *EnabledEvent) Payload() any          { return e.payload }
func (e *EnabledEvent) Timestamp() time.Time  { return e.timestamp }
func (e *EnabledEvent) CorrelationID() string { return e.correlationID }

type DisabledEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewDisabledEvent(skillID string) *DisabledEvent {
	return &DisabledEvent{
		eventType:     EventTypeDisabled,
		domain:        "skill",
		payload:       map[string]any{"skill_id": skillID},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *DisabledEvent) Type() string          { return e.eventType }
func (e *DisabledEvent) Domain() string        { return e.domain }
func (e *DisabledEvent) Payload() any          { return e.payload }
func (e *DisabledEvent) Timestamp() time.Time  { return e.timestamp }
func (e *DisabledEvent) CorrelationID() string { return e.correlationID }
