package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeCreated      = "model.created"
	EventTypeDeleted      = "model.deleted"
	EventTypePullProgress = "model.pull_progress"
	EventTypeVerified     = "model.verified"
)

type CreatedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewCreatedEvent(model *Model) *CreatedEvent {
	return &CreatedEvent{
		eventType: EventTypeCreated,
		domain:    "model",
		payload: map[string]any{
			"model_id":   model.ID,
			"name":       model.Name,
			"type":       model.Type,
			"format":     model.Format,
			"source":     model.Source,
			"created_at": model.CreatedAt,
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

type DeletedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewDeletedEvent(modelID, name string) *DeletedEvent {
	return &DeletedEvent{
		eventType: EventTypeDeleted,
		domain:    "model",
		payload: map[string]any{
			"model_id": modelID,
			"name":     name,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *DeletedEvent) Type() string          { return e.eventType }
func (e *DeletedEvent) Domain() string        { return e.domain }
func (e *DeletedEvent) Payload() any          { return e.payload }
func (e *DeletedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *DeletedEvent) CorrelationID() string { return e.correlationID }

type PullProgressEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewPullProgressEvent(progress *PullProgress) *PullProgressEvent {
	return &PullProgressEvent{
		eventType: EventTypePullProgress,
		domain:    "model",
		payload: map[string]any{
			"model_id":    progress.ModelID,
			"status":      progress.Status,
			"progress":    progress.Progress,
			"bytes_total": progress.BytesTotal,
			"bytes_done":  progress.BytesDone,
			"speed":       progress.Speed,
			"error":       progress.Error,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *PullProgressEvent) Type() string          { return e.eventType }
func (e *PullProgressEvent) Domain() string        { return e.domain }
func (e *PullProgressEvent) Payload() any          { return e.payload }
func (e *PullProgressEvent) Timestamp() time.Time  { return e.timestamp }
func (e *PullProgressEvent) CorrelationID() string { return e.correlationID }

type VerifiedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewVerifiedEvent(modelID string, result *VerificationResult) *VerifiedEvent {
	return &VerifiedEvent{
		eventType: EventTypeVerified,
		domain:    "model",
		payload: map[string]any{
			"model_id": modelID,
			"valid":    result.Valid,
			"issues":   result.Issues,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *VerifiedEvent) Type() string          { return e.eventType }
func (e *VerifiedEvent) Domain() string        { return e.domain }
func (e *VerifiedEvent) Payload() any          { return e.payload }
func (e *VerifiedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *VerifiedEvent) CorrelationID() string { return e.correlationID }
