# ASMS → AIMA 迁移分析报告

## 项目概述

### ASMS 现状

- **项目名称**: ASMS (AI Service Management System)
- **源码位置**: `/home/qujing/projects/asms`
- **总代码量**: ~97,000 行 Go 代码
- **架构风格**: 领域驱动设计 (DDD)，模块化微服务架构

### AIMA 目标

- **项目名称**: ai-inference-managed-by-ai
- **源码位置**: `/home/qujing/projects/ai-inference-managed-by-ai`
- **架构风格**: 原子单元 + 编排层
- **核心理念**: 一切皆原子服务，四种标准化接口

---

## 模块总览

| 领域 | ASMS 目录 | 代码行数 | 主要职责 |
|------|-----------|----------|----------|
| **Device** | `pkg/hal/` | ~2,200 | 硬件抽象层，GPU/NPU 设备管理 |
| **Model** | `pkg/model/` | ~9,500 | 多模态模型注册、下载、兼容性 |
| **Engine** | `pkg/engine/` | ~10,000 | 推理引擎生命周期与请求路由 |
| **Service** | `pkg/service/` | ~2,000 | 模型服务编排与生命周期 |
| **Resource** | `pkg/resource/` | ~1,100 | 资源槽位与内存预算管理 |
| **App** | `pkg/app/` | ~1,500 | Docker 应用生命周期管理 |
| **Pipeline** | `pkg/pipeline/` | ~2,600 | 多模型管道编排引擎 |
| **Remote** | `pkg/remote/` | ~1,300 | 远程访问隧道与沙箱执行 |
| **Alert** | `pkg/fleet/` (部分) | ~680 | 告警规则与通知通道 |
| **Fleet** | `pkg/fleet/` | ~11,400 | 多设备集群管理控制平面 |
| **Runtime** | `pkg/runtime/` | ~8,400 | 容器运行时与调度器 |
| **Store** | `pkg/store/` | ~4,600 | SQLite 持久化存储层 |
| **API** | `pkg/api/` | ~13,300 | REST API 与 OpenAI 兼容网关 |
| **MCP** | `pkg/mcp/` | ~10,300 | AI Agent 工具协议服务器 |

---

## 详细模块分析

### 1. Device Domain (HAL)

**源码**: `/home/qujing/projects/asms/pkg/hal/`

**主要文件**:
```
pkg/hal/
├── interfaces.go          # 核心接口定义
├── cache.go               # 设备指标缓存
├── v2/
│   ├── interfaces.go      # HAL v2 可扩展接口
│   └── manager.go         # HAL 管理器
├── nvidia/
│   └── provider.go        # NVIDIA GPU 提供者
└── generic/
    └── provider.go        # 通用 CPU 提供者
```

**核心接口** (位于 `pkg/hal/interfaces.go`):
```go
type DeviceProvider interface {
    Detect() ([]Device, error)
    Name() string
    Supported() bool
}

type Device interface {
    ID() string
    Name() string
    Vendor() string
    Architecture() string
    Capabilities() DeviceCapabilities
    TotalMemory() uint64
    AvailableMemory() uint64
    Utilization() (float64, error)
    Temperature() (float64, error)
    PowerUsage() (float64, error)
    HealthStatus() (HealthStatus, error)
    Metrics() (DeviceMetrics, error)
}
```

**已实现功能**:
- 多厂商 GPU 检测 (NVIDIA, 通用)
- 设备指标缓存 (TTL 可配置)
- 内存拓扑与分区管理
- 健康状态监控
- GB10/Jetson SoC 优化 (v2)

---

### 2. Model Domain

**源码**: `/home/qujing/projects/asms/pkg/model/`

**目录结构**:
```
pkg/model/
├── types.go               # 模型类型定义
├── manager.go             # 模型管理器
├── compatibility.go       # 引擎兼容性映射
├── downloader/            # 模型下载器
│   ├── ollama.go
│   ├── huggingface.go
│   └── modelscope.go
└── v2/
    ├── types.go           # 模型服务 v2 类型
    ├── service/           # 服务注册与联邦
    ├── search/            # 模型搜索聚合器
    ├── download/          # 下载管理器
    └── network/           # 网络探测
```

**核心结构体** (位于 `pkg/model/types.go`):
```go
type Model struct {
    ID           string
    Name         string
    Family       string
    Type         ModelType      // llm, vlm, asr, tts, embedding, diffusion...
    Modalities   ModelModalities
    Parameters   ParameterSpec
    Format       ModelFormat    // gguf, safetensors, onnx, tensorrt...
    Quantization string
    Requirements ResourceRequirements
    Source       ModelSource
    Status       ModelStatus
    InputTypes   []string
    OutputTypes  []string
    Languages    []string
    Streaming    bool
    RealTime     bool
}
```

**支持的模型类型** (9 种):
- `llm` - 大语言模型
- `vlm` - 视觉语言模型
- `asr` - 语音识别
- `tts` - 语音合成
- `embedding` - 文本嵌入
- `diffusion` - 图像生成
- `video_gen` - 视频生成
- `detection` - 目标检测
- `rerank` - 重排序

---

### 3. Engine Domain

**源码**: `/home/qujing/projects/asms/pkg/engine/`

**目录结构**:
```
pkg/engine/
├── types.go               # 引擎类型与请求/响应定义
├── manager.go             # 引擎生命周期管理
├── router.go              # 请求路由
├── loadbalancer.go        # 负载均衡
├── circuit_breaker.go     # 熔断器
├── failover.go            # 故障转移
├── pool.go                # 引擎池
└── adapters/
    ├── ollama.go          # Ollama 适配器
    ├── vllm.go            # vLLM 适配器
    ├── sglang.go          # SGLang 适配器
    ├── whisper.go         # Whisper ASR 适配器
    ├── tts.go             # TTS 适配器
    ├── diffusion.go       # 图像生成适配器
    ├── transformers.go    # Transformers 适配器
    ├── huggingface.go     # HuggingFace 适配器
    ├── video.go           # 视频处理适配器
    └── rerank.go          # 重排序适配器
```

**核心接口** (位于 `pkg/engine/adapters/`):
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

type LLMEngine interface {
    EngineAdapter
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
    Completion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}
```

**已实现适配器** (10 个):
- Ollama (LLM)
- vLLM (LLM, 高性能)
- SGLang (LLM, 高吞吐)
- Whisper (ASR)
- TTS (语音合成)
- Diffusion (图像生成)
- Transformers (通用)
- HuggingFace (多模态)
- Video (视频处理)
- Rerank (重排序)

---

### 4. Service Domain

**源码**: `/home/qujing/projects/asms/pkg/service/`

**主要文件**:
```
pkg/service/
├── manager.go             # 服务管理器
├── model.go               # 模型服务定义
├── lifecycle.go           # 生命周期管理
├── resource_table.go      # 资源查表
├── engine_selector.go     # 引擎选择器
├── hardware_selector.go   # 硬件选择器
├── telemetry.go           # 遥测收集
└── optimizer.go           # 服务优化
```

**核心结构体**:
```go
type ModelServiceManager struct {
    db               *sql.DB
    store            ServiceRepository
    modelRepo        interface{ Get(id string) (*model.Model, error) }
    engineMgr        *engine.Manager
    resourceMgr      *resource.Manager
    halMgr           *hal.Manager
    lifecycle        *LifecycleManager
    resourceTable    *ResourceTable
    engineSelector   *EngineSelector
    hardwareSelector *HardwareSelector
    telemetry        *TelemetryCollector
}

type ModelService struct {
    ID        string
    Name      string
    ModelID   string
    Status    ServiceStatus
    Config    *ServiceConfig
    Runtime   *ServiceRuntime
    Endpoint  *ServiceEndpoint
    Resources *ServiceResources
    Stats     *ServiceStats
}
```

---

### 5. Resource Domain

**源码**: `/home/qujing/projects/asms/pkg/resource/`

**核心结构体**:
```go
type ResourceSlot struct {
    ID           string
    Name         string
    Type         SlotType      // inference_native, docker_container, system_service
    ModelType    model.ModelType
    MemoryLimit  uint64
    MemoryTarget uint64
    GPUFraction  float64
    CPUCores     float64
    Priority     int
    Preemptible  bool
    Persistent   bool
    Status       SlotStatus
    CurrentModel string
    ProcessPID   int
    ActualMemory uint64
}

type MemoryBudget struct {
    TotalBytes     uint64
    SystemReserved uint64
    ASMSReserved   uint64
    InferencePool  uint64
    ContainerPool  uint64
    BufferFlexible uint64
}
```

---

### 6. App Domain

**源码**: `/home/qujing/projects/asms/pkg/app/`

**主要文件**:
```
pkg/app/
├── types.go               # 应用类型定义
├── manager.go             # 应用管理器
├── templates.go           # 应用模板
├── docker.go              # Docker 集成
└── docker_real.go         # 真实 Docker 操作
```

---

### 7. Pipeline Domain

**源码**: `/home/qujing/projects/asms/pkg/pipeline/`

**主要文件**:
```
pkg/pipeline/
├── types.go               # 管道类型定义
├── engine.go              # 管道执行引擎
├── template.go            # 管道模板
├── template_loader.go     # 模板加载器
├── validator.go           # 管道验证
├── condition.go           # 条件执行
├── retry.go               # 重试机制
├── cache.go               # 结果缓存
├── dependency_checker.go  # 依赖检查
└── metrics.go             # 管道指标
```

**预定义管道类型**:
- `voice-assistant`: ASR → LLM → TTS
- `rag`: Embed → Search → LLM
- `vision-chat`: Image → VLM → LLM
- `content-gen`: LLM → ImageGen
- `detect-describe`: YOLO → LLM
- `video-stream-analysis`: 提取帧 → VLM 分析

---

### 8. Alert Domain

**源码**: `/home/qujing/projects/asms/pkg/fleet/alert.go`, `alert_channel.go`

**核心结构体**:
```go
type Alert struct {
    ID          string
    DeviceID    string
    RuleID      string
    RuleName    string
    Severity    AlertSeverity     // info, warning, critical
    Status      AlertStatus       // firing, acknowledged, resolved
    Message     string
    Metrics     map[string]any
    TriggeredAt time.Time
}

type NotificationChannel struct {
    ID        string
    Name      string
    Type      ChannelType       // webhook, email, slack, wechat, sms
    Config    map[string]string
    Enabled   bool
}
```

**已实现通知发送器**:
- Webhook (带 HMAC 签名)
- Email (SMTP)
- Slack
- WeChat 企业微信

---

### 9. Remote Domain

**源码**: `/home/qujing/projects/asms/pkg/remote/`

**主要文件**:
```
pkg/remote/
├── types.go               # 远程访问类型
├── manager.go             # 远程管理器
├── tunnel_frp.go          # FRP 隧道实现
└── tunnel_cloudflare.go   # Cloudflare Tunnel 实现
```

**隧道支持**:
- FRP
- Cloudflare Tunnel

---

### 10. Fleet Domain

**源码**: `/home/qujing/projects/asms/pkg/fleet/`

**主要功能**:
- 设备注册与心跳
- 分布式任务调度
- 滚动更新
- 配置同步
- 配额管理
- RBAC 权限
- 备份恢复
- 设备分组

---

## 模块依赖关系

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Layer                                │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │   API   │ │   MCP   │ │  Fleet  │ │ Remote  │ │ Monitor │  │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘  │
└───────┼──────────┼──────────┼──────────┼──────────┼─────────────┘
        │          │          │          │          │
┌───────┴──────────┴──────────┴──────────┴──────────┴─────────────┐
│                      Service Layer                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │ Service │ │ Pipeline│ │   App   │ │  Alert  │ │Runtime  │  │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘  │
└───────┼──────────┼──────────┼──────────┼──────────┼─────────────┘
        │          │          │          │          │
┌───────┴──────────┴──────────┴──────────┴──────────┴─────────────┐
│                      Domain Layer                                │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │  Model  │ │ Engine  │ │Resource │ │   HAL   │ │  Store  │  │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 关键差距分析

### 需要新建的组件

| 组件 | 说明 |
|------|------|
| 原子单元抽象层 | Command/Query/Event/Resource 接口和 Registry |
| 统一 Gateway | HTTP/MCP/CLI 统一入口 |
| Resource 接口 | 可寻址资源 URI 访问和 Watch 订阅 |
| Workflow DSL | 比 Pipeline 更灵活的编排能力 |

### 需要新增的功能

| 原子单元 | 缺失功能 |
|----------|----------|
| `device.set_power_limit` | 设置功耗限制 |
| `model.verify` | 模型完整性验证 |
| `service.scale` | 服务扩缩容 |
| `pipeline.cancel` | 取消运行中的管道 |
| `inference.voices` | 列出可用语音 |

### 可直接复用的代码

| 模块 | 复用方式 |
|------|----------|
| `pkg/hal/` | 封装为 Device 原子单元 |
| `pkg/model/` | 封装为 Model 原子单元 |
| `pkg/engine/` | 封装为 Engine/Inference 原子单元 |
| `pkg/resource/` | 封装为 Resource 原子单元 |
| `pkg/service/` | 封装为 Service 原子单元 |
| `pkg/app/` | 封装为 App 原子单元 |
| `pkg/pipeline/` | 升级为 Workflow DSL |
| `pkg/store/` | 直接复用 |
| `pkg/eventbus/` | 直接复用 |
