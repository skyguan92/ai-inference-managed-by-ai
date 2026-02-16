package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type CreateRuleCommand struct {
	store Store
}

func NewCreateRuleCommand(store Store) *CreateRuleCommand {
	return &CreateRuleCommand{store: store}
}

func (c *CreateRuleCommand) Name() string {
	return "alert.create_rule"
}

func (c *CreateRuleCommand) Domain() string {
	return "alert"
}

func (c *CreateRuleCommand) Description() string {
	return "Create a new alert rule"
}

func (c *CreateRuleCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Rule name",
				},
			},
			"condition": {
				Name: "condition",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Alert condition expression",
				},
			},
			"severity": {
				Name: "severity",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Alert severity level",
					Enum:        []any{"info", "warning", "critical"},
				},
			},
			"channels": {
				Name: "channels",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Notification channels",
					Items:       &unit.Schema{Type: "string"},
				},
			},
			"cooldown": {
				Name: "cooldown",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Cooldown period in seconds",
					Min:         ptrFloat(0),
				},
			},
		},
		Required: []string{"name", "condition", "severity"},
	}
}

func (c *CreateRuleCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"rule_id": {
				Name:   "rule_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *CreateRuleCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"name":      "High CPU Usage",
				"condition": "cpu.utilization > 80",
				"severity":  "warning",
				"channels":  []string{"email", "slack"},
				"cooldown":  300,
			},
			Output:      map[string]any{"rule_id": "rule-123"},
			Description: "Create a CPU alert rule",
		},
	}
}

func (c *CreateRuleCommand) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	condition, _ := inputMap["condition"].(string)
	if condition == "" {
		return nil, fmt.Errorf("condition is required")
	}

	severity := AlertSeverity(getString(inputMap, "severity"))
	if !isValidSeverity(severity) {
		return nil, ErrInvalidSeverity
	}

	var channels []string
	if ch, ok := inputMap["channels"].([]any); ok {
		for _, c := range ch {
			if s, ok := c.(string); ok {
				channels = append(channels, s)
			}
		}
	} else if ch, ok := inputMap["channels"].([]string); ok {
		channels = ch
	}

	cooldown := getInt(inputMap, "cooldown")

	rule := &AlertRule{
		ID:        uuid.New().String(),
		Name:      name,
		Condition: condition,
		Severity:  severity,
		Channels:  channels,
		Cooldown:  cooldown,
		Enabled:   true,
	}

	if err := c.store.CreateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}

	return map[string]any{"rule_id": rule.ID}, nil
}

type UpdateRuleCommand struct {
	store Store
}

func NewUpdateRuleCommand(store Store) *UpdateRuleCommand {
	return &UpdateRuleCommand{store: store}
}

func (c *UpdateRuleCommand) Name() string {
	return "alert.update_rule"
}

func (c *UpdateRuleCommand) Domain() string {
	return "alert"
}

func (c *UpdateRuleCommand) Description() string {
	return "Update an existing alert rule"
}

func (c *UpdateRuleCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"rule_id": {
				Name: "rule_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Rule identifier",
				},
			},
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "New rule name",
				},
			},
			"condition": {
				Name: "condition",
				Schema: unit.Schema{
					Type:        "string",
					Description: "New condition expression",
				},
			},
			"enabled": {
				Name: "enabled",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Enable or disable the rule",
				},
			},
		},
		Required: []string{"rule_id"},
	}
}

func (c *UpdateRuleCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *UpdateRuleCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"rule_id":   "rule-123",
				"condition": "cpu.utilization > 90",
				"enabled":   true,
			},
			Output:      map[string]any{"success": true},
			Description: "Update alert rule condition",
		},
	}
}

func (c *UpdateRuleCommand) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	ruleID, _ := inputMap["rule_id"].(string)
	if ruleID == "" {
		return nil, ErrInvalidRuleID
	}

	rule, err := c.store.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}

	if name, ok := inputMap["name"].(string); ok && name != "" {
		rule.Name = name
	}
	if condition, ok := inputMap["condition"].(string); ok && condition != "" {
		rule.Condition = condition
	}
	if enabled, ok := inputMap["enabled"].(bool); ok {
		rule.Enabled = enabled
	}

	if err := c.store.UpdateRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}

	return map[string]any{"success": true}, nil
}

type DeleteRuleCommand struct {
	store Store
}

func NewDeleteRuleCommand(store Store) *DeleteRuleCommand {
	return &DeleteRuleCommand{store: store}
}

func (c *DeleteRuleCommand) Name() string {
	return "alert.delete_rule"
}

func (c *DeleteRuleCommand) Domain() string {
	return "alert"
}

func (c *DeleteRuleCommand) Description() string {
	return "Delete an alert rule"
}

func (c *DeleteRuleCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"rule_id": {
				Name: "rule_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Rule identifier",
				},
			},
		},
		Required: []string{"rule_id"},
	}
}

func (c *DeleteRuleCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *DeleteRuleCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"rule_id": "rule-123"},
			Output:      map[string]any{"success": true},
			Description: "Delete an alert rule",
		},
	}
}

func (c *DeleteRuleCommand) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	ruleID, _ := inputMap["rule_id"].(string)
	if ruleID == "" {
		return nil, ErrInvalidRuleID
	}

	if err := c.store.DeleteRule(ctx, ruleID); err != nil {
		return nil, fmt.Errorf("delete rule: %w", err)
	}

	return map[string]any{"success": true}, nil
}

type AcknowledgeCommand struct {
	store Store
}

func NewAcknowledgeCommand(store Store) *AcknowledgeCommand {
	return &AcknowledgeCommand{store: store}
}

func (c *AcknowledgeCommand) Name() string {
	return "alert.acknowledge"
}

func (c *AcknowledgeCommand) Domain() string {
	return "alert"
}

func (c *AcknowledgeCommand) Description() string {
	return "Acknowledge an alert"
}

func (c *AcknowledgeCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"alert_id": {
				Name: "alert_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Alert identifier",
				},
			},
		},
		Required: []string{"alert_id"},
	}
}

func (c *AcknowledgeCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *AcknowledgeCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"alert_id": "alert-456"},
			Output:      map[string]any{"success": true},
			Description: "Acknowledge an alert",
		},
	}
}

func (c *AcknowledgeCommand) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	alertID, _ := inputMap["alert_id"].(string)
	if alertID == "" {
		return nil, ErrInvalidAlertID
	}

	alert, err := c.store.GetAlert(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}

	now := currentTime()
	alert.Status = AlertStatusAcknowledged
	alert.AcknowledgedAt = &now

	if err := c.store.UpdateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("acknowledge alert: %w", err)
	}

	return map[string]any{"success": true}, nil
}

type ResolveCommand struct {
	store Store
}

func NewResolveCommand(store Store) *ResolveCommand {
	return &ResolveCommand{store: store}
}

func (c *ResolveCommand) Name() string {
	return "alert.resolve"
}

func (c *ResolveCommand) Domain() string {
	return "alert"
}

func (c *ResolveCommand) Description() string {
	return "Resolve an alert"
}

func (c *ResolveCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"alert_id": {
				Name: "alert_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Alert identifier",
				},
			},
		},
		Required: []string{"alert_id"},
	}
}

func (c *ResolveCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *ResolveCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"alert_id": "alert-456"},
			Output:      map[string]any{"success": true},
			Description: "Resolve an alert",
		},
	}
}

func (c *ResolveCommand) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	alertID, _ := inputMap["alert_id"].(string)
	if alertID == "" {
		return nil, ErrInvalidAlertID
	}

	alert, err := c.store.GetAlert(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}

	now := currentTime()
	alert.Status = AlertStatusResolved
	alert.ResolvedAt = &now

	if err := c.store.UpdateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("resolve alert: %w", err)
	}

	return map[string]any{"success": true}, nil
}

func ptrFloat(v float64) *float64 {
	return &v
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

var currentTime = func() time.Time {
	return time.Now()
}
