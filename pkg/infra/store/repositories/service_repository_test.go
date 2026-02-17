package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

func TestServiceRepository_Create_Get(t *testing.T) {
	repo := NewServiceRepository()
	ctx := context.Background()

	t.Run("create and get service", func(t *testing.T) {
		s := &service.ModelService{
			ID:            "svc-1",
			Name:          "test-service",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      2,
			ResourceClass: service.ResourceClassMedium,
		}

		err := repo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := repo.Get(ctx, "svc-1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if got.ID != "svc-1" || got.Name != "test-service" {
			t.Errorf("got %+v, want ID=svc-1, Name=test-service", got)
		}
	})

	t.Run("create duplicate fails", func(t *testing.T) {
		s := &service.ModelService{
			ID:            "svc-1",
			Name:          "duplicate-service",
			ModelID:       "model-2",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		}

		err := repo.Create(ctx, s)
		if !errors.Is(err, service.ErrServiceAlreadyExists) {
			t.Errorf("expected ErrServiceAlreadyExists, got %v", err)
		}
	})

	t.Run("get non-existent service", func(t *testing.T) {
		_, err := repo.Get(ctx, "nonexistent")
		if !errors.Is(err, service.ErrServiceNotFound) {
			t.Errorf("expected ErrServiceNotFound, got %v", err)
		}
	})

	t.Run("get by name", func(t *testing.T) {
		got, err := repo.GetByName(ctx, "test-service")
		if err != nil {
			t.Fatalf("GetByName failed: %v", err)
		}
		if got.Name != "test-service" {
			t.Errorf("got Name %s, want test-service", got.Name)
		}
	})
}

func TestServiceRepository_List(t *testing.T) {
	repo := NewServiceRepository()
	ctx := context.Background()

	t.Run("list with status filter", func(t *testing.T) {
		// Create 2 running services and 1 stopped service for this test
		repo.Create(ctx, &service.ModelService{
			ID:            "svc-running-1",
			Name:          "service-running-1",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})
		repo.Create(ctx, &service.ModelService{
			ID:            "svc-running-2",
			Name:          "service-running-2",
			ModelID:       "model-2",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})
		repo.Create(ctx, &service.ModelService{
			ID:            "svc-stopped-1",
			Name:          "service-stopped-1",
			ModelID:       "model-3",
			Status:        service.ServiceStatusStopped,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})

		runningServices, total, err := repo.List(ctx, service.ServiceFilter{Status: service.ServiceStatusRunning})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2 running services, got %d", total)
		}
		for _, s := range runningServices {
			if s.Status != service.ServiceStatusRunning {
				t.Errorf("expected running status, got %s", s.Status)
			}
		}
	})

	t.Run("list with model filter", func(t *testing.T) {
		// Clear previous data and create 2 services for model-1
		repo.Delete(ctx, "svc-running-1")
		repo.Delete(ctx, "svc-running-2")
		repo.Delete(ctx, "svc-stopped-1")

		repo.Create(ctx, &service.ModelService{
			ID:            "svc-model1-1",
			Name:          "service-model1-1",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})
		repo.Create(ctx, &service.ModelService{
			ID:            "svc-model1-2",
			Name:          "service-model1-2",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})

		services, total, err := repo.List(ctx, service.ServiceFilter{ModelID: "model-1"})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2 services for model-1, got %d", total)
		}
		for _, s := range services {
			if s.ModelID != "model-1" {
				t.Errorf("expected ModelID model-1, got %s", s.ModelID)
			}
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		// Clear previous data and create exactly 3 services for pagination test
		repo.Delete(ctx, "svc-model1-1")
		repo.Delete(ctx, "svc-model1-2")

		repo.Create(ctx, &service.ModelService{
			ID:            "svc-page-1",
			Name:          "service-page-1",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})
		repo.Create(ctx, &service.ModelService{
			ID:            "svc-page-2",
			Name:          "service-page-2",
			ModelID:       "model-2",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})
		repo.Create(ctx, &service.ModelService{
			ID:            "svc-page-3",
			Name:          "service-page-3",
			ModelID:       "model-3",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		})

		page1, total, err := repo.List(ctx, service.ServiceFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(page1) != 2 {
			t.Errorf("expected 2 services in page1, got %d", len(page1))
		}

		page2, _, err := repo.List(ctx, service.ServiceFilter{Limit: 2, Offset: 2})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(page2) != 1 {
			t.Errorf("expected 1 service in page2, got %d", len(page2))
		}
	})
}

func TestServiceRepository_Update_Delete(t *testing.T) {
	repo := NewServiceRepository()
	ctx := context.Background()

	t.Run("update existing service", func(t *testing.T) {
		s := &service.ModelService{
			ID:            "svc-update",
			Name:          "update-test",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		}
		repo.Create(ctx, s)

		s.Replicas = 5
		err := repo.Update(ctx, s)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := repo.Get(ctx, "svc-update")
		if got.Replicas != 5 {
			t.Errorf("replicas not updated, got %d", got.Replicas)
		}
	})

	t.Run("update non-existent service fails", func(t *testing.T) {
		s := &service.ModelService{
			ID:            "nonexistent",
			Name:          "nonexistent",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		}
		err := repo.Update(ctx, s)
		if !errors.Is(err, service.ErrServiceNotFound) {
			t.Errorf("expected ErrServiceNotFound, got %v", err)
		}
	})

	t.Run("delete existing service", func(t *testing.T) {
		s := &service.ModelService{
			ID:            "svc-delete",
			Name:          "delete-test",
			ModelID:       "model-1",
			Status:        service.ServiceStatusRunning,
			Replicas:      1,
			ResourceClass: service.ResourceClassSmall,
		}
		repo.Create(ctx, s)

		err := repo.Delete(ctx, "svc-delete")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = repo.Get(ctx, "svc-delete")
		if !errors.Is(err, service.ErrServiceNotFound) {
			t.Error("service should have been deleted")
		}
	})

	t.Run("delete non-existent service fails", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if !errors.Is(err, service.ErrServiceNotFound) {
			t.Errorf("expected ErrServiceNotFound, got %v", err)
		}
	})
}
