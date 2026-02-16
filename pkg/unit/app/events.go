package app

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeInstalled   = "app.installed"
	EventTypeStarted     = "app.started"
	EventTypeStopped     = "app.stopped"
	EventTypeOOMDetected = "app.oom_detected"
)

type InstalledEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewInstalledEvent(app *App) *InstalledEvent {
	return &InstalledEvent{
		eventType: EventTypeInstalled,
		domain:    "app",
		payload: map[string]any{
			"app_id":       app.ID,
			"name":         app.Name,
			"template":     app.Template,
			"status":       app.Status,
			"installed_at": time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *InstalledEvent) Type() string          { return e.eventType }
func (e *InstalledEvent) Domain() string        { return e.domain }
func (e *InstalledEvent) Payload() any          { return e.payload }
func (e *InstalledEvent) Timestamp() time.Time  { return e.timestamp }
func (e *InstalledEvent) CorrelationID() string { return e.correlationID }

type StartedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStartedEvent(app *App) *StartedEvent {
	return &StartedEvent{
		eventType: EventTypeStarted,
		domain:    "app",
		payload: map[string]any{
			"app_id":     app.ID,
			"name":       app.Name,
			"template":   app.Template,
			"status":     app.Status,
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

func NewStoppedEvent(app *App, reason string) *StoppedEvent {
	return &StoppedEvent{
		eventType: EventTypeStopped,
		domain:    "app",
		payload: map[string]any{
			"app_id":     app.ID,
			"name":       app.Name,
			"template":   app.Template,
			"status":     app.Status,
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

type OOMDetectedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewOOMDetectedEvent(app *App, memoryMB float64) *OOMDetectedEvent {
	return &OOMDetectedEvent{
		eventType: EventTypeOOMDetected,
		domain:    "app",
		payload: map[string]any{
			"app_id":      app.ID,
			"name":        app.Name,
			"template":    app.Template,
			"memory_mb":   memoryMB,
			"detected_at": time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *OOMDetectedEvent) Type() string          { return e.eventType }
func (e *OOMDetectedEvent) Domain() string        { return e.domain }
func (e *OOMDetectedEvent) Payload() any          { return e.payload }
func (e *OOMDetectedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *OOMDetectedEvent) CorrelationID() string { return e.correlationID }
