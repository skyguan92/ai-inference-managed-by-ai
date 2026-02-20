package app

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestInstallCommand_Name(t *testing.T) {
	cmd := NewInstallCommand(nil, nil)
	if cmd.Name() != "app.install" {
		t.Errorf("expected name 'app.install', got '%s'", cmd.Name())
	}
}

func TestInstallCommand_Domain(t *testing.T) {
	cmd := NewInstallCommand(nil, nil)
	if cmd.Domain() != "app" {
		t.Errorf("expected domain 'app', got '%s'", cmd.Domain())
	}
}

func TestInstallCommand_Schemas(t *testing.T) {
	cmd := NewInstallCommand(nil, nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestInstallCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      AppStore
		provider   AppProvider
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:     "successful install",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"template": "open-webui",
			},
			wantErr:    false,
			checkField: "app_id",
		},
		{
			name:     "install with name and config",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"template": "grafana",
				"name":     "my-monitoring",
				"config":   map[string]any{"port": 3000},
			},
			wantErr:    false,
			checkField: "app_id",
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"template": "open-webui"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"template": "open-webui"},
			wantErr:  true,
		},
		{
			name:     "missing template",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    NewMemoryStore(),
			provider: &MockProvider{installErr: errors.New("install failed")},
			input:    map[string]any{"template": "open-webui"},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(tt.store, tt.provider)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if tt.checkField != "" {
				if _, exists := resultMap[tt.checkField]; !exists {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}
		})
	}
}

func TestUninstallCommand_Name(t *testing.T) {
	cmd := NewUninstallCommand(nil, nil)
	if cmd.Name() != "app.uninstall" {
		t.Errorf("expected name 'app.uninstall', got '%s'", cmd.Name())
	}
}

func TestUninstallCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    AppStore
		provider AppProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful uninstall",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusInstalled),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  false,
		},
		{
			name:     "uninstall with remove_data",
			store:    createStoreWithApp("app-123", "grafana", AppStatusInstalled),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123", "remove_data": true},
			wantErr:  false,
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
			name:     "provider error",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusInstalled),
			provider: &MockProvider{uninstallErr: errors.New("uninstall failed")},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewUninstallCommand(tt.store, tt.provider)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if success, ok := resultMap["success"].(bool); !ok || !success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestStartCommand_Name(t *testing.T) {
	cmd := NewStartCommand(nil, nil)
	if cmd.Name() != "app.start" {
		t.Errorf("expected name 'app.start', got '%s'", cmd.Name())
	}
}

func TestStartCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    AppStore
		provider AppProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful start",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusInstalled),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  false,
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
			name:     "app already running",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusInstalled),
			provider: &MockProvider{startErr: errors.New("start failed")},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCommand(tt.store, tt.provider)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if success, ok := resultMap["success"].(bool); !ok || !success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestStopCommand_Name(t *testing.T) {
	cmd := NewStopCommand(nil, nil)
	if cmd.Name() != "app.stop" {
		t.Errorf("expected name 'app.stop', got '%s'", cmd.Name())
	}
}

func TestStopCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    AppStore
		provider AppProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful stop",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  false,
		},
		{
			name:     "stop with timeout",
			store:    createStoreWithApp("app-123", "grafana", AppStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123", "timeout": 60},
			wantErr:  false,
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
			name:     "app not running",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusInstalled),
			provider: &MockProvider{},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithApp("app-123", "open-webui", AppStatusRunning),
			provider: &MockProvider{stopErr: errors.New("stop failed")},
			input:    map[string]any{"app_id": "app-123"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStopCommand(tt.store, tt.provider)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if success, ok := resultMap["success"].(bool); !ok || !success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	if NewInstallCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for InstallCommand")
	}
	if NewUninstallCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for UninstallCommand")
	}
	if NewStartCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for StartCommand")
	}
	if NewStopCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for StopCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewInstallCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for InstallCommand")
	}
	if len(NewUninstallCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for UninstallCommand")
	}
	if len(NewStartCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for StartCommand")
	}
	if len(NewStopCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for StopCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewInstallCommand(nil, nil)
	var _ unit.Command = NewUninstallCommand(nil, nil)
	var _ unit.Command = NewStartCommand(nil, nil)
	var _ unit.Command = NewStopCommand(nil, nil)
}

func createStoreWithApp(id string, template string, status AppStatus) AppStore {
	store := NewMemoryStore()
	app := createTestApp(id, template, status)
	app.Status = status
	_ = store.Create(context.Background(), app)
	return store
}
