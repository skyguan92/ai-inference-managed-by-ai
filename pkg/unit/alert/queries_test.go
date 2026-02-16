package alert

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestListRulesQuery_Name(t *testing.T) {
	q := NewListRulesQuery(nil)
	if q.Name() != "alert.list_rules" {
		t.Errorf("expected name 'alert.list_rules', got '%s'", q.Name())
	}
}

func TestListRulesQuery_Domain(t *testing.T) {
	q := NewListRulesQuery(nil)
	if q.Domain() != "alert" {
		t.Errorf("expected domain 'alert', got '%s'", q.Domain())
	}
}

func TestListRulesQuery_Execute(t *testing.T) {
	store := NewMemoryStore()

	store.CreateRule(context.Background(), &AlertRule{
		ID: "rule-1", Name: "Rule 1", Condition: "cpu > 80", Severity: AlertSeverityWarning, Enabled: true,
	})
	store.CreateRule(context.Background(), &AlertRule{
		ID: "rule-2", Name: "Rule 2", Condition: "mem > 90", Severity: AlertSeverityCritical, Enabled: false,
	})

	q := NewListRulesQuery(store)

	t.Run("list all rules", func(t *testing.T) {
		result, err := q.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		resultMap := result.(map[string]any)
		rules := resultMap["rules"].([]map[string]any)
		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}
	})

	t.Run("list enabled only", func(t *testing.T) {
		result, err := q.Execute(context.Background(), map[string]any{"enabled_only": true})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		resultMap := result.(map[string]any)
		rules := resultMap["rules"].([]map[string]any)
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})
}

func TestHistoryQuery_Execute(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()

	store.CreateAlert(context.Background(), &Alert{
		ID: "alert-1", RuleID: "rule-1", RuleName: "R1", Severity: AlertSeverityWarning,
		Status: AlertStatusResolved, Message: "M1", TriggeredAt: now,
	})
	store.CreateAlert(context.Background(), &Alert{
		ID: "alert-2", RuleID: "rule-1", RuleName: "R1", Severity: AlertSeverityCritical,
		Status: AlertStatusFiring, Message: "M2", TriggeredAt: now,
	})
	store.CreateAlert(context.Background(), &Alert{
		ID: "alert-3", RuleID: "rule-2", RuleName: "R2", Severity: AlertSeverityWarning,
		Status: AlertStatusFiring, Message: "M3", TriggeredAt: now,
	})

	q := NewHistoryQuery(store)

	t.Run("list all alerts", func(t *testing.T) {
		result, err := q.Execute(context.Background(), map[string]any{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		resultMap := result.(map[string]any)
		alerts := resultMap["alerts"].([]map[string]any)
		if len(alerts) != 3 {
			t.Errorf("expected 3 alerts, got %d", len(alerts))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		result, err := q.Execute(context.Background(), map[string]any{"status": "firing"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		resultMap := result.(map[string]any)
		alerts := resultMap["alerts"].([]map[string]any)
		if len(alerts) != 2 {
			t.Errorf("expected 2 alerts, got %d", len(alerts))
		}
	})

	t.Run("filter by rule_id", func(t *testing.T) {
		result, err := q.Execute(context.Background(), map[string]any{"rule_id": "rule-1"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		resultMap := result.(map[string]any)
		alerts := resultMap["alerts"].([]map[string]any)
		if len(alerts) != 2 {
			t.Errorf("expected 2 alerts, got %d", len(alerts))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		result, err := q.Execute(context.Background(), map[string]any{"limit": 2})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		resultMap := result.(map[string]any)
		alerts := resultMap["alerts"].([]map[string]any)
		if len(alerts) > 2 {
			t.Errorf("expected at most 2 alerts, got %d", len(alerts))
		}
	})
}

func TestActiveQuery_Execute(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()

	store.CreateAlert(context.Background(), &Alert{
		ID: "alert-1", RuleID: "rule-1", RuleName: "R1", Severity: AlertSeverityWarning,
		Status: AlertStatusFiring, Message: "M1", TriggeredAt: now,
	})
	store.CreateAlert(context.Background(), &Alert{
		ID: "alert-2", RuleID: "rule-1", RuleName: "R1", Severity: AlertSeverityCritical,
		Status: AlertStatusAcknowledged, Message: "M2", TriggeredAt: now,
	})
	store.CreateAlert(context.Background(), &Alert{
		ID: "alert-3", RuleID: "rule-2", RuleName: "R2", Severity: AlertSeverityWarning,
		Status: AlertStatusResolved, Message: "M3", TriggeredAt: now,
	})

	q := NewActiveQuery(store)

	result, err := q.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	alerts := resultMap["alerts"].([]map[string]any)
	if len(alerts) != 2 {
		t.Errorf("expected 2 active alerts, got %d", len(alerts))
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewListRulesQuery(nil)
	var _ unit.Query = NewHistoryQuery(nil)
	var _ unit.Query = NewActiveQuery(nil)
}

func TestQueryDescriptionAndExamples(t *testing.T) {
	queries := []unit.Query{
		NewListRulesQuery(nil),
		NewHistoryQuery(nil),
		NewActiveQuery(nil),
	}

	for _, q := range queries {
		if q.Description() == "" {
			t.Errorf("%s: expected non-empty description", q.Name())
		}
		if len(q.Examples()) == 0 {
			t.Errorf("%s: expected at least one example", q.Name())
		}
	}
}
