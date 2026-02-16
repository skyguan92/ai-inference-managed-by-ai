package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableResolver_Resolve(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := &ExecutionContext{
		Input: map[string]any{
			"audio":   "base64data",
			"message": "Hello",
			"nested": map[string]any{
				"field": "nested_value",
			},
		},
		Config: map[string]any{
			"model": "llama3.2",
			"voice": "alloy",
		},
		Steps: map[string]map[string]any{
			"step1": {
				"text":     "transcribed text",
				"language": "en",
			},
			"step2": {
				"response": map[string]any{
					"content": "AI response",
				},
			},
		},
	}

	tests := []struct {
		name      string
		input     any
		want      any
		wantError bool
	}{
		{
			name:  "resolve input variable",
			input: "${input.audio}",
			want:  "base64data",
		},
		{
			name:  "resolve config variable",
			input: "${config.model}",
			want:  "llama3.2",
		},
		{
			name:  "resolve nested input variable",
			input: "${input.nested.field}",
			want:  "nested_value",
		},
		{
			name:  "resolve steps variable",
			input: "${steps.step1.text}",
			want:  "transcribed text",
		},
		{
			name:  "resolve nested steps variable",
			input: "${steps.step2.response.content}",
			want:  "AI response",
		},
		{
			name:  "resolve plain string",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "resolve string with embedded variable",
			input: "Using model ${config.model} with voice ${config.voice}",
			want:  "Using model llama3.2 with voice alloy",
		},
		{
			name:  "resolve map",
			input: map[string]any{"model": "${config.model}", "message": "${input.message}"},
			want:  map[string]any{"model": "llama3.2", "message": "Hello"},
		},
		{
			name:  "resolve array",
			input: []any{"${config.model}", "${config.voice}"},
			want:  []any{"llama3.2", "alloy"},
		},
		{
			name:  "resolve non-string values",
			input: 123,
			want:  123,
		},
		{
			name:      "error on nonexistent step",
			input:     "${steps.nonexistent.field}",
			wantError: true,
		},
		{
			name:      "error on nonexistent field",
			input:     "${input.nonexistent}",
			wantError: true,
		},
		{
			name:      "error on unknown source",
			input:     "${unknown.field}",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.input, ctx)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestVariableResolver_ResolveStepInput(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := &ExecutionContext{
		Input:  map[string]any{"text": "hello"},
		Config: map[string]any{"model": "llama3.2"},
		Steps: map[string]map[string]any{
			"transcribe": {"text": "transcribed"},
		},
	}

	step := &WorkflowStep{
		ID: "chat",
		Input: map[string]any{
			"model":   "${config.model}",
			"message": "${input.text}",
		},
	}

	result, err := resolver.ResolveStepInput(step, ctx)
	require.NoError(t, err)
	assert.Equal(t, "llama3.2", result["model"])
	assert.Equal(t, "hello", result["message"])
}

func TestVariableResolver_ResolveOutput(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := &ExecutionContext{
		Steps: map[string]map[string]any{
			"transcribe": {"text": "transcribed text"},
			"chat": {
				"response": map[string]any{
					"content": "AI response",
				},
			},
			"synthesize": {"audio": "audio_data"},
		},
	}

	outputDef := map[string]any{
		"text":     "${steps.transcribe.text}",
		"response": "${steps.chat.response.content}",
		"audio":    "${steps.synthesize.audio}",
	}

	result, err := resolver.ResolveOutput(outputDef, ctx)
	require.NoError(t, err)
	assert.Equal(t, "transcribed text", result["text"])
	assert.Equal(t, "AI response", result["response"])
	assert.Equal(t, "audio_data", result["audio"])
}

func TestVariableResolver_ComplexNesting(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := &ExecutionContext{
		Input: map[string]any{
			"messages": []any{
				map[string]any{"role": "user", "content": "Hello"},
			},
		},
		Config: map[string]any{
			"model": "llama3.2",
		},
		Steps: map[string]map[string]any{
			"transcribe": {"text": "Hello world"},
		},
	}

	step := &WorkflowStep{
		ID: "chat",
		Input: map[string]any{
			"model": "${config.model}",
			"messages": []any{
				map[string]any{
					"role":    "user",
					"content": "${steps.transcribe.text}",
				},
			},
		},
	}

	result, err := resolver.ResolveStepInput(step, ctx)
	require.NoError(t, err)
	assert.Equal(t, "llama3.2", result["model"])

	messages, ok := result["messages"].([]any)
	require.True(t, ok)
	require.Len(t, messages, 1)

	msg, ok := messages[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", msg["role"])
	assert.Equal(t, "Hello world", msg["content"])
}

func TestVariableResolver_EmptyPath(t *testing.T) {
	resolver := NewVariableResolver()

	ctx := &ExecutionContext{
		Input: map[string]any{"field": "value"},
	}

	result, err := resolver.Resolve("${input}", ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"field": "value"}, result)
}
