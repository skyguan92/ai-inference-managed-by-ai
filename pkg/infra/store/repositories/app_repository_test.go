package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
)

func TestAppRepository_Create_Get(t *testing.T) {
	repo := NewAppRepository()
	ctx := context.Background()

	t.Run("create and get app", func(t *testing.T) {
		a := &app.App{
			ID:       "app-1",
			Name:     "Test App",
			Template: "open-webui",
			Status:   app.AppStatusInstalled,
			Ports:    []int{8080},
		}

		err := repo.Create(ctx, a)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := repo.Get(ctx, "app-1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if got.ID != "app-1" || got.Name != "Test App" {
			t.Errorf("got %+v, want ID=app-1, Name=Test App", got)
		}
	})

	t.Run("create duplicate fails", func(t *testing.T) {
		a := &app.App{
			ID:       "app-1",
			Name:     "Duplicate App",
			Template: "grafana",
			Status:   app.AppStatusInstalled,
		}

		err := repo.Create(ctx, a)
		if !errors.Is(err, app.ErrAppAlreadyExists) {
			t.Errorf("expected ErrAppAlreadyExists, got %v", err)
		}
	})

	t.Run("get non-existent app", func(t *testing.T) {
		_, err := repo.Get(ctx, "nonexistent")
		if !errors.Is(err, app.ErrAppNotFound) {
			t.Errorf("expected ErrAppNotFound, got %v", err)
		}
	})
}

func TestAppRepository_List(t *testing.T) {
	repo := NewAppRepository()
	ctx := context.Background()

	t.Run("list with status filter", func(t *testing.T) {
		repo.Create(ctx, &app.App{
			ID:       "app-2",
			Name:     "App 2",
			Template: "open-webui",
			Status:   app.AppStatusRunning,
		})
		repo.Create(ctx, &app.App{
			ID:       "app-3",
			Name:     "App 3",
			Template: "grafana",
			Status:   app.AppStatusStopped,
		})

		runningApps, total, err := repo.List(ctx, app.AppFilter{Status: app.AppStatusRunning})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 running app, got %d", total)
		}
		for _, a := range runningApps {
			if a.Status != app.AppStatusRunning {
				t.Errorf("expected running status, got %s", a.Status)
			}
		}
	})

	t.Run("list with template filter", func(t *testing.T) {
		// Create a second open-webui app for this test
		repo.Create(ctx, &app.App{
			ID:       "app-webui-2",
			Name:     "App WebUI 2",
			Template: "open-webui",
			Status:   app.AppStatusInstalled,
		})

		apps, total, err := repo.List(ctx, app.AppFilter{Template: "open-webui"})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2 open-webui apps, got %d", total)
		}
		for _, a := range apps {
			if a.Template != "open-webui" {
				t.Errorf("expected open-webui template, got %s", a.Template)
			}
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		// Clear and create exactly 3 apps for pagination test
		repo.Delete(ctx, "app-2")
		repo.Delete(ctx, "app-3")
		repo.Delete(ctx, "app-webui-2")

		repo.Create(ctx, &app.App{
			ID:       "app-page-1",
			Name:     "App Page 1",
			Template: "open-webui",
			Status:   app.AppStatusInstalled,
		})
		repo.Create(ctx, &app.App{
			ID:       "app-page-2",
			Name:     "App Page 2",
			Template: "grafana",
			Status:   app.AppStatusRunning,
		})
		repo.Create(ctx, &app.App{
			ID:       "app-page-3",
			Name:     "App Page 3",
			Template: "open-webui",
			Status:   app.AppStatusStopped,
		})

		page1, total, err := repo.List(ctx, app.AppFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(page1) != 2 {
			t.Errorf("expected 2 apps in page1, got %d", len(page1))
		}

		page2, _, err := repo.List(ctx, app.AppFilter{Limit: 2, Offset: 2})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(page2) != 1 {
			t.Errorf("expected 1 app in page2, got %d", len(page2))
		}
	})
}

func TestAppRepository_Update_Delete(t *testing.T) {
	repo := NewAppRepository()
	ctx := context.Background()

	t.Run("update existing app", func(t *testing.T) {
		a := &app.App{
			ID:       "app-update",
			Name:     "Original Name",
			Template: "open-webui",
			Status:   app.AppStatusInstalled,
		}
		repo.Create(ctx, a)

		a.Name = "Updated Name"
		a.Status = app.AppStatusRunning
		err := repo.Update(ctx, a)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := repo.Get(ctx, "app-update")
		if got.Name != "Updated Name" {
			t.Errorf("name not updated, got %s", got.Name)
		}
		if got.Status != app.AppStatusRunning {
			t.Errorf("status not updated, got %s", got.Status)
		}
	})

	t.Run("update non-existent app fails", func(t *testing.T) {
		a := &app.App{
			ID:       "nonexistent",
			Name:     "Nonexistent",
			Template: "open-webui",
			Status:   app.AppStatusInstalled,
		}
		err := repo.Update(ctx, a)
		if !errors.Is(err, app.ErrAppNotFound) {
			t.Errorf("expected ErrAppNotFound, got %v", err)
		}
	})

	t.Run("delete existing app", func(t *testing.T) {
		a := &app.App{
			ID:       "app-delete",
			Name:     "To Delete",
			Template: "grafana",
			Status:   app.AppStatusInstalled,
		}
		repo.Create(ctx, a)

		err := repo.Delete(ctx, "app-delete")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = repo.Get(ctx, "app-delete")
		if !errors.Is(err, app.ErrAppNotFound) {
			t.Error("app should have been deleted")
		}
	})

	t.Run("delete non-existent app fails", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if !errors.Is(err, app.ErrAppNotFound) {
			t.Errorf("expected ErrAppNotFound, got %v", err)
		}
	})
}
