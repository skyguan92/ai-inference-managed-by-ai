package inference

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestChatCommand_Name(t *testing.T) {
	cmd := NewChatCommand(nil)
	if cmd.Name() != "inference.chat" {
		t.Errorf("expected name 'inference.chat', got '%s'", cmd.Name())
	}
}

func TestChatCommand_Domain(t *testing.T) {
	cmd := NewChatCommand(nil)
	if cmd.Domain() != "inference" {
		t.Errorf("expected domain 'inference', got '%s'", cmd.Domain())
	}
}

func TestChatCommand_Schemas(t *testing.T) {
	cmd := NewChatCommand(nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestChatCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful chat",
			provider: NewMockProvider(),
			input: map[string]any{
				"model": "llama3",
				"messages": []any{
					map[string]any{"role": "user", "content": "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name:     "successful chat with options",
			provider: NewMockProvider(),
			input: map[string]any{
				"model":       "gpt-4",
				"messages":    []any{map[string]any{"role": "user", "content": "Hi"}},
				"temperature": 0.7,
				"max_tokens":  100,
			},
			wantErr: false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "llama3", "messages": []any{map[string]any{"role": "user", "content": "Hi"}}},
			wantErr:  true,
		},
		{
			name:     "missing model",
			provider: NewMockProvider(),
			input:    map[string]any{"messages": []any{map[string]any{"role": "user", "content": "Hi"}}},
			wantErr:  true,
		},
		{
			name:     "missing messages",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "llama3"},
			wantErr:  true,
		},
		{
			name:     "empty messages",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "llama3", "messages": []any{}},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			provider: NewMockProvider(),
			input:    "invalid",
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{chatErr: errors.New("chat failed")},
			input:    map[string]any{"model": "llama3", "messages": []any{map[string]any{"role": "user", "content": "Hi"}}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewChatCommand(tt.provider)
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

			if _, ok := resultMap["content"]; !ok {
				t.Error("expected 'content' field")
			}
		})
	}
}

func TestCompleteCommand_Name(t *testing.T) {
	cmd := NewCompleteCommand(nil)
	if cmd.Name() != "inference.complete" {
		t.Errorf("expected name 'inference.complete', got '%s'", cmd.Name())
	}
}

func TestCompleteCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful completion",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "llama3", "prompt": "Once upon a time"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "llama3", "prompt": "test"},
			wantErr:  true,
		},
		{
			name:     "missing model",
			provider: NewMockProvider(),
			input:    map[string]any{"prompt": "test"},
			wantErr:  true,
		},
		{
			name:     "missing prompt",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "llama3"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{completeErr: errors.New("complete failed")},
			input:    map[string]any{"model": "llama3", "prompt": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCompleteCommand(tt.provider)
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

			if _, ok := resultMap["text"]; !ok {
				t.Error("expected 'text' field")
			}
		})
	}
}

func TestEmbedCommand_Name(t *testing.T) {
	cmd := NewEmbedCommand(nil)
	if cmd.Name() != "inference.embed" {
		t.Errorf("expected name 'inference.embed', got '%s'", cmd.Name())
	}
}

func TestEmbedCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "single text embedding",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "text-embedding-3-small", "input": "Hello world"},
			wantErr:  false,
		},
		{
			name:     "array of texts embedding",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "text-embedding-3-small", "input": []string{"Hello", "World"}},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "text-embedding-3-small", "input": "test"},
			wantErr:  true,
		},
		{
			name:     "missing model",
			provider: NewMockProvider(),
			input:    map[string]any{"input": "test"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{embedErr: errors.New("embed failed")},
			input:    map[string]any{"model": "text-embedding-3-small", "input": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewEmbedCommand(tt.provider)
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

			if _, ok := resultMap["embeddings"]; !ok {
				t.Error("expected 'embeddings' field")
			}
		})
	}
}

func TestTranscribeCommand_Name(t *testing.T) {
	cmd := NewTranscribeCommand(nil)
	if cmd.Name() != "inference.transcribe" {
		t.Errorf("expected name 'inference.transcribe', got '%s'", cmd.Name())
	}
}

func TestTranscribeCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful transcription",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "whisper-large-v3", "audio": "base64_audio_data"},
			wantErr:  false,
		},
		{
			name:     "with language",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "whisper-large-v3", "audio": "base64_audio", "language": "zh"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "whisper-large-v3", "audio": "test"},
			wantErr:  true,
		},
		{
			name:     "missing audio",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "whisper-large-v3"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{transcribeErr: errors.New("transcribe failed")},
			input:    map[string]any{"model": "whisper-large-v3", "audio": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewTranscribeCommand(tt.provider)
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

			if _, ok := resultMap["text"]; !ok {
				t.Error("expected 'text' field")
			}
		})
	}
}

func TestSynthesizeCommand_Name(t *testing.T) {
	cmd := NewSynthesizeCommand(nil)
	if cmd.Name() != "inference.synthesize" {
		t.Errorf("expected name 'inference.synthesize', got '%s'", cmd.Name())
	}
}

func TestSynthesizeCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful synthesis",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "tts-1", "text": "Hello world"},
			wantErr:  false,
		},
		{
			name:     "with voice",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "tts-1", "text": "Hello", "voice": "alloy"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "tts-1", "text": "test"},
			wantErr:  true,
		},
		{
			name:     "missing text",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "tts-1"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{synthesizeErr: errors.New("synthesize failed")},
			input:    map[string]any{"model": "tts-1", "text": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSynthesizeCommand(tt.provider)
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

			if _, ok := resultMap["audio"]; !ok {
				t.Error("expected 'audio' field")
			}
		})
	}
}

func TestGenerateImageCommand_Name(t *testing.T) {
	cmd := NewGenerateImageCommand(nil)
	if cmd.Name() != "inference.generate_image" {
		t.Errorf("expected name 'inference.generate_image', got '%s'", cmd.Name())
	}
}

func TestGenerateImageCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful image generation",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "dall-e-3", "prompt": "A cat on the moon"},
			wantErr:  false,
		},
		{
			name:     "with options",
			provider: NewMockProvider(),
			input: map[string]any{
				"model":  "stable-diffusion-xl",
				"prompt": "A sunset",
				"size":   "1024x1024",
				"steps":  50,
			},
			wantErr: false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "dall-e-3", "prompt": "test"},
			wantErr:  true,
		},
		{
			name:     "missing prompt",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "dall-e-3"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{genImageErr: errors.New("generate failed")},
			input:    map[string]any{"model": "dall-e-3", "prompt": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateImageCommand(tt.provider)
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

			if _, ok := resultMap["images"]; !ok {
				t.Error("expected 'images' field")
			}
		})
	}
}

func TestGenerateVideoCommand_Name(t *testing.T) {
	cmd := NewGenerateVideoCommand(nil)
	if cmd.Name() != "inference.generate_video" {
		t.Errorf("expected name 'inference.generate_video', got '%s'", cmd.Name())
	}
}

func TestGenerateVideoCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful video generation",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "video-gen-1", "prompt": "A sunset over the ocean"},
			wantErr:  false,
		},
		{
			name:     "with duration",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "video-gen-1", "prompt": "A sunset", "duration": 10},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "video-gen-1", "prompt": "test"},
			wantErr:  true,
		},
		{
			name:     "missing prompt",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "video-gen-1"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{genVideoErr: errors.New("generate failed")},
			input:    map[string]any{"model": "video-gen-1", "prompt": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGenerateVideoCommand(tt.provider)
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

			if _, ok := resultMap["video"]; !ok {
				t.Error("expected 'video' field")
			}
		})
	}
}

func TestRerankCommand_Name(t *testing.T) {
	cmd := NewRerankCommand(nil)
	if cmd.Name() != "inference.rerank" {
		t.Errorf("expected name 'inference.rerank', got '%s'", cmd.Name())
	}
}

func TestRerankCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful rerank",
			provider: NewMockProvider(),
			input: map[string]any{
				"model":     "rerank-1",
				"query":     "What is AI?",
				"documents": []any{"AI is technology", "Dogs are animals"},
			},
			wantErr: false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "rerank-1", "query": "test", "documents": []any{"doc1"}},
			wantErr:  true,
		},
		{
			name:     "missing query",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "rerank-1", "documents": []any{"doc1"}},
			wantErr:  true,
		},
		{
			name:     "empty documents",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "rerank-1", "query": "test", "documents": []any{}},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{rerankErr: errors.New("rerank failed")},
			input:    map[string]any{"model": "rerank-1", "query": "test", "documents": []any{"doc1"}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRerankCommand(tt.provider)
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

			if _, ok := resultMap["results"]; !ok {
				t.Error("expected 'results' field")
			}
		})
	}
}

func TestDetectCommand_Name(t *testing.T) {
	cmd := NewDetectCommand(nil)
	if cmd.Name() != "inference.detect" {
		t.Errorf("expected name 'inference.detect', got '%s'", cmd.Name())
	}
}

func TestDetectCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful detection",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "yolov8", "image": "base64_image_data"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model": "yolov8", "image": "test"},
			wantErr:  true,
		},
		{
			name:     "missing image",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "yolov8"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{detectErr: errors.New("detect failed")},
			input:    map[string]any{"model": "yolov8", "image": "test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewDetectCommand(tt.provider)
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

			if _, ok := resultMap["detections"]; !ok {
				t.Error("expected 'detections' field")
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	if NewChatCommand(nil).Description() == "" {
		t.Error("expected non-empty description for ChatCommand")
	}
	if NewCompleteCommand(nil).Description() == "" {
		t.Error("expected non-empty description for CompleteCommand")
	}
	if NewEmbedCommand(nil).Description() == "" {
		t.Error("expected non-empty description for EmbedCommand")
	}
	if NewTranscribeCommand(nil).Description() == "" {
		t.Error("expected non-empty description for TranscribeCommand")
	}
	if NewSynthesizeCommand(nil).Description() == "" {
		t.Error("expected non-empty description for SynthesizeCommand")
	}
	if NewGenerateImageCommand(nil).Description() == "" {
		t.Error("expected non-empty description for GenerateImageCommand")
	}
	if NewGenerateVideoCommand(nil).Description() == "" {
		t.Error("expected non-empty description for GenerateVideoCommand")
	}
	if NewRerankCommand(nil).Description() == "" {
		t.Error("expected non-empty description for RerankCommand")
	}
	if NewDetectCommand(nil).Description() == "" {
		t.Error("expected non-empty description for DetectCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewChatCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for ChatCommand")
	}
	if len(NewCompleteCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for CompleteCommand")
	}
	if len(NewEmbedCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for EmbedCommand")
	}
	if len(NewTranscribeCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for TranscribeCommand")
	}
	if len(NewSynthesizeCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for SynthesizeCommand")
	}
	if len(NewGenerateImageCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for GenerateImageCommand")
	}
	if len(NewGenerateVideoCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for GenerateVideoCommand")
	}
	if len(NewRerankCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for RerankCommand")
	}
	if len(NewDetectCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for DetectCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewChatCommand(nil)
	var _ unit.Command = NewCompleteCommand(nil)
	var _ unit.Command = NewEmbedCommand(nil)
	var _ unit.Command = NewTranscribeCommand(nil)
	var _ unit.Command = NewSynthesizeCommand(nil)
	var _ unit.Command = NewGenerateImageCommand(nil)
	var _ unit.Command = NewGenerateVideoCommand(nil)
	var _ unit.Command = NewRerankCommand(nil)
	var _ unit.Command = NewDetectCommand(nil)
}
