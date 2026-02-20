package skill

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// SkillsResource implements the static asms://skills resource.
type SkillsResource struct {
	store    SkillStore
	watchers []chan unit.ResourceUpdate
	mu       sync.Mutex
}

func NewSkillsResource(store SkillStore) *SkillsResource {
	return &SkillsResource{store: store}
}

func (r *SkillsResource) URI() string    { return "asms://skills" }
func (r *SkillsResource) Domain() string { return "skill" }

func (r *SkillsResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "All skills list",
		Properties: map[string]unit.Field{
			"skills": {
				Name: "skills",
				Schema: unit.Schema{
					Type:  "array",
					Items: skillItemSchema(),
				},
			},
			"total": {Name: "total", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *SkillsResource) Get(ctx context.Context) (any, error) {
	skills, total, err := r.store.List(ctx, SkillFilter{})
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	return map[string]any{
		"skills": skillsToMaps(skills),
		"total":  total,
	}, nil
}

func (r *SkillsResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
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

// SkillResource implements a single skill resource at asms://skill/{id}.
type SkillResource struct {
	store    SkillStore
	skillID  string
	watchers []chan unit.ResourceUpdate
	mu       sync.Mutex
}

func NewSkillResource(store SkillStore, skillID string) *SkillResource {
	return &SkillResource{store: store, skillID: skillID}
}

func (r *SkillResource) URI() string    { return "asms://skill/" + r.skillID }
func (r *SkillResource) Domain() string { return "skill" }

func (r *SkillResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Skill detail resource",
		Properties:  skillItemSchema().Properties,
	}
}

func (r *SkillResource) Get(ctx context.Context) (any, error) {
	sk, err := r.store.Get(ctx, r.skillID)
	if err != nil {
		return nil, fmt.Errorf("get skill: %w", err)
	}
	return skillToMap(sk), nil
}

func (r *SkillResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
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

// SkillResourceFactory creates SkillResource instances for asms://skill/{id} URIs.
type SkillResourceFactory struct {
	store SkillStore
}

func NewSkillResourceFactory(store SkillStore) *SkillResourceFactory {
	return &SkillResourceFactory{store: store}
}

func (f *SkillResourceFactory) Pattern() string { return "asms://skill/*" }

func (f *SkillResourceFactory) CanCreate(uri string) bool {
	return strings.HasPrefix(uri, "asms://skill/")
}

func (f *SkillResourceFactory) Create(uri string) (unit.Resource, error) {
	if !f.CanCreate(uri) {
		return nil, fmt.Errorf("unknown skill resource URI: %s", uri)
	}
	skillID := strings.TrimPrefix(uri, "asms://skill/")
	if skillID == "" {
		return nil, fmt.Errorf("skill ID is required in URI: %s", uri)
	}
	return NewSkillResource(f.store, skillID), nil
}
