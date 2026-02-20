package alert

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestRulesResource_URI(t *testing.T) {
	r := NewRulesResource(nil)
	if r.URI() != "asms://alerts/rules" {
		t.Errorf("expected URI 'asms://alerts/rules', got '%s'", r.URI())
	}
}

func TestRulesResource_Domain(t *testing.T) {
	r := NewRulesResource(nil)
	if r.Domain() != "alert" {
		t.Errorf("expected domain 'alert', got '%s'", r.Domain())
	}
}

func TestRulesResource_Get(t *testing.T) {
	store := NewMemoryStore()

	_ = store.CreateRule(context.Background(), &AlertRule{
		ID: "rule-1", Name: "Rule 1", Condition: "cpu > 80", Severity: AlertSeverityWarning, Enabled: true,
	})
	_ = store.CreateRule(context.Background(), &AlertRule{
		ID: "rule-2", Name: "Rule 2", Condition: "mem > 90", Severity: AlertSeverityCritical, Enabled: false,
	})

	r := NewRulesResource(store)
	result, err := r.Get(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	rules := resultMap["rules"].([]map[string]any)
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
}

func TestRulesResource_Watch(t *testing.T) {
	store := NewMemoryStore()
	r := NewRulesResource(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	select {
	case update := <-ch:
		if update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-time.After(35 * time.Second):
		t.Error("expected update within 35 seconds")
	}
}

func TestActiveResource_URI(t *testing.T) {
	r := NewActiveResource(nil)
	if r.URI() != "asms://alerts/active" {
		t.Errorf("expected URI 'asms://alerts/active', got '%s'", r.URI())
	}
}

func TestActiveResource_Domain(t *testing.T) {
	r := NewActiveResource(nil)
	if r.Domain() != "alert" {
		t.Errorf("expected domain 'alert', got '%s'", r.Domain())
	}
}

func TestActiveResource_Get(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()

	_ = store.CreateAlert(context.Background(), &Alert{
		ID: "alert-1", RuleID: "rule-1", RuleName: "R1", Severity: AlertSeverityWarning,
		Status: AlertStatusFiring, Message: "M1", TriggeredAt: now,
	})
	_ = store.CreateAlert(context.Background(), &Alert{
		ID: "alert-2", RuleID: "rule-1", RuleName: "R1", Severity: AlertSeverityCritical,
		Status: AlertStatusResolved, Message: "M2", TriggeredAt: now,
	})

	r := NewActiveResource(store)
	result, err := r.Get(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	alerts := resultMap["alerts"].([]map[string]any)
	if len(alerts) != 1 {
		t.Errorf("expected 1 active alert, got %d", len(alerts))
	}
}

func TestActiveResource_Watch(t *testing.T) {
	store := NewMemoryStore()
	r := NewActiveResource(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	select {
	case update := <-ch:
		if update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-time.After(10 * time.Second):
		t.Error("expected update within 10 seconds")
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewRulesResource(nil)
	var _ unit.Resource = NewActiveResource(nil)
}
