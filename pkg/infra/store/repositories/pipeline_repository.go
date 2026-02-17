package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
)

type PipelineRepository struct {
	store *pipeline.MemoryStore
}

func NewPipelineRepository() *PipelineRepository {
	return &PipelineRepository{
		store: pipeline.NewMemoryStore(),
	}
}

func (r *PipelineRepository) CreatePipeline(ctx context.Context, p *pipeline.Pipeline) error {
	return r.store.CreatePipeline(ctx, p)
}

func (r *PipelineRepository) GetPipeline(ctx context.Context, id string) (*pipeline.Pipeline, error) {
	return r.store.GetPipeline(ctx, id)
}

func (r *PipelineRepository) ListPipelines(ctx context.Context, filter pipeline.PipelineFilter) ([]pipeline.Pipeline, int, error) {
	return r.store.ListPipelines(ctx, filter)
}

func (r *PipelineRepository) DeletePipeline(ctx context.Context, id string) error {
	return r.store.DeletePipeline(ctx, id)
}

func (r *PipelineRepository) UpdatePipeline(ctx context.Context, p *pipeline.Pipeline) error {
	return r.store.UpdatePipeline(ctx, p)
}

func (r *PipelineRepository) CreateRun(ctx context.Context, run *pipeline.PipelineRun) error {
	return r.store.CreateRun(ctx, run)
}

func (r *PipelineRepository) GetRun(ctx context.Context, id string) (*pipeline.PipelineRun, error) {
	return r.store.GetRun(ctx, id)
}

func (r *PipelineRepository) ListRuns(ctx context.Context, pipelineID string) ([]pipeline.PipelineRun, error) {
	return r.store.ListRuns(ctx, pipelineID)
}

func (r *PipelineRepository) UpdateRun(ctx context.Context, run *pipeline.PipelineRun) error {
	return r.store.UpdateRun(ctx, run)
}

var _ pipeline.PipelineStore = (*PipelineRepository)(nil)
