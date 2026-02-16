# Engine Domain

推理引擎管理领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/engine/` | `pkg/engine/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `engine.start` | `{name, config?}` | `{process_id, status}` | 启动引擎 |
| `engine.stop` | `{name, force?, timeout?}` | `{success}` | 停止引擎 |
| `engine.restart` | `{name}` | `{process_id, status}` | 重启引擎 |
| `engine.install` | `{name, version?}` | `{success, path}` | 安装引擎 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `engine.get` | `{name}` | `{name, type, status, version, capabilities, models: []}` | 引擎信息 |
| `engine.list` | `{type?, status?}` | `{items: []}` | 列出引擎 |
| `engine.features` | `{name}` | `{supports_streaming, supports_batch, max_concurrent, ...}` | 引擎特性 |

## 已实现适配器

| 适配器 | 文件 | 模型类型 |
|--------|------|----------|
| Ollama | `adapters/ollama.go` | LLM |
| vLLM | `adapters/vllm.go` | LLM (高性能) |
| SGLang | `adapters/sglang.go` | LLM (高吞吐) |
| Whisper | `adapters/whisper.go` | ASR |
| TTS | `adapters/tts.go` | TTS |
| Diffusion | `adapters/diffusion.go` | ImageGen |
| Transformers | `adapters/transformers.go` | 通用 |
| HuggingFace | `adapters/huggingface.go` | 多模态 |
| Video | `adapters/video.go` | VideoGen |
| Rerank | `adapters/rerank.go` | Rerank |

## 核心接口

```go
type EngineAdapter interface {
    Name() string
    Version() string
    SupportedModelTypes() []model.ModelType
    SupportedFormats() []model.ModelFormat
    MaxConcurrentModels() int
    Install(version string) error
    IsInstalled() bool
    Start(config EngineConfig) (*EngineProcess, error)
    Stop(process *EngineProcess) error
    HealthCheck(process *EngineProcess) (HealthStatus, error)
    LoadModel(process *EngineProcess, m model.Model, opts LoadOptions) error
    UnloadModel(process *EngineProcess, modelID string) error
    EstimateMemory(m model.Model) (uint64, error)
}
```

## 实现文件

```
pkg/engine/
├── types.go               # 引擎类型
├── manager.go             # 生命周期管理
├── router.go              # 请求路由
├── loadbalancer.go        # 负载均衡
├── circuit_breaker.go     # 熔断器
├── failover.go            # 故障转移
├── pool.go                # 引擎池
└── adapters/
    ├── interfaces.go
    ├── ollama.go
    ├── vllm.go
    ├── sglang.go
    ├── whisper.go
    ├── tts.go
    └── ...
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `engine.start` | ✅ | `engine/manager.go` Start() |
| `engine.stop` | ✅ | `engine/manager.go` Stop() |
| `engine.get` | ✅ | `engine/manager.go` GetProcess() |
| `engine.list` | ✅ | `engine/manager.go` ListProcesses() |
| `engine.install` | ✅ | `engine/adapters/*.go` Install() |
| `engine.restart` | 🔧 | 组合调用 |
| `engine.features` | 🔧 | 需提取 |
