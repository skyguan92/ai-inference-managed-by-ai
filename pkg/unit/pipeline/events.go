package pipeline

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeStarted       = "pipeline.started"
	EventTypeStepCompleted = "pipeline.step_completed"
	EventTypeCompleted     = "pipeline.completed"
	EventTypeFailed        = "pipeline.failed"
)

type StartedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStartedEvent(pipeline *Pipeline, run *PipelineRun) *StartedEvent {
	return &StartedEvent{
		eventType: EventTypeStarted,
		domain:    "pipeline",
		payload: map[string]any{
			"pipeline_id":   pipeline.ID,
			"pipeline_name": pipeline.Name,
			"run_id":        run.ID,
			"status":        run.Status,
			"started_at":    run.StartedAt.Unix(),
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

type StepCompletedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewStepCompletedEvent(pipeline *Pipeline, run *PipelineRun, stepID string, result map[string]any) *StepCompletedEvent {
	return &StepCompletedEvent{
		eventType: EventTypeStepCompleted,
		domain:    "pipeline",
		payload: map[string]any{
			"pipeline_id": pipeline.ID,
			"run_id":      run.ID,
			"step_id":     stepID,
			"result":      result,
			"timestamp":   time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *StepCompletedEvent) Type() string          { return e.eventType }
func (e *StepCompletedEvent) Domain() string        { return e.domain }
func (e *StepCompletedEvent) Payload() any          { return e.payload }
func (e *StepCompletedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *StepCompletedEvent) CorrelationID() string { return e.correlationID }

type CompletedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewCompletedEvent(pipeline *Pipeline, run *PipelineRun) *CompletedEvent {
	var completedAt int64
	if run.CompletedAt != nil {
		completedAt = run.CompletedAt.Unix()
	}
	return &CompletedEvent{
		eventType: EventTypeCompleted,
		domain:    "pipeline",
		payload: map[string]any{
			"pipeline_id":  pipeline.ID,
			"run_id":       run.ID,
			"status":       run.Status,
			"step_results": run.StepResults,
			"completed_at": completedAt,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *CompletedEvent) Type() string          { return e.eventType }
func (e *CompletedEvent) Domain() string        { return e.domain }
func (e *CompletedEvent) Payload() any          { return e.payload }
func (e *CompletedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *CompletedEvent) CorrelationID() string { return e.correlationID }

type FailedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewFailedEvent(pipeline *Pipeline, run *PipelineRun, stepID string, errMsg string) *FailedEvent {
	return &FailedEvent{
		eventType: EventTypeFailed,
		domain:    "pipeline",
		payload: map[string]any{
			"pipeline_id": pipeline.ID,
			"run_id":      run.ID,
			"step_id":     stepID,
			"error":       errMsg,
			"timestamp":   time.Now().Unix(),
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
