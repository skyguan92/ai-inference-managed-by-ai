package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestStartCommand_Name(t *testing.T) {
	cmd := NewStartCommand(nil, nil)
	if cmd.Name() != "engine.start" {
		t.Errorf("expected name 'engine.start', got '%s'", cmd.Name())
	}
}

func TestStartCommand_Domain(t *testing.T) {
	cmd := NewStartCommand(nil, nil)
	if cmd.Domain() != "engine" {
		t.Errorf("expected domain 'engine', got '%s'", cmd.Domain())
	}
}

func TestStartCommand_Schemas(t *testing.T) {
	cmd := NewStartCommand(nil, nil)

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

func TestStartCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      EngineStore
		provider   EngineProvider
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:     "successful start",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusStopped),
			provider: &MockProvider{},
			input: map[string]any{
				"name": "ollama",
			},
			wantErr:    false,
			checkField: "process_id",
		},
		{
			name:     "start with config",
			store:    createStoreWithEngine("vllm", EngineTypeVLLM, EngineStatusStopped),
			provider: &MockProvider{},
			input: map[string]any{
				"name":   "vllm",
				"config": map[string]any{"gpu_memory_utilization": 0.9},
			},
			wantErr:    false,
			checkField: "process_id",
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "missing name",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "engine not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"name": "nonexistent"},
			wantErr:  true,
		},
		{
			name:     "engine already running",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusStopped),
			provider: &MockProvider{startErr: errors.New("start failed")},
			input:    map[string]any{"name": "ollama"},
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

			if tt.checkField != "" {
				if _, exists := resultMap[tt.checkField]; !exists {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}
		})
	}
}

func TestStopCommand_Name(t *testing.T) {
	cmd := NewStopCommand(nil, nil)
	if cmd.Name() != "engine.stop" {
		t.Errorf("expected name 'engine.stop', got '%s'", cmd.Name())
	}
}

func TestStopCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    EngineStore
		provider EngineProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful stop",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  false,
		},
		{
			name:     "stop with force",
			store:    createStoreWithEngine("vllm", EngineTypeVLLM, EngineStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"name": "vllm", "force": true},
			wantErr:  false,
		},
		{
			name:     "stop with timeout",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama", "timeout": 60},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "missing name",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "engine not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"name": "nonexistent"},
			wantErr:  true,
		},
		{
			name:     "engine not running",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusStopped),
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusRunning),
			provider: &MockProvider{stopErr: errors.New("stop failed")},
			input:    map[string]any{"name": "ollama"},
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

func TestRestartCommand_Name(t *testing.T) {
	cmd := NewRestartCommand(nil, nil)
	if cmd.Name() != "engine.restart" {
		t.Errorf("expected name 'engine.restart', got '%s'", cmd.Name())
	}
}

func TestRestartCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      EngineStore
		provider   EngineProvider
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:       "successful restart running engine",
			store:      createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusRunning),
			provider:   &MockProvider{},
			input:      map[string]any{"name": "ollama"},
			wantErr:    false,
			checkField: "process_id",
		},
		{
			name:       "successful restart stopped engine",
			store:      createStoreWithEngine("vllm", EngineTypeVLLM, EngineStatusStopped),
			provider:   &MockProvider{},
			input:      map[string]any{"name": "vllm"},
			wantErr:    false,
			checkField: "process_id",
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "missing name",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "engine not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"name": "nonexistent"},
			wantErr:  true,
		},
		{
			name:     "provider start error",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusStopped),
			provider: &MockProvider{startErr: errors.New("start failed")},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "provider stop error",
			store:    createStoreWithEngine("ollama", EngineTypeOllama, EngineStatusRunning),
			provider: &MockProvider{stopErr: errors.New("stop failed")},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRestartCommand(tt.store, tt.provider)
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

func TestInstallCommand_Name(t *testing.T) {
	cmd := NewInstallCommand(nil, nil)
	if cmd.Name() != "engine.install" {
		t.Errorf("expected name 'engine.install', got '%s'", cmd.Name())
	}
}

func TestInstallCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      EngineStore
		provider   EngineProvider
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:       "successful install",
			store:      NewMemoryStore(),
			provider:   &MockProvider{},
			input:      map[string]any{"name": "ollama"},
			wantErr:    false,
			checkField: "success",
		},
		{
			name:       "install with version",
			store:      NewMemoryStore(),
			provider:   &MockProvider{},
			input:      map[string]any{"name": "vllm", "version": "0.4.0"},
			wantErr:    false,
			checkField: "path",
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "missing name",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    NewMemoryStore(),
			provider: &MockProvider{installErr: errors.New("install failed")},
			input:    map[string]any{"name": "ollama"},
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

func TestCommand_Description(t *testing.T) {
	if NewStartCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for StartCommand")
	}
	if NewStopCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for StopCommand")
	}
	if NewRestartCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for RestartCommand")
	}
	if NewInstallCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for InstallCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewStartCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for StartCommand")
	}
	if len(NewStopCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for StopCommand")
	}
	if len(NewRestartCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for RestartCommand")
	}
	if len(NewInstallCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for InstallCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewStartCommand(nil, nil)
	var _ unit.Command = NewStopCommand(nil, nil)
	var _ unit.Command = NewRestartCommand(nil, nil)
	var _ unit.Command = NewInstallCommand(nil, nil)
}

func createStoreWithEngine(name string, engineType EngineType, status EngineStatus) EngineStore {
	store := NewMemoryStore()
	engine := createTestEngine(name, engineType)
	engine.Status = status
	store.Create(context.Background(), engine)
	return store
}
