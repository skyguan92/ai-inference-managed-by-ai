package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestCreateCommand_Name(t *testing.T) {
	cmd := NewCreateCommand(nil, nil)
	if cmd.Name() != "service.create" {
		t.Errorf("expected name 'service.create', got '%s'", cmd.Name())
	}
}

func TestCreateCommand_Domain(t *testing.T) {
	cmd := NewCreateCommand(nil, nil)
	if cmd.Domain() != "service" {
		t.Errorf("expected domain 'service', got '%s'", cmd.Domain())
	}
}

func TestCreateCommand_Schemas(t *testing.T) {
	cmd := NewCreateCommand(nil, nil)

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

func TestCreateCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      ServiceStore
		provider   ServiceProvider
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:     "successful create",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"model_id": "llama3-70b",
			},
			wantErr:    false,
			checkField: "service_id",
		},
		{
			name:     "create with all options",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"model_id":       "mistral-7b",
				"resource_class": "large",
				"replicas":       3,
				"persistent":     true,
			},
			wantErr:    false,
			checkField: "service_id",
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"model_id": "llama3-70b"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"model_id": "llama3-70b"},
			wantErr:  true,
		},
		{
			name:     "missing model_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    NewMemoryStore(),
			provider: &MockProvider{createErr: errors.New("create failed")},
			input:    map[string]any{"model_id": "llama3-70b"},
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
			cmd := NewCreateCommand(tt.store, tt.provider)
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

func TestDeleteCommand_Name(t *testing.T) {
	cmd := NewDeleteCommand(nil, nil)
	if cmd.Name() != "service.delete" {
		t.Errorf("expected name 'service.delete', got '%s'", cmd.Name())
	}
}

func TestDeleteCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ServiceStore
		provider ServiceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful delete",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "missing service_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "service not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "nonexistent"},
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
			cmd := NewDeleteCommand(tt.store, tt.provider)
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

func TestScaleCommand_Name(t *testing.T) {
	cmd := NewScaleCommand(nil, nil)
	if cmd.Name() != "service.scale" {
		t.Errorf("expected name 'service.scale', got '%s'", cmd.Name())
	}
}

func TestScaleCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ServiceStore
		provider ServiceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful scale up",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123", "replicas": 5},
			wantErr:  false,
		},
		{
			name:     "successful scale down",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123", "replicas": 1},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123", "replicas": 5},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"service_id": "svc-123", "replicas": 5},
			wantErr:  true,
		},
		{
			name:     "missing service_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"replicas": 5},
			wantErr:  true,
		},
		{
			name:     "missing replicas",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "service not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "nonexistent", "replicas": 5},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{scaleErr: errors.New("scale failed")},
			input:    map[string]any{"service_id": "svc-123", "replicas": 5},
			wantErr:  true,
		},
		{
			name:     "negative replicas",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123", "replicas": -1},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewScaleCommand(tt.store, tt.provider)
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
	if cmd.Name() != "service.start" {
		t.Errorf("expected name 'service.start', got '%s'", cmd.Name())
	}
}

func TestStartCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ServiceStore
		provider ServiceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful start",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusStopped),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "missing service_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "service not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "nonexistent"},
			wantErr:  true,
		},
		{
			name:     "service already running",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusStopped),
			provider: &MockProvider{startErr: errors.New("start failed")},
			input:    map[string]any{"service_id": "svc-123"},
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

// TestStartCommand_FailedStatusTransition verifies Bug #24: when Start fails,
// service transitions to "failed" instead of staying at "creating".
func TestStartCommand_FailedStatusTransition(t *testing.T) {
	store := createStoreWithService("svc-123", "model-1", ServiceStatusCreating)
	provider := &MockProvider{startErr: errors.New("docker container exited")}
	cmd := NewStartCommand(store, provider)

	_, err := cmd.Execute(context.Background(), map[string]any{"service_id": "svc-123"})
	if err == nil {
		t.Fatal("expected error from failed start")
	}

	// Verify service status transitioned to "failed"
	svc, getErr := store.Get(context.Background(), "svc-123")
	if getErr != nil {
		t.Fatalf("failed to get service: %v", getErr)
	}
	if svc.Status != ServiceStatusFailed {
		t.Errorf("expected service status %q, got %q", ServiceStatusFailed, svc.Status)
	}
}

func TestStopCommand_Name(t *testing.T) {
	cmd := NewStopCommand(nil, nil)
	if cmd.Name() != "service.stop" {
		t.Errorf("expected name 'service.stop', got '%s'", cmd.Name())
	}
}

func TestStopCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ServiceStore
		provider ServiceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful stop",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "stop with force",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123", "force": true},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "missing service_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "service not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "nonexistent"},
			wantErr:  true,
		},
		{
			name:     "service already stopped",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusStopped),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "provider error",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{stopErr: errors.New("stop failed")},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "stop creating service succeeds",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusCreating),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "stop failed service succeeds",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusFailed),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "stop creating service ignores provider error",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusCreating),
			provider: &MockProvider{stopErr: errors.New("nothing to stop")},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "stop failed service ignores provider error",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusFailed),
			provider: &MockProvider{stopErr: errors.New("nothing to stop")},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
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

// TestStopCommand_StatusTransitions verifies Bug #25: stop on non-running
// services transitions them to "stopped" in the store.
func TestStopCommand_StatusTransitions(t *testing.T) {
	for _, fromStatus := range []ServiceStatus{ServiceStatusCreating, ServiceStatusFailed, ServiceStatusRunning} {
		t.Run(string(fromStatus), func(t *testing.T) {
			store := createStoreWithService("svc-123", "model-1", fromStatus)
			cmd := NewStopCommand(store, &MockProvider{})

			result, err := cmd.Execute(context.Background(), map[string]any{"service_id": "svc-123"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resultMap := result.(map[string]any)
			if !resultMap["success"].(bool) {
				t.Fatal("expected success=true")
			}

			svc, getErr := store.Get(context.Background(), "svc-123")
			if getErr != nil {
				t.Fatalf("failed to get service: %v", getErr)
			}
			if svc.Status != ServiceStatusStopped {
				t.Errorf("expected status %q after stop, got %q", ServiceStatusStopped, svc.Status)
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	if NewCreateCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for CreateCommand")
	}
	if NewDeleteCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for DeleteCommand")
	}
	if NewScaleCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for ScaleCommand")
	}
	if NewStartCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for StartCommand")
	}
	if NewStopCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for StopCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewCreateCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for CreateCommand")
	}
	if len(NewDeleteCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for DeleteCommand")
	}
	if len(NewScaleCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for ScaleCommand")
	}
	if len(NewStartCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for StartCommand")
	}
	if len(NewStopCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for StopCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewCreateCommand(nil, nil)
	var _ unit.Command = NewDeleteCommand(nil, nil)
	var _ unit.Command = NewScaleCommand(nil, nil)
	var _ unit.Command = NewStartCommand(nil, nil)
	var _ unit.Command = NewStopCommand(nil, nil)
}

func createStoreWithService(id string, modelID string, status ServiceStatus) ServiceStore {
	store := NewMemoryStore()
	service := createTestService(id, modelID, status)
	_ = store.Create(context.Background(), service)
	return store
}
