package remote

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeEnabled         = "remote.enabled"
	EventTypeDisabled        = "remote.disabled"
	EventTypeCommandExecuted = "remote.command_executed"
)

type EnabledEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewEnabledEvent(tunnel *TunnelInfo) *EnabledEvent {
	return &EnabledEvent{
		eventType: EventTypeEnabled,
		domain:    "remote",
		payload: map[string]any{
			"tunnel_id":  tunnel.ID,
			"provider":   tunnel.Provider,
			"public_url": tunnel.PublicURL,
			"started_at": tunnel.StartedAt.Unix(),
		},
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

func NewDisabledEvent(tunnelID string) *DisabledEvent {
	return &DisabledEvent{
		eventType: EventTypeDisabled,
		domain:    "remote",
		payload: map[string]any{
			"tunnel_id":   tunnelID,
			"disabled_at": time.Now().Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *DisabledEvent) Type() string          { return e.eventType }
func (e *DisabledEvent) Domain() string        { return e.domain }
func (e *DisabledEvent) Payload() any          { return e.payload }
func (e *DisabledEvent) Timestamp() time.Time  { return e.timestamp }
func (e *DisabledEvent) CorrelationID() string { return e.correlationID }

type CommandExecutedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewCommandExecutedEvent(record *AuditRecord) *CommandExecutedEvent {
	return &CommandExecutedEvent{
		eventType: EventTypeCommandExecuted,
		domain:    "remote",
		payload: map[string]any{
			"audit_id":    record.ID,
			"command":     record.Command,
			"exit_code":   record.ExitCode,
			"duration_ms": record.Duration,
			"timestamp":   record.Timestamp.Unix(),
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *CommandExecutedEvent) Type() string          { return e.eventType }
func (e *CommandExecutedEvent) Domain() string        { return e.domain }
func (e *CommandExecutedEvent) Payload() any          { return e.payload }
func (e *CommandExecutedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *CommandExecutedEvent) CorrelationID() string { return e.correlationID }
