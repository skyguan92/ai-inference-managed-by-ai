package gateway

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPAdapter_GetPrompts(t *testing.T) {
	adapter := NewMCPAdapter(nil)
	prompts := adapter.GetPrompts()

	if len(prompts) != 5 {
		t.Errorf("expected 5 prompts, got %d", len(prompts))
	}

	expectedNames := map[string]bool{
		"model_management":    false,
		"inference_assistant": false,
		"resource_optimizer":  false,
		"troubleshooting":     false,
		"pipeline_builder":    false,
	}

	for _, p := range prompts {
		if _, exists := expectedNames[p.Name]; !exists {
			t.Errorf("unexpected prompt name: %s", p.Name)
		}
		expectedNames[p.Name] = true

		if p.Description == "" {
			t.Errorf("prompt %s has no description", p.Name)
		}

		if p.Template == "" {
			t.Errorf("prompt %s has no template", p.Name)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected prompt %s not found", name)
		}
	}
}

func TestMCPAdapter_handlePromptsList(t *testing.T) {
	adapter := NewMCPAdapter(nil)
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-1",
		Method:  "prompts/list",
	}

	resp := adapter.handlePromptsList(context.Background(), req)

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(*MCPPromptsListResult)
	if !ok {
		t.Fatal("expected *MCPPromptsListResult")
	}

	if len(result.Prompts) != 5 {
		t.Errorf("expected 5 prompts, got %d", len(result.Prompts))
	}

	for _, p := range result.Prompts {
		if p.Template != "" {
			t.Errorf("prompt %s should not expose template in list", p.Name)
		}
	}
}

func TestMCPAdapter_handlePromptsGet(t *testing.T) {
	adapter := NewMCPAdapter(nil)

	tests := []struct {
		name       string
		params     map[string]any
		wantError  bool
		errorCode  int
		checkText  string
		shouldHave string
	}{
		{
			name:       "get model_management prompt",
			params:     map[string]any{"name": "model_management"},
			wantError:  false,
			checkText:  "模型管理助手",
			shouldHave: "model.pull",
		},
		{
			name:       "get inference_assistant prompt",
			params:     map[string]any{"name": "inference_assistant"},
			wantError:  false,
			checkText:  "推理助手",
			shouldHave: "inference.chat",
		},
		{
			name:       "get resource_optimizer prompt",
			params:     map[string]any{"name": "resource_optimizer"},
			wantError:  false,
			checkText:  "资源优化助手",
			shouldHave: "resource.status",
		},
		{
			name:      "missing name parameter",
			params:    map[string]any{},
			wantError: true,
			errorCode: MCPErrorCodeInvalidParams,
		},
		{
			name:      "non-existent prompt",
			params:    map[string]any{"name": "not_exist"},
			wantError: true,
			errorCode: MCPErrorCodeInvalidParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramsJSON, _ := json.Marshal(tt.params)
			req := &MCPRequest{
				JSONRPC: "2.0",
				ID:      "test-1",
				Method:  "prompts/get",
				Params:  paramsJSON,
			}

			resp := adapter.handlePromptsGet(context.Background(), req)

			if tt.wantError {
				if resp.Error == nil {
					t.Error("expected error, got nil")
					return
				}
				if resp.Error.Code != tt.errorCode {
					t.Errorf("expected error code %d, got %d", tt.errorCode, resp.Error.Code)
				}
				return
			}

			if resp.Error != nil {
				t.Errorf("unexpected error: %v", resp.Error)
				return
			}

			result, ok := resp.Result.(*MCPPromptGetResult)
			if !ok {
				t.Fatal("expected *MCPPromptGetResult")
			}

			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
			}

			text := result.Messages[0].Content.Text
			if !promptContains(text, tt.checkText) {
				t.Errorf("expected text to contain %q, got:\n%s", tt.checkText, text)
			}

			if tt.shouldHave != "" && !promptContains(text, tt.shouldHave) {
				t.Errorf("expected text to contain %q, got:\n%s", tt.shouldHave, text)
			}
		})
	}
}

func TestMCPAdapter_handlePromptsGet_WithArgs(t *testing.T) {
	adapter := NewMCPAdapter(nil)

	tests := []struct {
		name          string
		promptName    string
		args          map[string]string
		shouldHave    string
		shouldNotHave string
	}{
		{
			name:       "troubleshooting with issue_type",
			promptName: "troubleshooting",
			args:       map[string]string{"issue_type": "GPU_OOM"},
			shouldHave: "当前关注问题类型: GPU_OOM",
		},
		{
			name:          "troubleshooting without issue_type",
			promptName:    "troubleshooting",
			args:          map[string]string{},
			shouldNotHave: "当前关注问题类型",
		},
		{
			name:       "pipeline_builder with pipeline_type",
			promptName: "pipeline_builder",
			args:       map[string]string{"pipeline_type": "rag"},
			shouldHave: "当前选择管道: rag",
		},
		{
			name:          "pipeline_builder without pipeline_type",
			promptName:    "pipeline_builder",
			args:          map[string]string{},
			shouldNotHave: "当前选择管道",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{
				"name":      tt.promptName,
				"arguments": tt.args,
			}
			paramsJSON, _ := json.Marshal(params)
			req := &MCPRequest{
				JSONRPC: "2.0",
				ID:      "test-1",
				Method:  "prompts/get",
				Params:  paramsJSON,
			}

			resp := adapter.handlePromptsGet(context.Background(), req)

			if resp.Error != nil {
				t.Errorf("unexpected error: %v", resp.Error)
				return
			}

			result := resp.Result.(*MCPPromptGetResult)
			text := result.Messages[0].Content.Text

			if tt.shouldHave != "" && !promptContains(text, tt.shouldHave) {
				t.Errorf("expected text to contain %q, got:\n%s", tt.shouldHave, text)
			}

			if tt.shouldNotHave != "" && promptContains(text, tt.shouldNotHave) {
				t.Errorf("expected text NOT to contain %q, got:\n%s", tt.shouldNotHave, text)
			}
		})
	}
}

func TestMCPAdapter_renderPrompt(t *testing.T) {
	adapter := NewMCPAdapter(nil)

	tests := []struct {
		name     string
		prompt   MCPPrompt
		args     map[string]string
		expected string
		wantErr  bool
	}{
		{
			name: "simple replacement",
			prompt: MCPPrompt{
				Template: "Hello {{.name}}!",
				Arguments: []MCPPromptArgument{
					{Name: "name", Required: false},
				},
			},
			args:     map[string]string{"name": "World"},
			expected: "Hello World!",
		},
		{
			name: "conditional block with value",
			prompt: MCPPrompt{
				Template: "Start{{if .extra}} Extra: {{.extra}}{{end}} End",
				Arguments: []MCPPromptArgument{
					{Name: "extra", Required: false},
				},
			},
			args:     map[string]string{"extra": "data"},
			expected: "Start Extra: data End",
		},
		{
			name: "conditional block without value",
			prompt: MCPPrompt{
				Template: "Start{{if .extra}} Extra: {{.extra}}{{end}} End",
				Arguments: []MCPPromptArgument{
					{Name: "extra", Required: false},
				},
			},
			args:     map[string]string{},
			expected: "Start End",
		},
		{
			name: "missing required arg",
			prompt: MCPPrompt{
				Template: "Hello {{.name}}!",
				Arguments: []MCPPromptArgument{
					{Name: "name", Required: true},
				},
			},
			args:    map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adapter.renderPrompt(&tt.prompt, tt.args)
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
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestMCPAdapter_findPrompt(t *testing.T) {
	adapter := NewMCPAdapter(nil)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"model_management", false},
		{"inference_assistant", false},
		{"resource_optimizer", false},
		{"troubleshooting", false},
		{"pipeline_builder", false},
		{"non_existent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := adapter.findPrompt(tt.name)
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
			if prompt.Name != tt.name {
				t.Errorf("expected prompt name %q, got %q", tt.name, prompt.Name)
			}
		})
	}
}

func TestMCPAdapter_ListPrompts(t *testing.T) {
	adapter := NewMCPAdapter(nil)

	prompts := adapter.ListPrompts()
	if len(prompts) != 5 {
		t.Errorf("expected 5 prompts, got %d", len(prompts))
	}
}

func promptContains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || promptContainsHelper(s, substr))
}

func promptContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
