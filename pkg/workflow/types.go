package workflow

import (
	"time"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

type WorkflowDef struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Config      map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
	Steps       []WorkflowStep `yaml:"steps" json:"steps"`
	Output      map[string]any `yaml:"output,omitempty" json:"output,omitempty"`
}

type WorkflowStep struct {
	ID        string         `yaml:"id" json:"id"`
	Type      string         `yaml:"type" json:"type"`
	Input     map[string]any `yaml:"input" json:"input"`
	DependsOn []string       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	OnFailure string         `yaml:"on_failure,omitempty" json:"on_failure,omitempty"`
	Retry     *RetryConfig   `yaml:"retry,omitempty" json:"retry,omitempty"`
}

type RetryConfig struct {
	MaxAttempts  int `yaml:"max_attempts" json:"max_attempts"`
	DelaySeconds int `yaml:"delay_seconds" json:"delay_seconds"`
}

type ExecutionContext struct {
	Input  map[string]any
	Config map[string]any
	Steps  map[string]map[string]any
}

type StepResult struct {
	StepID    string
	Status    ExecutionStatus
	Output    map[string]any
	Error     string
	StartedAt time.Time
	EndedAt   *time.Time
	Duration  time.Duration
}

type ExecutionResult struct {
	WorkflowID  string
	RunID       string
	Status      ExecutionStatus
	StepResults map[string]StepResult
	Output      map[string]any
	Error       string
	Duration    time.Duration
	StartedAt   time.Time
	CompletedAt *time.Time
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}
