package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
)

func TestAlertRepository_CreateRule(t *testing.T) {
	repo := NewAlertRepository()
	ctx := context.Background()

	t.Run("create and get rule", func(t *testing.T) {
		rule := &alert.AlertRule{
			ID:        "rule-1",
			Name:      "Test Rule",
			Condition: "cpu > 80",
			Severity:  alert.AlertSeverityWarning,
			Enabled:   true,
		}

		err := repo.CreateRule(ctx, rule)
		if err != nil {
			t.Fatalf("CreateRule failed: %v", err)
		}

		got, err := repo.GetRule(ctx, "rule-1")
		if err != nil {
			t.Fatalf("GetRule failed: %v", err)
		}
		if got.ID != "rule-1" || got.Name != "Test Rule" {
			t.Errorf("got %+v, want ID=rule-1, Name=Test Rule", got)
		}
	})

	t.Run("create rule with empty id", func(t *testing.T) {
		rule := &alert.AlertRule{
			Name:      "Auto ID Rule",
			Condition: "memory > 90",
			Severity:  alert.AlertSeverityCritical,
			Enabled:   true,
		}

		err := repo.CreateRule(ctx, rule)
		if err != nil {
			t.Fatalf("CreateRule failed: %v", err)
		}
		if rule.ID == "" {
			t.Error("expected ID to be auto-generated")
		}
	})

	t.Run("get non-existent rule", func(t *testing.T) {
		_, err := repo.GetRule(ctx, "nonexistent")
		if !errors.Is(err, alert.ErrRuleNotFound) {
			t.Errorf("expected ErrRuleNotFound, got %v", err)
		}
	})
}

func TestAlertRepository_ListRules(t *testing.T) {
	repo := NewAlertRepository()
	ctx := context.Background()

	t.Run("list rules", func(t *testing.T) {
		repo.CreateRule(ctx, &alert.AlertRule{
			ID:       "rule-2",
			Name:     "Rule 2",
			Enabled:  true,
			Severity: alert.AlertSeverityWarning,
		})
		repo.CreateRule(ctx, &alert.AlertRule{
			ID:       "rule-3",
			Name:     "Rule 3",
			Enabled:  false,
			Severity: alert.AlertSeverityInfo,
		})

		rules, err := repo.ListRules(ctx, alert.RuleFilter{})
		if err != nil {
			t.Fatalf("ListRules failed: %v", err)
		}
		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}
	})

	t.Run("list enabled rules only", func(t *testing.T) {
		rules, err := repo.ListRules(ctx, alert.RuleFilter{EnabledOnly: true})
		if err != nil {
			t.Fatalf("ListRules failed: %v", err)
		}
		for _, r := range rules {
			if !r.Enabled {
				t.Errorf("expected only enabled rules, got disabled rule %s", r.ID)
			}
		}
	})
}

func TestAlertRepository_Update_DeleteRule(t *testing.T) {
	repo := NewAlertRepository()
	ctx := context.Background()

	t.Run("update existing rule", func(t *testing.T) {
		rule := &alert.AlertRule{
			ID:       "rule-update",
			Name:     "Original Name",
			Enabled:  true,
			Severity: alert.AlertSeverityWarning,
		}
		repo.CreateRule(ctx, rule)

		rule.Name = "Updated Name"
		err := repo.UpdateRule(ctx, rule)
		if err != nil {
			t.Fatalf("UpdateRule failed: %v", err)
		}

		got, _ := repo.GetRule(ctx, "rule-update")
		if got.Name != "Updated Name" {
			t.Errorf("name not updated, got %s", got.Name)
		}
		if got.UpdatedAt.Before(rule.CreatedAt) {
			t.Error("updated_at should be updated")
		}
	})

	t.Run("delete existing rule", func(t *testing.T) {
		rule := &alert.AlertRule{
			ID:       "rule-delete",
			Name:     "To Delete",
			Enabled:  true,
			Severity: alert.AlertSeverityWarning,
		}
		repo.CreateRule(ctx, rule)

		err := repo.DeleteRule(ctx, "rule-delete")
		if err != nil {
			t.Fatalf("DeleteRule failed: %v", err)
		}

		_, err = repo.GetRule(ctx, "rule-delete")
		if !errors.Is(err, alert.ErrRuleNotFound) {
			t.Error("rule should have been deleted")
		}
	})
}

func TestAlertRepository_CreateAlert(t *testing.T) {
	repo := NewAlertRepository()
	ctx := context.Background()

	t.Run("create and get alert", func(t *testing.T) {
		a := &alert.Alert{
			ID:       "alert-1",
			RuleID:   "rule-1",
			RuleName: "Test Rule",
			Severity: alert.AlertSeverityWarning,
			Status:   alert.AlertStatusFiring,
			Message:  "CPU usage is high",
		}

		err := repo.CreateAlert(ctx, a)
		if err != nil {
			t.Fatalf("CreateAlert failed: %v", err)
		}

		got, err := repo.GetAlert(ctx, "alert-1")
		if err != nil {
			t.Fatalf("GetAlert failed: %v", err)
		}
		if got.ID != "alert-1" || got.Message != "CPU usage is high" {
			t.Errorf("got %+v, want ID=alert-1, Message=CPU usage is high", got)
		}
	})

	t.Run("list alerts with filter", func(t *testing.T) {
		repo.CreateAlert(ctx, &alert.Alert{
			ID:       "alert-2",
			RuleID:   "rule-1",
			Severity: alert.AlertSeverityCritical,
			Status:   alert.AlertStatusFiring,
		})
		repo.CreateAlert(ctx, &alert.Alert{
			ID:       "alert-3",
			RuleID:   "rule-2",
			Severity: alert.AlertSeverityWarning,
			Status:   alert.AlertStatusResolved,
		})

		alerts, total, err := repo.ListAlerts(ctx, alert.AlertFilter{Status: alert.AlertStatusFiring})
		if err != nil {
			t.Fatalf("ListAlerts failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2 firing alerts, got %d", total)
		}
		for _, a := range alerts {
			if a.Status != alert.AlertStatusFiring {
				t.Errorf("expected firing status, got %s", a.Status)
			}
		}
	})

	t.Run("list active alerts", func(t *testing.T) {
		alerts, err := repo.ListActiveAlerts(ctx)
		if err != nil {
			t.Fatalf("ListActiveAlerts failed: %v", err)
		}
		for _, a := range alerts {
			if a.Status != alert.AlertStatusFiring && a.Status != alert.AlertStatusAcknowledged {
				t.Errorf("expected active alert, got status %s", a.Status)
			}
		}
	})
}
