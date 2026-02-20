package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestAppResource_URI(t *testing.T) {
	r := NewAppResource("app-123", nil, nil)
	expected := "asms://app/app-123"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestAppResource_Domain(t *testing.T) {
	r := NewAppResource("app-123", nil, nil)
	if r.Domain() != "app" {
		t.Errorf("expected domain 'app', got '%s'", r.Domain())
	}
}

func TestAppResource_Schema(t *testing.T) {
	r := NewAppResource("app-123", nil, nil)
	schema := r.Schema()
	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected 'name' property in schema")
	}
	if _, ok := schema.Properties["template"]; !ok {
		t.Error("expected 'template' property in schema")
	}
}

func TestAppResource_Get(t *testing.T) {
	tests := []struct {
		name       string
		store      AppStore
		provider   AppProvider
		appID      string
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "successful get",
			store: func() AppStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusInstalled))
				return s
			}(),
			provider:   &MockProvider{},
			appID:      "app-123",
			wantErr:    false,
			checkField: "template",
			checkValue: "open-webui",
		},
		{
			name: "get running app with metrics",
			store: func() AppStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusRunning))
				return s
			}(),
			provider:   &MockProvider{},
			appID:      "app-123",
			wantErr:    false,
			checkField: "metrics",
		},
		{
			name:    "nil store",
			store:   nil,
			appID:   "app-123",
			wantErr: true,
		},
		{
			name:    "app not found",
			store:   NewMemoryStore(),
			appID:   "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewAppResource(tt.appID, tt.store, tt.provider)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if tt.checkValue != nil {
				if val, exists := resultMap[tt.checkField]; exists {
					if val != tt.checkValue {
						t.Errorf("expected %s=%v, got %v", tt.checkField, tt.checkValue, val)
					}
				} else {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}
		})
	}
}

func TestAppResource_Watch(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusInstalled))

	r := NewAppResource("app-123", store, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok {
			if update.URI != r.URI() {
				t.Errorf("expected URI=%s, got %s", r.URI(), update.URI)
			}
		}
	case <-ctx.Done():
	}
}

func TestTemplatesResource_URI(t *testing.T) {
	r := NewTemplatesResource(nil)
	expected := "asms://apps/templates"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestTemplatesResource_Domain(t *testing.T) {
	r := NewTemplatesResource(nil)
	if r.Domain() != "app" {
		t.Errorf("expected domain 'app', got '%s'", r.Domain())
	}
}

func TestTemplatesResource_Schema(t *testing.T) {
	r := NewTemplatesResource(nil)
	schema := r.Schema()
	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["templates"]; !ok {
		t.Error("expected 'templates' property in schema")
	}
}

func TestTemplatesResource_Get(t *testing.T) {
	tests := []struct {
		name      string
		provider  AppProvider
		wantErr   bool
		wantCount int
	}{
		{
			name:      "successful get",
			provider:  &MockProvider{},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:     "nil provider",
			provider: nil,
			wantErr:  true,
		},
		{
			name:      "provider error",
			provider:  &MockProvider{templatesErr: errors.New("templates error")},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewTemplatesResource(tt.provider)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			templates, ok := resultMap["templates"].([]map[string]any)
			if !ok {
				t.Error("expected 'templates' to be []map[string]any")
				return
			}

			if len(templates) != tt.wantCount {
				t.Errorf("expected %d templates, got %d", tt.wantCount, len(templates))
			}
		})
	}
}

func TestTemplatesResource_Watch(t *testing.T) {
	r := NewTemplatesResource(&MockProvider{})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok {
			if update.URI != r.URI() {
				t.Errorf("expected URI=%s, got %s", r.URI(), update.URI)
			}
		}
	case <-ctx.Done():
	}
}

func TestParseAppResourceURI(t *testing.T) {
	tests := []struct {
		uri    string
		wantID string
		wantOK bool
	}{
		{"asms://app/app-123", "app-123", true},
		{"asms://app/open-webui", "open-webui", true},
		{"asms://app/", "", false},
		{"asms://apps/templates", "", false},
		{"asms://model/model-123", "", false},
		{"invalid-uri", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			id, ok := ParseAppResourceURI(tt.uri)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if id != tt.wantID {
				t.Errorf("expected id=%s, got %s", tt.wantID, id)
			}
		})
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewAppResource("app-123", nil, nil)
	var _ unit.Resource = NewTemplatesResource(nil)
}

func TestMemoryStore_CRUD(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("app-123", "open-webui", AppStatusInstalled)

	if err := store.Create(ctx, app); err != nil {
		t.Errorf("Create failed: %v", err)
	}

	if err := store.Create(ctx, app); !errors.Is(err, ErrAppAlreadyExists) {
		t.Errorf("expected ErrAppAlreadyExists, got %v", err)
	}

	got, err := store.Get(ctx, "app-123")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if got.Template != "open-webui" {
		t.Errorf("expected template=open-webui, got %s", got.Template)
	}

	_, err = store.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrAppNotFound) {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}

	apps, _, err := store.List(ctx, AppFilter{})
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("expected 1 app, got %d", len(apps))
	}

	app.Status = AppStatusRunning
	if err := store.Update(ctx, app); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	got, _ = store.Get(ctx, "app-123")
	if got.Status != AppStatusRunning {
		t.Errorf("expected status=running, got %s", got.Status)
	}

	nonexistent := createTestApp("nonexistent", "grafana", AppStatusInstalled)
	if err := store.Update(ctx, nonexistent); !errors.Is(err, ErrAppNotFound) {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}

	if err := store.Delete(ctx, "app-123"); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	if err := store.Delete(ctx, "app-123"); !errors.Is(err, ErrAppNotFound) {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}
}

func TestMemoryStore_ListWithFilter(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	apps := []*App{
		createTestApp("app-1", "open-webui", AppStatusRunning),
		createTestApp("app-2", "grafana", AppStatusStopped),
	}

	for _, a := range apps {
		_ = store.Create(ctx, a)
	}

	openWebuiApps, total, err := store.List(ctx, AppFilter{Template: "open-webui"})
	if err != nil {
		t.Errorf("List with filter failed: %v", err)
	}
	if len(openWebuiApps) != 1 || total != 1 {
		t.Errorf("expected 1 open-webui app, got %d (total: %d)", len(openWebuiApps), total)
	}

	runningApps, total, err := store.List(ctx, AppFilter{Status: AppStatusRunning})
	if err != nil {
		t.Errorf("List with status filter failed: %v", err)
	}
	if len(runningApps) != 1 || total != 1 {
		t.Errorf("expected 1 running app, got %d (total: %d)", len(runningApps), total)
	}

	_ = store.Create(ctx, createTestApp("app-3", "nginx", AppStatusInstalled))
	_ = store.Create(ctx, createTestApp("app-4", "redis", AppStatusInstalled))

	pagedApps, total, err := store.List(ctx, AppFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Errorf("List with pagination failed: %v", err)
	}
	if len(pagedApps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(pagedApps))
	}
	if total != 4 {
		t.Errorf("expected total=4, got %d", total)
	}
}
