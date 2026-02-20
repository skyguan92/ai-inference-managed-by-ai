package app

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestGetQuery_Name(t *testing.T) {
	q := NewGetQuery(nil, nil)
	if q.Name() != "app.get" {
		t.Errorf("expected name 'app.get', got '%s'", q.Name())
	}
}

func TestGetQuery_Domain(t *testing.T) {
	q := NewGetQuery(nil, nil)
	if q.Domain() != "app" {
		t.Errorf("expected domain 'app', got '%s'", q.Domain())
	}
}

func TestGetQuery_Schemas(t *testing.T) {
	q := NewGetQuery(nil, nil)

	inputSchema := q.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(inputSchema.Required))
	}

	outputSchema := q.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestGetQuery_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      AppStore
		provider   AppProvider
		input      any
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "successful get",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusInstalled))
				return s
			}(),
			provider:   &MockProvider{},
			input:      map[string]any{"app_id": "app-123"},
			wantErr:    false,
			checkField: "template",
			checkValue: "open-webui",
		},
		{
			name: "get running app with metrics",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusRunning))
				return s
			}(),
			provider:   &MockProvider{},
			input:      map[string]any{"app_id": "app-123"},
			wantErr:    false,
			checkField: "metrics",
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"app_id": "app-123"},
			wantErr: true,
		},
		{
			name:    "missing app_id",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "app not found",
			store:   NewMemoryStore(),
			input:   map[string]any{"app_id": "nonexistent"},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   NewMemoryStore(),
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewGetQuery(tt.store, tt.provider)
			result, err := q.Execute(context.Background(), tt.input)

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

func TestListQuery_Name(t *testing.T) {
	q := NewListQuery(nil)
	if q.Name() != "app.list" {
		t.Errorf("expected name 'app.list', got '%s'", q.Name())
	}
}

func TestListQuery_Execute(t *testing.T) {
	tests := []struct {
		name      string
		store     AppStore
		input     any
		wantErr   bool
		wantCount int
	}{
		{
			name: "list all apps",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-1", "open-webui", AppStatusRunning))
				_ = s.Create(context.Background(), createTestApp("app-2", "grafana", AppStatusStopped))
				return s
			}(),
			input:     map[string]any{},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "list with status filter",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-1", "open-webui", AppStatusRunning))
				_ = s.Create(context.Background(), createTestApp("app-2", "grafana", AppStatusStopped))
				return s
			}(),
			input:     map[string]any{"status": "running"},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "list with template filter",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-1", "open-webui", AppStatusRunning))
				_ = s.Create(context.Background(), createTestApp("app-2", "grafana", AppStatusStopped))
				return s
			}(),
			input:     map[string]any{"template": "grafana"},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:      "empty store",
			store:     NewMemoryStore(),
			input:     map[string]any{},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewListQuery(tt.store)
			result, err := q.Execute(context.Background(), tt.input)

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

			apps, ok := resultMap["apps"].([]map[string]any)
			if !ok {
				t.Error("expected 'apps' to be []map[string]any")
				return
			}

			if len(apps) != tt.wantCount {
				t.Errorf("expected %d apps, got %d", tt.wantCount, len(apps))
			}
		})
	}
}

func TestLogsQuery_Name(t *testing.T) {
	q := NewLogsQuery(nil, nil)
	if q.Name() != "app.logs" {
		t.Errorf("expected name 'app.logs', got '%s'", q.Name())
	}
}

func TestLogsQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    AppStore
		provider AppProvider
		input    any
		wantErr  bool
		wantLogs bool
	}{
		{
			name: "successful get logs",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusRunning))
				return s
			}(),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  false,
			wantLogs: true,
		},
		{
			name: "get logs with tail",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusRunning))
				return s
			}(),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123", "tail": 50},
			wantErr:  false,
			wantLogs: true,
		},
		{
			name: "get logs with since",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusRunning))
				return s
			}(),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123", "since": 1700000000},
			wantErr:  false,
			wantLogs: true,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
		{
			name:     "missing app_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "app not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "nonexistent"},
			wantErr:  true,
		},
		{
			name: "provider error",
			store: func() AppStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestApp("app-123", "open-webui", AppStatusRunning))
				return s
			}(),
			provider: &MockProvider{logsErr: errors.New("logs error")},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewLogsQuery(tt.store, tt.provider)
			result, err := q.Execute(context.Background(), tt.input)

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

			if tt.wantLogs {
				logs, ok := resultMap["logs"].([]map[string]any)
				if !ok || len(logs) == 0 {
					t.Error("expected non-empty logs array")
				}
			}
		})
	}
}

func TestTemplatesQuery_Name(t *testing.T) {
	q := NewTemplatesQuery(nil)
	if q.Name() != "app.templates" {
		t.Errorf("expected name 'app.templates', got '%s'", q.Name())
	}
}

func TestTemplatesQuery_Execute(t *testing.T) {
	tests := []struct {
		name           string
		provider       AppProvider
		input          any
		wantErr        bool
		wantCount      int
		wantFirstField string
	}{
		{
			name:           "list all templates",
			provider:       &MockProvider{},
			input:          map[string]any{},
			wantErr:        false,
			wantCount:      2,
			wantFirstField: "id",
		},
		{
			name: "list with category filter",
			provider: &MockProvider{
				templates: []Template{
					{ID: "open-webui", Name: "Open WebUI", Category: AppCategoryAIChat, Description: "AI Chat", Image: "open-webui:latest"},
				},
			},
			input:          map[string]any{"category": "ai-chat"},
			wantErr:        false,
			wantCount:      1,
			wantFirstField: "id",
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{templatesErr: errors.New("templates error")},
			input:    map[string]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewTemplatesQuery(tt.provider)
			result, err := q.Execute(context.Background(), tt.input)

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

func TestQuery_Description(t *testing.T) {
	if NewGetQuery(nil, nil).Description() == "" {
		t.Error("expected non-empty description for GetQuery")
	}
	if NewListQuery(nil).Description() == "" {
		t.Error("expected non-empty description for ListQuery")
	}
	if NewLogsQuery(nil, nil).Description() == "" {
		t.Error("expected non-empty description for LogsQuery")
	}
	if NewTemplatesQuery(nil).Description() == "" {
		t.Error("expected non-empty description for TemplatesQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewGetQuery(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for GetQuery")
	}
	if len(NewListQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for ListQuery")
	}
	if len(NewLogsQuery(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for LogsQuery")
	}
	if len(NewTemplatesQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for TemplatesQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewGetQuery(nil, nil)
	var _ unit.Query = NewListQuery(nil)
	var _ unit.Query = NewLogsQuery(nil, nil)
	var _ unit.Query = NewTemplatesQuery(nil)
}
