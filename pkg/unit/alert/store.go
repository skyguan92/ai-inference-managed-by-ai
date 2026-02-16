package alert

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRuleNotFound    = errors.New("rule not found")
	ErrAlertNotFound   = errors.New("alert not found")
	ErrRuleExists      = errors.New("rule already exists")
	ErrInvalidRuleID   = errors.New("invalid rule id")
	ErrInvalidAlertID  = errors.New("invalid alert id")
	ErrInvalidSeverity = errors.New("invalid severity")
)

type Store interface {
	CreateRule(ctx context.Context, rule *AlertRule) error
	GetRule(ctx context.Context, id string) (*AlertRule, error)
	ListRules(ctx context.Context, filter RuleFilter) ([]AlertRule, error)
	UpdateRule(ctx context.Context, rule *AlertRule) error
	DeleteRule(ctx context.Context, id string) error

	CreateAlert(ctx context.Context, alert *Alert) error
	GetAlert(ctx context.Context, id string) (*Alert, error)
	ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, int, error)
	UpdateAlert(ctx context.Context, alert *Alert) error
	ListActiveAlerts(ctx context.Context) ([]Alert, error)
}

type MemoryStore struct {
	rules  map[string]*AlertRule
	alerts map[string]*Alert
	mu     sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
}

func (s *MemoryStore) CreateRule(ctx context.Context, rule *AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	if _, exists := s.rules[rule.ID]; exists {
		return ErrRuleExists
	}

	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	s.rules[rule.ID] = rule
	return nil
}

func (s *MemoryStore) GetRule(ctx context.Context, id string) (*AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, exists := s.rules[id]
	if !exists {
		return nil, ErrRuleNotFound
	}
	return rule, nil
}

func (s *MemoryStore) ListRules(ctx context.Context, filter RuleFilter) ([]AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []AlertRule
	for _, r := range s.rules {
		if filter.EnabledOnly && !r.Enabled {
			continue
		}
		result = append(result, *r)
	}
	return result, nil
}

func (s *MemoryStore) UpdateRule(ctx context.Context, rule *AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[rule.ID]; !exists {
		return ErrRuleNotFound
	}

	rule.UpdatedAt = time.Now()
	s.rules[rule.ID] = rule
	return nil
}

func (s *MemoryStore) DeleteRule(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[id]; !exists {
		return ErrRuleNotFound
	}

	delete(s.rules, id)
	return nil
}

func (s *MemoryStore) CreateAlert(ctx context.Context, alert *Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}

	s.alerts[alert.ID] = alert
	return nil
}

func (s *MemoryStore) GetAlert(ctx context.Context, id string) (*Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, exists := s.alerts[id]
	if !exists {
		return nil, ErrAlertNotFound
	}
	return alert, nil
}

func (s *MemoryStore) ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Alert
	for _, a := range s.alerts {
		if filter.RuleID != "" && a.RuleID != filter.RuleID {
			continue
		}
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.Severity != "" && a.Severity != filter.Severity {
			continue
		}
		result = append(result, *a)
	}

	total := len(result)

	offset := filter.Offset
	if offset > len(result) {
		offset = len(result)
	}

	end := len(result)
	if filter.Limit > 0 {
		end = offset + filter.Limit
		if end > len(result) {
			end = len(result)
		}
	}

	return result[offset:end], total, nil
}

func (s *MemoryStore) UpdateAlert(ctx context.Context, alert *Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alerts[alert.ID]; !exists {
		return ErrAlertNotFound
	}

	s.alerts[alert.ID] = alert
	return nil
}

func (s *MemoryStore) ListActiveAlerts(ctx context.Context) ([]Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Alert
	for _, a := range s.alerts {
		if a.Status == AlertStatusFiring || a.Status == AlertStatusAcknowledged {
			result = append(result, *a)
		}
	}
	return result, nil
}

func isValidSeverity(s AlertSeverity) bool {
	switch s {
	case AlertSeverityInfo, AlertSeverityWarning, AlertSeverityCritical:
		return true
	default:
		return false
	}
}
