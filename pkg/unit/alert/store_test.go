package alert

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newTestRule(id, name string, enabled bool) *AlertRule {
	return &AlertRule{
		ID:        id,
		Name:      name,
		Condition: "cpu > 90",
		Severity:  AlertSeverityWarning,
		Enabled:   enabled,
	}
}

func newTestAlert(id, ruleID string, status AlertStatus, severity AlertSeverity) *Alert {
	return &Alert{
		ID:          id,
		RuleID:      ruleID,
		RuleName:    "test-rule",
		Severity:    severity,
		Status:      status,
		Message:     "test alert",
		TriggeredAt: time.Now(),
	}
}

// --- NewMemoryStore ---

func TestNewMemoryStore_NotNil(t *testing.T) {
	s := NewMemoryStore()
	require.NotNil(t, s)
}

// --- CreateRule ---

func TestMemoryStore_CreateRule_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("", "cpu-alert", true)
	err := s.CreateRule(ctx, rule)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID, "expected ID to be auto-generated")
}

func TestMemoryStore_CreateRule_SetsTimestamps(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	before := time.Now()

	rule := newTestRule("", "cpu-alert", true)
	require.NoError(t, s.CreateRule(ctx, rule))

	assert.True(t, rule.CreatedAt.After(before) || rule.CreatedAt.Equal(before))
	assert.True(t, rule.UpdatedAt.After(before) || rule.UpdatedAt.Equal(before))
}

func TestMemoryStore_CreateRule_WithExplicitID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("rule-123", "cpu-alert", true)
	require.NoError(t, s.CreateRule(ctx, rule))

	got, err := s.GetRule(ctx, "rule-123")
	require.NoError(t, err)
	assert.Equal(t, "rule-123", got.ID)
}

func TestMemoryStore_CreateRule_Duplicate_ReturnsError(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("rule-123", "cpu-alert", true)
	require.NoError(t, s.CreateRule(ctx, rule))

	err := s.CreateRule(ctx, rule)
	assert.ErrorIs(t, err, ErrRuleExists)
}

// --- GetRule ---

func TestMemoryStore_GetRule_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("rule-1", "mem-alert", false)
	require.NoError(t, s.CreateRule(ctx, rule))

	got, err := s.GetRule(ctx, "rule-1")
	require.NoError(t, err)
	assert.Equal(t, "rule-1", got.ID)
	assert.Equal(t, "mem-alert", got.Name)
}

func TestMemoryStore_GetRule_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.GetRule(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRuleNotFound)
}

// --- UpdateRule ---

func TestMemoryStore_UpdateRule_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("rule-1", "old-name", true)
	require.NoError(t, s.CreateRule(ctx, rule))

	rule.Name = "new-name"
	require.NoError(t, s.UpdateRule(ctx, rule))

	got, err := s.GetRule(ctx, "rule-1")
	require.NoError(t, err)
	assert.Equal(t, "new-name", got.Name)
}

func TestMemoryStore_UpdateRule_SetsUpdatedAt(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("rule-1", "cpu-alert", true)
	require.NoError(t, s.CreateRule(ctx, rule))

	before := time.Now()
	require.NoError(t, s.UpdateRule(ctx, rule))

	got, err := s.GetRule(ctx, "rule-1")
	require.NoError(t, err)
	assert.True(t, got.UpdatedAt.After(before) || got.UpdatedAt.Equal(before))
}

func TestMemoryStore_UpdateRule_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("nonexistent", "cpu-alert", true)
	err := s.UpdateRule(ctx, rule)
	assert.ErrorIs(t, err, ErrRuleNotFound)
}

// --- DeleteRule ---

func TestMemoryStore_DeleteRule_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rule := newTestRule("rule-1", "cpu-alert", true)
	require.NoError(t, s.CreateRule(ctx, rule))

	require.NoError(t, s.DeleteRule(ctx, "rule-1"))

	_, err := s.GetRule(ctx, "rule-1")
	assert.ErrorIs(t, err, ErrRuleNotFound)
}

func TestMemoryStore_DeleteRule_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.DeleteRule(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRuleNotFound)
}

// --- ListRules ---

func TestMemoryStore_ListRules_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateRule(ctx, newTestRule("r1", "rule-1", true)))
	require.NoError(t, s.CreateRule(ctx, newTestRule("r2", "rule-2", false)))
	require.NoError(t, s.CreateRule(ctx, newTestRule("r3", "rule-3", true)))

	rules, err := s.ListRules(ctx, RuleFilter{})
	require.NoError(t, err)
	assert.Len(t, rules, 3)
}

func TestMemoryStore_ListRules_EnabledOnly(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateRule(ctx, newTestRule("r1", "rule-1", true)))
	require.NoError(t, s.CreateRule(ctx, newTestRule("r2", "rule-2", false)))
	require.NoError(t, s.CreateRule(ctx, newTestRule("r3", "rule-3", true)))

	rules, err := s.ListRules(ctx, RuleFilter{EnabledOnly: true})
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	for _, r := range rules {
		assert.True(t, r.Enabled)
	}
}

func TestMemoryStore_ListRules_Empty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	rules, err := s.ListRules(ctx, RuleFilter{})
	require.NoError(t, err)
	assert.Empty(t, rules)
}

// --- CreateAlert ---

func TestMemoryStore_CreateAlert_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	alert := newTestAlert("", "rule-1", AlertStatusFiring, AlertSeverityWarning)
	err := s.CreateAlert(ctx, alert)
	require.NoError(t, err)
	assert.NotEmpty(t, alert.ID, "expected ID auto-generated")
}

func TestMemoryStore_CreateAlert_WithExplicitID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	alert := newTestAlert("alert-123", "rule-1", AlertStatusFiring, AlertSeverityWarning)
	require.NoError(t, s.CreateAlert(ctx, alert))

	got, err := s.GetAlert(ctx, "alert-123")
	require.NoError(t, err)
	assert.Equal(t, "alert-123", got.ID)
}

// --- GetAlert ---

func TestMemoryStore_GetAlert_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	alert := newTestAlert("alert-1", "rule-1", AlertStatusFiring, AlertSeverityCritical)
	require.NoError(t, s.CreateAlert(ctx, alert))

	got, err := s.GetAlert(ctx, "alert-1")
	require.NoError(t, err)
	assert.Equal(t, "alert-1", got.ID)
	assert.Equal(t, AlertSeverityCritical, got.Severity)
}

func TestMemoryStore_GetAlert_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.GetAlert(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrAlertNotFound)
}

// --- UpdateAlert ---

func TestMemoryStore_UpdateAlert_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	alert := newTestAlert("alert-1", "rule-1", AlertStatusFiring, AlertSeverityWarning)
	require.NoError(t, s.CreateAlert(ctx, alert))

	alert.Status = AlertStatusAcknowledged
	require.NoError(t, s.UpdateAlert(ctx, alert))

	got, err := s.GetAlert(ctx, "alert-1")
	require.NoError(t, err)
	assert.Equal(t, AlertStatusAcknowledged, got.Status)
}

func TestMemoryStore_UpdateAlert_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	alert := newTestAlert("nonexistent", "rule-1", AlertStatusFiring, AlertSeverityWarning)
	err := s.UpdateAlert(ctx, alert)
	assert.ErrorIs(t, err, ErrAlertNotFound)
}

// --- ListAlerts ---

func TestMemoryStore_ListAlerts_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a1", "r1", AlertStatusFiring, AlertSeverityWarning)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a2", "r1", AlertStatusResolved, AlertSeverityInfo)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a3", "r2", AlertStatusFiring, AlertSeverityCritical)))

	alerts, total, err := s.ListAlerts(ctx, AlertFilter{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, alerts, 3)
}

func TestMemoryStore_ListAlerts_FilterByRuleID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a1", "r1", AlertStatusFiring, AlertSeverityWarning)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a2", "r2", AlertStatusFiring, AlertSeverityWarning)))

	alerts, total, err := s.ListAlerts(ctx, AlertFilter{RuleID: "r1"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "r1", alerts[0].RuleID)
}

func TestMemoryStore_ListAlerts_FilterByStatus(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a1", "r1", AlertStatusFiring, AlertSeverityWarning)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a2", "r1", AlertStatusResolved, AlertSeverityWarning)))

	alerts, total, err := s.ListAlerts(ctx, AlertFilter{Status: AlertStatusFiring})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, AlertStatusFiring, alerts[0].Status)
}

func TestMemoryStore_ListAlerts_FilterBySeverity(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a1", "r1", AlertStatusFiring, AlertSeverityCritical)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a2", "r1", AlertStatusFiring, AlertSeverityInfo)))

	alerts, total, err := s.ListAlerts(ctx, AlertFilter{Severity: AlertSeverityCritical})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, AlertSeverityCritical, alerts[0].Severity)
}

func TestMemoryStore_ListAlerts_Limit(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "a" + string(rune('0'+i))
		require.NoError(t, s.CreateAlert(ctx, newTestAlert(id, "r1", AlertStatusFiring, AlertSeverityWarning)))
	}

	alerts, total, err := s.ListAlerts(ctx, AlertFilter{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, alerts, 2)
}

// --- ListActiveAlerts ---

func TestMemoryStore_ListActiveAlerts_ReturnsFiringAndAcknowledged(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a1", "r1", AlertStatusFiring, AlertSeverityWarning)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a2", "r1", AlertStatusAcknowledged, AlertSeverityWarning)))
	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a3", "r1", AlertStatusResolved, AlertSeverityWarning)))

	active, err := s.ListActiveAlerts(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 2)
	for _, a := range active {
		assert.True(t, a.Status == AlertStatusFiring || a.Status == AlertStatusAcknowledged)
	}
}

func TestMemoryStore_ListActiveAlerts_Empty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateAlert(ctx, newTestAlert("a1", "r1", AlertStatusResolved, AlertSeverityInfo)))

	active, err := s.ListActiveAlerts(ctx)
	require.NoError(t, err)
	assert.Empty(t, active)
}

// --- isValidSeverity ---

func TestIsValidSeverity(t *testing.T) {
	tests := []struct {
		severity AlertSeverity
		want     bool
	}{
		{AlertSeverityInfo, true},
		{AlertSeverityWarning, true},
		{AlertSeverityCritical, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.want, isValidSeverity(tt.severity))
		})
	}
}

// --- Concurrent access ---

func TestMemoryStore_ConcurrentRuleOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	// Seed rules
	for i := 0; i < 5; i++ {
		id := "rule-" + string(rune('0'+i))
		require.NoError(t, s.CreateRule(ctx, newTestRule(id, id, true)))
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = s.ListRules(ctx, RuleFilter{})
			_, _ = s.GetRule(ctx, "rule-0")
		}(i)
	}
	wg.Wait()
}

func TestMemoryStore_ConcurrentAlertOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			alert := newTestAlert("", "r1", AlertStatusFiring, AlertSeverityWarning)
			_ = s.CreateAlert(ctx, alert)
			_, _ = s.ListActiveAlerts(ctx)
		}(i)
	}
	wg.Wait()
}
