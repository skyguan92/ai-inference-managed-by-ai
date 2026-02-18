package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
)

func TestHandleStream_Validation(t *testing.T) {
	registry := unit.NewRegistry()
	gateway := NewGateway(registry)

	tests := []struct {
		name    string
		req     *Request
		wantErr bool
		errCode string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
		{
			name: "query type not supported",
			req: &Request{
				Type: TypeQuery,
				Unit: "test.query",
				Input: map[string]any{
					"stream": true,
				},
			},
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
		{
			name: "command not found",
			req: &Request{
				Type: TypeCommand,
				Unit: "nonexistent.command",
				Input: map[string]any{
					"stream": true,
				},
			},
			wantErr: true,
			errCode: ErrCodeUnitNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gateway.HandleStream(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				errInfo, ok := err.(*ErrorInfo)
				if !ok {
					t.Errorf("expected ErrorInfo, got %T", err)
					return
				}
				if errInfo.Code != tt.errCode {
					t.Errorf("expected error code %s, got %s", tt.errCode, errInfo.Code)
				}
			}
		})
	}
}

func TestHandleStream_StreamingCommand(t *testing.T) {
	registry := unit.NewRegistry()
	provider := inference.NewMockProvider()
	chatCmd := inference.NewChatCommand(provider)
	registry.RegisterCommand(chatCmd)

	gateway := NewGateway(registry)

	req := &Request{
		Type: TypeCommand,
		Unit: "inference.chat",
		Input: map[string]any{
			"model": "llama3",
			"messages": []any{
				map[string]any{"role": "user", "content": "Hello"},
			},
			"stream": true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := gateway.HandleStream(ctx, req)
	if err != nil {
		t.Fatalf("HandleStream() error = %v", err)
	}

	var receivedChunks int
	var done bool
	for resp := range stream {
		receivedChunks++
		if resp.Done {
			done = true
			break
		}
		if resp.Error != nil {
			t.Errorf("unexpected error in stream: %v", resp.Error)
			return
		}
	}

	if !done {
		t.Error("expected stream to end with Done=true")
	}

	if receivedChunks == 0 {
		t.Error("expected to receive chunks")
	}
}

func TestHandleStream_NonStreamingCommand(t *testing.T) {
	registry := unit.NewRegistry()
	// Register a command that doesn't support streaming
	provider := inference.NewMockProvider()
	embedCmd := inference.NewEmbedCommand(provider)
	registry.RegisterCommand(embedCmd)

	gateway := NewGateway(registry)

	req := &Request{
		Type: TypeCommand,
		Unit: "inference.embed",
		Input: map[string]any{
			"model":  "text-embedding-3-small",
			"input":  "Hello",
			"stream": true,
		},
	}

	_, err := gateway.HandleStream(context.Background(), req)
	if err == nil {
		t.Error("expected error for non-streaming command")
		return
	}

	errInfo, ok := err.(*ErrorInfo)
	if !ok {
		t.Errorf("expected ErrorInfo, got %T", err)
		return
	}

	if errInfo.Code != ErrCodeInvalidRequest {
		t.Errorf("expected error code %s, got %s", ErrCodeInvalidRequest, errInfo.Code)
	}
}

func TestIsStreamingRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *Request
		expected bool
	}{
		{
			name:     "nil request",
			req:      nil,
			expected: false,
		},
		{
			name: "no input",
			req: &Request{
				Type: TypeCommand,
				Unit: "test.cmd",
			},
			expected: false,
		},
		{
			name: "stream=false",
			req: &Request{
				Type:  TypeCommand,
				Unit:  "test.cmd",
				Input: map[string]any{"stream": false},
			},
			expected: false,
		},
		{
			name: "stream=true",
			req: &Request{
				Type:  TypeCommand,
				Unit:  "test.cmd",
				Input: map[string]any{"stream": true},
			},
			expected: true,
		},
		{
			name: "stream as string",
			req: &Request{
				Type:  TypeCommand,
				Unit:  "test.cmd",
				Input: map[string]any{"stream": "true"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStreamingRequest(tt.req)
			if result != tt.expected {
				t.Errorf("isStreamingRequest() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFormatSSEData(t *testing.T) {
	tests := []struct {
		name     string
		resp     StreamResponse
		contains string
	}{
		{
			name: "basic content chunk",
			resp: StreamResponse{
				Data: "Hello",
			},
			contains: `"content":"Hello"`,
		},
		{
			name: "chunk with finish_reason",
			resp: StreamResponse{
				Data: "Hello",
				Metadata: map[string]any{
					"finish_reason": "stop",
					"model":         "gpt-4",
				},
			},
			contains: `"finish_reason":"stop"`,
		},
		{
			name:     "empty data",
			resp:     StreamResponse{},
			contains: `"content":null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSSEData(tt.resp)
			if !contains(result, tt.contains) {
				t.Errorf("formatSSEData() = %s, expected to contain %s", result, tt.contains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
