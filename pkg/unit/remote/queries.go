package remote

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type StatusQuery struct {
	store  RemoteStore
	events unit.EventPublisher
}

func NewStatusQuery(store RemoteStore) *StatusQuery {
	return &StatusQuery{store: store}
}

func NewStatusQueryWithEvents(store RemoteStore, events unit.EventPublisher) *StatusQuery {
	return &StatusQuery{store: store, events: events}
}

func (q *StatusQuery) Name() string {
	return "remote.status"
}

func (q *StatusQuery) Domain() string {
	return "remote"
}

func (q *StatusQuery) Description() string {
	return "Get remote access tunnel status"
}

func (q *StatusQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type:       "object",
		Properties: map[string]unit.Field{},
	}
}

func (q *StatusQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"enabled":        {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
			"provider":       {Name: "provider", Schema: unit.Schema{Type: "string"}},
			"public_url":     {Name: "public_url", Schema: unit.Schema{Type: "string"}},
			"uptime_seconds": {Name: "uptime_seconds", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (q *StatusQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"enabled": true, "provider": "cloudflare", "public_url": "https://test.tunnel.example.com", "uptime_seconds": 3600},
			Description: "Get tunnel status when connected",
		},
		{
			Input:       map[string]any{},
			Output:      map[string]any{"enabled": false, "uptime_seconds": 0},
			Description: "Get tunnel status when disconnected",
		},
	}
}

func (q *StatusQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	tunnel, err := q.store.GetTunnel(ctx)
	if err != nil {
		output := map[string]any{
			"enabled":        false,
			"provider":       "",
			"public_url":     "",
			"uptime_seconds": 0,
		}
		ec.PublishCompleted(output)
		return output, nil
	}

	uptime := int64(0)
	if !tunnel.StartedAt.IsZero() {
		uptime = int64(time.Since(tunnel.StartedAt).Seconds())
	}

	output := map[string]any{
		"enabled":        tunnel.Status == TunnelStatusConnected,
		"provider":       string(tunnel.Provider),
		"public_url":     tunnel.PublicURL,
		"uptime_seconds": uptime,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type AuditQuery struct {
	store  RemoteStore
	events unit.EventPublisher
}

func NewAuditQuery(store RemoteStore) *AuditQuery {
	return &AuditQuery{store: store}
}

func NewAuditQueryWithEvents(store RemoteStore, events unit.EventPublisher) *AuditQuery {
	return &AuditQuery{store: store, events: events}
}

func (q *AuditQuery) Name() string {
	return "remote.audit"
}

func (q *AuditQuery) Domain() string {
	return "remote"
}

func (q *AuditQuery) Description() string {
	return "Get remote command audit log"
}

func (q *AuditQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"since": {
				Name: "since",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter records since this time (RFC3339)",
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of records to return",
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(1000),
					Default:     100,
				},
			},
		},
	}
}

func (q *AuditQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"records": {
				Name: "records",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
							"command":     {Name: "command", Schema: unit.Schema{Type: "string"}},
							"exit_code":   {Name: "exit_code", Schema: unit.Schema{Type: "number"}},
							"timestamp":   {Name: "timestamp", Schema: unit.Schema{Type: "string"}},
							"duration_ms": {Name: "duration_ms", Schema: unit.Schema{Type: "number"}},
						},
					},
				},
			},
		},
	}
}

func (q *AuditQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"records": []map[string]any{{"id": "audit-123", "command": "ls -la", "exit_code": 0, "timestamp": "2024-01-01T00:00:00Z", "duration_ms": 50}}},
			Description: "Get all audit records",
		},
		{
			Input:       map[string]any{"limit": 10},
			Output:      map[string]any{"records": []map[string]any{}},
			Description: "Get last 10 audit records",
		},
	}
}

func (q *AuditQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, _ := input.(map[string]any)

	filter := AuditFilter{
		Limit: 100,
	}

	if sinceStr, ok := inputMap["since"].(string); ok && sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			filter.Since = since
		}
	}

	if limit, ok := toInt(inputMap["limit"]); ok && limit > 0 {
		filter.Limit = limit
	}

	records, err := q.store.ListAuditRecords(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list audit records: %w", err)
	}

	result := make([]map[string]any, len(records))
	for i, r := range records {
		result[i] = map[string]any{
			"id":          r.ID,
			"command":     r.Command,
			"exit_code":   r.ExitCode,
			"timestamp":   r.Timestamp.Format(time.RFC3339),
			"duration_ms": r.Duration,
		}
	}

	output := map[string]any{"records": result}
	ec.PublishCompleted(output)
	return output, nil
}
