package unit

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionEventType represents the type of execution event
type ExecutionEventType string

const (
	// ExecutionStarted is fired when a command/query execution starts
	ExecutionStarted ExecutionEventType = "execution_started"
	// ExecutionCompleted is fired when a command/query execution succeeds
	ExecutionCompleted ExecutionEventType = "execution_completed"
	// ExecutionFailed is fired when a command/query execution fails
	ExecutionFailed ExecutionEventType = "execution_failed"
)

// ExecutionEvent represents an event during command/query execution
type ExecutionEvent struct {
	EventType         string    `json:"event_type"`
	EventDomain       string    `json:"domain"`
	UnitName          string    `json:"unit_name"`
	Input             any       `json:"input,omitempty"`
	Output            any       `json:"output,omitempty"`
	Error             string    `json:"error,omitempty"`
	EventTimestamp    time.Time `json:"timestamp"`
	EventCorrelationID string    `json:"correlation_id"`
	DurationMs        int64     `json:"duration_ms,omitempty"`
}

// Type returns the event type
func (e *ExecutionEvent) Type() string { return e.EventType }

// Domain returns the domain
func (e *ExecutionEvent) Domain() string { return e.EventDomain }

// Payload returns the event payload
func (e *ExecutionEvent) Payload() any { return e }

// Timestamp returns the timestamp
func (e *ExecutionEvent) Timestamp() time.Time { return e.EventTimestamp }

// CorrelationID returns the correlation ID
func (e *ExecutionEvent) CorrelationID() string { return e.EventCorrelationID }

// EventPublisher interface for publishing events
type EventPublisher interface {
	Publish(event any) error
}

// ExecutionContext holds execution context for event publishing
type ExecutionContext struct {
	Publisher     EventPublisher
	Domain        string
	UnitName      string
	CorrelationID string
	StartTime     time.Time
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(publisher EventPublisher, domain, unitName string) *ExecutionContext {
	return &ExecutionContext{
		Publisher:     publisher,
		Domain:        domain,
		UnitName:      unitName,
		CorrelationID: uuid.New().String(),
		StartTime:     time.Now(),
	}
}

// PublishStarted publishes an execution started event
func (ec *ExecutionContext) PublishStarted(input any) {
	if ec.Publisher == nil {
		return
	}

	event := &ExecutionEvent{
		EventType:          string(ExecutionStarted),
		EventDomain:        ec.Domain,
		UnitName:           ec.UnitName,
		Input:              input,
		EventTimestamp:     time.Now(),
		EventCorrelationID: ec.CorrelationID,
	}

	_ = ec.Publisher.Publish(event)
}

// PublishCompleted publishes an execution completed event
func (ec *ExecutionContext) PublishCompleted(output any) {
	if ec.Publisher == nil {
		return
	}

	duration := time.Since(ec.StartTime).Milliseconds()

	event := &ExecutionEvent{
		EventType:          string(ExecutionCompleted),
		EventDomain:        ec.Domain,
		UnitName:           ec.UnitName,
		Output:             output,
		EventTimestamp:     time.Now(),
		EventCorrelationID: ec.CorrelationID,
		DurationMs:         duration,
	}

	_ = ec.Publisher.Publish(event)
}

// PublishFailed publishes an execution failed event
func (ec *ExecutionContext) PublishFailed(err error) {
	if ec.Publisher == nil {
		return
	}

	duration := time.Since(ec.StartTime).Milliseconds()
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	event := &ExecutionEvent{
		EventType:          string(ExecutionFailed),
		EventDomain:        ec.Domain,
		UnitName:           ec.UnitName,
		Error:              errMsg,
		EventTimestamp:     time.Now(),
		EventCorrelationID: ec.CorrelationID,
		DurationMs:         duration,
	}

	_ = ec.Publisher.Publish(event)
}

// NoopEventPublisher is an event publisher that does nothing
type NoopEventPublisher struct{}

// Publish implements EventPublisher but does nothing
func (n *NoopEventPublisher) Publish(event any) error { return nil }

// Ensure NoopEventPublisher implements EventPublisher
var _ EventPublisher = (*NoopEventPublisher)(nil)
