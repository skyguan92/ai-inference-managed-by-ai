package alert

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type ListRulesQuery struct {
	store  Store
	events unit.EventPublisher
}

func NewListRulesQuery(store Store) *ListRulesQuery {
	return &ListRulesQuery{store: store}
}

func NewListRulesQueryWithEvents(store Store, events unit.EventPublisher) *ListRulesQuery {
	return &ListRulesQuery{store: store, events: events}
}

func (q *ListRulesQuery) Name() string {
	return "alert.list_rules"
}

func (q *ListRulesQuery) Domain() string {
	return "alert"
}

func (q *ListRulesQuery) Description() string {
	return "List all alert rules"
}

func (q *ListRulesQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"enabled_only": {
				Name: "enabled_only",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Only return enabled rules",
				},
			},
		},
	}
}

func (q *ListRulesQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"rules": {
				Name: "rules",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":        {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":      {Name: "name", Schema: unit.Schema{Type: "string"}},
							"condition": {Name: "condition", Schema: unit.Schema{Type: "string"}},
							"severity":  {Name: "severity", Schema: unit.Schema{Type: "string"}},
							"channels":  {Name: "channels", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
							"cooldown":  {Name: "cooldown", Schema: unit.Schema{Type: "number"}},
							"enabled":   {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
						},
					},
				},
			},
		},
	}
}

func (q *ListRulesQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"enabled_only": true},
			Output:      map[string]any{"rules": []map[string]any{{"id": "rule-1", "name": "High CPU", "condition": "cpu > 80", "severity": "warning", "enabled": true}}},
			Description: "List enabled alert rules",
		},
	}
}

func (q *ListRulesQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	inputMap, _ := input.(map[string]any)
	enabledOnly, _ := inputMap["enabled_only"].(bool)

	filter := RuleFilter{EnabledOnly: enabledOnly}

	rules, err := q.store.ListRules(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list rules: %w", err)
	}

	result := make([]map[string]any, len(rules))
	for i, r := range rules {
		result[i] = map[string]any{
			"id":        r.ID,
			"name":      r.Name,
			"condition": r.Condition,
			"severity":  r.Severity,
			"channels":  r.Channels,
			"cooldown":  r.Cooldown,
			"enabled":   r.Enabled,
		}
	}

	output := map[string]any{"rules": result}
	ec.PublishCompleted(output)
	return output, nil
}

type HistoryQuery struct {
	store  Store
	events unit.EventPublisher
}

func NewHistoryQuery(store Store) *HistoryQuery {
	return &HistoryQuery{store: store}
}

func NewHistoryQueryWithEvents(store Store, events unit.EventPublisher) *HistoryQuery {
	return &HistoryQuery{store: store, events: events}
}

func (q *HistoryQuery) Name() string {
	return "alert.history"
}

func (q *HistoryQuery) Domain() string {
	return "alert"
}

func (q *HistoryQuery) Description() string {
	return "Get alert history"
}

func (q *HistoryQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"rule_id": {
				Name: "rule_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by rule ID",
				},
			},
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by status",
					Enum:        []any{"firing", "acknowledged", "resolved"},
				},
			},
			"severity": {
				Name: "severity",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by severity",
					Enum:        []any{"info", "warning", "critical"},
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(1000),
				},
			},
		},
	}
}

func (q *HistoryQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"alerts": {
				Name: "alerts",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
							"rule_id":      {Name: "rule_id", Schema: unit.Schema{Type: "string"}},
							"rule_name":    {Name: "rule_name", Schema: unit.Schema{Type: "string"}},
							"severity":     {Name: "severity", Schema: unit.Schema{Type: "string"}},
							"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
							"message":      {Name: "message", Schema: unit.Schema{Type: "string"}},
							"triggered_at": {Name: "triggered_at", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (q *HistoryQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"limit": 10},
			Output:      map[string]any{"alerts": []map[string]any{{"id": "alert-1", "rule_name": "High CPU", "status": "resolved"}}},
			Description: "Get recent alert history",
		},
	}
}

func (q *HistoryQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	inputMap, _ := input.(map[string]any)

	filter := AlertFilter{
		RuleID:   getString(inputMap, "rule_id"),
		Status:   AlertStatus(getString(inputMap, "status")),
		Severity: AlertSeverity(getString(inputMap, "severity")),
		Limit:    getInt(inputMap, "limit"),
	}

	if filter.Limit == 0 {
		filter.Limit = 100
	}

	alerts, _, err := q.store.ListAlerts(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list alerts: %w", err)
	}

	result := make([]map[string]any, len(alerts))
	for i, a := range alerts {
		result[i] = map[string]any{
			"id":           a.ID,
			"rule_id":      a.RuleID,
			"rule_name":    a.RuleName,
			"severity":     a.Severity,
			"status":       a.Status,
			"message":      a.Message,
			"triggered_at": a.TriggeredAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	output := map[string]any{"alerts": result}
	ec.PublishCompleted(output)
	return output, nil
}

type ActiveQuery struct {
	store  Store
	events unit.EventPublisher
}

func NewActiveQuery(store Store) *ActiveQuery {
	return &ActiveQuery{store: store}
}

func NewActiveQueryWithEvents(store Store, events unit.EventPublisher) *ActiveQuery {
	return &ActiveQuery{store: store, events: events}
}

func (q *ActiveQuery) Name() string {
	return "alert.active"
}

func (q *ActiveQuery) Domain() string {
	return "alert"
}

func (q *ActiveQuery) Description() string {
	return "Get all active alerts"
}

func (q *ActiveQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type:       "object",
		Properties: map[string]unit.Field{},
	}
}

func (q *ActiveQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"alerts": {
				Name: "alerts",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
							"rule_id":      {Name: "rule_id", Schema: unit.Schema{Type: "string"}},
							"rule_name":    {Name: "rule_name", Schema: unit.Schema{Type: "string"}},
							"severity":     {Name: "severity", Schema: unit.Schema{Type: "string"}},
							"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
							"message":      {Name: "message", Schema: unit.Schema{Type: "string"}},
							"triggered_at": {Name: "triggered_at", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (q *ActiveQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"alerts": []map[string]any{{"id": "alert-1", "rule_name": "High CPU", "status": "firing"}}},
			Description: "Get all active alerts",
		},
	}
}

func (q *ActiveQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	alerts, err := q.store.ListActiveAlerts(ctx)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list active alerts: %w", err)
	}

	result := make([]map[string]any, len(alerts))
	for i, a := range alerts {
		result[i] = map[string]any{
			"id":           a.ID,
			"rule_id":      a.RuleID,
			"rule_name":    a.RuleName,
			"severity":     a.Severity,
			"status":       a.Status,
			"message":      a.Message,
			"triggered_at": a.TriggeredAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	output := map[string]any{"alerts": result}
	ec.PublishCompleted(output)
	return output, nil
}
