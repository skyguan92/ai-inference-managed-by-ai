package remote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestEnableCommand_Name(t *testing.T) {
	cmd := NewEnableCommand(nil, nil)
	if cmd.Name() != "remote.enable" {
		t.Errorf("expected name 'remote.enable', got '%s'", cmd.Name())
	}
}

func TestEnableCommand_Domain(t *testing.T) {
	cmd := NewEnableCommand(nil, nil)
	if cmd.Domain() != "remote" {
		t.Errorf("expected domain 'remote', got '%s'", cmd.Domain())
	}
}

func TestEnableCommand_Schemas(t *testing.T) {
	cmd := NewEnableCommand(nil, nil)

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

func TestEnableCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      RemoteStore
		provider   RemoteProvider
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:     "successful enable",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"provider": "cloudflare",
			},
			wantErr:    false,
			checkField: "tunnel_id",
		},
		{
			name:     "enable with config",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"provider": "frp",
				"config": map[string]any{
					"server":     "frp.example.com:7000",
					"expose_api": true,
					"expose_mcp": true,
				},
			},
			wantErr:    false,
			checkField: "tunnel_id",
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"provider": "cloudflare"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"provider": "cloudflare"},
			wantErr:  true,
		},
		{
			name:     "missing provider",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    NewMemoryStore(),
			provider: &MockProvider{enableErr: errors.New("enable failed")},
			input:    map[string]any{"provider": "cloudflare"},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    "invalid",
			wantErr:  true,
		},
		{
			name:     "tunnel already enabled",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{},
			input:    map[string]any{"provider": "cloudflare"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewEnableCommand(tt.store, tt.provider)
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

func TestDisableCommand_Name(t *testing.T) {
	cmd := NewDisableCommand(nil, nil)
	if cmd.Name() != "remote.disable" {
		t.Errorf("expected name 'remote.disable', got '%s'", cmd.Name())
	}
}

func TestDisableCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    RemoteStore
		provider RemoteProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful disable",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "no tunnel",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{disableErr: errors.New("disable failed")},
			input:    map[string]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewDisableCommand(tt.store, tt.provider)
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

func TestExecCommand_Name(t *testing.T) {
	cmd := NewExecCommand(nil, nil)
	if cmd.Name() != "remote.exec" {
		t.Errorf("expected name 'remote.exec', got '%s'", cmd.Name())
	}
}

func TestExecCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    RemoteStore
		provider RemoteProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful exec",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{},
			input:    map[string]any{"command": "ls -la"},
			wantErr:  false,
		},
		{
			name:     "exec with timeout",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{},
			input:    map[string]any{"command": "sleep 5", "timeout": 10},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"command": "ls"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"command": "ls"},
			wantErr:  true,
		},
		{
			name:     "missing command",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "no tunnel",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"command": "ls"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{execErr: errors.New("exec failed")},
			input:    map[string]any{"command": "ls"},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			store:    createStoreWithTunnel(),
			provider: &MockProvider{},
			input:    "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewExecCommand(tt.store, tt.provider)
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

			if _, exists := resultMap["stdout"]; !exists {
				t.Error("expected field 'stdout' not found")
			}
			if _, exists := resultMap["exit_code"]; !exists {
				t.Error("expected field 'exit_code' not found")
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	if NewEnableCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for EnableCommand")
	}
	if NewDisableCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for DisableCommand")
	}
	if NewExecCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for ExecCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewEnableCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for EnableCommand")
	}
	if len(NewDisableCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for DisableCommand")
	}
	if len(NewExecCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for ExecCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewEnableCommand(nil, nil)
	var _ unit.Command = NewDisableCommand(nil, nil)
	var _ unit.Command = NewExecCommand(nil, nil)
}

func createStoreWithTunnel() RemoteStore {
	store := NewMemoryStore()
	store.SetTunnel(context.Background(), &TunnelInfo{
		ID:        "tunnel-test",
		Status:    TunnelStatusConnected,
		Provider:  TunnelProviderCloudflare,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	})
	return store
}
