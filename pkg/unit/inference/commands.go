package inference

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
\t"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type ChatCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewChatCommand(provider InferenceProvider) *ChatCommand {
	return &ChatCommand{provider: provider}
}

func NewChatCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *ChatCommand {
	return &ChatCommand{provider: provider, events: events}
}

func (c *ChatCommand) Name() string {
	return "inference.chat"
}

func (c *ChatCommand) Domain() string {
	return "inference"
}

func (c *ChatCommand) Description() string {
	return "Perform a chat completion with an AI model"
}

func (c *ChatCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model identifier (e.g., llama3, gpt-4)",
				},
			},
			"messages": {
				Name: "messages",
				Schema: unit.Schema{
					Type:        "array",
					Description: "List of chat messages",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"role":    {Name: "role", Schema: unit.Schema{Type: "string", Enum: []any{"system", "user", "assistant"}}},
							"content": {Name: "content", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
			"temperature": {
				Name: "temperature",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Sampling temperature (0-2)",
					Min:         ptrs.Float64(0),
					Max:         ptrs.Float64(2),
				},
			},
			"max_tokens": {
				Name: "max_tokens",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum tokens to generate",
					Min:         ptrs.Float64(1),
				},
			},
			"top_p": {
				Name: "top_p",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Nucleus sampling parameter",
					Min:         ptrs.Float64(0),
					Max:         ptrs.Float64(1),
				},
			},
			"top_k": {
				Name: "top_k",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Top-k sampling parameter",
					Min:         ptrs.Float64(1),
				},
			},
			"frequency_penalty": {
				Name: "frequency_penalty",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Frequency penalty (-2 to 2)",
					Min:         ptrs.Float64(-2),
					Max:         ptrs.Float64(2),
				},
			},
			"presence_penalty": {
				Name: "presence_penalty",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Presence penalty (-2 to 2)",
					Min:         ptrs.Float64(-2),
					Max:         ptrs.Float64(2),
				},
			},
			"stop": {
				Name: "stop",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Stop sequences",
					Items:       &unit.Schema{Type: "string"},
				},
			},
			"stream": {
				Name: "stream",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Enable streaming response",
				},
			},
		},
		Required: []string{"model", "messages"},
	}
}

func (c *ChatCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"content":       {Name: "content", Schema: unit.Schema{Type: "string"}},
			"finish_reason": {Name: "finish_reason", Schema: unit.Schema{Type: "string"}},
			"usage": {
				Name: "usage",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"prompt_tokens":     {Name: "prompt_tokens", Schema: unit.Schema{Type: "number"}},
						"completion_tokens": {Name: "completion_tokens", Schema: unit.Schema{Type: "number"}},
						"total_tokens":      {Name: "total_tokens", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
			"model": {Name: "model", Schema: unit.Schema{Type: "string"}},
			"id":    {Name: "id", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (c *ChatCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"model": "llama3",
				"messages": []map[string]any{
					{"role": "system", "content": "You are a helpful assistant."},
					{"role": "user", "content": "Hello, how are you?"},
				},
			},
			Output: map[string]any{
				"content":       "Hello! I'm doing well, thank you for asking.",
				"finish_reason": "stop",
				"usage":         map[string]any{"prompt_tokens": 20, "completion_tokens": 15, "total_tokens": 35},
			},
			Description: "Simple chat completion",
		},
		{
			Input: map[string]any{
				"model":       "gpt-4",
				"messages":    []map[string]any{{"role": "user", "content": "Write a poem"}},
				"temperature": 0.8,
				"max_tokens":  500,
			},
			Output: map[string]any{
				"content":       "In digital streams of thought...",
				"finish_reason": "stop",
				"usage":         map[string]any{"prompt_tokens": 5, "completion_tokens": 100, "total_tokens": 105},
			},
			Description: "Chat with custom parameters",
		},
	}
}

func (c *ChatCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	msgsRaw, ok := inputMap["messages"].([]any)
	if !ok || len(msgsRaw) == 0 {
		err := fmt.Errorf("messages are required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	messages := make([]Message, len(msgsRaw))
	for i, m := range msgsRaw {
		mMap, ok := m.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid message format: %w", ErrInvalidInput)
		}
		messages[i] = Message{
			Role:    mMap["role"].(string),
			Content: mMap["content"].(string),
		}
	}

	opts := ChatOptions{}
	if v, ok := inputMap["temperature"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.Temperature = &f
		}
	}
	if v, ok := inputMap["max_tokens"]; ok {
		if i, ok := toInt(v); ok {
			opts.MaxTokens = &i
		}
	}
	if v, ok := inputMap["top_p"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.TopP = &f
		}
	}
	if v, ok := inputMap["top_k"]; ok {
		if i, ok := toInt(v); ok {
			opts.TopK = &i
		}
	}
	if v, ok := inputMap["frequency_penalty"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.FrequencyPenalty = &f
		}
	}
	if v, ok := inputMap["presence_penalty"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.PresencePenalty = &f
		}
	}
	if v, ok := inputMap["stop"].([]any); ok {
		opts.Stop = make([]string, len(v))
		for i, s := range v {
			opts.Stop[i] = s.(string)
		}
	}
	if v, ok := inputMap["stream"].(bool); ok {
		opts.Stream = v
	}

	resp, err := c.provider.Chat(ctx, model, messages, opts)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	output := map[string]any{
		"content":       resp.Content,
		"finish_reason": resp.FinishReason,
		"usage": map[string]any{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
		"model": resp.Model,
		"id":    resp.ID,
	}
	ec.PublishCompleted(output)
	return output, nil
}

// SupportsStreaming returns true as chat command supports streaming
func (c *ChatCommand) SupportsStreaming() bool {
	return true
}

// ExecuteStream executes the chat command in streaming mode
func (c *ChatCommand) ExecuteStream(ctx context.Context, input any, stream chan<- unit.StreamChunk) error {
	if c.provider == nil {
		return ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		return ErrModelNotSpecified
	}

	msgsRaw, ok := inputMap["messages"].([]any)
	if !ok || len(msgsRaw) == 0 {
		return fmt.Errorf("messages are required: %w", ErrInvalidInput)
	}

	messages := make([]Message, len(msgsRaw))
	for i, m := range msgsRaw {
		mMap, ok := m.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid message format: %w", ErrInvalidInput)
		}
		messages[i] = Message{
			Role:    mMap["role"].(string),
			Content: mMap["content"].(string),
		}
	}

	opts := ChatOptions{Stream: true}
	if v, ok := inputMap["temperature"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.Temperature = &f
		}
	}
	if v, ok := inputMap["max_tokens"]; ok {
		if i, ok := toInt(v); ok {
			opts.MaxTokens = &i
		}
	}

	// Create internal channel for provider stream
	providerStream := make(chan ChatStreamChunk, 10)
	defer close(providerStream)

	// Run provider stream in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- c.provider.ChatStream(ctx, model, messages, opts, providerStream)
	}()

	// Forward chunks from provider to unit stream
	for {
		select {
		case chunk, ok := <-providerStream:
			if !ok {
				return <-errChan
			}
			stream <- unit.StreamChunk{
				Type: "content",
				Data: chunk.Content,
				Metadata: map[string]any{
					"finish_reason": chunk.FinishReason,
					"model":         chunk.Model,
					"id":            chunk.ID,
				},
			}
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type CompleteCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewCompleteCommand(provider InferenceProvider) *CompleteCommand {
	return &CompleteCommand{provider: provider}
}

func NewCompleteCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *CompleteCommand {
	return &CompleteCommand{provider: provider, events: events}
}

func (c *CompleteCommand) Name() string {
	return "inference.complete"
}

func (c *CompleteCommand) Domain() string {
	return "inference"
}

func (c *CompleteCommand) Description() string {
	return "Perform a text completion with an AI model"
}

func (c *CompleteCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model identifier",
				},
			},
			"prompt": {
				Name: "prompt",
				Schema: unit.Schema{
					Type:        "string",
					Description: "The prompt to complete",
				},
			},
			"temperature": {
				Name: "temperature",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Sampling temperature",
					Min:         ptrs.Float64(0),
					Max:         ptrs.Float64(2),
				},
			},
			"max_tokens": {
				Name: "max_tokens",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum tokens to generate",
				},
			},
			"top_p": {
				Name: "top_p",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Nucleus sampling parameter",
				},
			},
			"stop": {
				Name: "stop",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Stop sequences",
					Items:       &unit.Schema{Type: "string"},
				},
			},
			"stream": {
				Name: "stream",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Enable streaming",
				},
			},
		},
		Required: []string{"model", "prompt"},
	}
}

func (c *CompleteCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"text":          {Name: "text", Schema: unit.Schema{Type: "string"}},
			"finish_reason": {Name: "finish_reason", Schema: unit.Schema{Type: "string"}},
			"usage": {
				Name: "usage",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"prompt_tokens":     {Name: "prompt_tokens", Schema: unit.Schema{Type: "number"}},
						"completion_tokens": {Name: "completion_tokens", Schema: unit.Schema{Type: "number"}},
						"total_tokens":      {Name: "total_tokens", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
		},
	}
}

func (c *CompleteCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "llama3", "prompt": "Once upon a time"},
			Output:      map[string]any{"text": " in a land far away, there lived a brave knight...", "finish_reason": "stop", "usage": map[string]any{"prompt_tokens": 4, "completion_tokens": 20, "total_tokens": 24}},
			Description: "Simple text completion",
		},
	}
}

func (c *CompleteCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	prompt, _ := inputMap["prompt"].(string)
	if prompt == "" {
		err := fmt.Errorf("prompt is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	opts := CompleteOptions{}
	if v, ok := inputMap["temperature"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.Temperature = &f
		}
	}
	if v, ok := inputMap["max_tokens"]; ok {
		if i, ok := toInt(v); ok {
			opts.MaxTokens = &i
		}
	}
	if v, ok := inputMap["top_p"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.TopP = &f
		}
	}
	if v, ok := inputMap["stop"].([]any); ok {
		opts.Stop = make([]string, len(v))
		for i, s := range v {
			opts.Stop[i] = s.(string)
		}
	}
	if v, ok := inputMap["stream"].(bool); ok {
		opts.Stream = v
	}

	resp, err := c.provider.Complete(ctx, model, prompt, opts)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	output := map[string]any{
		"text":          resp.Text,
		"finish_reason": resp.FinishReason,
		"usage": map[string]any{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
	}
	ec.PublishCompleted(output)
	return output, nil
}

// SupportsStreaming returns true as complete command supports streaming
func (c *CompleteCommand) SupportsStreaming() bool {
	return true
}

// ExecuteStream executes the complete command in streaming mode
func (c *CompleteCommand) ExecuteStream(ctx context.Context, input any, stream chan<- unit.StreamChunk) error {
	if c.provider == nil {
		return ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		return ErrModelNotSpecified
	}

	prompt, _ := inputMap["prompt"].(string)
	if prompt == "" {
		return fmt.Errorf("prompt is required: %w", ErrInvalidInput)
	}

	opts := CompleteOptions{Stream: true}
	if v, ok := inputMap["temperature"]; ok {
		if f, ok := toFloat64(v); ok {
			opts.Temperature = &f
		}
	}
	if v, ok := inputMap["max_tokens"]; ok {
		if i, ok := toInt(v); ok {
			opts.MaxTokens = &i
		}
	}

	// Create internal channel for provider stream
	providerStream := make(chan CompleteStreamChunk, 10)
	defer close(providerStream)

	// Run provider stream in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- c.provider.CompleteStream(ctx, model, prompt, opts, providerStream)
	}()

	// Forward chunks from provider to unit stream
	for {
		select {
		case chunk, ok := <-providerStream:
			if !ok {
				return <-errChan
			}
			stream <- unit.StreamChunk{
				Type: "content",
				Data: chunk.Text,
				Metadata: map[string]any{
					"finish_reason": chunk.FinishReason,
					"model":         chunk.Model,
					"id":            chunk.ID,
				},
			}
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type EmbedCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewEmbedCommand(provider InferenceProvider) *EmbedCommand {
	return &EmbedCommand{provider: provider}
}

func NewEmbedCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *EmbedCommand {
	return &EmbedCommand{provider: provider, events: events}
}

func (c *EmbedCommand) Name() string {
	return "inference.embed"
}

func (c *EmbedCommand) Domain() string {
	return "inference"
}

func (c *EmbedCommand) Description() string {
	return "Generate text embeddings using an embedding model"
}

func (c *EmbedCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Embedding model identifier",
				},
			},
			"input": {
				Name: "input",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Text or array of texts to embed",
				},
			},
		},
		Required: []string{"model", "input"},
	}
}

func (c *EmbedCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"embeddings": {
				Name: "embeddings",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Array of embedding vectors",
					Items: &unit.Schema{
						Type:  "array",
						Items: &unit.Schema{Type: "number"},
					},
				},
			},
			"usage": {
				Name: "usage",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"prompt_tokens": {Name: "prompt_tokens", Schema: unit.Schema{Type: "number"}},
						"total_tokens":  {Name: "total_tokens", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
		},
	}
}

func (c *EmbedCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "text-embedding-3-small", "input": "Hello world"},
			Output:      map[string]any{"embeddings": [][]float64{{0.1, 0.2, 0.3}}, "usage": map[string]any{"prompt_tokens": 2, "total_tokens": 2}},
			Description: "Single text embedding",
		},
	}
}

func (c *EmbedCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	var texts []string
	switch v := inputMap["input"].(type) {
	case string:
		texts = []string{v}
	case []any:
		texts = make([]string, len(v))
		for i, s := range v {
			texts[i] = s.(string)
		}
	case []string:
		texts = v
	default:
		err := fmt.Errorf("input must be string or array: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	resp, err := c.provider.Embed(ctx, model, texts)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	output := map[string]any{
		"embeddings": resp.Embeddings,
		"usage": map[string]any{
			"prompt_tokens": resp.Usage.PromptTokens,
			"total_tokens":  resp.Usage.TotalTokens,
		},
	}
	ec.PublishCompleted(output)
	return output, nil
}

type TranscribeCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewTranscribeCommand(provider InferenceProvider) *TranscribeCommand {
	return &TranscribeCommand{provider: provider}
}

func NewTranscribeCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *TranscribeCommand {
	return &TranscribeCommand{provider: provider, events: events}
}

func (c *TranscribeCommand) Name() string {
	return "inference.transcribe"
}

func (c *TranscribeCommand) Domain() string {
	return "inference"
}

func (c *TranscribeCommand) Description() string {
	return "Transcribe audio to text using an ASR model"
}

func (c *TranscribeCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "ASR model identifier (e.g., whisper-large-v3)",
				},
			},
			"audio": {
				Name: "audio",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Audio data (base64 encoded or file path)",
				},
			},
			"language": {
				Name: "language",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Audio language (e.g., en, zh)",
				},
			},
		},
		Required: []string{"model", "audio"},
	}
}

func (c *TranscribeCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"text":     {Name: "text", Schema: unit.Schema{Type: "string"}},
			"language": {Name: "language", Schema: unit.Schema{Type: "string"}},
			"duration": {Name: "duration", Schema: unit.Schema{Type: "number"}},
			"segments": {
				Name: "segments",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "object"},
				},
			},
		},
	}
}

func (c *TranscribeCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "whisper-large-v3", "audio": "base64_audio_data", "language": "en"},
			Output:      map[string]any{"text": "Hello, this is a transcription.", "language": "en", "duration": 3.5},
			Description: "Transcribe English audio",
		},
	}
}

func (c *TranscribeCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	audioRaw, _ := inputMap["audio"].(string)
	if audioRaw == "" {
		err := fmt.Errorf("audio is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	audio := []byte(audioRaw)
	language, _ := inputMap["language"].(string)

	resp, err := c.provider.Transcribe(ctx, model, audio, language)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	segments := make([]map[string]any, len(resp.Segments))
	for i, s := range resp.Segments {
		segments[i] = map[string]any{
			"id":    s.ID,
			"start": s.Start,
			"end":   s.End,
			"text":  s.Text,
		}
	}

	output := map[string]any{
		"text":     resp.Text,
		"language": resp.Language,
		"duration": resp.Duration,
		"segments": segments,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type SynthesizeCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewSynthesizeCommand(provider InferenceProvider) *SynthesizeCommand {
	return &SynthesizeCommand{provider: provider}
}

func NewSynthesizeCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *SynthesizeCommand {
	return &SynthesizeCommand{provider: provider, events: events}
}

func (c *SynthesizeCommand) Name() string {
	return "inference.synthesize"
}

func (c *SynthesizeCommand) Domain() string {
	return "inference"
}

func (c *SynthesizeCommand) Description() string {
	return "Synthesize speech from text using a TTS model"
}

func (c *SynthesizeCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "TTS model identifier",
				},
			},
			"text": {
				Name: "text",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Text to synthesize",
				},
			},
			"voice": {
				Name: "voice",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Voice ID or name",
				},
			},
			"stream": {
				Name: "stream",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Enable streaming output",
				},
			},
		},
		Required: []string{"model", "text"},
	}
}

func (c *SynthesizeCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"audio":    {Name: "audio", Schema: unit.Schema{Type: "string", Description: "Base64 encoded audio"}},
			"format":   {Name: "format", Schema: unit.Schema{Type: "string"}},
			"duration": {Name: "duration", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (c *SynthesizeCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "tts-1", "text": "Hello, world!", "voice": "alloy"},
			Output:      map[string]any{"audio": "base64_audio", "format": "wav", "duration": 1.5},
			Description: "Synthesize speech",
		},
	}
}

func (c *SynthesizeCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	text, _ := inputMap["text"].(string)
	if text == "" {
		err := fmt.Errorf("text is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	voice, _ := inputMap["voice"].(string)

	resp, err := c.provider.Synthesize(ctx, model, text, voice)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	output := map[string]any{
		"audio":    string(resp.Audio),
		"format":   resp.Format,
		"duration": resp.Duration,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type GenerateImageCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewGenerateImageCommand(provider InferenceProvider) *GenerateImageCommand {
	return &GenerateImageCommand{provider: provider}
}

func NewGenerateImageCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *GenerateImageCommand {
	return &GenerateImageCommand{provider: provider, events: events}
}

func (c *GenerateImageCommand) Name() string {
	return "inference.generate_image"
}

func (c *GenerateImageCommand) Domain() string {
	return "inference"
}

func (c *GenerateImageCommand) Description() string {
	return "Generate images from text using a diffusion model"
}

func (c *GenerateImageCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Image generation model (e.g., dall-e-3, stable-diffusion-xl)",
				},
			},
			"prompt": {
				Name: "prompt",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Text description of the image",
				},
			},
			"size": {
				Name: "size",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Image size (e.g., 1024x1024)",
				},
			},
			"steps": {
				Name: "steps",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Number of diffusion steps",
				},
			},
			"width": {
				Name: "width",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Image width in pixels",
				},
			},
			"height": {
				Name: "height",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Image height in pixels",
				},
			},
			"negative_prompt": {
				Name: "negative_prompt",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Negative prompt for what to avoid",
				},
			},
			"seed": {
				Name: "seed",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Random seed for reproducibility",
				},
			},
		},
		Required: []string{"model", "prompt"},
	}
}

func (c *GenerateImageCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"images": {
				Name: "images",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Generated images",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"base64": {Name: "base64", Schema: unit.Schema{Type: "string"}},
							"url":    {Name: "url", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
			"format": {Name: "format", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (c *GenerateImageCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "dall-e-3", "prompt": "A cat sitting on a moon", "size": "1024x1024"},
			Output:      map[string]any{"images": []map[string]any{{"base64": "image_data"}}, "format": "png"},
			Description: "Generate an image",
		},
	}
}

func (c *GenerateImageCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	prompt, _ := inputMap["prompt"].(string)
	if prompt == "" {
		err := fmt.Errorf("prompt is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	opts := ImageOptions{}
	opts.Size, _ = inputMap["size"].(string)
	opts.NegativePrompt, _ = inputMap["negative_prompt"].(string)
	if v, ok := toInt(inputMap["steps"]); ok {
		opts.Steps = v
	}
	if v, ok := toInt(inputMap["width"]); ok {
		opts.Width = v
	}
	if v, ok := toInt(inputMap["height"]); ok {
		opts.Height = v
	}
	if v, ok := toInt64(inputMap["seed"]); ok {
		opts.Seed = &v
	}

	resp, err := c.provider.GenerateImage(ctx, model, prompt, opts)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("image generation failed: %w", err)
	}

	images := make([]map[string]any, len(resp.Images))
	for i, img := range resp.Images {
		images[i] = map[string]any{
			"base64": img.Base64,
			"url":    img.URL,
			"data":   img.Data,
		}
	}

	output := map[string]any{
		"images": images,
		"format": resp.Format,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type GenerateVideoCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewGenerateVideoCommand(provider InferenceProvider) *GenerateVideoCommand {
	return &GenerateVideoCommand{provider: provider}
}

func NewGenerateVideoCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *GenerateVideoCommand {
	return &GenerateVideoCommand{provider: provider, events: events}
}

func (c *GenerateVideoCommand) Name() string {
	return "inference.generate_video"
}

func (c *GenerateVideoCommand) Domain() string {
	return "inference"
}

func (c *GenerateVideoCommand) Description() string {
	return "Generate video from text prompt"
}

func (c *GenerateVideoCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Video generation model",
				},
			},
			"prompt": {
				Name: "prompt",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Text description of the video",
				},
			},
			"duration": {
				Name: "duration",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Video duration in seconds",
				},
			},
			"fps": {
				Name: "fps",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Frames per second",
				},
			},
			"width": {
				Name: "width",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Video width",
				},
			},
			"height": {
				Name: "height",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Video height",
				},
			},
		},
		Required: []string{"model", "prompt"},
	}
}

func (c *GenerateVideoCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"video":    {Name: "video", Schema: unit.Schema{Type: "string", Description: "Base64 encoded video"}},
			"format":   {Name: "format", Schema: unit.Schema{Type: "string"}},
			"duration": {Name: "duration", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (c *GenerateVideoCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "video-gen-1", "prompt": "A sunset over the ocean", "duration": 5},
			Output:      map[string]any{"video": "base64_video", "format": "mp4", "duration": 5.0},
			Description: "Generate a short video",
		},
	}
}

func (c *GenerateVideoCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	prompt, _ := inputMap["prompt"].(string)
	if prompt == "" {
		err := fmt.Errorf("prompt is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	opts := VideoOptions{}
	if v, ok := toFloat64(inputMap["duration"]); ok {
		opts.Duration = v
	}
	if v, ok := toInt(inputMap["fps"]); ok {
		opts.FPS = v
	}
	if v, ok := toInt(inputMap["width"]); ok {
		opts.Width = v
	}
	if v, ok := toInt(inputMap["height"]); ok {
		opts.Height = v
	}
	if v, ok := toInt64(inputMap["seed"]); ok {
		opts.Seed = &v
	}

	resp, err := c.provider.GenerateVideo(ctx, model, prompt, opts)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("video generation failed: %w", err)
	}

	output := map[string]any{
		"video":    string(resp.Video),
		"format":   resp.Format,
		"duration": resp.Duration,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type RerankCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewRerankCommand(provider InferenceProvider) *RerankCommand {
	return &RerankCommand{provider: provider}
}

func NewRerankCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *RerankCommand {
	return &RerankCommand{provider: provider, events: events}
}

func (c *RerankCommand) Name() string {
	return "inference.rerank"
}

func (c *RerankCommand) Domain() string {
	return "inference"
}

func (c *RerankCommand) Description() string {
	return "Rerank documents by relevance to a query"
}

func (c *RerankCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Reranking model identifier",
				},
			},
			"query": {
				Name: "query",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Search query",
				},
			},
			"documents": {
				Name: "documents",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Documents to rerank",
					Items:       &unit.Schema{Type: "string"},
				},
			},
		},
		Required: []string{"model", "query", "documents"},
	}
}

func (c *RerankCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"results": {
				Name: "results",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "object"},
				},
			},
		},
	}
}

func (c *RerankCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"model":     "rerank-1",
				"query":     "What is machine learning?",
				"documents": []string{"Machine learning is AI.", "Dogs are pets."},
			},
			Output: map[string]any{
				"results": []map[string]any{
					{"document": "Machine learning is AI.", "score": 0.95, "index": 0},
					{"document": "Dogs are pets.", "score": 0.1, "index": 1},
				},
			},
			Description: "Rerank documents",
		},
	}
}

func (c *RerankCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	query, _ := inputMap["query"].(string)
	if query == "" {
		err := fmt.Errorf("query is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	docsRaw, ok := inputMap["documents"].([]any)
	if !ok || len(docsRaw) == 0 {
		err := fmt.Errorf("documents are required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	documents := make([]string, len(docsRaw))
	for i, d := range docsRaw {
		documents[i] = d.(string)
	}

	resp, err := c.provider.Rerank(ctx, model, query, documents)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("rerank failed: %w", err)
	}

	results := make([]map[string]any, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = map[string]any{
			"document": r.Document,
			"score":    r.Score,
			"index":    r.Index,
		}
	}

	output := map[string]any{"results": results}
	ec.PublishCompleted(output)
	return output, nil
}

type DetectCommand struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewDetectCommand(provider InferenceProvider) *DetectCommand {
	return &DetectCommand{provider: provider}
}

func NewDetectCommandWithEvents(provider InferenceProvider, events unit.EventPublisher) *DetectCommand {
	return &DetectCommand{provider: provider, events: events}
}

func (c *DetectCommand) Name() string {
	return "inference.detect"
}

func (c *DetectCommand) Domain() string {
	return "inference"
}

func (c *DetectCommand) Description() string {
	return "Detect objects in an image"
}

func (c *DetectCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Detection model identifier",
				},
			},
			"image": {
				Name: "image",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Image data (base64 or URL)",
				},
			},
		},
		Required: []string{"model", "image"},
	}
}

func (c *DetectCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"detections": {
				Name: "detections",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Detected objects",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"label":      {Name: "label", Schema: unit.Schema{Type: "string"}},
							"confidence": {Name: "confidence", Schema: unit.Schema{Type: "number"}},
							"bbox": {
								Name: "bbox",
								Schema: unit.Schema{
									Type:  "array",
									Items: &unit.Schema{Type: "number"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *DetectCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model": "yolov8", "image": "base64_image_data"},
			Output:      map[string]any{"detections": []map[string]any{{"label": "person", "confidence": 0.95, "bbox": []float64{100, 100, 200, 300}}}},
			Description: "Detect objects in image",
		},
	}
}

func (c *DetectCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	model, _ := inputMap["model"].(string)
	if model == "" {
		err := ErrModelNotSpecified
		ec.PublishFailed(err)
		return nil, err
	}

	imageRaw, _ := inputMap["image"].(string)
	if imageRaw == "" {
		err := fmt.Errorf("image is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	resp, err := c.provider.Detect(ctx, model, []byte(imageRaw))
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("detection failed: %w", err)
	}

	detections := make([]map[string]any, len(resp.Detections))
	for i, d := range resp.Detections {
		detections[i] = map[string]any{
			"label":      d.Label,
			"confidence": d.Confidence,
			"bbox":       []float64{d.BBox[0], d.BBox[1], d.BBox[2], d.BBox[3]},
		}
	}

	output := map[string]any{"detections": detections}
	ec.PublishCompleted(output)
	return output, nil
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case float64:
		return int64(val), true
	default:
		return 0, false
	}
}
