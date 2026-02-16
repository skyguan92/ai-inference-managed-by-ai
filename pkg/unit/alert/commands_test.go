package alert

import (
	"context"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestCreateRuleCommand_Name(t *testing.T) {
	cmd := NewCreateRuleCommand(nil)
	if cmd.Name() != "alert.create_rule" {
		t.Errorf("expected name 'alert.create_rule', got '%s'", cmd.Name())
	}
}

func TestCreateRuleCommand_Domain(t *testing.T) {
	cmd := NewCreateRuleCommand(nil)
	if cmd.Domain() != "alert" {
		t.Errorf("expected domain 'alert', got '%s'", cmd.Domain())
	}
}

func TestCreateRuleCommand_Schemas(t *testing.T) {
	cmd := NewCreateRuleCommand(nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 3 {
		t.Errorf("expected 3 required fields, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestCreateRuleCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewCreateRuleCommand(store)

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "successful creation",
			input: map[string]any{
				"name":      "High CPU",
				"condition": "cpu.utilization > 80",
				"severity":  "warning",
				"channels":  []string{"email", "slack"},
				"cooldown":  300,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			input: map[string]any{
				"condition": "cpu.utilization > 80",
				"severity":  "warning",
			},
			wantErr: true,
		},
		{
			name: "missing condition",
			input: map[string]any{
				"name":     "High CPU",
				"severity": "warning",
			},
			wantErr: true,
		},
		{
			name: "invalid severity",
			input: map[string]any{
				"name":      "High CPU",
				"condition": "cpu.utilization > 80",
				"severity":  "invalid",
			},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cmd.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, ok := resultMap["rule_id"].(string); !ok {
				t.Error("expected 'rule_id' to be string")
			}
		})
	}
}

func TestUpdateRuleCommand_Execute(t *testing.T) {
	store := NewMemoryStore()

	rule := &AlertRule{
		ID:        "rule-1",
		Name:      "Test Rule",
		Condition: "cpu > 50",
		Severity:  AlertSeverityWarning,
		Enabled:   true,
	}
	store.CreateRule(context.Background(), rule)

	cmd := NewUpdateRuleCommand(store)

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "successful update",
			input: map[string]any{
				"rule_id":   "rule-1",
				"condition": "cpu > 90",
			},
			wantErr: false,
		},
		{
			name: "rule not found",
			input: map[string]any{
				"rule_id":   "nonexistent",
				"condition": "cpu > 90",
			},
			wantErr: true,
		},
		{
			name: "missing rule_id",
			input: map[string]any{
				"condition": "cpu > 90",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cmd.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if success, _ := resultMap["success"].(bool); !success {
				t.Error("expected success to be true")
			}
		})
	}
}

func TestDeleteRuleCommand_Execute(t *testing.T) {
	store := NewMemoryStore()

	rule := &AlertRule{
		ID:        "rule-1",
		Name:      "Test Rule",
		Condition: "cpu > 50",
		Severity:  AlertSeverityWarning,
		Enabled:   true,
	}
	store.CreateRule(context.Background(), rule)

	cmd := NewDeleteRuleCommand(store)

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "successful delete",
			input:   map[string]any{"rule_id": "rule-1"},
			wantErr: false,
		},
		{
			name:    "rule not found",
			input:   map[string]any{"rule_id": "nonexistent"},
			wantErr: true,
		},
		{
			name:    "missing rule_id",
			input:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cmd.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if success, _ := resultMap["success"].(bool); !success {
				t.Error("expected success to be true")
			}
		})
	}
}

func TestAcknowledgeCommand_Execute(t *testing.T) {
	store := NewMemoryStore()

	alert := &Alert{
		ID:          "alert-1",
		RuleID:      "rule-1",
		RuleName:    "Test",
		Severity:    AlertSeverityWarning,
		Status:      AlertStatusFiring,
		Message:     "Test alert",
		TriggeredAt: currentTime(),
	}
	store.CreateAlert(context.Background(), alert)

	cmd := NewAcknowledgeCommand(store)

	result, err := cmd.Execute(context.Background(), map[string]any{"alert_id": "alert-1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Error("expected result to be map[string]any")
		return
	}

	if success, _ := resultMap["success"].(bool); !success {
		t.Error("expected success to be true")
	}

	updated, _ := store.GetAlert(context.Background(), "alert-1")
	if updated.Status != AlertStatusAcknowledged {
		t.Errorf("expected status to be acknowledged, got %s", updated.Status)
	}
	if updated.AcknowledgedAt == nil {
		t.Error("expected acknowledged_at to be set")
	}
}

func TestResolveCommand_Execute(t *testing.T) {
	store := NewMemoryStore()

	alert := &Alert{
		ID:          "alert-1",
		RuleID:      "rule-1",
		RuleName:    "Test",
		Severity:    AlertSeverityWarning,
		Status:      AlertStatusFiring,
		Message:     "Test alert",
		TriggeredAt: currentTime(),
	}
	store.CreateAlert(context.Background(), alert)

	cmd := NewResolveCommand(store)

	result, err := cmd.Execute(context.Background(), map[string]any{"alert_id": "alert-1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Error("expected result to be map[string]any")
		return
	}

	if success, _ := resultMap["success"].(bool); !success {
		t.Error("expected success to be true")
	}

	updated, _ := store.GetAlert(context.Background(), "alert-1")
	if updated.Status != AlertStatusResolved {
		t.Errorf("expected status to be resolved, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Error("expected resolved_at to be set")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewCreateRuleCommand(nil)
	var _ unit.Command = NewUpdateRuleCommand(nil)
	var _ unit.Command = NewDeleteRuleCommand(nil)
	var _ unit.Command = NewAcknowledgeCommand(nil)
	var _ unit.Command = NewResolveCommand(nil)
}

func TestCommandDescriptionAndExamples(t *testing.T) {
	commands := []unit.Command{
		NewCreateRuleCommand(nil),
		NewUpdateRuleCommand(nil),
		NewDeleteRuleCommand(nil),
		NewAcknowledgeCommand(nil),
		NewResolveCommand(nil),
	}

	for _, cmd := range commands {
		if cmd.Description() == "" {
			t.Errorf("%s: expected non-empty description", cmd.Name())
		}
		if len(cmd.Examples()) == 0 {
			t.Errorf("%s: expected at least one example", cmd.Name())
		}
	}
}
