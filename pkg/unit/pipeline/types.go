package pipeline

import "time"

type PipelineStatus string

const (
	PipelineStatusIdle    PipelineStatus = "idle"
	PipelineStatusRunning PipelineStatus = "running"
	PipelineStatusPaused  PipelineStatus = "paused"
	PipelineStatusError   PipelineStatus = "error"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

type PipelineStep struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Input     map[string]any `json:"input"`
	DependsOn []string       `json:"depends_on,omitempty"`
}

type Pipeline struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Steps     []PipelineStep `json:"steps"`
	Status    PipelineStatus `json:"status"`
	Config    map[string]any `json:"config,omitempty"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

type PipelineRun struct {
	ID          string         `json:"id"`
	PipelineID  string         `json:"pipeline_id"`
	Status      RunStatus      `json:"status"`
	Input       map[string]any `json:"input,omitempty"`
	StepResults map[string]any `json:"step_results"`
	Error       string         `json:"error,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

type PipelineFilter struct {
	Status PipelineStatus `json:"status,omitempty"`
	Limit  int            `json:"limit,omitempty"`
	Offset int            `json:"offset,omitempty"`
}

type CreateResult struct {
	PipelineID string `json:"pipeline_id"`
}

type DeleteResult struct {
	Success bool `json:"success"`
}

type RunResult struct {
	RunID  string    `json:"run_id"`
	Status RunStatus `json:"status"`
}

type CancelResult struct {
	Success bool `json:"success"`
}

type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Issues []string `json:"issues"`
}

type StepResult struct {
	StepID    string         `json:"step_id"`
	Status    RunStatus      `json:"status"`
	Output    map[string]any `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"`
}
