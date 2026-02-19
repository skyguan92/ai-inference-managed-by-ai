package inference

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestChatCommand_SupportsStreaming(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewChatCommand(provider)

	if !cmd.SupportsStreaming() {
		t.Error("ChatCommand should support streaming")
	}
}

func TestChatCommand_ExecuteStream(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewChatCommand(provider)

	input := map[string]any{
		"model": "llama3",
		"messages": []any{
			map[string]any{"role": "user", "content": "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := make(chan unit.StreamChunk, 10)

	go func() {
		defer close(stream)
		err := cmd.ExecuteStream(ctx, input, stream)
		if err != nil {
			t.Errorf("ExecuteStream failed: %v", err)
		}
	}()

	var receivedChunks int
	for chunk := range stream {
		receivedChunks++
		if chunk.Type != "content" {
			t.Errorf("expected type 'content', got %s", chunk.Type)
		}
		if chunk.Data == nil {
			t.Error("expected non-nil data")
		}
	}

	if receivedChunks == 0 {
		t.Error("expected to receive chunks, got none")
	}
}

func TestChatCommand_ExecuteStream_NilProvider(t *testing.T) {
	cmd := NewChatCommand(nil)

	input := map[string]any{
		"model": "llama3",
		"messages": []any{
			map[string]any{"role": "user", "content": "Hello"},
		},
	}

	stream := make(chan unit.StreamChunk, 10)
	defer close(stream)

	err := cmd.ExecuteStream(context.Background(), input, stream)
	if err != ErrProviderNotSet {
		t.Errorf("expected ErrProviderNotSet, got %v", err)
	}
}

func TestChatCommand_ExecuteStream_MissingModel(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewChatCommand(provider)

	input := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "Hello"},
		},
	}

	stream := make(chan unit.StreamChunk, 10)
	defer close(stream)

	err := cmd.ExecuteStream(context.Background(), input, stream)
	if err != ErrModelNotSpecified {
		t.Errorf("expected ErrModelNotSpecified, got %v", err)
	}
}

func TestChatCommand_ExecuteStream_InvalidInput(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewChatCommand(provider)

	stream := make(chan unit.StreamChunk, 10)
	defer close(stream)

	err := cmd.ExecuteStream(context.Background(), "invalid input", stream)
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestCompleteCommand_SupportsStreaming(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewCompleteCommand(provider)

	if !cmd.SupportsStreaming() {
		t.Error("CompleteCommand should support streaming")
	}
}

func TestCompleteCommand_ExecuteStream(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewCompleteCommand(provider)

	input := map[string]any{
		"model":  "llama3",
		"prompt": "Once upon a time",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := make(chan unit.StreamChunk, 10)

	go func() {
		defer close(stream)
		err := cmd.ExecuteStream(ctx, input, stream)
		if err != nil {
			t.Errorf("ExecuteStream failed: %v", err)
		}
	}()

	var receivedChunks int
	for chunk := range stream {
		receivedChunks++
		if chunk.Type != "content" {
			t.Errorf("expected type 'content', got %s", chunk.Type)
		}
		if chunk.Data == nil {
			t.Error("expected non-nil data")
		}
	}

	if receivedChunks == 0 {
		t.Error("expected to receive chunks, got none")
	}
}

func TestCompleteCommand_ExecuteStream_NilProvider(t *testing.T) {
	cmd := NewCompleteCommand(nil)

	input := map[string]any{
		"model":  "llama3",
		"prompt": "Once upon a time",
	}

	stream := make(chan unit.StreamChunk, 10)
	defer close(stream)

	err := cmd.ExecuteStream(context.Background(), input, stream)
	if err != ErrProviderNotSet {
		t.Errorf("expected ErrProviderNotSet, got %v", err)
	}
}

func TestCompleteCommand_ExecuteStream_MissingModel(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewCompleteCommand(provider)

	input := map[string]any{
		"prompt": "Once upon a time",
	}

	stream := make(chan unit.StreamChunk, 10)
	defer close(stream)

	err := cmd.ExecuteStream(context.Background(), input, stream)
	if err != ErrModelNotSpecified {
		t.Errorf("expected ErrModelNotSpecified, got %v", err)
	}
}

func TestCompleteCommand_ExecuteStream_MissingPrompt(t *testing.T) {
	provider := NewMockProvider()
	cmd := NewCompleteCommand(provider)

	input := map[string]any{
		"model": "llama3",
	}

	stream := make(chan unit.StreamChunk, 10)
	defer close(stream)

	err := cmd.ExecuteStream(context.Background(), input, stream)
	if err == nil {
		t.Error("expected error for missing prompt")
	}
}

func TestMockProvider_ChatStream(t *testing.T) {
	provider := NewMockProvider()

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	opts := ChatOptions{Stream: true}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := make(chan ChatStreamChunk, 10)

	go func() {
		defer close(stream)
		err := provider.ChatStream(ctx, "llama3", messages, opts, stream)
		if err != nil {
			t.Errorf("ChatStream failed: %v", err)
		}
	}()

	var receivedChunks int
	var finalChunk *ChatStreamChunk

	for chunk := range stream {
		receivedChunks++
		if chunk.FinishReason != "" {
			finalChunk = &chunk
		}
	}

	if receivedChunks == 0 {
		t.Error("expected to receive chunks, got none")
	}

	if finalChunk == nil {
		t.Error("expected final chunk with finish_reason")
	} else {
		if finalChunk.FinishReason != "stop" {
			t.Errorf("expected finish_reason 'stop', got %s", finalChunk.FinishReason)
		}
		if finalChunk.Usage == nil {
			t.Error("expected usage in final chunk")
		}
	}
}

func TestMockProvider_CompleteStream(t *testing.T) {
	provider := NewMockProvider()

	opts := CompleteOptions{Stream: true}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := make(chan CompleteStreamChunk, 10)

	go func() {
		defer close(stream)
		err := provider.CompleteStream(ctx, "llama3", "Once upon a time", opts, stream)
		if err != nil {
			t.Errorf("CompleteStream failed: %v", err)
		}
	}()

	var receivedChunks int
	var finalChunk *CompleteStreamChunk

	for chunk := range stream {
		receivedChunks++
		if chunk.FinishReason != "" {
			finalChunk = &chunk
		}
	}

	if receivedChunks == 0 {
		t.Error("expected to receive chunks, got none")
	}

	if finalChunk == nil {
		t.Error("expected final chunk with finish_reason")
	} else {
		if finalChunk.FinishReason != "stop" {
			t.Errorf("expected finish_reason 'stop', got %s", finalChunk.FinishReason)
		}
		if finalChunk.Usage == nil {
			t.Error("expected usage in final chunk")
		}
	}
}

func TestMockProvider_ChatStream_ContextCancel(t *testing.T) {
	provider := NewMockProvider()

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	opts := ChatOptions{Stream: true}

	ctx, cancel := context.WithCancel(context.Background())
	stream := make(chan ChatStreamChunk, 10)

	go func() {
		defer close(stream)
		provider.ChatStream(ctx, "llama3", messages, opts, stream)
	}()

	// Cancel context immediately
	cancel()

	// Give some time for the goroutine to process
	time.Sleep(50 * time.Millisecond)

	// Should not panic or hang
	for range stream {
		// Drain channel
	}
}
