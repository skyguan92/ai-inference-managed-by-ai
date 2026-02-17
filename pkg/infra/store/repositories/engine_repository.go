package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
)

type EngineRepository struct {
	store *engine.MemoryStore
}

func NewEngineRepository() *EngineRepository {
	return &EngineRepository{
		store: engine.NewMemoryStore(),
	}
}

func (r *EngineRepository) Create(ctx context.Context, e *engine.Engine) error {
	return r.store.Create(ctx, e)
}

func (r *EngineRepository) Get(ctx context.Context, name string) (*engine.Engine, error) {
	return r.store.Get(ctx, name)
}

func (r *EngineRepository) GetByID(ctx context.Context, id string) (*engine.Engine, error) {
	return r.store.GetByID(ctx, id)
}

func (r *EngineRepository) List(ctx context.Context, filter engine.EngineFilter) ([]engine.Engine, int, error) {
	return r.store.List(ctx, filter)
}

func (r *EngineRepository) Delete(ctx context.Context, name string) error {
	return r.store.Delete(ctx, name)
}

func (r *EngineRepository) Update(ctx context.Context, e *engine.Engine) error {
	return r.store.Update(ctx, e)
}

var _ engine.EngineStore = (*EngineRepository)(nil)
