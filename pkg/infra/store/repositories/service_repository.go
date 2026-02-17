package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

type ServiceRepository struct {
	store *service.MemoryStore
}

func NewServiceRepository() *ServiceRepository {
	return &ServiceRepository{
		store: service.NewMemoryStore(),
	}
}

func (r *ServiceRepository) Create(ctx context.Context, svc *service.ModelService) error {
	return r.store.Create(ctx, svc)
}

func (r *ServiceRepository) Get(ctx context.Context, id string) (*service.ModelService, error) {
	return r.store.Get(ctx, id)
}

func (r *ServiceRepository) GetByName(ctx context.Context, name string) (*service.ModelService, error) {
	return r.store.GetByName(ctx, name)
}

func (r *ServiceRepository) List(ctx context.Context, filter service.ServiceFilter) ([]service.ModelService, int, error) {
	return r.store.List(ctx, filter)
}

func (r *ServiceRepository) Delete(ctx context.Context, id string) error {
	return r.store.Delete(ctx, id)
}

func (r *ServiceRepository) Update(ctx context.Context, svc *service.ModelService) error {
	return r.store.Update(ctx, svc)
}

var _ service.ServiceStore = (*ServiceRepository)(nil)
