package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
)

func TestAlertService_NewAlertService(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	tests := []struct {
		name     string
		registry *unit.Registry
		store    alert.Store
		bus      *eventbus.InMemoryEventBus
	}{
		{
			name:     "with all dependencies",
			registry: unit.NewRegistry(),
			store:    store,
			bus:      bus,
		},
		{
			name:     "with nil bus",
			registry: unit.NewRegistry(),
			store:    store,
			bus:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAlertService(tt.registry, tt.store, tt.bus)
			if svc == nil {
				t.Error("expected non-nil AlertService")
			}
		})
	}
}

func TestAlertService_CreateRuleWithTest_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "alert.create_rule",
		execute: func(ctx context.Context, input any) (any, error) {
			rule := &alert.AlertRule{
				ID:        "rule-123",
				Name:      "test-rule",
				Condition: "cpu > 80",
				Severity:  alert.AlertSeverityWarning,
				Enabled:   true,
			}
			_ = store.CreateRule(ctx, rule)
			return map[string]any{"rule_id": rule.ID}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	input := CreateRuleInput{
		Name:      "test-rule",
		Condition: "cpu > 80",
		Severity:  alert.AlertSeverityWarning,
		Channels:  []string{"email"},
		Cooldown:  300,
	}

	result, err := svc.CreateRuleWithTest(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RuleID == "" {
		t.Error("expected non-empty rule_id")
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestAlertService_CreateRuleWithTest_InvalidInput(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	tests := []struct {
		name  string
		input CreateRuleInput
	}{
		{
			name: "empty name",
			input: CreateRuleInput{
				Condition: "cpu > 80",
				Severity:  alert.AlertSeverityWarning,
			},
		},
		{
			name: "empty condition",
			input: CreateRuleInput{
				Name:     "test-rule",
				Severity: alert.AlertSeverityWarning,
			},
		},
		{
			name: "invalid severity",
			input: CreateRuleInput{
				Name:      "test-rule",
				Condition: "cpu > 80",
				Severity:  "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CreateRuleWithTest(context.Background(), tt.input)
			if err == nil {
				t.Error("expected error for invalid input")
			}
			if result != nil {
				t.Errorf("expected nil result, got %+v", result)
			}
		})
	}
}

func TestAlertService_CreateRuleWithTest_CommandNotFound(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	input := CreateRuleInput{
		Name:      "test-rule",
		Condition: "cpu > 80",
		Severity:  alert.AlertSeverityWarning,
	}

	result, err := svc.CreateRuleWithTest(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_UpdateRuleWithValidation_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateRule(context.Background(), &alert.AlertRule{
		ID:        "rule-123",
		Name:      "test-rule",
		Condition: "cpu > 80",
		Severity:  alert.AlertSeverityWarning,
		Enabled:   true,
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "alert.update_rule",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			rule, _ := store.GetRule(ctx, inputMap["rule_id"].(string))
			if name, ok := inputMap["name"].(string); ok {
				rule.Name = name
			}
			_ = store.UpdateRule(ctx, rule)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	enabled := true
	input := UpdateRuleInput{
		Name:    "updated-rule",
		Enabled: &enabled,
	}

	result, err := svc.UpdateRuleWithValidation(context.Background(), "rule-123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestAlertService_UpdateRuleWithValidation_EmptyRuleID(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	input := UpdateRuleInput{Name: "updated"}

	result, err := svc.UpdateRuleWithValidation(context.Background(), "", input)
	if err == nil {
		t.Fatal("expected error for empty rule_id")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_DeleteRuleWithCleanup_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateRule(context.Background(), &alert.AlertRule{
		ID:        "rule-123",
		Name:      "test-rule",
		Condition: "cpu > 80",
		Severity:  alert.AlertSeverityWarning,
		Enabled:   true,
	})
	_ = store.CreateAlert(context.Background(), &alert.Alert{
		ID:          "alert-1",
		RuleID:      "rule-123",
		Status:      alert.AlertStatusResolved,
		Severity:    alert.AlertSeverityWarning,
		TriggeredAt: time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "alert.delete_rule",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			_ = store.DeleteRule(ctx, inputMap["rule_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.DeleteRuleWithCleanup(context.Background(), "rule-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestAlertService_DeleteRuleWithCleanup_RuleNotFound(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "alert.delete_rule",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.DeleteRuleWithCleanup(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rule")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_AcknowledgeAlert_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateAlert(context.Background(), &alert.Alert{
		ID:          "alert-123",
		RuleID:      "rule-123",
		Status:      alert.AlertStatusFiring,
		Severity:    alert.AlertSeverityWarning,
		TriggeredAt: time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "alert.acknowledge",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			a, _ := store.GetAlert(ctx, inputMap["alert_id"].(string))
			a.Status = alert.AlertStatusAcknowledged
			now := time.Now()
			a.AcknowledgedAt = &now
			_ = store.UpdateAlert(ctx, a)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.AcknowledgeAlert(context.Background(), "alert-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Alert.Status != alert.AlertStatusAcknowledged {
		t.Errorf("expected status=acknowledged, got %s", result.Alert.Status)
	}
}

func TestAlertService_AcknowledgeAlert_EmptyAlertID(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	result, err := svc.AcknowledgeAlert(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty alert_id")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_ResolveAlert_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateAlert(context.Background(), &alert.Alert{
		ID:          "alert-123",
		RuleID:      "rule-123",
		Status:      alert.AlertStatusFiring,
		Severity:    alert.AlertSeverityWarning,
		TriggeredAt: time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "alert.resolve",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			a, _ := store.GetAlert(ctx, inputMap["alert_id"].(string))
			a.Status = alert.AlertStatusResolved
			now := time.Now()
			a.ResolvedAt = &now
			_ = store.UpdateAlert(ctx, a)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.ResolveAlert(context.Background(), "alert-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Alert.Status != alert.AlertStatusResolved {
		t.Errorf("expected status=resolved, got %s", result.Alert.Status)
	}
}

func TestAlertService_GetActiveAlerts_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "alert.active",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"alerts": []any{
					map[string]any{"id": "alert-1", "rule_id": "rule-1", "rule_name": "Rule 1", "severity": "warning", "status": "firing", "message": "CPU high"},
					map[string]any{"id": "alert-2", "rule_id": "rule-2", "rule_name": "Rule 2", "severity": "critical", "status": "acknowledged", "message": "Memory low"},
				},
			}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetActiveAlerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
	if len(result.Alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(result.Alerts))
	}
}

func TestAlertService_GetHistoryByRule_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "alert.history",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"alerts": []any{
					map[string]any{"id": "alert-1", "rule_id": "rule-123", "rule_name": "Rule 1", "severity": "warning", "status": "resolved", "message": "CPU high"},
				},
			}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetHistoryByRule(context.Background(), "rule-123", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestAlertService_GetHistoryByRule_EmptyRuleID(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetHistoryByRule(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected error for empty rule_id")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_GetHistoryBySeverity_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "alert.history",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"alerts": []any{
					map[string]any{"id": "alert-1", "rule_id": "rule-1", "rule_name": "Rule 1", "severity": "critical", "status": "resolved", "message": "Critical issue"},
				},
			}, nil
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetHistoryBySeverity(context.Background(), alert.AlertSeverityCritical, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestAlertService_GetHistoryBySeverity_InvalidSeverity(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetHistoryBySeverity(context.Background(), "invalid", 10)
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_GetRuleSummary_Success(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateRule(context.Background(), &alert.AlertRule{
		ID:        "rule-123",
		Name:      "test-rule",
		Condition: "cpu > 80",
		Severity:  alert.AlertSeverityWarning,
		Enabled:   true,
	})
	_ = store.CreateAlert(context.Background(), &alert.Alert{
		ID:          "alert-1",
		RuleID:      "rule-123",
		Status:      alert.AlertStatusFiring,
		Severity:    alert.AlertSeverityWarning,
		TriggeredAt: time.Now(),
	})
	_ = store.CreateAlert(context.Background(), &alert.Alert{
		ID:          "alert-2",
		RuleID:      "rule-123",
		Status:      alert.AlertStatusResolved,
		Severity:    alert.AlertSeverityWarning,
		TriggeredAt: time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetRuleSummary(context.Background(), "rule-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Rule == nil {
		t.Fatal("expected non-nil rule")
	}
	if result.ActiveCount != 1 {
		t.Errorf("expected active_count=1, got %d", result.ActiveCount)
	}
}

func TestAlertService_ListRules(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateRule(context.Background(), &alert.AlertRule{
		ID:      "rule-1",
		Name:    "Rule 1",
		Enabled: true,
	})
	_ = store.CreateRule(context.Background(), &alert.AlertRule{
		ID:      "rule-2",
		Name:    "Rule 2",
		Enabled: false,
	})

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	rules, err := svc.ListRules(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rules) != 1 {
		t.Errorf("expected 1 enabled rule, got %d", len(rules))
	}
}

func TestAlertService_AcknowledgeAlert_CommandNotFound(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.CreateAlert(context.Background(), &alert.Alert{
		ID:          "alert-123",
		RuleID:      "rule-123",
		Status:      alert.AlertStatusFiring,
		TriggeredAt: time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewAlertService(registry, store, bus)

	result, err := svc.AcknowledgeAlert(context.Background(), "alert-123")
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAlertService_GetActiveAlerts_CommandFails(t *testing.T) {
	store := alert.NewMemoryStore()
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "alert.active",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("query failed")
		},
	})

	svc := NewAlertService(registry, store, bus)

	result, err := svc.GetActiveAlerts(context.Background())
	if err == nil {
		t.Fatal("expected error for failed query")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}
