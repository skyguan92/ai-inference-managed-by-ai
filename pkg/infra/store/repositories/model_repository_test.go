package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

func TestModelRepository_Create_Get(t *testing.T) {
	repo := NewModelRepository()
	ctx := context.Background()

	t.Run("create and get model", func(t *testing.T) {
		m := &model.Model{
			ID:     "model-1",
			Name:   "Test Model",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		}

		err := repo.Create(ctx, m)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := repo.Get(ctx, "model-1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if got.ID != "model-1" || got.Name != "Test Model" {
			t.Errorf("got %+v, want ID=model-1, Name=Test Model", got)
		}
	})

	t.Run("create duplicate fails", func(t *testing.T) {
		m := &model.Model{
			ID:     "model-1",
			Name:   "Duplicate",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		}

		err := repo.Create(ctx, m)
		if !errors.Is(err, model.ErrModelAlreadyExists) {
			t.Errorf("expected ErrModelAlreadyExists, got %v", err)
		}
	})

	t.Run("get non-existent model", func(t *testing.T) {
		_, err := repo.Get(ctx, "nonexistent")
		if !errors.Is(err, model.ErrModelNotFound) {
			t.Errorf("expected ErrModelNotFound, got %v", err)
		}
	})
}

func TestModelRepository_List(t *testing.T) {
	repo := NewModelRepository()
	ctx := context.Background()

	t.Run("list empty repository", func(t *testing.T) {
		models, total, err := repo.List(ctx, model.ModelFilter{})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 0 || len(models) != 0 {
			t.Errorf("expected empty list, got %d models, total %d", len(models), total)
		}
	})

	t.Run("list all models", func(t *testing.T) {
		repo.Create(ctx, &model.Model{
			ID:     "model-1",
			Name:   "Model One",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		})
		repo.Create(ctx, &model.Model{
			ID:     "model-2",
			Name:   "Model Two",
			Type:   model.ModelTypeVLM,
			Format: model.FormatSafetensors,
			Status: model.StatusPending,
		})
		repo.Create(ctx, &model.Model{
			ID:     "model-3",
			Name:   "Model Three",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		})

		models, total, err := repo.List(ctx, model.ModelFilter{})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 || len(models) != 3 {
			t.Errorf("expected 3 models, got %d models, total %d", len(models), total)
		}
	})

	t.Run("list with type filter", func(t *testing.T) {
		llmModels, total, err := repo.List(ctx, model.ModelFilter{Type: model.ModelTypeLLM})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2 LLM models, got %d", total)
		}
		if len(llmModels) != 2 {
			t.Errorf("expected 2 LLM models in result, got %d", len(llmModels))
		}
		for _, m := range llmModels {
			if m.Type != model.ModelTypeLLM {
				t.Errorf("expected LLM type, got %s", m.Type)
			}
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		readyModels, total, err := repo.List(ctx, model.ModelFilter{Status: model.StatusReady})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2 ready models, got %d", total)
		}
		for _, m := range readyModels {
			if m.Status != model.StatusReady {
				t.Errorf("expected ready status, got %s", m.Status)
			}
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		page1, total, err := repo.List(ctx, model.ModelFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(page1) != 2 {
			t.Errorf("expected 2 models in page1, got %d", len(page1))
		}

		page2, _, err := repo.List(ctx, model.ModelFilter{Limit: 2, Offset: 2})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(page2) != 1 {
			t.Errorf("expected 1 model in page2, got %d", len(page2))
		}
	})

	t.Run("list with format filter", func(t *testing.T) {
		ggufModels, total, err := repo.List(ctx, model.ModelFilter{Format: model.FormatGGUF})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2 GGUF models, got %d", total)
		}
		for _, m := range ggufModels {
			if m.Format != model.FormatGGUF {
				t.Errorf("expected GGUF format, got %s", m.Format)
			}
		}
	})

	t.Run("list with offset beyond total", func(t *testing.T) {
		models, total, err := repo.List(ctx, model.ModelFilter{Offset: 10})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(models) != 0 {
			t.Errorf("expected empty result, got %d models", len(models))
		}
	})
}

func TestModelRepository_Update_Delete(t *testing.T) {
	repo := NewModelRepository()
	ctx := context.Background()

	t.Run("update existing model", func(t *testing.T) {
		m := &model.Model{
			ID:     "model-update",
			Name:   "Original Name",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		}
		repo.Create(ctx, m)

		m.Name = "Updated Name"
		err := repo.Update(ctx, m)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := repo.Get(ctx, "model-update")
		if got.Name != "Updated Name" {
			t.Errorf("name not updated, got %s", got.Name)
		}
	})

	t.Run("update non-existent model fails", func(t *testing.T) {
		m := &model.Model{
			ID:     "nonexistent",
			Name:   "Name",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		}
		err := repo.Update(ctx, m)
		if !errors.Is(err, model.ErrModelNotFound) {
			t.Errorf("expected ErrModelNotFound, got %v", err)
		}
	})

	t.Run("delete existing model", func(t *testing.T) {
		m := &model.Model{
			ID:     "model-delete",
			Name:   "To Delete",
			Type:   model.ModelTypeLLM,
			Format: model.FormatGGUF,
			Status: model.StatusReady,
		}
		repo.Create(ctx, m)

		err := repo.Delete(ctx, "model-delete")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = repo.Get(ctx, "model-delete")
		if !errors.Is(err, model.ErrModelNotFound) {
			t.Error("model should have been deleted")
		}
	})

	t.Run("delete non-existent model fails", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if !errors.Is(err, model.ErrModelNotFound) {
			t.Errorf("expected ErrModelNotFound, got %v", err)
		}
	})
}
