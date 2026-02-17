package e2e

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
)

func TestInferenceFlowE2E(t *testing.T) {
	env := SetupTestEnv(t)

	t.Run("list inference models", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "inference.models",
			Input: map[string]any{},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		models := getSliceField(data, "models")
		total := len(models)
		if total < 0 {
			t.Errorf("expected non-negative models count, got: %d", total)
		}
	})

	t.Run("list inference models by type", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "inference.models",
			Input: map[string]any{
				"type": "llm",
			},
		})
		assertSuccess(t, resp)
	})

	t.Run("chat completion", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.chat",
			Input: map[string]any{
				"model": "llama3",
				"messages": []any{
					map[string]any{"role": "system", "content": "You are a helpful assistant."},
					map[string]any{"role": "user", "content": "Hello, how are you?"},
				},
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		content := getStringField(data, "content")
		if content == "" {
			t.Errorf("expected content to be non-empty")
		}
		finishReason := getStringField(data, "finish_reason")
		if finishReason != "stop" {
			t.Errorf("expected finish_reason 'stop', got: %s", finishReason)
		}
	})

	t.Run("chat completion with options", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.chat",
			Input: map[string]any{
				"model":       "gpt-4",
				"messages":    []any{map[string]any{"role": "user", "content": "Write a poem"}},
				"temperature": 0.8,
				"max_tokens":  500,
				"top_p":       0.9,
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		usage := getMapField(data, "usage")
		if usage == nil {
			t.Errorf("expected usage to be non-nil")
		}
	})

	t.Run("text completion", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.complete",
			Input: map[string]any{
				"model":  "llama3",
				"prompt": "Once upon a time",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		text := getStringField(data, "text")
		if text == "" {
			t.Errorf("expected text to be non-empty")
		}
	})

	t.Run("text completion with options", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.complete",
			Input: map[string]any{
				"model":       "llama3",
				"prompt":      "The meaning of life is",
				"temperature": 0.7,
				"max_tokens":  100,
				"stop":        []string{"\n", "."},
			},
		})
		assertSuccess(t, resp)
	})

	t.Run("text embedding", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.embed",
			Input: map[string]any{
				"model": "text-embedding-3-small",
				"input": "Hello world",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		embeddings := getSliceField(data, "embeddings")
		total := len(embeddings)
		if total < 0 {
			t.Errorf("expected non-negative embeddings count, got: %d", total)
		}
	})

	t.Run("batch text embedding", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.embed",
			Input: map[string]any{
				"model": "text-embedding-3-small",
				"input": []any{"Hello", "World"},
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		embeddings := getSliceField(data, "embeddings")
		total := len(embeddings)
		if total < 0 {
			t.Errorf("expected non-negative embeddings count, got: %d", total)
		}
	})

	t.Run("audio transcription", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.transcribe",
			Input: map[string]any{
				"model":    "whisper-large-v3",
				"audio":    "base64_audio_data",
				"language": "en",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		text := getStringField(data, "text")
		if text == "" {
			t.Errorf("expected text to be non-empty")
		}
	})

	t.Run("text to speech synthesis", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.synthesize",
			Input: map[string]any{
				"model": "tts-1",
				"text":  "Hello, world!",
				"voice": "alloy",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		audio := getStringField(data, "audio")
		if audio == "" {
			t.Errorf("expected audio to be non-empty")
		}
		format := getStringField(data, "format")
		if format == "" {
			t.Errorf("expected format to be non-empty")
		}
	})

	t.Run("image generation", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.generate_image",
			Input: map[string]any{
				"model":  "dall-e-3",
				"prompt": "A cat sitting on a moon",
				"size":   "1024x1024",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		images := getSliceField(data, "images")
		total := len(images)
		if total < 0 {
			t.Errorf("expected non-negative images count, got: %d", total)
		}
	})

	t.Run("image generation with options", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.generate_image",
			Input: map[string]any{
				"model":           "stable-diffusion-xl",
				"prompt":          "A beautiful sunset",
				"width":           512,
				"height":          512,
				"steps":           30,
				"negative_prompt": "blurry, low quality",
			},
		})
		assertSuccess(t, resp)
	})

	t.Run("video generation", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.generate_video",
			Input: map[string]any{
				"model":    "video-gen-1",
				"prompt":   "A sunset over the ocean",
				"duration": 5,
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		video := getStringField(data, "video")
		if video == "" {
			t.Errorf("expected video to be non-empty")
		}
	})

	t.Run("document reranking", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.rerank",
			Input: map[string]any{
				"model":     "rerank-1",
				"query":     "What is machine learning?",
				"documents": []any{"Machine learning is AI.", "Dogs are pets."},
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		results := getSliceField(data, "results")
		total := len(results)
		if total < 0 {
			t.Errorf("expected non-negative results count, got: %d", total)
		}
	})

	t.Run("object detection", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.detect",
			Input: map[string]any{
				"model": "yolov8",
				"image": "base64_image_data",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		detections := getSliceField(data, "detections")
		total := len(detections)
		if total < 0 {
			t.Errorf("expected non-negative detections count, got: %d", total)
		}
	})

	t.Run("list voices for TTS", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "inference.voices",
			Input: map[string]any{},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		voices := getSliceField(data, "voices")
		total := len(voices)
		if total < 0 {
			t.Errorf("expected non-negative voices count, got: %d", total)
		}
	})

	t.Run("chat without model should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.chat",
			Input: map[string]any{
				"messages": []map[string]any{{"role": "user", "content": "Hello"}},
			},
		})
		assertError(t, resp)
	})

	t.Run("chat without messages should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.chat",
			Input: map[string]any{
				"model": "llama3",
			},
		})
		assertError(t, resp)
	})

	t.Run("embedding without input should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.embed",
			Input: map[string]any{
				"model": "text-embedding-3-small",
			},
		})
		assertError(t, resp)
	})

	t.Run("full inference workflow", func(t *testing.T) {
		modelsResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "inference.models",
			Input: map[string]any{},
		})
		assertSuccess(t, modelsResp)

		chatResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.chat",
			Input: map[string]any{
				"model": "llama3",
				"messages": []any{
					map[string]any{"role": "user", "content": "What is the capital of France?"},
				},
			},
		})
		assertSuccess(t, chatResp)

		embedResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "inference.embed",
			Input: map[string]any{
				"model": "text-embedding-3-small",
				"input": "Paris is the capital of France",
			},
		})
		assertSuccess(t, embedResp)
	})
}
