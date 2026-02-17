package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
)

type AlertService struct {
	registry *unit.Registry
	store    alert.Store
	bus      *eventbus.InMemoryEventBus
}

func NewAlertService(registry *unit.Registry, store alert.Store, bus *eventbus.InMemoryEventBus) *AlertService {
	return &AlertService{
		registry: registry,
		store:    store,
		bus:      bus,
	}
}

type CreateRuleInput struct {
	Name      string
	Condition string
	Severity  alert.AlertSeverity
	Channels  []string
	Cooldown  int
}

type CreateRuleWithTestResult struct {
	RuleID    string
	Valid     bool
	TestError string
	Rule      *alert.AlertRule
}

func (s *AlertService) CreateRuleWithTest(ctx context.Context, input CreateRuleInput) (*CreateRuleWithTestResult, error) {
	if err := s.validateRuleInput(input); err != nil {
		return nil, fmt.Errorf("validate rule: %w", err)
	}

	createCmd := s.registry.GetCommand("alert.create_rule")
	if createCmd == nil {
		return nil, fmt.Errorf("alert.create_rule command not found")
	}

	result, err := createCmd.Execute(ctx, map[string]any{
		"name":      input.Name,
		"condition": input.Condition,
		"severity":  string(input.Severity),
		"channels":  input.Channels,
		"cooldown":  input.Cooldown,
	})
	if err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected create result type")
	}

	ruleID, _ := resultMap["rule_id"].(string)
	if ruleID == "" {
		return nil, fmt.Errorf("rule_id not found in create result")
	}

	testResult := &CreateRuleWithTestResult{
		RuleID: ruleID,
		Valid:  true,
	}

	if err := s.testRule(ctx, ruleID); err != nil {
		testResult.Valid = false
		testResult.TestError = err.Error()
	}

	rule, err := s.store.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}
	testResult.Rule = rule

	s.publishEvent(ctx, "alert.rule_created", map[string]any{
		"rule_id": ruleID,
		"name":    input.Name,
		"valid":   testResult.Valid,
	})

	return testResult, nil
}

type UpdateRuleInput struct {
	Name      string
	Condition string
	Severity  alert.AlertSeverity
	Channels  []string
	Cooldown  *int
	Enabled   *bool
}

type UpdateRuleWithValidationResult struct {
	Success       bool
	RuleID        string
	Valid         bool
	ValidateError string
	Rule          *alert.AlertRule
}

func (s *AlertService) UpdateRuleWithValidation(ctx context.Context, ruleID string, input UpdateRuleInput) (*UpdateRuleWithValidationResult, error) {
	if ruleID == "" {
		return nil, alert.ErrInvalidRuleID
	}

	existingRule, err := s.store.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}

	if input.Condition != "" && input.Condition != existingRule.Condition {
		if err := s.validateCondition(input.Condition); err != nil {
			return nil, fmt.Errorf("validate condition: %w", err)
		}
	}

	updateCmd := s.registry.GetCommand("alert.update_rule")
	if updateCmd == nil {
		return nil, fmt.Errorf("alert.update_rule command not found")
	}

	updateInput := map[string]any{"rule_id": ruleID}
	if input.Name != "" {
		updateInput["name"] = input.Name
	}
	if input.Condition != "" {
		updateInput["condition"] = input.Condition
	}
	if input.Enabled != nil {
		updateInput["enabled"] = *input.Enabled
	}

	_, err = updateCmd.Execute(ctx, updateInput)
	if err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}

	result := &UpdateRuleWithValidationResult{
		Success: true,
		RuleID:  ruleID,
		Valid:   true,
	}

	if input.Condition != "" {
		if err := s.testRule(ctx, ruleID); err != nil {
			result.Valid = false
			result.ValidateError = err.Error()
		}
	}

	updatedRule, err := s.store.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get updated rule: %w", err)
	}
	result.Rule = updatedRule

	s.publishEvent(ctx, "alert.rule_updated", map[string]any{
		"rule_id": ruleID,
		"valid":   result.Valid,
	})

	return result, nil
}

type DeleteRuleWithCleanupResult struct {
	Success        bool
	RuleID         string
	DeletedAlerts  int
	DisabledAlerts int
}

func (s *AlertService) DeleteRuleWithCleanup(ctx context.Context, ruleID string) (*DeleteRuleWithCleanupResult, error) {
	if ruleID == "" {
		return nil, alert.ErrInvalidRuleID
	}

	_, err := s.store.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}

	alertFilter := alert.AlertFilter{
		RuleID: ruleID,
		Limit:  1000,
	}
	alerts, _, err := s.store.ListAlerts(ctx, alertFilter)
	if err != nil {
		return nil, fmt.Errorf("list alerts for rule: %w", err)
	}

	var deletedAlerts, disabledAlerts int
	for _, a := range alerts {
		if a.Status == alert.AlertStatusResolved {
			deletedAlerts++
		} else {
			disabledAlerts++
		}
	}

	deleteCmd := s.registry.GetCommand("alert.delete_rule")
	if deleteCmd == nil {
		return nil, fmt.Errorf("alert.delete_rule command not found")
	}

	_, err = deleteCmd.Execute(ctx, map[string]any{"rule_id": ruleID})
	if err != nil {
		return nil, fmt.Errorf("delete rule: %w", err)
	}

	result := &DeleteRuleWithCleanupResult{
		Success:        true,
		RuleID:         ruleID,
		DeletedAlerts:  deletedAlerts,
		DisabledAlerts: disabledAlerts,
	}

	s.publishEvent(ctx, "alert.rule_deleted", map[string]any{
		"rule_id":        ruleID,
		"deleted_alerts": deletedAlerts,
	})

	return result, nil
}

type AcknowledgeAlertResult struct {
	Success bool
	AlertID string
	Alert   *alert.Alert
}

func (s *AlertService) AcknowledgeAlert(ctx context.Context, alertID string) (*AcknowledgeAlertResult, error) {
	if alertID == "" {
		return nil, alert.ErrInvalidAlertID
	}

	ackCmd := s.registry.GetCommand("alert.acknowledge")
	if ackCmd == nil {
		return nil, fmt.Errorf("alert.acknowledge command not found")
	}

	_, err := ackCmd.Execute(ctx, map[string]any{"alert_id": alertID})
	if err != nil {
		return nil, fmt.Errorf("acknowledge alert: %w", err)
	}

	updatedAlert, err := s.store.GetAlert(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}

	s.publishEvent(ctx, "alert.acknowledged", map[string]any{
		"alert_id": alertID,
		"rule_id":  updatedAlert.RuleID,
	})

	return &AcknowledgeAlertResult{
		Success: true,
		AlertID: alertID,
		Alert:   updatedAlert,
	}, nil
}

type ResolveAlertResult struct {
	Success bool
	AlertID string
	Alert   *alert.Alert
}

func (s *AlertService) ResolveAlert(ctx context.Context, alertID string) (*ResolveAlertResult, error) {
	if alertID == "" {
		return nil, alert.ErrInvalidAlertID
	}

	resolveCmd := s.registry.GetCommand("alert.resolve")
	if resolveCmd == nil {
		return nil, fmt.Errorf("alert.resolve command not found")
	}

	_, err := resolveCmd.Execute(ctx, map[string]any{"alert_id": alertID})
	if err != nil {
		return nil, fmt.Errorf("resolve alert: %w", err)
	}

	updatedAlert, err := s.store.GetAlert(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}

	s.publishEvent(ctx, "alert.resolved", map[string]any{
		"alert_id": alertID,
		"rule_id":  updatedAlert.RuleID,
	})

	return &ResolveAlertResult{
		Success: true,
		AlertID: alertID,
		Alert:   updatedAlert,
	}, nil
}

type ActiveAlertsResult struct {
	Alerts     []alert.Alert
	Total      int
	ByStatus   map[alert.AlertStatus]int
	BySeverity map[alert.AlertSeverity]int
}

func (s *AlertService) GetActiveAlerts(ctx context.Context) (*ActiveAlertsResult, error) {
	activeQuery := s.registry.GetQuery("alert.active")
	if activeQuery == nil {
		return nil, fmt.Errorf("alert.active query not found")
	}

	result, err := activeQuery.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("get active alerts: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected active alerts result type")
	}

	alertsRaw, ok := resultMap["alerts"].([]any)
	if !ok {
		return nil, fmt.Errorf("alerts not found in result")
	}

	alerts := make([]alert.Alert, len(alertsRaw))
	byStatus := make(map[alert.AlertStatus]int)
	bySeverity := make(map[alert.AlertSeverity]int)

	for i, a := range alertsRaw {
		aMap, ok := a.(map[string]any)
		if !ok {
			continue
		}

		alerts[i] = alert.Alert{
			ID:       getString(aMap, "id"),
			RuleID:   getString(aMap, "rule_id"),
			RuleName: getString(aMap, "rule_name"),
			Severity: alert.AlertSeverity(getString(aMap, "severity")),
			Status:   alert.AlertStatus(getString(aMap, "status")),
			Message:  getString(aMap, "message"),
		}

		byStatus[alerts[i].Status]++
		bySeverity[alerts[i].Severity]++
	}

	return &ActiveAlertsResult{
		Alerts:     alerts,
		Total:      len(alerts),
		ByStatus:   byStatus,
		BySeverity: bySeverity,
	}, nil
}

type HistoryResult struct {
	Alerts []alert.Alert
	Total  int
}

func (s *AlertService) GetHistoryByRule(ctx context.Context, ruleID string, limit int) (*HistoryResult, error) {
	if ruleID == "" {
		return nil, alert.ErrInvalidRuleID
	}

	if limit <= 0 {
		limit = 100
	}

	historyQuery := s.registry.GetQuery("alert.history")
	if historyQuery == nil {
		return nil, fmt.Errorf("alert.history query not found")
	}

	result, err := historyQuery.Execute(ctx, map[string]any{
		"rule_id": ruleID,
		"limit":   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get history by rule: %w", err)
	}

	alerts, total, err := s.parseHistoryResult(result)
	if err != nil {
		return nil, err
	}

	return &HistoryResult{
		Alerts: alerts,
		Total:  total,
	}, nil
}

func (s *AlertService) GetHistoryBySeverity(ctx context.Context, severity alert.AlertSeverity, limit int) (*HistoryResult, error) {
	if !isValidSeverity(severity) {
		return nil, alert.ErrInvalidSeverity
	}

	if limit <= 0 {
		limit = 100
	}

	historyQuery := s.registry.GetQuery("alert.history")
	if historyQuery == nil {
		return nil, fmt.Errorf("alert.history query not found")
	}

	result, err := historyQuery.Execute(ctx, map[string]any{
		"severity": string(severity),
		"limit":    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get history by severity: %w", err)
	}

	alerts, total, err := s.parseHistoryResult(result)
	if err != nil {
		return nil, err
	}

	return &HistoryResult{
		Alerts: alerts,
		Total:  total,
	}, nil
}

type RuleSummary struct {
	Rule          *alert.AlertRule
	ActiveCount   int
	ResolvedCount int
	LastTriggered *time.Time
}

func (s *AlertService) GetRuleSummary(ctx context.Context, ruleID string) (*RuleSummary, error) {
	rule, err := s.store.GetRule(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}

	activeAlerts, err := s.store.ListActiveAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active alerts: %w", err)
	}

	var activeCount int
	var lastTriggered *time.Time
	for _, a := range activeAlerts {
		if a.RuleID == ruleID {
			activeCount++
			if lastTriggered == nil || a.TriggeredAt.After(*lastTriggered) {
				lastTriggered = &a.TriggeredAt
			}
		}
	}

	historyFilter := alert.AlertFilter{
		RuleID: ruleID,
		Status: alert.AlertStatusResolved,
		Limit:  1000,
	}
	resolvedAlerts, _, err := s.store.ListAlerts(ctx, historyFilter)
	if err != nil {
		return nil, fmt.Errorf("get resolved alerts: %w", err)
	}

	return &RuleSummary{
		Rule:          rule,
		ActiveCount:   activeCount,
		ResolvedCount: len(resolvedAlerts),
		LastTriggered: lastTriggered,
	}, nil
}

func (s *AlertService) ListRules(ctx context.Context, enabledOnly bool) ([]alert.AlertRule, error) {
	filter := alert.RuleFilter{EnabledOnly: enabledOnly}
	return s.store.ListRules(ctx, filter)
}

func (s *AlertService) validateRuleInput(input CreateRuleInput) error {
	if input.Name == "" {
		return fmt.Errorf("name is required")
	}
	if input.Condition == "" {
		return fmt.Errorf("condition is required")
	}
	if !isValidSeverity(input.Severity) {
		return alert.ErrInvalidSeverity
	}
	return s.validateCondition(input.Condition)
}

func (s *AlertService) validateCondition(condition string) error {
	if len(condition) == 0 {
		return fmt.Errorf("condition cannot be empty")
	}
	if len(condition) > 1000 {
		return fmt.Errorf("condition too long")
	}
	return nil
}

func (s *AlertService) testRule(ctx context.Context, ruleID string) error {
	_, err := s.store.GetRule(ctx, ruleID)
	return err
}

func (s *AlertService) parseHistoryResult(result any) ([]alert.Alert, int, error) {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected history result type")
	}

	alertsRaw, ok := resultMap["alerts"].([]any)
	if !ok {
		return nil, 0, fmt.Errorf("alerts not found in result")
	}

	alerts := make([]alert.Alert, len(alertsRaw))
	for i, a := range alertsRaw {
		aMap, ok := a.(map[string]any)
		if !ok {
			continue
		}

		alerts[i] = alert.Alert{
			ID:       getString(aMap, "id"),
			RuleID:   getString(aMap, "rule_id"),
			RuleName: getString(aMap, "rule_name"),
			Severity: alert.AlertSeverity(getString(aMap, "severity")),
			Status:   alert.AlertStatus(getString(aMap, "status")),
			Message:  getString(aMap, "message"),
		}
	}

	return alerts, len(alerts), nil
}

func (s *AlertService) publishEvent(ctx context.Context, eventType string, payload any) {
	if s.bus == nil {
		return
	}

	evt := &alertEvent{
		eventType: eventType,
		domain:    "alert",
		payload:   payload,
	}

	_ = s.bus.Publish(evt)
}

type alertEvent struct {
	eventType string
	domain    string
	payload   any
}

func (e *alertEvent) Type() string          { return e.eventType }
func (e *alertEvent) Domain() string        { return e.domain }
func (e *alertEvent) Payload() any          { return e.payload }
func (e *alertEvent) Timestamp() time.Time  { return time.Now() }
func (e *alertEvent) CorrelationID() string { return "" }

func isValidSeverity(s alert.AlertSeverity) bool {
	switch s {
	case alert.AlertSeverityInfo, alert.AlertSeverityWarning, alert.AlertSeverityCritical:
		return true
	default:
		return false
	}
}
