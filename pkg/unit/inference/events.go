package inference

import (
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

const (
	EventTypeRequestStarted   = "inference.request_started"
	EventTypeRequestCompleted = "inference.request_completed"
	EventTypeRequestFailed    = "inference.request_failed"
)

type RequestStartedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRequestStartedEvent(requestID, model, requestType string) *RequestStartedEvent {
	return &RequestStartedEvent{
		eventType: EventTypeRequestStarted,
		domain:    "inference",
		payload: map[string]any{
			"request_id": requestID,
			"model":      model,
			"type":       requestType,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RequestStartedEvent) Type() string          { return e.eventType }
func (e *RequestStartedEvent) Domain() string        { return e.domain }
func (e *RequestStartedEvent) Payload() any          { return e.payload }
func (e *RequestStartedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RequestStartedEvent) CorrelationID() string { return e.correlationID }

type RequestCompletedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRequestCompletedEvent(requestID string, duration time.Duration, tokens int) *RequestCompletedEvent {
	return &RequestCompletedEvent{
		eventType: EventTypeRequestCompleted,
		domain:    "inference",
		payload: map[string]any{
			"request_id":   requestID,
			"duration_ms":  duration.Milliseconds(),
			"total_tokens": tokens,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RequestCompletedEvent) Type() string          { return e.eventType }
func (e *RequestCompletedEvent) Domain() string        { return e.domain }
func (e *RequestCompletedEvent) Payload() any          { return e.payload }
func (e *RequestCompletedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RequestCompletedEvent) CorrelationID() string { return e.correlationID }

type RequestFailedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRequestFailedEvent(requestID string, errMsg string) *RequestFailedEvent {
	return &RequestFailedEvent{
		eventType: EventTypeRequestFailed,
		domain:    "inference",
		payload: map[string]any{
			"request_id": requestID,
			"error":      errMsg,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RequestFailedEvent) Type() string          { return e.eventType }
func (e *RequestFailedEvent) Domain() string        { return e.domain }
func (e *RequestFailedEvent) Payload() any          { return e.payload }
func (e *RequestFailedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RequestFailedEvent) CorrelationID() string { return e.correlationID }

var _ unit.Event = (*RequestStartedEvent)(nil)
var _ unit.Event = (*RequestCompletedEvent)(nil)
var _ unit.Event = (*RequestFailedEvent)(nil)
