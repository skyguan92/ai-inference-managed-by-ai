package alert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type RulesResource struct {
	store    Store
	watchers []chan unit.ResourceUpdate
	mu       sync.Mutex
}

func NewRulesResource(store Store) *RulesResource {
	return &RulesResource{store: store}
}

// AlertResourceFactory creates alert Resource instances dynamically based on URI patterns.
type AlertResourceFactory struct {
	store Store
}

func NewAlertResourceFactory(store Store) *AlertResourceFactory {
	return &AlertResourceFactory{store: store}
}

func (f *AlertResourceFactory) CanCreate(uri string) bool {
	return uri == "asms://alerts/rules" || uri == "asms://alerts/active"
}

func (f *AlertResourceFactory) Create(uri string) (unit.Resource, error) {
	switch uri {
	case "asms://alerts/rules":
		return NewRulesResource(f.store), nil
	case "asms://alerts/active":
		return NewActiveResource(f.store), nil
	default:
		return nil, fmt.Errorf("unknown alert resource URI: %s", uri)
	}
}

func (f *AlertResourceFactory) Pattern() string {
	return "asms://alerts/*"
}

func (r *RulesResource) URI() string {
	return "asms://alerts/rules"
}

func (r *RulesResource) Domain() string {
	return "alert"
}

func (r *RulesResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Alert rules resource",
		Properties: map[string]unit.Field{
			"rules": {
				Name: "rules",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":        {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":      {Name: "name", Schema: unit.Schema{Type: "string"}},
							"condition": {Name: "condition", Schema: unit.Schema{Type: "string"}},
							"severity":  {Name: "severity", Schema: unit.Schema{Type: "string"}},
							"channels":  {Name: "channels", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
							"cooldown":  {Name: "cooldown", Schema: unit.Schema{Type: "number"}},
							"enabled":   {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
						},
					},
				},
			},
		},
	}
}

func (r *RulesResource) Get(ctx context.Context) (any, error) {
	rules, err := r.store.ListRules(ctx, RuleFilter{})
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}

	result := make([]map[string]any, len(rules))
	for i, rule := range rules {
		result[i] = map[string]any{
			"id":        rule.ID,
			"name":      rule.Name,
			"condition": rule.Condition,
			"severity":  rule.Severity,
			"channels":  rule.Channels,
			"cooldown":  rule.Cooldown,
			"enabled":   rule.Enabled,
		}
	}

	return map[string]any{"rules": result}, nil
}

func (r *RulesResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	r.mu.Lock()
	r.watchers = append(r.watchers, ch)
	r.mu.Unlock()

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.mu.Lock()
				for i, w := range r.watchers {
					if w == ch {
						r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
						break
					}
				}
				r.mu.Unlock()
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
					Error:     err,
				}
			}
		}
	}()

	return ch, nil
}

type ActiveResource struct {
	store    Store
	watchers []chan unit.ResourceUpdate
	mu       sync.Mutex
}

func NewActiveResource(store Store) *ActiveResource {
	return &ActiveResource{store: store}
}

func (r *ActiveResource) URI() string {
	return "asms://alerts/active"
}

func (r *ActiveResource) Domain() string {
	return "alert"
}

func (r *ActiveResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Active alerts resource",
		Properties: map[string]unit.Field{
			"alerts": {
				Name: "alerts",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
							"rule_id":      {Name: "rule_id", Schema: unit.Schema{Type: "string"}},
							"rule_name":    {Name: "rule_name", Schema: unit.Schema{Type: "string"}},
							"severity":     {Name: "severity", Schema: unit.Schema{Type: "string"}},
							"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
							"message":      {Name: "message", Schema: unit.Schema{Type: "string"}},
							"triggered_at": {Name: "triggered_at", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (r *ActiveResource) Get(ctx context.Context) (any, error) {
	alerts, err := r.store.ListActiveAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}

	result := make([]map[string]any, len(alerts))
	for i, a := range alerts {
		result[i] = map[string]any{
			"id":           a.ID,
			"rule_id":      a.RuleID,
			"rule_name":    a.RuleName,
			"severity":     a.Severity,
			"status":       a.Status,
			"message":      a.Message,
			"triggered_at": a.TriggeredAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return map[string]any{"alerts": result}, nil
}

func (r *ActiveResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	r.mu.Lock()
	r.watchers = append(r.watchers, ch)
	r.mu.Unlock()

	go func() {
		defer close(ch)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.mu.Lock()
				for i, w := range r.watchers {
					if w == ch {
						r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
						break
					}
				}
				r.mu.Unlock()
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "update",
					Data:      data,
					Error:     err,
				}
			}
		}
	}()

	return ch, nil
}
