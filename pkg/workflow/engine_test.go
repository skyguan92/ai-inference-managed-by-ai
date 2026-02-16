package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockExecutor(ctx context.Context, unitType string, input map[string]any) (map[string]any, error) {
	switch unitType {
	case "inference.chat":
		return map[string]any{
			"content":       "Mock response",
			"finish_reason": "stop",
		}, nil
	case "inference.transcribe":
		text, _ := input["audio"].(string)
		return map[string]any{
			"text":     "Transcribed: " + text,
			"language": "en",
		}, nil
	case "inference.synthesize":
		text, _ := input["text"].(string)
		return map[string]any{
			"audio":    "audio_data_for_" + text,
			"format":   "mp3",
			"duration": 2.5,
		}, nil
	case "failing.unit":
		return nil, errors.New("unit execution failed")
	default:
		return map[string]any{"mock": true, "type": unitType}, nil
	}
}

func TestWorkflowEngine_Execute(t *testing.T) {
	store := NewInMemoryWorkflowStore()
	engine := NewWorkflowEngine(nil, store, mockExecutor)

	t.Run("simple workflow", func(t *testing.T) {
		def := &WorkflowDef{
			Name: "simple",
			Steps: []WorkflowStep{
				{ID: "chat", Type: "inference.chat", Input: map[string]any{"model": "test"}},
			},
		}

		result, err := engine.Execute(context.Background(), def, map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, ExecutionStatusCompleted, result.Status)
		assert.Contains(t, result.StepResults, "chat")
		assert.Equal(t, ExecutionStatusCompleted, result.StepResults["chat"].Status)
	})

	t.Run("workflow with dependencies", func(t *testing.T) {
		def := &WorkflowDef{
			Name:   "chained",
			Config: map[string]any{"model": "test"},
			Steps: []WorkflowStep{
				{ID: "transcribe", Type: "inference.transcribe", Input: map[string]any{"audio": "${input.audio}"}},
				{ID: "chat", Type: "inference.chat", Input: map[string]any{"model": "${config.model}", "message": "${steps.transcribe.text}"}, DependsOn: []string{"transcribe"}},
			},
		}

		result, err := engine.Execute(context.Background(), def, map[string]any{"audio": "audio_data"})
		require.NoError(t, err)
		assert.Equal(t, ExecutionStatusCompleted, result.Status)
		assert.Contains(t, result.StepResults, "transcribe")
		assert.Contains(t, result.StepResults, "chat")
	})

	t.Run("workflow with output resolution", func(t *testing.T) {
		def := &WorkflowDef{
			Name: "with_output",
			Steps: []WorkflowStep{
				{ID: "chat", Type: "inference.chat", Input: map[string]any{}},
			},
			Output: map[string]any{"response": "${steps.chat.content}"},
		}

		result, err := engine.Execute(context.Background(), def, map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, ExecutionStatusCompleted, result.Status)
		assert.Equal(t, "Mock response", result.Output["response"])
	})

	t.Run("workflow with failure", func(t *testing.T) {
		def := &WorkflowDef{
			Name: "failing",
			Steps: []WorkflowStep{
				{ID: "fail", Type: "failing.unit", Input: map[string]any{}},
			},
		}

		result, err := engine.Execute(context.Background(), def, map[string]any{})
		require.Error(t, err)
		assert.Equal(t, ExecutionStatusFailed, result.Status)
		assert.NotEmpty(t, result.Error)
	})

	t.Run("workflow with continue on failure", func(t *testing.T) {
		def := &WorkflowDef{
			Name: "continue_on_fail",
			Steps: []WorkflowStep{
				{ID: "fail", Type: "failing.unit", Input: map[string]any{}, OnFailure: "continue"},
				{ID: "success", Type: "inference.chat", Input: map[string]any{}, DependsOn: []string{"fail"}},
			},
		}

		result, err := engine.Execute(context.Background(), def, map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, ExecutionStatusCompleted, result.Status)
	})

	t.Run("invalid workflow validation", func(t *testing.T) {
		def := &WorkflowDef{
			Name: "invalid",
			Steps: []WorkflowStep{
				{ID: "s1", Type: "test", DependsOn: []string{"nonexistent"}},
			},
		}

		_, err := engine.Execute(context.Background(), def, map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

func TestWorkflowEngine_ExecuteWithRetry(t *testing.T) {
	attemptCount := 0
	retryExecutor := func(ctx context.Context, unitType string, input map[string]any) (map[string]any, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, errors.New("transient error")
		}
		return map[string]any{"success": true}, nil
	}

	store := NewInMemoryWorkflowStore()
	engine := NewWorkflowEngine(nil, store, retryExecutor)

	def := &WorkflowDef{
		Name: "retry_test",
		Steps: []WorkflowStep{
			{
				ID:    "retry",
				Type:  "test",
				Input: map[string]any{},
				Retry: &RetryConfig{
					MaxAttempts:  3,
					DelaySeconds: 0,
				},
			},
		},
	}

	attemptCount = 0
	result, err := engine.Execute(context.Background(), def, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, ExecutionStatusCompleted, result.Status)
	assert.Equal(t, 3, attemptCount)
}

func TestWorkflowEngine_Cancel(t *testing.T) {
	executor := func(ctx context.Context, unitType string, input map[string]any) (map[string]any, error) {
		time.Sleep(100 * time.Millisecond)
		return map[string]any{"done": true}, nil
	}

	store := NewInMemoryWorkflowStore()
	engine := NewWorkflowEngine(nil, store, executor)

	def := &WorkflowDef{
		Name: "long_running",
		Steps: []WorkflowStep{
			{ID: "long", Type: "long.operation", Input: map[string]any{}},
		},
	}

	result, err := engine.ExecuteAsync(context.Background(), def, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, ExecutionStatusRunning, result.Status)

	assert.True(t, engine.IsRunning(result.RunID))

	canceled := engine.Cancel(result.RunID)
	assert.True(t, canceled)

	time.Sleep(150 * time.Millisecond)
	assert.False(t, engine.IsRunning(result.RunID))
}

func TestWorkflowEngine_WorkflowManagement(t *testing.T) {
	store := NewInMemoryWorkflowStore()
	engine := NewWorkflowEngine(nil, store, mockExecutor)
	ctx := context.Background()

	def := &WorkflowDef{
		Name: "managed",
		Steps: []WorkflowStep{
			{ID: "s1", Type: "test", Input: map[string]any{}},
		},
	}

	err := engine.RegisterWorkflow(ctx, def)
	require.NoError(t, err)

	retrieved, err := engine.GetWorkflow(ctx, "managed")
	require.NoError(t, err)
	assert.Equal(t, "managed", retrieved.Name)

	workflows, err := engine.ListWorkflows(ctx)
	require.NoError(t, err)
	assert.Len(t, workflows, 1)

	err = engine.DeleteWorkflow(ctx, "managed")
	require.NoError(t, err)

	retrieved, err = engine.GetWorkflow(ctx, "managed")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestWorkflowEngine_ExecutionStorage(t *testing.T) {
	store := NewInMemoryWorkflowStore()
	engine := NewWorkflowEngine(nil, store, mockExecutor)
	ctx := context.Background()

	def := &WorkflowDef{
		Name: "stored",
		Steps: []WorkflowStep{
			{ID: "s1", Type: "inference.chat", Input: map[string]any{}},
		},
	}

	result, err := engine.Execute(ctx, def, map[string]any{})
	require.NoError(t, err)

	stored, err := engine.GetExecution(ctx, result.RunID)
	require.NoError(t, err)
	assert.Equal(t, result.RunID, stored.RunID)
	assert.Equal(t, ExecutionStatusCompleted, stored.Status)

	executions, err := engine.ListExecutions(ctx, "stored", 10)
	require.NoError(t, err)
	assert.Len(t, executions, 1)
}

func TestWorkflowEngine_VoiceAssistant(t *testing.T) {
	store := NewInMemoryWorkflowStore()
	engine := NewWorkflowEngine(nil, store, mockExecutor)

	def := &WorkflowDef{
		Name:        "voice_assistant",
		Description: "Voice input to audio output",
		Config: map[string]any{
			"llm_model": "llama3.2",
			"tts_model": "tts-1",
			"voice":     "alloy",
		},
		Steps: []WorkflowStep{
			{
				ID:   "transcribe",
				Type: "inference.transcribe",
				Input: map[string]any{
					"audio": "${input.audio}",
				},
			},
			{
				ID:   "chat",
				Type: "inference.chat",
				Input: map[string]any{
					"model": "${config.llm_model}",
					"messages": []any{
						map[string]any{
							"role":    "user",
							"content": "${steps.transcribe.text}",
						},
					},
				},
				DependsOn: []string{"transcribe"},
			},
			{
				ID:   "synthesize",
				Type: "inference.synthesize",
				Input: map[string]any{
					"model": "${config.tts_model}",
					"text":  "${steps.chat.content}",
					"voice": "${config.voice}",
				},
				DependsOn: []string{"chat"},
			},
		},
		Output: map[string]any{
			"text":     "${steps.transcribe.text}",
			"response": "${steps.chat.content}",
			"audio":    "${steps.synthesize.audio}",
		},
	}

	result, err := engine.Execute(context.Background(), def, map[string]any{
		"audio": "user_audio_data",
	})
	require.NoError(t, err)
	assert.Equal(t, ExecutionStatusCompleted, result.Status)
	assert.Contains(t, result.Output["text"], "Transcribed")
	assert.Contains(t, result.Output["audio"], "audio_data")
}
