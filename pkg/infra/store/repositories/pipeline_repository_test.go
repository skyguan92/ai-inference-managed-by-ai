package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
)

func TestPipelineRepository_Create_Get(t *testing.T) {
	repo := NewPipelineRepository()
	ctx := context.Background()

	t.Run("create and get pipeline", func(t *testing.T) {
		p := &pipeline.Pipeline{
			ID:     "pipe-1",
			Name:   "test-pipeline",
			Status: pipeline.PipelineStatusIdle,
			Steps: []pipeline.PipelineStep{
				{ID: "step1", Name: "Step 1", Type: "inference.chat"},
			},
		}

		err := repo.CreatePipeline(ctx, p)
		if err != nil {
			t.Fatalf("CreatePipeline failed: %v", err)
		}

		got, err := repo.GetPipeline(ctx, "pipe-1")
		if err != nil {
			t.Fatalf("GetPipeline failed: %v", err)
		}
		if got.ID != "pipe-1" || got.Name != "test-pipeline" {
			t.Errorf("got %+v, want ID=pipe-1, Name=test-pipeline", got)
		}
	})

	t.Run("create duplicate fails", func(t *testing.T) {
		p := &pipeline.Pipeline{
			ID:     "pipe-1",
			Name:   "duplicate-pipeline",
			Status: pipeline.PipelineStatusIdle,
			Steps:  []pipeline.PipelineStep{},
		}

		err := repo.CreatePipeline(ctx, p)
		if !errors.Is(err, pipeline.ErrPipelineAlreadyExists) {
			t.Errorf("expected ErrPipelineAlreadyExists, got %v", err)
		}
	})

	t.Run("get non-existent pipeline", func(t *testing.T) {
		_, err := repo.GetPipeline(ctx, "nonexistent")
		if !errors.Is(err, pipeline.ErrPipelineNotFound) {
			t.Errorf("expected ErrPipelineNotFound, got %v", err)
		}
	})
}

func TestPipelineRepository_List(t *testing.T) {
	repo := NewPipelineRepository()
	ctx := context.Background()

	t.Run("list with status filter", func(t *testing.T) {
		repo.CreatePipeline(ctx, &pipeline.Pipeline{
			ID:     "pipe-2",
			Name:   "pipeline-2",
			Status: pipeline.PipelineStatusRunning,
			Steps:  []pipeline.PipelineStep{},
		})
		repo.CreatePipeline(ctx, &pipeline.Pipeline{
			ID:     "pipe-3",
			Name:   "pipeline-3",
			Status: pipeline.PipelineStatusPaused,
			Steps:  []pipeline.PipelineStep{},
		})

		runningPipelines, total, err := repo.ListPipelines(ctx, pipeline.PipelineFilter{Status: pipeline.PipelineStatusRunning})
		if err != nil {
			t.Fatalf("ListPipelines failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 running pipeline, got %d", total)
		}
		for _, p := range runningPipelines {
			if p.Status != pipeline.PipelineStatusRunning {
				t.Errorf("expected running status, got %s", p.Status)
			}
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		// Create additional pipelines to have enough data for pagination test
		repo.CreatePipeline(ctx, &pipeline.Pipeline{
			ID:     "pipe-4",
			Name:   "pipeline-4",
			Status: pipeline.PipelineStatusIdle,
			Steps:  []pipeline.PipelineStep{},
		})

		page1, total, err := repo.ListPipelines(ctx, pipeline.PipelineFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("ListPipelines failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(page1) != 2 {
			t.Errorf("expected 2 pipelines in page1, got %d", len(page1))
		}
	})
}

func TestPipelineRepository_Update_Delete(t *testing.T) {
	repo := NewPipelineRepository()
	ctx := context.Background()

	t.Run("update existing pipeline", func(t *testing.T) {
		p := &pipeline.Pipeline{
			ID:     "pipe-update",
			Name:   "update-test",
			Status: pipeline.PipelineStatusIdle,
			Steps:  []pipeline.PipelineStep{},
		}
		repo.CreatePipeline(ctx, p)

		p.Status = pipeline.PipelineStatusRunning
		err := repo.UpdatePipeline(ctx, p)
		if err != nil {
			t.Fatalf("UpdatePipeline failed: %v", err)
		}

		got, _ := repo.GetPipeline(ctx, "pipe-update")
		if got.Status != pipeline.PipelineStatusRunning {
			t.Errorf("status not updated, got %s", got.Status)
		}
	})

	t.Run("delete existing pipeline", func(t *testing.T) {
		p := &pipeline.Pipeline{
			ID:     "pipe-delete",
			Name:   "delete-test",
			Status: pipeline.PipelineStatusIdle,
			Steps:  []pipeline.PipelineStep{},
		}
		repo.CreatePipeline(ctx, p)

		err := repo.DeletePipeline(ctx, "pipe-delete")
		if err != nil {
			t.Fatalf("DeletePipeline failed: %v", err)
		}

		_, err = repo.GetPipeline(ctx, "pipe-delete")
		if !errors.Is(err, pipeline.ErrPipelineNotFound) {
			t.Error("pipeline should have been deleted")
		}
	})
}

func TestPipelineRepository_Run(t *testing.T) {
	repo := NewPipelineRepository()
	ctx := context.Background()

	t.Run("create and get run", func(t *testing.T) {
		run := &pipeline.PipelineRun{
			ID:          "run-1",
			PipelineID:  "pipe-1",
			Status:      pipeline.RunStatusPending,
			StepResults: make(map[string]any),
			StartedAt:   time.Now(),
		}

		err := repo.CreateRun(ctx, run)
		if err != nil {
			t.Fatalf("CreateRun failed: %v", err)
		}

		got, err := repo.GetRun(ctx, "run-1")
		if err != nil {
			t.Fatalf("GetRun failed: %v", err)
		}
		if got.ID != "run-1" || got.Status != pipeline.RunStatusPending {
			t.Errorf("got %+v, want ID=run-1, Status=pending", got)
		}
	})

	t.Run("list runs", func(t *testing.T) {
		repo.CreateRun(ctx, &pipeline.PipelineRun{
			ID:          "run-2",
			PipelineID:  "pipe-1",
			Status:      pipeline.RunStatusRunning,
			StepResults: make(map[string]any),
			StartedAt:   time.Now(),
		})

		runs, err := repo.ListRuns(ctx, "pipe-1")
		if err != nil {
			t.Fatalf("ListRuns failed: %v", err)
		}
		if len(runs) != 2 {
			t.Errorf("expected 2 runs, got %d", len(runs))
		}
	})

	t.Run("update run", func(t *testing.T) {
		run := &pipeline.PipelineRun{
			ID:          "run-update",
			PipelineID:  "pipe-1",
			Status:      pipeline.RunStatusPending,
			StepResults: make(map[string]any),
			StartedAt:   time.Now(),
		}
		repo.CreateRun(ctx, run)

		run.Status = pipeline.RunStatusCompleted
		err := repo.UpdateRun(ctx, run)
		if err != nil {
			t.Fatalf("UpdateRun failed: %v", err)
		}

		got, _ := repo.GetRun(ctx, "run-update")
		if got.Status != pipeline.RunStatusCompleted {
			t.Errorf("status not updated, got %s", got.Status)
		}
	})
}
