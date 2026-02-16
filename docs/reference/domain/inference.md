# Inference Domain

推理服务领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/inference/` | `pkg/engine/adapters/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `inference.chat` | `{model, messages, stream?, temperature?, max_tokens?, ...}` | `{content, finish_reason, usage}` | 聊天补全 |
| `inference.complete` | `{model, prompt, stream?, ...}` | `{text, finish_reason, usage}` | 文本补全 |
| `inference.embed` | `{model, input}` | `{embeddings: [], usage}` | 文本嵌入 |
| `inference.transcribe` | `{model, audio, language?}` | `{text, segments, language}` | 语音转文字 |
| `inference.synthesize` | `{model, text, voice?, stream?}` | `{audio, format, duration}` | 文字转语音 |
| `inference.generate_image` | `{model, prompt, size?, steps?, ...}` | `{images: [], format}` | 图像生成 |
| `inference.generate_video` | `{model, prompt, duration?, ...}` | `{video, format, duration}` | 视频生成 |
| `inference.rerank` | `{model, query, documents}` | `{results: []}` | 重排序 |
| `inference.detect` | `{model, image}` | `{detections: []}` | 目标检测 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `inference.models` | `{type?}` | `{models: []}` | 可用模型 |
| `inference.voices` | `{model?}` | `{voices: []}` | 可用语音 |

## 扩展接口

```go
type LLMEngine interface {
    EngineAdapter
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
    Completion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

type ASREngine interface {
    EngineAdapter
    Transcribe(ctx context.Context, audio AudioInput) (*TranscriptionResult, error)
}

type TTSEngine interface {
    EngineAdapter
    Synthesize(ctx context.Context, req SynthesizeRequest) (*AudioOutput, error)
}
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `inference.chat` | ✅ | `engine/adapters/*.go` ChatCompletion() |
| `inference.complete` | ✅ | `engine/adapters/*.go` Completion() |
| `inference.embed` | ✅ | `engine/adapters/*.go` Embed() |
| `inference.transcribe` | ✅ | `engine/adapters/whisper.go` Transcribe() |
| `inference.synthesize` | ✅ | `engine/adapters/tts.go` Synthesize() |
| `inference.generate_image` | ✅ | `engine/adapters/diffusion.go` GenerateImage() |
| `inference.generate_video` | ✅ | `engine/adapters/video.go` GenerateVideo() |
| `inference.rerank` | ✅ | `engine/adapters/rerank.go` Rerank() |
| `inference.detect` | ✅ | `engine/adapters/*.go` Detect() |
| `inference.models` | ✅ | `service/model.go` |
| `inference.voices` | 🔧 | TTS 适配器中需提取 |
