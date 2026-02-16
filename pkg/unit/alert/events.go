package alert

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeTriggered    = "alert.triggered"
	EventTypeAcknowledged = "alert.acknowledged"
	EventTypeResolved     = "alert.resolved"
)

type TriggeredEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewTriggeredEvent(alert *Alert) *TriggeredEvent {
	return &TriggeredEvent{
		eventType:     EventTypeTriggered,
		domain:        "alert",
		payload:       alertToMap(alert),
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *TriggeredEvent) Type() string          { return e.eventType }
func (e *TriggeredEvent) Domain() string        { return e.domain }
func (e *TriggeredEvent) Payload() any          { return e.payload }
func (e *TriggeredEvent) Timestamp() time.Time  { return e.timestamp }
func (e *TriggeredEvent) CorrelationID() string { return e.correlationID }

type AcknowledgedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewAcknowledgedEvent(alert *Alert) *AcknowledgedEvent {
	return &AcknowledgedEvent{
		eventType:     EventTypeAcknowledged,
		domain:        "alert",
		payload:       alertToMap(alert),
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *AcknowledgedEvent) Type() string          { return e.eventType }
func (e *AcknowledgedEvent) Domain() string        { return e.domain }
func (e *AcknowledgedEvent) Payload() any          { return e.payload }
func (e *AcknowledgedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *AcknowledgedEvent) CorrelationID() string { return e.correlationID }

type ResolvedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewResolvedEvent(alert *Alert) *ResolvedEvent {
	return &ResolvedEvent{
		eventType:     EventTypeResolved,
		domain:        "alert",
		payload:       alertToMap(alert),
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ResolvedEvent) Type() string          { return e.eventType }
func (e *ResolvedEvent) Domain() string        { return e.domain }
func (e *ResolvedEvent) Payload() any          { return e.payload }
func (e *ResolvedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ResolvedEvent) CorrelationID() string { return e.correlationID }

func alertToMap(alert *Alert) map[string]any {
	return map[string]any{
		"id":           alert.ID,
		"rule_id":      alert.RuleID,
		"rule_name":    alert.RuleName,
		"severity":     alert.Severity,
		"status":       alert.Status,
		"message":      alert.Message,
		"triggered_at": alert.TriggeredAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
