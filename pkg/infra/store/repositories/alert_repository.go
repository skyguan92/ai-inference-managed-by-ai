package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
)

type AlertRepository struct {
	store *alert.MemoryStore
}

func NewAlertRepository() *AlertRepository {
	return &AlertRepository{
		store: alert.NewMemoryStore(),
	}
}

func (r *AlertRepository) CreateRule(ctx context.Context, rule *alert.AlertRule) error {
	return r.store.CreateRule(ctx, rule)
}

func (r *AlertRepository) GetRule(ctx context.Context, id string) (*alert.AlertRule, error) {
	return r.store.GetRule(ctx, id)
}

func (r *AlertRepository) ListRules(ctx context.Context, filter alert.RuleFilter) ([]alert.AlertRule, error) {
	return r.store.ListRules(ctx, filter)
}

func (r *AlertRepository) UpdateRule(ctx context.Context, rule *alert.AlertRule) error {
	return r.store.UpdateRule(ctx, rule)
}

func (r *AlertRepository) DeleteRule(ctx context.Context, id string) error {
	return r.store.DeleteRule(ctx, id)
}

func (r *AlertRepository) CreateAlert(ctx context.Context, a *alert.Alert) error {
	return r.store.CreateAlert(ctx, a)
}

func (r *AlertRepository) GetAlert(ctx context.Context, id string) (*alert.Alert, error) {
	return r.store.GetAlert(ctx, id)
}

func (r *AlertRepository) ListAlerts(ctx context.Context, filter alert.AlertFilter) ([]alert.Alert, int, error) {
	return r.store.ListAlerts(ctx, filter)
}

func (r *AlertRepository) UpdateAlert(ctx context.Context, a *alert.Alert) error {
	return r.store.UpdateAlert(ctx, a)
}

func (r *AlertRepository) ListActiveAlerts(ctx context.Context) ([]alert.Alert, error) {
	return r.store.ListActiveAlerts(ctx)
}

var _ alert.Store = (*AlertRepository)(nil)
