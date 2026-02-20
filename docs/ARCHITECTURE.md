# ai-inference-managed-by-ai

> 让 AI 管理的 AI 推理基础设施

## 项目概述

**ai-inference-managed-by-ai** 是一个面向 AI Agent 和机器用户的 AI 推理基础设施管理平台。

### 核心设计理念

1. **一切皆原子服务** - 所有功能拆分为最小可组合单元
2. **四种接口类型** - Command、Query、Event、Resource
3. **统一入口** - HTTP/MCP/CLI 共享同一套原子单元
4. **可编排性** - 高级功能通过编排原子单元实现
5. **AI First** - 主要用户是 AI Agent，接口设计优先考虑机器可理解性

---


---

## 设计哲学

### 为什么是四种接口类型？

AIMA 选择 Command / Query / Event / Resource 四种接口类型，基于以下考量：

| 接口类型 | 本质 | 对应概念 | 为什么需要 |
|----------|------|----------|-----------|
| Command | 有副作用的操作 | CQS 的 Command | 所有写操作的统一抽象 |
| Query | 无副作用的查询 | CQS 的 Query | 读写分离，Query 可安全重试和缓存 |
| Event | 异步通知 | 领域事件 | 解耦生产者和消费者，支持事件溯源 |
| Resource | 可寻址状态 | REST 资源 | 状态的统一寻址和订阅 |

**为什么不是三种？** 如果合并 Command 和 Query（如通用的 Action 接口），会丧失"查询无副作用"的保证，导致：
- 无法安全地对 Query 做缓存
- 无法并发执行多个 Query
- AI Agent 无法判断调用某个 tool 是否会改变系统状态

**为什么不是五种或更多？** 四种类型已经覆盖了所有交互模式：同步写（Command）、同步读（Query）、异步通知（Event）、状态订阅（Resource）。增加更多类型会引入不必要的认知负担。

### 为什么 Command 和 Query 签名相同？

Command 和 Query 接口的方法签名完全相同（`Execute(ctx, input) (output, error)`），这是刻意的设计：

1. **统一执行路径** - Gateway 可以用相同的逻辑分发 Command 和 Query
2. **MCP 统一映射** - 两者都映射为 MCP Tool，AI Agent 无需区分底层类型
3. **语义约束而非签名约束** - 区别在于约定（Query 不允许副作用），而非编译器强制

### 为什么使用 `any` 类型？

Input/Output 使用 `any` 而非泛型 `Command[I, O]`，基于以下权衡：

| 方案 | 优势 | 劣势 |
|------|------|------|
| `any` (当前) | Registry 可存储异构类型；序列化/反序列化简单 | 运行时类型断言 |
| 泛型 `Command[I, O]` | 编译时类型安全 | Registry 无法存储异构泛型类型；需要类型擦除 |

Go 的泛型不支持协变，因此 `Registry` 无法同时存储 `Command[PullInput, PullOutput]` 和 `Command[CreateInput, CreateOutput]`。我们选择在实现层面使用强类型（每个 Command 内部定义具体的 Input/Output struct），在接口层面使用 `any` 做桥接。

> **改进方向**: 未来可引入 `TypedCommand[I, O]` 适配器，在不破坏接口的前提下提供类型安全的实现辅助。

### 为什么是 Registry 而非依赖注入？

| 方案 | 说明 |
|------|------|
| Registry (当前) | 原子单元主动注册到全局表，Gateway 通过名称查找 |
| DI Container | 容器管理所有依赖的创建和注入 |

选择 Registry 的理由：
1. **简单直接** - 无需引入 DI 框架（wire/dig），降低 AI Agent 的理解成本
2. **运行时可发现** - AI Agent 可以列出所有已注册单元，做动态选择
3. **热插拔** - 可以在运行时注册/注销原子单元
4. **单二进制约束** - 不需要外部配置文件来描述依赖关系

## 架构分层

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AI Agent Layer (智能代理层)                     │
│    Agent Operator · Skills · Conversation · LLM Client Adapters    │
├─────────────────────────────────────────────────────────────────────┤
│                      Orchestration Layer                            │
│          Pipelines · Workflows · Pre-built Flows · DSL              │
├─────────────────────────────────────────────────────────────────────┤
│                      Service Layer                                  │
│    ModelService · InferenceService · CatalogService · AppService   │
├─────────────────────────────────────────────────────────────────────┤
│                   Atomic Unit Layer (核心)                          │
│     Command · Query · Event · Resource (4 种接口类型)               │
│     13 个领域: device model engine inference resource service      │
│                app pipeline alert remote catalog agent skill       │
├─────────────────────────────────────────────────────────────────────┤
│                   Infrastructure Layer                              │
│      HAL · Store · EventBus · Docker · Registry · Network          │
└─────────────────────────────────────────────────────────────────────┘
```

### 新增分层说明

| 层 | 新增内容 | 说明 |
|----|---------|------|
| AI Agent Layer | Agent Operator, Skills, LLM Client | 面向终端用户的智能对话代理，通过 MCP Tools 操作 AIMA |
| Service Layer | CatalogService | 编排 Recipe 的一键部署流程（拉取引擎镜像 + 拉取模型）|
| Atomic Unit Layer | catalog / agent / skill 三个新领域 | 硬件最佳实践、AI 代理、技能知识库 |
| Infrastructure Layer | Registry Provider | Docker 镜像仓库抽象层，支持 DockerHub/GHCR/国内镜像 |

---


---

## 原子单元设计原则

### 自闭环（Self-contained）定义

每个原子单元必须是一个**自闭环**——即能够独立产生价值的最小功能单位。自闭环是 AIMA 的核心设计原则。

一个合格的自闭环原子单元必须满足以下 5 个条件：

#### 1. 自描述（Self-describing）

原子单元必须携带足够的元信息，使得 AI Agent 或人类开发者**无需阅读源码**即可理解其能力。

```go
// 必须实现：
Name() string           // 唯一标识，格式 {domain}.{action}
Domain() string         // 所属领域
Description() string    // 人类可读的功能描述
InputSchema() Schema    // 输入参数的完整定义（类型、约束、默认值）
OutputSchema() Schema   // 输出结构的完整定义
Examples() []Example    // 至少一个输入/输出示例
```

**检查标准**：
- 描述是否足够让 AI Agent 判断"什么时候应该用这个工具"？
- Schema 是否包含所有参数的类型、描述、约束？
- Examples 是否覆盖了主要使用场景？

#### 2. 自验证（Self-validating）

原子单元必须在执行前验证输入的合法性，拒绝无效输入而非在执行过程中崩溃。

**检查标准**：
- 是否在执行业务逻辑前完成了所有输入验证？
- 验证失败的错误信息是否清晰？（告诉调用者哪个字段有什么问题）
- 是否处理了类型不匹配的情况？

#### 3. 自执行（Self-executing）

原子单元**只依赖注入的基础设施**（Store、Provider、EventBus），**不依赖其他原子单元**。

```
正确: PullCommand 依赖 ModelStore + ModelProvider（基础设施）
错误: PullCommand 内部调用 VerifyCommand（其他原子单元）
```

原子单元之间的组合发生在**服务层**（Service Layer），而非原子单元内部。

**检查标准**：
- Execute 方法是否只调用注入的 Store/Provider/EventBus？
- 是否有直接调用 Registry 获取其他 Command/Query 的行为？（应移到 Service 层）

#### 4. 自报告（Self-reporting）

执行结果必须包含足够的上下文，使得调用者无需额外查询即可理解发生了什么。

**检查标准**：
- 成功时是否返回了操作产生的关键信息（ID、状态、摘要）？
- 失败时错误信息是否包含上下文（哪个模型、什么原因、建议动作）？
- AI Agent 收到结果后是否能独立判断下一步？

#### 5. 自恢复（Self-recovering）

执行失败时不留下不一致的中间状态。

**检查标准**：
- 是否使用 defer 清理临时状态？
- 如果创建了中间资源（如数据库记录），失败时是否回滚？
- 并发保护是否正确？（mutex 的 Lock/Unlock 配对）

### 自闭环 vs 组合

```
+--------------------------------------------------+
|                  服务层 (组合)                     |
|  PullAndVerify = model.pull + model.verify       |
|  独立使用 model.pull 也完全有效                    |
+--------------------------------------------------+
|              原子单元层 (自闭环)                   |
|  model.pull    model.verify    model.create      |
|  每个都可以单独调用，独立产生价值                   |
+--------------------------------------------------+
```

**核心原则**：
- 原子单元 = 自闭环（独立可用）
- 服务方法 = 原子单元的编排（组合增值）
- AI Agent 可以直接调用原子单元，也可以通过服务方法调用编排好的流程

## 四种接口类型 (限定)

所有功能单元必须实现以下四种接口之一：

### Command - 有副作用的操作

```go
type Command interface {
    // 单元标识
    Name() string
    Domain() string
    
    // 输入输出 Schema
    InputSchema() Schema
    OutputSchema() Schema
    
    // 执行
    Execute(ctx context.Context, input any) (output any, err error)
    
    // 元信息
    Description() string
    Examples() []Example
}
```

### Query - 无副作用的查询

```go
type Query interface {
    Name() string
    Domain() string
    InputSchema() Schema
    OutputSchema() Schema
    Execute(ctx context.Context, input any) (output any, err error)
    Description() string
    Examples() []Example
}
```

### Event - 异步事件

```go
type Event interface {
    Type() string
    Domain() string
    Payload() any
    Timestamp() time.Time
    CorrelationID() string
}
```

### Resource - 可寻址资源

```go
type Resource interface {
    URI() string
    Domain() string
    Schema() Schema
    Get(ctx context.Context) (any, error)
    Watch(ctx context.Context) (<-chan ResourceUpdate, error)
}
```

---


### URI Scheme 规范

AIMA 使用 `asms://` URI scheme 唯一标识所有可寻址资源。

#### URI 格式

```
asms://{domain}/{identifier}[/{sub-resource}]
```

#### 完整 URI 列表

| URI | 类型 | 说明 |
|-----|------|------|
| `asms://device/{id}/info` | 动态 | 设备信息 |
| `asms://device/{id}/metrics` | 动态 | 设备实时指标 |
| `asms://device/{id}/health` | 动态 | 设备健康状态 |
| `asms://model/{id}` | 动态 | 模型详情 |
| `asms://models/registry` | 静态 | 模型注册表 |
| `asms://models/compatibility` | 静态 | 兼容性矩阵 |
| `asms://engine/{name}` | 动态 | 引擎详情 |
| `asms://engines/status` | 静态 | 引擎状态总览 |
| `asms://inference/models` | 静态 | 可用推理模型 |
| `asms://resource/status` | 静态 | 资源状态 |
| `asms://resource/budget` | 静态 | 资源预算 |
| `asms://resource/allocations` | 静态 | 分配列表 |
| `asms://service/{id}` | 动态 | 服务详情 |
| `asms://services` | 静态 | 服务列表 |
| `asms://app/{id}` | 动态 | 应用详情 |
| `asms://apps/templates` | 静态 | 应用模板 |
| `asms://pipeline/{id}` | 动态 | 管道详情 |
| `asms://pipelines` | 静态 | 管道列表 |
| `asms://alerts/rules` | 静态 | 告警规则 |
| `asms://alerts/active` | 静态 | 活动告警 |
| `asms://remote/status` | 静态 | 远程状态 |
| `asms://remote/audit` | 静态 | 审计日志 |
| `asms://catalog/recipes` | 静态 | Recipe 列表 |
| `asms://catalog/recipe/{id}` | 动态 | Recipe 详情 |
| `asms://agent/status` | 静态 | Agent 状态 |
| `asms://agent/conversations` | 静态 | 活跃会话 |
| `asms://skills` | 静态 | Skills 列表 |
| `asms://skill/{id}` | 动态 | Skill 详情 |

#### 静态 vs 动态资源

- **静态资源**: URI 固定，直接注册到 Registry（`RegisterResource`）
- **动态资源**: URI 含占位符（如 `{id}`），通过 `ResourceFactory` 动态创建

```go
// 静态资源注册
registry.RegisterResource(NewEnginesStatusResource(store))

// 动态资源通过 Factory 创建
registry.RegisterResourceFactory(&ModelResourceFactory{store: store})
// 当请求 asms://model/llama3.2 时，Factory.Create("asms://model/llama3.2") 被调用
```

## Schema 定义

```go
type Schema struct {
    Type       string             `json:"type"`                 // object, array, string, number, boolean
    Properties map[string]Field   `json:"properties,omitempty"`
    Items      *Schema            `json:"items,omitempty"`      // for arrays
    Required   []string           `json:"required,omitempty"`
    Optional   []string           `json:"optional,omitempty"`
    
    // 文档
    Title       string            `json:"title,omitempty"`
    Description string            `json:"description,omitempty"`
    Examples    []any             `json:"examples,omitempty"`
    
    // 验证
    Min         *float64          `json:"min,omitempty"`
    Max         *float64          `json:"max,omitempty"`
    MinLength   *int              `json:"minLength,omitempty"`
    MaxLength   *int              `json:"maxLength,omitempty"`
    Pattern     string            `json:"pattern,omitempty"`
    Enum        []any             `json:"enum,omitempty"`
    Default     any               `json:"default,omitempty"`
}

type Field struct {
    Schema
    Name string `json:"name"`
}
```

---

## 原子单元定义

### 命名规范

```
{domain}.{action}

示例:
- model.pull
- model.list
- inference.chat
- engine.start
- resource.allocate
```

---

## 领域划分

### 1. Device Domain

硬件设备管理。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `device.detect` | 检测硬件设备 | `{}` | `{devices: [{id, name, vendor, type, memory}]}` |
| `device.set_power_limit` | 设置功耗限制 | `{device_id, limit_watts}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `device.info` | 获取设备信息 | `{device_id?}` | `{id, name, vendor, architecture, capabilities, memory}` |
| `device.metrics` | 获取实时指标 | `{device_id?, history?}` | `{utilization, temperature, power, memory_used, memory_total}` |
| `device.health` | 健康检查 | `{device_id?}` | `{status, issues: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://device/{id}/info` | 设备信息 |
| `asms://device/{id}/metrics` | 实时指标 |
| `asms://device/{id}/health` | 健康状态 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `device.detected` | 检测到新设备 | `{device}` |
| `device.health_changed` | 健康状态变化 | `{device_id, old_status, new_status}` |
| `device.metrics_alert` | 指标告警 | `{device_id, metric, value, threshold}` |

---

### 2. Model Domain

模型管理。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `model.create` | 创建模型记录 | `{name, type, source?, format?, path?}` | `{model_id}` |
| `model.delete` | 删除模型 | `{model_id, force?}` | `{success}` |
| `model.pull` | 从源拉取模型 | `{source, repo, tag?, mirror?}` | `{model_id, status}` |
| `model.import` | 导入本地模型 | `{path, name?, type?, auto_detect?}` | `{model_id}` |
| `model.verify` | 验证模型完整性 | `{model_id, checksum?}` | `{valid, issues: []}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `model.get` | 获取模型详情 | `{model_id}` | `{id, name, type, format, status, size, requirements}` |
| `model.list` | 列出模型 | `{type?, status?, format?, limit?, offset?}` | `{items: [], total}` |
| `model.search` | 搜索模型 | `{query, source?, type?, limit?}` | `{results: []}` |
| `model.estimate_resources` | 预估资源需求 | `{model_id}` | `{memory_min, memory_recommended, gpu_type}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://model/{id}` | 模型详情 |
| `asms://models/registry` | 模型注册表 |
| `asms://models/compatibility` | 兼容性矩阵 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `model.created` | 模型创建 | `{model}` |
| `model.deleted` | 模型删除 | `{model_id}` |
| `model.pull_progress` | 拉取进度 | `{model_id, progress, status}` |
| `model.verified` | 验证完成 | `{model_id, valid, issues}` |

---

### 3. Engine Domain

推理引擎管理。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `engine.start` | 启动引擎 | `{name, config?}` | `{process_id, status}` |
| `engine.stop` | 停止引擎 | `{name, force?, timeout?}` | `{success}` |
| `engine.restart` | 重启引擎 | `{name}` | `{process_id, status}` |
| `engine.install` | 安装引擎 | `{name, version?}` | `{success, path}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `engine.get` | 获取引擎信息 | `{name}` | `{name, type, status, version, capabilities, models: []}` |
| `engine.list` | 列出引擎 | `{type?, status?}` | `{items: []}` |
| `engine.features` | 获取引擎特性 | `{name}` | `{supports_streaming, supports_batch, max_concurrent, ...}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://engine/{name}` | 引擎详情 |
| `asms://engines/status` | 引擎状态 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `engine.started` | 引擎启动 | `{engine_name, process_id}` |
| `engine.stopped` | 引擎停止 | `{engine_name}` |
| `engine.error` | 引擎错误 | `{engine_name, error}` |
| `engine.health_changed` | 健康状态变化 | `{engine_name, status}` |

---

### 4. Inference Domain

推理服务。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `inference.chat` | 聊天补全 | `{model, messages, stream?, temperature?, max_tokens?, ...}` | `{content, finish_reason, usage}` |
| `inference.complete` | 文本补全 | `{model, prompt, stream?, ...}` | `{text, finish_reason, usage}` |
| `inference.embed` | 文本嵌入 | `{model, input}` | `{embeddings: [], usage}` |
| `inference.transcribe` | 语音转文字 | `{model, audio, language?}` | `{text, segments, language}` |
| `inference.synthesize` | 文字转语音 | `{model, text, voice?, stream?}` | `{audio, format, duration}` |
| `inference.generate_image` | 图像生成 | `{model, prompt, size?, steps?, ...}` | `{images: [], format}` |
| `inference.generate_video` | 视频生成 | `{model, prompt, duration?, ...}` | `{video, format, duration}` |
| `inference.rerank` | 重排序 | `{model, query, documents}` | `{results: []}` |
| `inference.detect` | 目标检测 | `{model, image}` | `{detections: []}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `inference.models` | 列出可用模型 | `{type?}` | `{models: []}` |
| `inference.voices` | 列出可用语音 | `{model?}` | `{voices: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://inference/models` | 可用模型列表 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `inference.request_started` | 请求开始 | `{request_id, model, type}` |
| `inference.request_completed` | 请求完成 | `{request_id, duration, tokens}` |
| `inference.request_failed` | 请求失败 | `{request_id, error}` |

---

### 5. Resource Domain

资源管理。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `resource.allocate` | 分配资源 | `{name, type, memory_bytes, gpu_fraction?, priority?}` | `{slot_id}` |
| `resource.release` | 释放资源 | `{slot_id}` | `{success}` |
| `resource.update_slot` | 更新槽位 | `{slot_id, memory_limit?, status?}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `resource.status` | 资源状态 | `{}` | `{memory, storage, slots: [], pressure}` |
| `resource.budget` | 资源预算 | `{}` | `{total, reserved, pools: {}}` |
| `resource.allocations` | 分配列表 | `{slot_id?, type?}` | `{allocations: []}` |
| `resource.can_allocate` | 检查是否可分配 | `{memory_bytes, priority?}` | `{can_allocate, reason?}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://resource/status` | 资源状态 |
| `asms://resource/budget` | 资源预算 |
| `asms://resource/allocations` | 分配列表 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `resource.allocated` | 资源分配 | `{slot_id, memory}` |
| `resource.released` | 资源释放 | `{slot_id}` |
| `resource.pressure_warning` | 资源压力警告 | `{pressure, threshold}` |
| `resource.preemption` | 资源抢占 | `{slot_id, reason}` |

---

### 6. Service Domain

模型服务化（长期运行的服务实例）。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `service.create` | 创建服务 | `{model_id, resource_class?, replicas?, persistent?}` | `{service_id}` |
| `service.delete` | 删除服务 | `{service_id}` | `{success}` |
| `service.scale` | 扩缩容 | `{service_id, replicas}` | `{success}` |
| `service.start` | 启动服务 | `{service_id}` | `{success}` |
| `service.stop` | 停止服务 | `{service_id, force?}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `service.get` | 获取服务详情 | `{service_id}` | `{id, model_id, status, replicas, endpoints, metrics}` |
| `service.list` | 列出服务 | `{status?, model_id?}` | `{services: []}` |
| `service.recommend` | 推荐配置 | `{model_id, hint?}` | `{resource_class, replicas, expected_throughput}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://service/{id}` | 服务详情 |
| `asms://services` | 服务列表 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `service.created` | 服务创建 | `{service}` |
| `service.scaled` | 服务扩缩容 | `{service_id, replicas}` |
| `service.failed` | 服务失败 | `{service_id, error}` |

---

### 7. App Domain

Docker 应用管理。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `app.install` | 安装应用 | `{template, name?, config?}` | `{app_id}` |
| `app.uninstall` | 卸载应用 | `{app_id, remove_data?}` | `{success}` |
| `app.start` | 启动应用 | `{app_id}` | `{success}` |
| `app.stop` | 停止应用 | `{app_id, timeout?}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `app.get` | 获取应用详情 | `{app_id}` | `{id, name, template, status, ports, volumes, metrics}` |
| `app.list` | 列出应用 | `{status?}` | `{apps: []}` |
| `app.logs` | 获取日志 | `{app_id, tail?, since?}` | `{logs: []}` |
| `app.templates` | 列出模板 | `{category?}` | `{templates: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://app/{id}` | 应用详情 |
| `asms://apps/templates` | 应用模板 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `app.installed` | 应用安装 | `{app}` |
| `app.started` | 应用启动 | `{app_id}` |
| `app.stopped` | 应用停止 | `{app_id}` |
| `app.oom_detected` | OOM 检测 | `{app_id}` |

---

### 8. Pipeline Domain

管道编排。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `pipeline.create` | 创建管道 | `{name, steps, config?}` | `{pipeline_id}` |
| `pipeline.delete` | 删除管道 | `{pipeline_id}` | `{success}` |
| `pipeline.run` | 运行管道 | `{pipeline_id, input, async?}` | `{run_id, status}` |
| `pipeline.cancel` | 取消运行 | `{run_id}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `pipeline.get` | 获取管道详情 | `{pipeline_id}` | `{id, name, steps, status, config}` |
| `pipeline.list` | 列出管道 | `{}` | `{pipelines: []}` |
| `pipeline.status` | 获取运行状态 | `{run_id}` | `{status, step_results, error?}` |
| `pipeline.validate` | 验证管道定义 | `{steps}` | `{valid, issues: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://pipeline/{id}` | 管道详情 |
| `asms://pipelines` | 管道列表 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `pipeline.started` | 管道开始运行 | `{run_id, pipeline_id}` |
| `pipeline.step_completed` | 步骤完成 | `{run_id, step_id, result}` |
| `pipeline.completed` | 管道完成 | `{run_id, result}` |
| `pipeline.failed` | 管道失败 | `{run_id, step_id, error}` |

---

### 9. Alert Domain

告警管理。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `alert.create_rule` | 创建告警规则 | `{name, condition, severity, channels?, cooldown?}` | `{rule_id}` |
| `alert.update_rule` | 更新规则 | `{rule_id, name?, condition?, enabled?}` | `{success}` |
| `alert.delete_rule` | 删除规则 | `{rule_id}` | `{success}` |
| `alert.acknowledge` | 确认告警 | `{alert_id}` | `{success}` |
| `alert.resolve` | 解决告警 | `{alert_id}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `alert.list_rules` | 列出规则 | `{enabled_only?}` | `{rules: []}` |
| `alert.history` | 告警历史 | `{rule_id?, status?, severity?, limit?}` | `{alerts: []}` |
| `alert.active` | 活动告警 | `{}` | `{alerts: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://alerts/rules` | 告警规则 |
| `asms://alerts/active` | 活动告警 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `alert.triggered` | 告警触发 | `{alert}` |
| `alert.acknowledged` | 告警确认 | `{alert_id}` |
| `alert.resolved` | 告警解决 | `{alert_id}` |

---

### 10. Remote Domain

远程操作。

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `remote.enable` | 启用远程访问 | `{provider, config?}` | `{tunnel_id, public_url}` |
| `remote.disable` | 禁用远程访问 | `{}` | `{success}` |
| `remote.exec` | 执行远程命令 | `{command, timeout?}` | `{stdout, stderr, exit_code}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `remote.status` | 远程状态 | `{}` | `{enabled, provider, public_url, uptime}` |
| `remote.audit` | 审计日志 | `{since?, limit?}` | `{records: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://remote/status` | 远程状态 |
| `asms://remote/audit` | 审计日志 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `remote.enabled` | 远程启用 | `{provider, url}` |
| `remote.disabled` | 远程禁用 | `{}` |
| `remote.command_executed` | 命令执行 | `{command, exit_code}` |

---

### 11. Catalog Domain

硬件最佳实践管理。将经过验证的"硬件配置 → 推理引擎镜像 → 可运行模型 → 最优参数"组合沉淀为 **Recipe（配方）**，用户可基于自己的硬件快速匹配并一键部署。

#### 设计理念

AIMA 的核心资产是在不同硬件上验证过的推理方案。Recipe 将这些方案结构化，使得：
1. **新用户零配置启动** — 检测硬件后自动匹配最佳 Recipe
2. **社区共享** — 用户可贡献和下载经验证的 Recipe
3. **AI Agent 可理解** — Recipe 的 YAML 格式对 AI Agent 完全透明

#### 核心类型

```go
// HardwareProfile 描述特定硬件配置
type HardwareProfile struct {
    GPUVendor   string   `json:"gpu_vendor" yaml:"gpu_vendor"`     // "NVIDIA", "AMD", "Apple", ""
    GPUModel    string   `json:"gpu_model" yaml:"gpu_model"`       // "RTX 4090", "A100", "GB10"
    GPUArch     string   `json:"gpu_arch" yaml:"gpu_arch"`         // "sm_89", "sm_121a"
    VRAMMinGB   int      `json:"vram_min_gb" yaml:"vram_min_gb"`   // 最低显存 (GB)
    CPUArch     string   `json:"cpu_arch" yaml:"cpu_arch"`         // "x86_64", "aarch64"
    OS          string   `json:"os" yaml:"os"`                     // "linux", "windows", "darwin"
    UnifiedMem  bool     `json:"unified_memory" yaml:"unified_memory"` // Apple Silicon / Jetson Thor
    Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Recipe 映射硬件到验证过的引擎+模型+配置组合
type Recipe struct {
    ID             string           `json:"id" yaml:"id"`
    Name           string           `json:"name" yaml:"name"`
    Description    string           `json:"description" yaml:"description"`
    Version        string           `json:"version" yaml:"version"`
    Author         string           `json:"author,omitempty" yaml:"author,omitempty"`
    Profile        HardwareProfile  `json:"profile" yaml:"profile"`
    Engine         RecipeEngine     `json:"engine" yaml:"engine"`
    Models         []RecipeModel    `json:"models" yaml:"models"`
    ResourceLimits ResourceLimits   `json:"resource_limits" yaml:"resource_limits"`
    Verified       bool             `json:"verified" yaml:"verified"`
    Tags           []string         `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// RecipeEngine 引擎镜像配置
type RecipeEngine struct {
    Type           string         `json:"type" yaml:"type"`           // "vllm", "ollama"
    Image          string         `json:"image" yaml:"image"`         // Docker 镜像
    FallbackImages []string       `json:"fallback_images,omitempty"`  // 备用镜像
    Config         map[string]any `json:"config,omitempty"`           // 引擎特定配置
}

// RecipeModel 模型下载配置
type RecipeModel struct {
    Name           string `json:"name" yaml:"name"`
    Source         string `json:"source" yaml:"source"`           // "ollama", "huggingface", "modelscope"
    Repo           string `json:"repo" yaml:"repo"`               // 仓库路径
    Tag            string `json:"tag,omitempty" yaml:"tag,omitempty"`
    Type           string `json:"type" yaml:"type"`               // "llm", "vlm", "asr"
    Format         string `json:"format,omitempty" yaml:"format,omitempty"` // "gguf", "safetensors"
    Mirror         string `json:"mirror,omitempty" yaml:"mirror,omitempty"` // 国内镜像源
    MemoryRequired int64  `json:"memory_required,omitempty"`      // 所需内存 (bytes)
}
```

#### Recipe YAML 示例

```yaml
# pkg/catalog/recipes/nvidia-rtx4090-llm.yaml
id: "nvidia-rtx4090-llm-vllm"
name: "NVIDIA RTX 4090 - LLM with vLLM"
description: "适用于 RTX 4090 (24GB VRAM) 的大语言模型推理方案"
version: "1.0.0"
author: "aima-community"

profile:
  gpu_vendor: "NVIDIA"
  gpu_model: "RTX 4090"
  gpu_arch: "sm_89"
  vram_min_gb: 24
  cpu_arch: "x86_64"
  os: "linux"
  unified_memory: false
  tags: ["consumer", "ada-lovelace"]

engine:
  type: "vllm"
  image: "vllm/vllm-openai:v0.15.0"
  config:
    gpu_memory_utilization: 0.90
    max_model_len: 32768
    tensor_parallel_size: 1
    quantization: "awq"

models:
  - name: "Qwen 2.5 7B AWQ"
    source: "huggingface"
    repo: "Qwen/Qwen2.5-7B-Instruct-AWQ"
    type: "llm"
    format: "safetensors"
    mirror: "modelscope"
    memory_required: 8000000000
  - name: "Llama 3.2 3B"
    source: "ollama"
    repo: "llama3.2:3b"
    type: "llm"
    format: "gguf"
    memory_required: 4000000000

resource_limits:
  gpu_memory_utilization: 0.90
  max_model_len: 32768
  tensor_parallel: 1

verified: true
tags: ["llm", "24gb", "consumer-gpu"]
```

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `catalog.create_recipe` | 创建/添加 Recipe | `{recipe (YAML/JSON)}` | `{recipe_id}` |
| `catalog.validate_recipe` | 验证 Recipe 格式正确性 | `{recipe (YAML/JSON)}` | `{valid, issues: []}` |
| `catalog.apply_recipe` | 一键部署：拉取引擎镜像 + 拉取模型 | `{recipe_id, skip_engine?, skip_models?}` | `{engine_ready, models: [{name, status}]}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `catalog.match` | 基于当前硬件匹配最佳 Recipe | `{gpu_vendor?, tags?, limit?}` | `{recipes: [{recipe, score}]}` |
| `catalog.list` | 列出所有 Recipe | `{tags?, gpu_vendor?, verified_only?}` | `{recipes: [], total}` |
| `catalog.get` | 获取特定 Recipe | `{recipe_id}` | `{recipe}` |
| `catalog.check_status` | 检查 Recipe 所需制品是否本地已有 | `{recipe_id}` | `{engine_ready, models_ready: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://catalog/recipes` | 所有 Recipe 列表 |
| `asms://catalog/recipe/{id}` | 特定 Recipe 详情 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `catalog.recipe_created` | Recipe 创建 | `{recipe_id, name}` |
| `catalog.recipe_matched` | Recipe 匹配成功 | `{recipe_id, hardware_profile, score}` |
| `catalog.recipe_applied` | Recipe 部署完成 | `{recipe_id, engine_ready, models_ready}` |

#### 硬件匹配算法

`catalog.match` 使用评分制匹配当前硬件与 Recipe：

```
匹配流程:
1. HAL Provider.Detect() -> 获取当前硬件信息 (HardwareInfo)
2. 遍历所有 Recipe 的 HardwareProfile
3. 评分规则:
   - GPU Vendor 完全匹配: +40 分
   - GPU Model 完全匹配: +30 分
   - GPU Arch 兼容: +15 分
   - VRAM 满足最低要求: +10 分
   - OS 匹配: +5 分
4. 按分数降序排列，返回 Top N
```

#### 存储方式

- **内置 Recipe**: 通过 `//go:embed` 编译进二进制（与 workflow templates 相同模式）
- **用户 Recipe**: 存储在 `~/.aima/recipes/` 目录，YAML 格式
- **Recipe 注册表**: 运行时合并内置和用户 Recipe

---

### 12. Agent Domain

AI 代理操作系统。提供一个可配置的 AI Agent，代替用户操作 AIMA 平台。Agent 通过 LLM（如 Claude Haiku、GPT-4o、本地 Ollama 模型）驱动，使用 AIMA 的 MCP Tools 作为其能力集。

#### 设计理念

AIMA 的 MCP 适配器已经将所有原子单元自动生成为 Tool 定义。Agent 是一个坐在 MCP Tools 之上的对话循环：接收用户自然语言 → LLM 思考 → 调用 AIMA Tools → 返回结果。类似于 Claude Code 对代码仓库的操作，但 Agent 只对 AIMA 的原子单元有权限。

```
┌────────────────────────────────────────────────────────────────┐
│                        Agent Operator                          │
│                                                                │
│  用户消息 ──► LLM (Claude/GPT/Ollama) ──► Tool Calls          │
│                      ▲                       │                 │
│                      │                       ▼                 │
│              Tool Results ◄── MCPAdapter.ExecuteTool()         │
│                                                                │
│  System Prompt = 基础描述 + 活跃 Skills + Tool 列表            │
└────────────────────────────────────────────────────────────────┘
```

#### 核心类型

```go
// LLMClient 抽象不同 LLM 提供商的通信接口
type LLMClient interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDef, opts ChatOptions) (*ChatResponse, error)
    Name() string
    ModelName() string
}

// 支持的 LLM 提供商:
// - Anthropic Claude (anthropic.go) — claude-haiku / claude-sonnet
// - OpenAI 兼容 (openai.go)        — GPT-4o / 本地 OpenAI API 服务器
// - Ollama (ollama.go)              — 本地模型

// Message 对话消息
type Message struct {
    Role       string     `json:"role"`         // "system", "user", "assistant", "tool"
    Content    string     `json:"content"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Agent 核心 AI 代理
type Agent struct {
    llm           LLMClient
    mcpAdapter    *gateway.MCPAdapter     // 复用现有 MCP 工具
    skillRegistry *skill.SkillRegistry    // 技能注册表
    conversations map[string]*Conversation
}
```

#### Agent 对话循环

```go
func (a *Agent) Chat(ctx, conversationID, userMessage string) (string, error) {
    // 1. 获取或创建会话
    conv := a.getOrCreateConversation(conversationID)
    conv.Messages = append(conv.Messages, Message{Role: "user", Content: userMessage})

    // 2. 构建 System Prompt (基础描述 + Skills + Tool 列表)
    systemPrompt := a.buildSystemPrompt()

    // 3. 获取可用工具 (从 MCP 适配器自动生成)
    toolDefs := a.mcpToolsToLLMTools(a.mcpAdapter.GenerateToolDefinitions())

    // 4. 对话循环 (可能涉及多次 Tool 调用)
    for {
        response := a.llm.Chat(ctx, conv.Messages, toolDefs, opts)
        conv.Messages = append(conv.Messages, response.Message)

        if len(response.ToolCalls) == 0 {
            return response.Message.Content, nil  // 无 Tool 调用，返回文本
        }

        // 5. 执行每个 Tool 调用 (通过 MCP 适配器)
        for _, tc := range response.ToolCalls {
            result := a.mcpAdapter.ExecuteTool(ctx, tc.Name, tc.Arguments)
            conv.Messages = append(conv.Messages, Message{
                Role: "tool", Content: result, ToolCallID: tc.ID,
            })
        }
        // 继续循环: LLM 看到 Tool 结果后可能调用更多工具或直接回复
    }
}
```

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `agent.chat` | 向 AI Agent 发送消息并获取回复 | `{message, conversation_id?}` | `{response, conversation_id, tool_calls_count}` |
| `agent.reset` | 重置/清空会话 | `{conversation_id}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `agent.status` | 获取 Agent 状态 | `{}` | `{enabled, provider, model, active_conversations}` |
| `agent.history` | 获取会话历史 | `{conversation_id, limit?}` | `{messages: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://agent/status` | Agent 状态 |
| `asms://agent/conversations` | 活跃会话列表 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `agent.message_sent` | Agent 发送回复 | `{conversation_id, message_length}` |
| `agent.tool_called` | Agent 调用了 Tool | `{conversation_id, tool_name, success}` |
| `agent.conversation_created` | 新会话创建 | `{conversation_id}` |

#### CLI 集成

```bash
# 交互式对话模式
aima agent chat
# > 帮我在当前硬件上部署一个 LLM
# Agent: 让我先检查你的硬件配置...
# [tool_call: device_detect]
# Agent: 你有一块 RTX 4090，24GB 显存。我找到了 3 个匹配的方案...

# 单条问题
aima agent ask "当前安装了哪些模型？"

# 查看状态
aima agent status
```

#### 安全约束

- Agent 的 Tool 调用通过 `MCPAdapter.ExecuteTool()` 执行，与外部 MCP 客户端走相同路径
- Auth 中间件同样生效：Agent 的操作受制于调用者的认证级别
- Agent 不能直接调用原子单元，必须经过 Gateway 路径

---

### 13. Skill Domain

AI Agent 技能知识库。Skills 是结构化的最佳实践文档，以 Markdown + YAML Front-matter 格式存储，加载到 Agent 的 System Prompt 中指导其决策。

#### 设计理念

Agent 需要领域知识才能有效操作 AIMA。Skills 解决了"Agent 知道有哪些 Tool，但不知道什么场景下用哪些 Tool、用什么参数"的问题。

```
Skills 类比:
- AIMA Tools = Agent 的"手脚"（能做什么）
- Skills     = Agent 的"经验"（什么时候做、怎么做）
```

Skills 可以是：
1. **内置 (builtin)** — 随 AIMA 二进制发布，覆盖基础场景
2. **用户自建 (user)** — 用户根据自己的使用场景创建
3. **社区共享 (community)** — 从社区下载的经验集

#### Skill 文件格式

使用 YAML Front-matter + Markdown 正文（与 Hugo/Jekyll 相同模式）：

```markdown
---
id: setup-llm
name: "在新硬件上部署 LLM"
category: setup
description: "指导 Agent 在新机器上设置 LLM 推理服务"
trigger:
  keywords: ["setup", "install", "configure", "部署", "安装"]
  tool_names: ["catalog_match", "catalog_apply_recipe"]
  always_on: false
priority: 10
enabled: true
source: builtin
---

# 在新硬件上部署 LLM

当用户要求设置 LLM 推理时，按以下步骤操作：

## 步骤 1: 检测硬件
调用 `device_detect` 获取 GPU、CPU、内存信息。

## 步骤 2: 匹配 Recipe
调用 `catalog_match` 查找匹配的 Recipe。展示前 3 个选项给用户。

## 步骤 3: 部署 Recipe
用户选择后，调用 `catalog_apply_recipe` 一键部署。

## 步骤 4: 验证
调用 `inference_chat` 发送测试消息确认一切正常。

## 常见问题
- 如果 `catalog_match` 无结果，改为手动推荐合适的引擎和模型
- 如果 Docker pull 失败，检查是否配置了国内镜像
```

#### 核心类型

```go
// Skill 表示一个可加载到 Agent 上下文中的知识单元
type Skill struct {
    ID          string       `json:"id" yaml:"id"`
    Name        string       `json:"name" yaml:"name"`
    Category    string       `json:"category" yaml:"category"`       // setup, troubleshoot, optimize, manage
    Description string       `json:"description" yaml:"description"`
    Trigger     SkillTrigger `json:"trigger" yaml:"trigger"`
    Content     string       `json:"content" yaml:"content"`         // Markdown 正文
    Priority    int          `json:"priority" yaml:"priority"`       // 优先级 (越高越重要)
    Enabled     bool         `json:"enabled" yaml:"enabled"`
    Source      string       `json:"source" yaml:"source"`           // "builtin", "user", "community"
}

// SkillTrigger 定义 Skill 的激活条件
type SkillTrigger struct {
    Keywords  []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`     // 用户消息含这些词时激活
    ToolNames []string `json:"tool_names,omitempty" yaml:"tool_names,omitempty"` // 涉及这些 Tool 时激活
    AlwaysOn  bool     `json:"always_on,omitempty" yaml:"always_on,omitempty"`   // 始终加载到 System Prompt
}
```

#### Agent 如何使用 Skills

```go
func (a *Agent) buildSystemPrompt() string {
    // 1. 基础描述: "你是 AIMA AI Agent，负责管理 AI 推理基础设施..."
    // 2. Tool 列表摘要: 从 MCPAdapter 获取
    // 3. 始终加载的 Skills (always_on=true)
    // 4. 触发式 Skills: 分析最近对话，匹配 keywords/tool_names
    // 拼接为完整 System Prompt
}
```

#### Commands

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `skill.add` | 添加新 Skill | `{path \| content, source?}` | `{skill_id}` |
| `skill.remove` | 移除用户 Skill | `{skill_id}` | `{success}` |
| `skill.enable` | 启用 Skill | `{skill_id}` | `{success}` |
| `skill.disable` | 禁用 Skill | `{skill_id}` | `{success}` |

#### Queries

| 名称 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `skill.list` | 列出 Skills | `{category?, source?, enabled_only?}` | `{skills: [], total}` |
| `skill.get` | 获取 Skill 详情 | `{skill_id}` | `{skill}` |
| `skill.search` | 搜索 Skills | `{query, category?}` | `{skills: []}` |

#### Resources

| URI | 描述 |
|-----|------|
| `asms://skills` | 所有 Skills 列表 |
| `asms://skill/{id}` | 特定 Skill 详情 |

#### Events

| 类型 | 描述 | 载荷 |
|------|------|------|
| `skill.added` | Skill 添加 | `{skill_id, name, source}` |
| `skill.removed` | Skill 移除 | `{skill_id}` |
| `skill.enabled` | Skill 启用 | `{skill_id}` |
| `skill.disabled` | Skill 禁用 | `{skill_id}` |

#### 内置 Skills

| Skill ID | 类别 | 说明 |
|----------|------|------|
| `setup-llm` | setup | 在新硬件上部署 LLM 推理 |
| `troubleshoot-gpu` | troubleshoot | GPU 问题排查 |
| `optimize-inference` | optimize | 推理性能优化 |
| `manage-models` | manage | 模型管理最佳实践 |
| `recipe-advisor` | setup | Recipe 选择和自定义指导 |

#### 存储方式

- **内置 Skills**: `skills/builtin/*.md`，通过 `//go:embed` 编译进二进制
- **用户 Skills**: `~/.aima/skills/*.md`，文件系统直接管理
- **加载顺序**: 内置 → 用户，同 ID 用户版本覆盖内置版本

---

## 外部仓库集成

### Registry Provider 抽象

Engine Docker 镜像来自 DockerHub、GHCR 等容器仓库，模型来自 HuggingFace、ModelScope 等模型仓库。`RegistryProvider` 统一抽象镜像仓库操作。

```go
// RegistryProvider 抽象 Docker 镜像仓库操作
type RegistryProvider interface {
    Name() string
    PullImage(ctx context.Context, image string, progress chan<- PullProgress) error
    ImageExists(ctx context.Context, image string) (bool, error)
}

// 实现:
// - DockerHub (dockerhub.go)
// - GitHub Container Registry (ghcr.go)
```

### 国内镜像支持

中国用户访问 DockerHub 和 HuggingFace 可能受限。通过配置镜像源自动替换：

```toml
[registry]
region = "cn"  # 启用国内镜像

[[registry.mirrors]]
source = "docker.io"
mirror = "registry.cn-hangzhou.aliyuncs.com"
region = "cn"

[[registry.mirrors]]
source = "huggingface.co"
mirror = "hf-mirror.com"
region = "cn"
```

Recipe 中的 `mirror` 字段支持模型从 ModelScope 下载（替代 HuggingFace）：

```yaml
models:
  - name: "Qwen 2.5 7B"
    source: "huggingface"
    repo: "Qwen/Qwen2.5-7B-Instruct-AWQ"
    mirror: "modelscope"  # region=cn 时自动使用 ModelScope
```

### 一键部署数据流

```
catalog.apply_recipe(recipe_id):
  1. 加载 Recipe
  2. 检查引擎镜像:
     RegistryProvider.ImageExists(recipe.Engine.Image)?
     -> 否: RegistryProvider.PullImage(image)  // 自动选择镜像源
     -> 失败: 依次尝试 FallbackImages
  3. 拉取模型 (对每个 recipe.Models):
     -> 调用 model.pull(source, repo, tag)     // 复用现有原子单元
     -> region=cn 且有 mirror: 替换 source 为 mirror
  4. 返回部署状态
```

---

## 服务层

服务层聚合多个原子单元，提供更高级别的业务逻辑。

### ModelService

```go
type ModelService struct {
    commands   *UnitRegistry  // model.* commands
    queries    *UnitRegistry  // model.* queries
    store      ModelRepository
    downloader Downloader
    bus        *EventBus
}

// 聚合业务方法 (内部调用原子单元)
func (s *ModelService) PullAndVerify(ctx context.Context, source, repo string) (*Model, error) {
    // 1. 拉取模型
    result, err := s.commands.Execute(ctx, "model.pull", map[string]any{
        "source": source,
        "repo":   repo,
    })
    if err != nil {
        return nil, err
    }
    
    modelID := result.(map[string]any)["model_id"].(string)
    
    // 2. 验证模型
    _, err = s.commands.Execute(ctx, "model.verify", map[string]any{
        "model_id": modelID,
    })
    if err != nil {
        // 回滚：删除模型
        s.commands.Execute(ctx, "model.delete", map[string]any{"model_id": modelID})
        return nil, err
    }
    
    // 3. 返回模型详情
    return s.queries.Execute(ctx, "model.get", map[string]any{"model_id": modelID})
}
```

### InferenceService

```go
type InferenceService struct {
    modelService    *ModelService
    engineService   *EngineService
    resourceService *ResourceService
    router          *InferenceRouter
}

func (s *InferenceService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // 1. 获取模型信息
    model, err := s.modelService.Get(ctx, req.Model)
    if err != nil {
        return nil, err
    }
    
    // 2. 选择引擎
    engine := s.router.SelectEngine(model.Type, model.Format)
    
    // 3. 检查资源
    canAlloc, _ := s.resourceService.CanAllocate(ctx, model.Requirements.MemoryMin)
    if !canAlloc {
        return nil, ErrInsufficientResources
    }
    
    // 4. 执行推理
    return s.engineService.Chat(ctx, engine, req)
}
```

---

## 编排层

### Pipeline DSL

```yaml
# 示例: 语音助手 Pipeline
name: voice_assistant
description: 语音输入 → ASR → LLM → TTS → 音频输出

config:
  llm_model: "llama3.2"
  tts_model: "tts-1"
  voice: "alloy"

steps:
  - id: transcribe
    type: inference.transcribe
    input:
      model: "whisper-large-v3"
      audio: "${input.audio}"
    output: text
  
  - id: chat
    type: inference.chat
    input:
      model: "${config.llm_model}"
      messages:
        - role: user
          content: "${steps.transcribe.text}"
    output: response
  
  - id: synthesize
    type: inference.synthesize
    input:
      model: "${config.tts_model}"
      text: "${steps.chat.response.content}"
      voice: "${config.voice}"
    output: audio

output:
  text: "${steps.transcribe.text}"
  response: "${steps.chat.response.content}"
  audio: "${steps.synthesize.audio}"
```

### Workflow Engine

```go
type WorkflowEngine struct {
    registry *UnitRegistry
    store    WorkflowStore
    bus      *EventBus
}

func (e *WorkflowEngine) Execute(ctx context.Context, def *WorkflowDef, input any) (*Result, error) {
    // 1. 验证 DAG
    if err := def.Validate(); err != nil {
        return nil, err
    }
    
    // 2. 拓扑排序
    order := topologicalSort(def.Steps)
    
    // 3. 构建执行上下文
    ctx = context.WithValue(ctx, "input", input)
    ctx = context.WithValue(ctx, "config", def.Config)
    
    // 4. 按顺序执行步骤
    for _, step := range order {
        // 解析输入变量
        resolvedInput := resolveVariables(step.Input, ctx)
        
        // 获取原子单元
        unit := e.registry.Get(step.Type)
        
        // 执行
        result, err := unit.Execute(ctx, resolvedInput)
        if err != nil {
            // 失败处理
            if step.OnFailure == "abort" {
                return nil, err
            }
            // 继续或重试
        }
        
        // 更新上下文
        ctx = context.WithValue(ctx, "steps."+step.ID, result)
        
        // 发布事件
        e.bus.Publish(Event{
            Type: "workflow.step_completed",
            Payload: map[string]any{
                "run_id":  ctx.Value("run_id"),
                "step_id": step.ID,
            },
        })
    }
    
    // 5. 组装输出
    return assembleOutput(def.Output, ctx)
}
```

### 预构建模板

```
templates/
├── voice_assistant.yaml      # 语音助手
├── rag_pipeline.yaml         # RAG 问答
├── batch_inference.yaml      # 批量推理
├── multimodal_chat.yaml      # 多模态对话
└── video_analysis.yaml       # 视频分析
```

---

## 统一入口 Gateway

### 请求/响应格式

```go
// 统一请求格式
type Request struct {
    Type    string         `json:"type"`    // "command" | "query" | "resource" | "workflow"
    Unit    string         `json:"unit"`    // "model.pull" | "inference.chat"
    Input   map[string]any `json:"input"`
    Options RequestOptions `json:"options"` // timeout, async, trace_id, etc.
}

// 统一响应格式
type Response struct {
    Success bool           `json:"success"`
    Data    any            `json:"data,omitempty"`
    Error   *ErrorInfo     `json:"error,omitempty"`
    Meta    *ResponseMeta  `json:"meta,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`     // "MODEL_NOT_FOUND", "INSUFFICIENT_RESOURCES"
    Message string `json:"message"`
    Details any    `json:"details,omitempty"`
}

type ResponseMeta struct {
    RequestID  string        `json:"request_id"`
    Duration   time.Duration `json:"duration_ms"`
    TraceID    string        `json:"trace_id,omitempty"`
    Pagination *Pagination   `json:"pagination,omitempty"`
}
```

### Gateway 实现

```go
type Gateway struct {
    registry      *UnitRegistry
    services      *ServiceLayer
    workflowEngine *WorkflowEngine
    bus           *EventBus
}

func (g *Gateway) Handle(ctx context.Context, req *Request) (*Response, error) {
    start := time.Now()
    requestID := generateRequestID()
    
    // 记录请求开始
    g.bus.Publish(Event{
        Type: "gateway.request_started",
        Payload: map[string]any{
            "request_id": requestID,
            "type":       req.Type,
            "unit":       req.Unit,
        },
    })
    
    var result any
    var err error
    
    switch req.Type {
    case "command":
        unit := g.registry.GetCommand(req.Unit)
        if unit == nil {
            err = ErrUnitNotFound
            break
        }
        result, err = unit.Execute(ctx, req.Input)
        
    case "query":
        unit := g.registry.GetQuery(req.Unit)
        if unit == nil {
            err = ErrUnitNotFound
            break
        }
        result, err = unit.Execute(ctx, req.Input)
        
    case "resource":
        resource := g.registry.GetResource(req.Unit)
        if resource == nil {
            err = ErrResourceNotFound
            break
        }
        result, err = resource.Get(ctx)
        
    case "workflow":
        result, err = g.workflowEngine.Run(ctx, req.Unit, req.Input)
    }
    
    // 构建响应
    resp := &Response{
        Success: err == nil,
        Data:    result,
        Meta: &ResponseMeta{
            RequestID: requestID,
            Duration:  time.Since(start),
        },
    }
    
    if err != nil {
        resp.Error = toErrorInfo(err)
    }
    
    return resp, nil
}
```

### 适配器

#### HTTP 适配器

```go
// POST /api/v2/execute
func (g *Gateway) HTTPHandler(w http.ResponseWriter, r *http.Request) {
    var req Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, ErrInvalidRequest)
        return
    }
    
    resp, _ := g.Handle(r.Context(), &req)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

// RESTful 风格映射
// POST /api/v2/models/pull -> {type: "command", unit: "model.pull", input: body}
```

#### MCP 适配器

```go
// MCP Tool 定义自动生成
func (g *Gateway) MCPToolDefinitions() []ToolDefinition {
    var tools []ToolDefinition
    
    // 从所有 Command 和 Query 生成
    for _, cmd := range g.registry.ListCommands() {
        tools = append(tools, ToolDefinition{
            Name:        strings.ReplaceAll(cmd.Name(), ".", "_"),
            Description: cmd.Description(),
            InputSchema: cmd.InputSchema(),
        })
    }
    
    return tools
}

// MCP Tool 执行
func (g *Gateway) MCPToolHandler(toolName string, params json.RawMessage) (any, error) {
    // 转换 tool_name -> unit.name
    unitName := strings.ReplaceAll(toolName, "_", ".")
    
    req := &Request{
        Type:  "command", // 或 query
        Unit:  unitName,
        Input: parseInput(params),
    }
    
    resp, err := g.Handle(context.Background(), req)
    if err != nil {
        return nil, err
    }
    if !resp.Success {
        return nil, resp.Error
    }
    return resp.Data, nil
}
```

#### CLI 适配器

```go
// 统一 CLI 格式
// aima exec <unit> [flags]
// aima exec model.pull --source ollama --repo llama3.2
// aima exec inference.chat --model llama3.2 --message "Hello"

func (g *Gateway) CLIHandler(cmd *cobra.Command, args []string) {
    unit := cmd.Annotations["unit"]
    input := extractInputFromFlags(cmd, args)
    
    reqType := "command"
    if cmd.Annotations["type"] == "query" {
        reqType = "query"
    }
    
    req := &Request{
        Type:  reqType,
        Unit:  unit,
        Input: input,
    }
    
    resp, _ := g.Handle(cmd.Context(), req)
    
    if resp.Success {
        printOutput(resp.Data)
    } else {
        printError(resp.Error)
        os.Exit(1)
    }
}
```

---


### 中间件架构

Gateway 采用洋葱模型（Onion Model）的中间件链处理请求。

#### 执行顺序

```
请求 --> Recovery --> Logging --> Auth --> RateLimit --> CORS --> Handler
响应 <-- Recovery <-- Logging <-- Auth <-- RateLimit <-- CORS <-- Handler
```

#### 标准中间件

| 中间件 | 职责 | 文件 |
|--------|------|------|
| Recovery | panic 恢复，返回 500 | `pkg/gateway/middleware/recovery.go` |
| Logging | 请求/响应日志 | `pkg/gateway/middleware/logging.go` |
| Auth | API Key 认证 | 待实现 |
| RateLimit | 请求频率限制 | `pkg/infra/ratelimit/ratelimit.go` |
| CORS | 跨域支持 | `pkg/gateway/middleware/cors.go` |

#### 中间件接口

```go
type Middleware func(http.Handler) http.Handler

// 使用方式
handler := Recovery(Logging(Auth(RateLimit(CORS(actualHandler)))))
```

## 项目结构

```
ai-inference-managed-by-ai/
├── cmd/
│   └── aima/
│       └── main.go              # 单二进制入口
│
├── pkg/
│   ├── unit/                    # 原子单元 (核心, 13 个领域)
│   │   ├── types.go             # Command/Query/Event/Resource 接口
│   │   ├── registry.go          # 单元注册表
│   │   ├── schema.go            # Schema 定义和验证
│   │   ├── context.go           # 执行上下文
│   │   │
│   │   ├── device/              # Device 领域 — 硬件设备管理
│   │   ├── model/               # Model 领域 — 模型管理
│   │   ├── engine/              # Engine 领域 — 推理引擎管理
│   │   ├── inference/           # Inference 领域 — 推理服务
│   │   ├── resource/            # Resource 领域 — 资源管理
│   │   ├── service/             # Service 领域 — 模型服务化
│   │   ├── app/                 # App 领域 — Docker 应用
│   │   ├── pipeline/            # Pipeline 领域 — 管道编排
│   │   ├── alert/               # Alert 领域 — 告警管理
│   │   ├── remote/              # Remote 领域 — 远程操作
│   │   ├── catalog/             # Catalog 领域 — 硬件最佳实践 ★ NEW
│   │   ├── agent/               # Agent 领域 — AI 代理操作 ★ NEW
│   │   └── skill/               # Skill 领域 — 技能知识库 ★ NEW
│   │
│   ├── agent/                   # AI Agent 核心实现 ★ NEW
│   │   ├── agent.go             # Agent 对话循环
│   │   ├── types.go             # Agent/Message/Conversation 类型
│   │   ├── tools.go             # MCP Tool → LLM Tool 转换桥
│   │   ├── conversation.go      # 会话状态管理
│   │   ├── store.go             # ConversationStore 接口
│   │   ├── llm/                 # LLM 客户端适配器
│   │   │   ├── types.go         # LLMClient 接口
│   │   │   ├── anthropic.go     # Anthropic Claude
│   │   │   ├── openai.go        # OpenAI 兼容
│   │   │   └── ollama.go        # 本地 Ollama
│   │   └── skill/               # 技能加载和管理
│   │       ├── types.go         # Skill/SkillTrigger 类型
│   │       ├── loader.go        # 加载内置+用户 Skills
│   │       ├── registry.go      # SkillRegistry
│   │       └── prompt.go        # System Prompt 构建
│   │
│   ├── catalog/                 # Recipe 管理 ★ NEW
│   │   ├── recipes/             # 内置 Recipe YAML (//go:embed)
│   │   │   ├── nvidia-rtx4090-llm.yaml
│   │   │   ├── nvidia-a100-llm.yaml
│   │   │   ├── apple-m-series-llm.yaml
│   │   │   └── generic-cpu-llm.yaml
│   │   ├── loader.go            # Recipe 加载器
│   │   └── matcher.go           # 硬件-Recipe 评分匹配
│   │
│   ├── service/                 # 服务层 (聚合)
│   │   ├── model_service.go
│   │   ├── inference_service.go
│   │   ├── engine_service.go
│   │   ├── resource_service.go
│   │   ├── device_service.go
│   │   ├── app_service.go
│   │   ├── pipeline_service.go
│   │   ├── alert_service.go
│   │   ├── remote_service.go
│   │   └── catalog_service.go   # Recipe 部署编排 ★ NEW
│   │
│   ├── workflow/                # 编排层
│   │   ├── engine.go            # 工作流引擎
│   │   ├── dsl.go               # DSL 解析
│   │   ├── validator.go         # DAG 验证
│   │   ├── executor.go          # 步骤执行器
│   │   └── templates/           # 预构建模板
│   │
│   ├── gateway/                 # 统一入口
│   │   ├── gateway.go           # 核心 Gateway
│   │   ├── http_adapter.go      # HTTP 适配
│   │   ├── mcp_adapter.go       # MCP 协议适配
│   │   ├── mcp_tools.go         # MCP Tool 自动生成
│   │   ├── grpc_adapter.go      # gRPC 适配
│   │   └── middleware/          # 中间件 (auth, logging, cors, recovery)
│   │
│   ├── infra/                   # 基础设施层
│   │   ├── hal/                 # 硬件抽象
│   │   ├── store/               # 数据存储
│   │   ├── eventbus/            # 事件总线
│   │   ├── docker/              # Docker 客户端
│   │   ├── metrics/             # Prometheus 指标
│   │   ├── network/             # 网络工具
│   │   ├── ratelimit/           # 速率限制
│   │   └── provider/            # 外部集成
│   │       ├── ollama/          # Ollama 推理引擎
│   │       ├── vllm/            # vLLM 推理引擎
│   │       ├── huggingface/     # HuggingFace 模型仓库
│   │       ├── modelscope/      # ModelScope 模型仓库
│   │       ├── registry/        # Docker 镜像仓库 ★ NEW
│   │       │   ├── types.go     # RegistryProvider 接口
│   │       │   ├── dockerhub.go # DockerHub 实现
│   │       │   └── mirror.go    # 国内镜像支持
│   │       ├── hybrid_engine_provider.go
│   │       ├── multi_engine_provider.go
│   │       └── docker_engine_provider.go
│   │
│   ├── config/                  # 配置管理
│   ├── registry/                # 注册工具
│   └── cli/                     # CLI 命令行
│       ├── root.go
│       ├── device.go, model.go, engine.go, inference.go
│       ├── exec.go, workflow.go, mcp.go, service.go
│       ├── agent.go             # Agent 交互命令 ★ NEW
│       ├── start.go
│       └── version.go
│
├── skills/                      # 内置 Skills (//go:embed) ★ NEW
│   └── builtin/
│       ├── setup-llm.md
│       ├── troubleshoot-gpu.md
│       ├── optimize-inference.md
│       ├── manage-models.md
│       └── recipe-advisor.md
│
├── configs/
│   └── aima.toml               # 默认配置
│
├── docs/
│   └── ARCHITECTURE.md         # 本文档
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---


---

## 生命周期管理

### 启动顺序

```
main()
  |
  v
1. 加载配置 (config.Load)
  |
  v
2. 初始化基础设施
  +-- Store (SQLite / Memory)
  +-- EventBus (InMemory / Persistent)
  +-- HAL Provider (NVIDIA / Generic)
  +-- Docker Client
  +-- Registry Provider (DockerHub + Mirrors)    ★ NEW
  |
  v
3. 创建 Registry
  |
  v
4. 注册原子单元（所有 13 个域的 Commands/Queries/Resources）
  +-- 10 个原有领域
  +-- Catalog 领域 (加载内置+用户 Recipes)     ★ NEW
  +-- Skill 领域 (加载内置+用户 Skills)         ★ NEW
  +-- Agent 领域                                 ★ NEW
  |
  v
5. 创建 Service 层
  +-- CatalogService (编排 Recipe 部署)          ★ NEW
  |
  v
6. 创建 Gateway
  |
  v
7. 初始化 AI Agent (如果 agent.enabled=true)    ★ NEW
  +-- 创建 LLM Client (Anthropic/OpenAI/Ollama)
  +-- 加载 SkillRegistry
  +-- 注册 Agent 到 Gateway
  |
  v
8. 启动 Adapters
  +-- HTTP Server (:9090)
  +-- MCP Server (stdio/SSE)
  +-- gRPC Server (:50051)
  +-- CLI (如果是 CLI 模式)
  |
  v
9. 系统就绪 -> 接受请求
```

### 优雅关闭

```
收到 SIGTERM/SIGINT
  |
  v
1. 停止接受新请求
  |
  v
2. 等待进行中的请求完成 (最长 30s)
  |
  v
3. 关闭 Adapters (HTTP/MCP/gRPC)
  |
  v
4. 关闭基础设施 (EventBus/Store/Docker)
  |
  v
5. 退出
```

### 健康检查

| 类型 | 端点 | 说明 |
|------|------|------|
| Liveness | `GET /api/v2/health` | 进程是否存活 |
| Readiness | `GET /api/v2/health?ready=true` | 是否可接受请求（检查 Provider、Store）|

```json
{
  "status": "ready",
  "checks": {
    "store": "ok",
    "ollama": "ok",
    "nvidia_smi": "ok"
  },
  "uptime_seconds": 3600
}
```


---

## 错误处理架构

### 错误分层

```
+--------------------------------------+
|  系统错误 (System Error)              |  panic, OOM, 信号
+--------------------------------------+
|  基础设施错误 (Infra Error)           |  数据库断连, 网络超时, Provider 不可用
+--------------------------------------+
|  领域错误 (Domain Error)              |  模型不存在, 资源不足, 输入无效
+--------------------------------------+
```

### 错误码体系

| 错误码 | 层级 | HTTP | 说明 |
|--------|------|------|------|
| `INVALID_REQUEST` | 领域 | 400 | 请求格式错误 |
| `VALIDATION_ERROR` | 领域 | 400 | 参数验证失败 |
| `UNIT_NOT_FOUND` | 领域 | 404 | 原子单元不存在 |
| `MODEL_NOT_FOUND` | 领域 | 404 | 模型不存在 |
| `RESOURCE_NOT_FOUND` | 领域 | 404 | 资源不存在 |
| `UNAUTHORIZED` | 领域 | 401 | 未认证 |
| `FORBIDDEN` | 领域 | 403 | 无权限 |
| `RATE_LIMITED` | 基础设施 | 429 | 请求频率超限 |
| `INSUFFICIENT_RESOURCES` | 领域 | 503 | GPU/内存不足 |
| `ENGINE_NOT_RUNNING` | 基础设施 | 503 | 推理引擎未运行 |
| `EXECUTION_FAILED` | 基础设施 | 500 | 执行异常 |
| `TIMEOUT` | 基础设施 | 504 | 执行超时 |
| `INTERNAL_ERROR` | 系统 | 500 | 内部错误 |

### 错误传播策略

```
原子单元 -> Service 层 (wrap context) -> Gateway (ToErrorInfo, 映射码) -> Adapter (格式化)
```

每一层用 `fmt.Errorf("context: %w", err)` 包装上下文，Gateway 层统一转换为 ErrorInfo。

### 重试与幂等性

| 操作类型 | 安全重试？ | 说明 |
|----------|-----------|------|
| Query | 总是安全 | 无副作用 |
| Command (创建类) | 需幂等键 | 如 model.create 用 name 做幂等 |
| Command (删除类) | 安全 | 删除已删除的资源返回成功 |
| Command (状态变更) | 需检查 | 如 engine.start 检查当前状态 |


---

## 可观测性

### 三大支柱

| 支柱 | 技术 | 说明 |
|------|------|------|
| Metrics | Prometheus | QPS、延迟、错误率、GPU 利用率 |
| Tracing | OpenTelemetry | 请求链路追踪 (RequestID + TraceID) |
| Logging | 结构化 JSON | 按级别过滤，关联 RequestID |

### 请求追踪

每个请求携带两个 ID：

| ID | 说明 | 生命周期 |
|----|------|---------|
| RequestID | 单次请求标识 | 单个请求 |
| TraceID | 跨请求关联 | 由调用方传入或自动生成 |

```go
ctx = unit.WithRequestID(ctx, requestID)
ctx = unit.WithTraceID(ctx, traceID)
```

### 关键指标

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `aima_request_total` | Counter | 总请求数 (按 type, unit, status) |
| `aima_request_duration_ms` | Histogram | 请求延迟 |
| `aima_command_errors_total` | Counter | 命令执行错误数 |
| `aima_gpu_utilization` | Gauge | GPU 利用率 |
| `aima_gpu_memory_used_bytes` | Gauge | GPU 已用内存 |
| `aima_gpu_temperature_celsius` | Gauge | GPU 温度 |
| `aima_models_total` | Gauge | 已注册模型数 |
| `aima_active_inference_requests` | Gauge | 活跃推理请求数 |

### 日志格式

使用结构化 JSON 日志：

```json
{
  "level": "info",
  "time": "2026-02-17T10:30:00.123Z",
  "msg": "command executed",
  "request_id": "req_abc123",
  "trace_id": "trace_xyz",
  "unit": "model.pull",
  "duration_ms": 5234,
  "success": true
}
```


---

## 安全架构

### 认证

| 方式 | 适用场景 | 配置 |
|------|---------|------|
| API Key | HTTP/gRPC 远程访问 | `Authorization: Bearer <key>` |
| 无认证 | 本地 CLI / MCP stdio | 依赖操作系统权限 |

### 授权

按操作风险等级分层：

| 风险等级 | 操作示例 | 认证要求 |
|----------|---------|---------|
| 低 | Query (model.list, device.info) | 可选 |
| 中 | Command (model.pull, engine.start) | 推荐 |
| 高 | Command (remote.exec, app.uninstall) | 强制 |
| 危险 | 系统配置变更 | 强制 + 确认 |

### 输入安全

| 风险点 | 防护措施 |
|--------|---------|
| `remote.exec` 命令注入 | 白名单命令 + 参数转义 |
| 模型路径遍历 | 路径规范化 + 限制在 data_dir |
| API 请求体过大 | `max_request_size = 10MB` |
| 频率攻击 | `rate_limit_per_min = 120` |

### 敏感数据处理

| 数据类型 | 处理方式 |
|----------|---------|
| API Key | 环境变量或配置文件，不记录到日志 |
| 模型权重 | 本地存储，不经网络传输 |
| 推理内容 | 不持久化，不记录到日志 |
| 审计日志 | 记录操作元数据，不记录输入内容 |

## 配置结构

```toml
# aima.toml

[general]
data_dir = "~/.aima"
hostname = ""
device_id = ""

[api]
listen_addr = "127.0.0.1:9090"
enable_cors = false
tls_cert = ""
tls_key = ""

[gateway]
request_timeout = "30s"
max_request_size = "10MB"
enable_tracing = false

[resource]
system_reserved_mb = 10240
inference_pool_pct = 0.6
container_pool_pct = 0.2
pressure_threshold = 0.9

[model]
storage_dir = "~/.aima/models"
default_source = "ollama"
max_cache_gb = 50

[engine]
auto_start = true
ollama_addr = "localhost:11434"

[workflow]
max_concurrent_steps = 10
step_timeout = "5m"
enable_caching = true

[alert]
enabled = true
check_interval = "1m"

[remote]
enabled = false
provider = "frp"

[security]
api_key = ""
rate_limit_per_min = 120

[logging]
level = "info"
format = "json"
file = "~/.aima/logs/aima.log"

# ─── 以下为新增配置 ─────────────────────────────

[catalog]
recipes_dir = "~/.aima/recipes"        # 用户自定义 Recipe 目录

[registry]
region = ""                            # 设为 "cn" 启用国内镜像

[[registry.mirrors]]
source = "docker.io"
mirror = "registry.cn-hangzhou.aliyuncs.com"
region = "cn"

[[registry.mirrors]]
source = "huggingface.co"
mirror = "hf-mirror.com"
region = "cn"

[agent]
enabled = true
provider = "anthropic"                 # "anthropic", "openai", "ollama"
api_endpoint = ""                      # 自定义 API 端点 (代理或本地)
api_key = ""                           # 建议使用 AIMA_AGENT_API_KEY 环境变量
model = "claude-haiku-4-5-20251001"    # LLM 模型 ID
max_tokens = 4096
temperature = 0.7
max_history = 50                       # 会话中保留的最大消息数

[skill]
builtin_enabled = true                 # 加载内置 Skills
user_skills_dir = "~/.aima/skills"     # 用户 Skills 目录
max_active_skills = 10                 # System Prompt 中最大活跃 Skills 数
```

### 新增环境变量覆盖

| 环境变量 | 配置项 | 说明 |
|---------|--------|------|
| `AIMA_AGENT_PROVIDER` | `agent.provider` | LLM 提供商 |
| `AIMA_AGENT_API_KEY` | `agent.api_key` | LLM API Key (敏感，推荐用环境变量) |
| `AIMA_AGENT_MODEL` | `agent.model` | LLM 模型 ID |
| `AIMA_AGENT_ENDPOINT` | `agent.api_endpoint` | 自定义 API 端点 |
| `AIMA_REGISTRY_REGION` | `registry.region` | 镜像区域 |
| `AIMA_CATALOG_DIR` | `catalog.recipes_dir` | Recipe 目录 |
| `AIMA_SKILLS_DIR` | `skill.user_skills_dir` | Skills 目录 |

---

## 部署目标

- **主要平台**: NVIDIA DGX Spark (GB10 SoC, 128GB 统一内存)
- **次要平台**: NVIDIA Jetson, RTX 显卡, 通用 Linux/Windows/macOS
- **部署方式**: 单二进制文件，零外部依赖

---

## API 示例

### HTTP API

```bash
# 执行命令
POST /api/v2/execute
{
  "type": "command",
  "unit": "model.pull",
  "input": {
    "source": "ollama",
    "repo": "llama3.2"
  }
}

# 响应
{
  "success": true,
  "data": {
    "model_id": "llama3.2:latest",
    "status": "ready"
  },
  "meta": {
    "request_id": "req_abc123",
    "duration_ms": 5234
  }
}

# 执行查询
POST /api/v2/execute
{
  "type": "query",
  "unit": "model.list",
  "input": {
    "type": "llm"
  }
}

# 运行工作流
POST /api/v2/workflow/voice_assistant/run
{
  "input": {
    "audio": "base64..."
  },
  "config": {
    "llm_model": "llama3.2"
  }
}
```

### MCP Tool

```json
{
  "name": "aima_model_pull",
  "description": "Pull a model from source registry",
  "inputSchema": {
    "type": "object",
    "properties": {
      "source": {
        "type": "string",
        "enum": ["ollama", "huggingface", "modelscope"]
      },
      "repo": {
        "type": "string",
        "description": "Model repository name"
      }
    },
    "required": ["source", "repo"]
  }
}
```

### CLI

```bash
# 统一格式
aima exec <unit> [flags]

# 示例
aima exec model.pull --source ollama --repo llama3.2
aima exec model.list --type llm
aima exec inference.chat --model llama3.2 --message "Hello"
aima exec resource.status

# 工作流
aima workflow run voice_assistant --input.audio @audio.wav

# 启动服务
aima start
aima mcp serve
```

---

## 事件系统

### Event Bus 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Event Bus                                │
│                                                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐                  │
│  │Publisher │───▶│  Topic   │───▶│Subscriber│                  │
│  └──────────┘    │  Router  │    └──────────┘                  │
│  ┌──────────┐    └──────────┘    ┌──────────┐                  │
│  │Publisher │───────────────────▶│Subscriber│                  │
│  └──────────┘                    └──────────┘                  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │               Persistent Store (SQLite)                  │   │
│  │         - Event Replay    - Query History               │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Event 接口

```go
type Event interface {
    Type() string           // 事件类型 (e.g., "model.created")
    Domain() string         // 所属领域 (e.g., "model")
    Payload() any           // 事件载荷
    Timestamp() time.Time   // 时间戳
    CorrelationID() string  // 关联 ID
}
```

### 订阅模式

```go
// 订阅特定领域
events := bus.Subscribe(ctx, "model")

// 订阅特定类型
events := bus.Subscribe(ctx, "model.pull_progress")

// 订阅所有事件
events := bus.Subscribe(ctx, "")

// 使用 Handler
handler := &MyEventHandler{}
bus.SubscribeWithHandler(ctx, "model", handler)
```

### 事件持久化

```go
// 创建持久化事件总线
persistentBus, err := eventbus.NewPersistentEventBus(
    "~/.aima/events.db",
    eventbus.WithRetention(7*24*time.Hour),
    eventbus.WithMaxEvents(100000),
)

// 查询历史事件
history, err := persistentBus.Query(ctx, eventbus.Query{
    Domain:    "model",
    Type:      "model.created",
    Since:     time.Now().Add(-24*time.Hour),
    Limit:     100,
})

// 回放事件
err := persistentBus.Replay(ctx, "model.created", func(e Event) error {
    // 处理事件
    return nil
})
```

### 核心事件类型

| 领域 | 事件类型 | 说明 |
|------|----------|------|
| model | `model.created` | 模型创建 |
| model | `model.pull_progress` | 拉取进度 |
| model | `model.verified` | 验证完成 |
| engine | `engine.started` | 引擎启动 |
| engine | `engine.error` | 引擎错误 |
| inference | `inference.request_started` | 推理开始 |
| inference | `inference.request_completed` | 推理完成 |
| resource | `resource.pressure_warning` | 资源压力警告 |
| alert | `alert.triggered` | 告警触发 |

---


---

## Provider 架构

Provider 是连接原子单元与外部系统的适配层，负责实际的 I/O 操作。

### Provider 类型

```
+----------------------------------------------------------+
|                   原子单元 (Atomic Units)                  |
|   model.pull   engine.start   inference.chat   ...       |
+----------------------------------------------------------+
|                   Provider 接口层                          |
|   ModelProvider   EngineProvider   InferenceProvider      |
+----------------------------------------------------------+
|                   Provider 实现                            |
|   Ollama | vLLM | HuggingFace | ModelScope | TensorRT   |
+----------------------------------------------------------+
```

### Provider 接口定义

```go
// ModelProvider - 模型操作的底层实现
type ModelProvider interface {
    Pull(ctx, source, repo, tag string, progressCh chan<- PullProgress) (*Model, error)
    Search(ctx, query, source string, modelType ModelType, limit int) ([]ModelSearchResult, error)
    ImportLocal(ctx, path string, autoDetect bool) (*Model, error)
    Verify(ctx, modelID, checksum string) (*VerificationResult, error)
    EstimateResources(ctx, modelID string) (*ModelRequirements, error)
}

// InferenceProvider - 推理能力的底层实现
type InferenceProvider interface {
    Chat(ctx, modelName string, messages []Message, opts ChatOptions) (*ChatResponse, error)
    Complete(ctx, modelName, prompt string, opts CompleteOptions) (*CompletionResponse, error)
    Embed(ctx, modelName string, input []string) (*EmbeddingResponse, error)
    // ... 更多推理能力
}
```

### Provider 选择策略

当同一类型有多个 Provider 时（如 Ollama 和 vLLM 都支持 Chat），选择策略如下：

1. 检查模型在哪个 Provider 已加载 -> 优先使用已加载的
2. 检查模型格式匹配 -> GGUF 优先 Ollama，safetensors 优先 vLLM
3. 默认使用配置的 default_engine

实现文件：`pkg/infra/provider/hybrid_engine_provider.go`

### Provider 降级策略

```
优先 Provider 不可用时的行为：

1. 探测 (Probe)  -> 定期检查 Provider 健康状态
2. 降级 (Fallback) -> Ollama 不可用时尝试 vLLM
3. 恢复 (Recovery) -> 后台持续探测，恢复后自动纳入选择池
```

## 适配器列表

### HTTP Adapter

```go
// 端点映射
POST   /api/v2/execute              // 通用执行
POST   /api/v2/stream               // 流式执行
POST   /api/v2/command/{unit}       // 执行命令
POST   /api/v2/query/{unit}         // 执行查询
GET    /api/v2/resource/{uri}       // 获取资源
GET    /api/v2/resource/{uri}/watch  // 订阅资源 (SSE)

// 工作流端点
GET    /api/v2/workflows            // 列出工作流
POST   /api/v2/workflow/{name}/run  // 运行工作流
GET    /api/v2/workflow/{name}/runs/{run_id}  // 获取状态

// 系统端点
GET    /api/v2/health               // 健康检查
GET    /api/v2/metrics              // Prometheus 指标
GET    /api/v2/units                // 列出所有单元
```

### MCP Adapter

```go
// MCP Tool 自动生成
tools := adapter.GenerateTools(registry)

// 输出示例
{
  "name": "model_pull",
  "description": "Pull a model from source registry",
  "inputSchema": { ... }
}

// 传输方式
- stdio (默认): aima mcp serve
- sse: aima mcp serve --transport sse --port 9091
```

### gRPC Adapter

```protobuf
service AIMAService {
  rpc Execute(Request) returns (Response);
  rpc ExecuteStream(Request) returns (stream StreamChunk);
  rpc WatchResource(WatchRequest) returns (stream ResourceUpdate);
  rpc HealthCheck(HealthRequest) returns (HealthResponse);
}
```

| 特性 | 支持 |
|------|------|
| 默认端口 | 50051 |
| 流式响应 | ✅ |
| 双向流 | ✅ |
| TLS | ✅ |
| 元数据传递 | ✅ |

### CLI Adapter

```bash
# 统一执行格式
aima exec <unit> [flags]

# 示例
aima exec model.pull --source ollama --repo llama3.2
aima exec model.list --type llm --limit 10
aima exec inference.chat --model llama3.2 --message "Hello"

# 工作流
aima workflow run rag --input.query "What is AI?"
aima workflow list
aima workflow status <run_id>

# 服务管理
aima start
aima stop
aima status

# MCP
aima mcp serve
aima mcp tools  # 列出所有可用 tools
```

### 适配器对比

| 特性 | HTTP | MCP | gRPC | CLI |
|------|------|-----|------|-----|
| 传输协议 | HTTP/1.1, HTTP/2 | stdio/SSE | HTTP/2 | 本地执行 |
| 流式支持 | SSE | ✅ | ✅ | ❌ |
| 适用场景 | Web/通用 | AI Agent | 高性能 | 运维/脚本 |
| 认证 | API Key | 环境变量 | TLS/API Key | 本地权限 |
| 工具发现 | OpenAPI | MCP Tools | Reflection | --help |

---


---

## 关键场景数据流

### 场景 1: AI Agent 通过 MCP 拉取模型

```
AI Agent -> MCP tools/call(model_pull)
  -> MCPAdapter.ExecuteTool()
    -> Gateway.Handle(type=command, unit=model.pull)
      -> Registry.GetCommand("model.pull")
      -> PullCommand.Execute(ctx, input)
        -> OllamaProvider.Pull(source, repo, tag)
          -> HTTP POST ollama:11434/api/pull
        -> ModelStore.Create(model)
        -> EventBus.Publish(model.pull.completed)
      -> Response{success: true, data: {model_id, status}}
    -> MCPToolResult{content: [{type: text, text: JSON}]}
  -> AI Agent 收到结果
```

### 场景 2: HTTP 流式推理

```
HTTP Client -> POST /api/v2/stream
  -> HTTPAdapter.handleStream()
    -> Gateway.HandleStream(req)
      -> Registry.GetCommand("inference.chat")
      -> StreamingCommand.ExecuteStream(ctx, input, stream)
        -> OllamaProvider.Chat(ctx, model, messages, stream=true)
          -> HTTP POST ollama:11434/api/chat (stream)
          -> 逐 chunk 返回
        -> chunk -> unitStream -> gatewayStream
      -> SSE: event: chunk / data: {...}
    -> 最终: event: chunk / data: {"done": true}
```

### 场景 3: AI Agent 一键部署 (新增)

```
用户: "帮我部署一个 LLM"

  -> aima agent chat
    -> Agent.Chat(ctx, "帮我部署一个 LLM")
      -> LLM 思考: 需要先检测硬件
      -> [Tool Call: device_detect]
        -> MCPAdapter.ExecuteTool("device_detect", {})
        -> Gateway.Handle(command, device.detect)
        -> HAL.Detect() -> {GPU: "RTX 4090", VRAM: 24GB}
      -> LLM 思考: 有 RTX 4090，查找匹配方案
      -> [Tool Call: catalog_match]
        -> 评分匹配 -> 返回 Top 3 Recipes
      -> LLM 回复: "找到 3 个方案: 1. vLLM + Qwen 2.5  2. ..."
      -> 用户: "用第一个"
      -> [Tool Call: catalog_apply_recipe]
        -> CatalogService.ApplyRecipe()
          -> RegistryProvider.PullImage("vllm/vllm-openai:v0.15.0")
          -> model.pull(source=huggingface, repo=Qwen/Qwen2.5-7B-AWQ)
        -> 返回 {engine_ready: true, models: [{name: "Qwen 2.5", status: "ready"}]}
      -> [Tool Call: inference_chat] (测试)
        -> 返回测试成功
      -> LLM 回复: "部署完成！Qwen 2.5 7B 已就绪，可以开始对话。"
```

### 场景 4: Workflow 编排

```
API Request: POST /api/v2/workflow/voice_assistant/run
  -> Gateway.execute(TypeWorkflow)
    -> WorkflowEngine.Execute(def, input)
      -> DAGValidator.Validate(def)              <- 验证 DAG 无环
      -> TopologicalSort(def.Steps)               <- 拓扑排序确定执行顺序
      -> for step in sortedSteps:
          -> VariableResolver.ResolveStepInput()  <- 解析 ${input.xxx}
          -> Registry.Get(step.Type)              <- 获取对应原子单元
          -> unit.Execute(ctx, resolvedInput)     <- 执行
          -> execCtx.Steps[step.ID] = result      <- 存储中间结果
      -> VariableResolver.ResolveOutput()         <- 组装最终输出
    -> Response
```


---

## AI Agent 交互模型

AIMA 的主要用户是 AI Agent。以下描述 AI Agent 与系统交互的完整工作流。

### 交互协议

AI Agent 通过 MCP (Model Context Protocol) 或 HTTP API 与 AIMA 交互。推荐使用 MCP，因为：

1. **工具自动发现** - MCP 的 `tools/list` 自动列出所有可用原子单元
2. **Schema 传递** - 每个 Tool 携带完整的 InputSchema，AI Agent 能理解参数要求
3. **示例学习** - Examples 帮助 AI Agent 理解正确的输入格式

### AI Agent 工作流

```
+--------------+    +--------------+    +--------------+    +--------------+
|  Discovery   | -> |  Planning    | -> |  Execution   | -> | Verification |
|  发现能力    |    |  规划步骤    |    |  执行操作    |    |  验证结果    |
+--------------+    +--------------+    +--------------+    +--------------+
```

#### Phase 1: Discovery（发现能力）

AI Agent 首先获取可用工具列表：

```
MCP: tools/list -> 返回所有 Command + Query 的 Tool 定义
HTTP: GET /api/v2/units -> 返回所有注册的原子单元
```

每个 Tool 包含：
- **name**: 工具名称（如 `model_pull`）
- **description**: 工具描述（如 "Pull a model from a remote source"）
- **inputSchema**: 参数定义（类型、约束、默认值）

#### Phase 2: Planning（规划步骤）

AI Agent 基于用户需求和可用工具，规划执行步骤。例如：

```
用户: "帮我部署 llama3.2 模型并启动推理服务"

AI Agent 规划:
1. model_pull(source="ollama", repo="llama3.2")    -> 拉取模型
2. model_list(type="llm")                          -> 确认模型就绪
3. engine_start(name="ollama")                      -> 确保引擎运行
4. inference_chat(model="llama3.2", messages=[...]) -> 测试推理
```

#### Phase 3: Execution（执行操作）

AI Agent 按计划调用 Tool：

```json
{
  "method": "tools/call",
  "params": {
    "name": "model_pull",
    "arguments": {
      "source": "ollama",
      "repo": "llama3.2"
    }
  }
}
```

#### Phase 4: Verification（验证结果）

AI Agent 检查执行结果，决定下一步：
- `status=ready` -> 继续下一步
- `status=error` -> 重试或向用户报告

### MCP Tool 自动生成原理

关键转换规则：
- 名称：`model.pull` -> `model_pull`（点号替换为下划线）
- Schema：`unit.Schema` -> JSON Schema（自动映射）
- 类型：Command 和 Query 都映射为 MCP Tool

自动生成代码位于 `pkg/gateway/mcp_tools.go`，`GenerateToolDefinitions()` 方法遍历 Registry 中所有 Command 和 Query，转换为 MCP ToolDefinition。

### AI Agent 友好性设计指南

编写新的原子单元时，请遵循以下原则确保 AI Agent 友好：

| 维度 | 要求 | 示例 |
|------|------|------|
| Description | 用一句话说明**做什么**和**何时用** | "Pull a model from a remote source. Use when you need to download a new model." |
| InputSchema | 每个字段都有 description | `"source": {"type": "string", "description": "Model source registry"}` |
| Required | 明确标记必填 vs 可选 | `"required": ["source", "repo"]` |
| Default | 合理的默认值 | `"tag": {"type": "string", "default": "latest"}` |
| Enum | 有限选项列出所有值 | `"source": {"enum": ["ollama", "huggingface", "modelscope"]}` |
| Examples | 覆盖典型场景 | 至少提供一个成功场景的输入/输出 |
| Error Message | 包含修复建议 | `"model not found: llama4. Available: llama3.2, mistral"` |

## 性能数据

### 基准测试结果

| 指标 | HTTP | gRPC | 说明 |
|------|------|------|------|
| P50 延迟 | 12ms | 5ms | 简单查询 |
| P99 延迟 | 45ms | 18ms | 简单查询 |
| 并发 QPS | 5,000 | 15,000 | 8 核 CPU |
| 内存占用 | 128MB | 128MB | 基础服务 |
| 启动时间 | 2s | 2s | 包含初始化 |

### 流式性能

| 场景 | 吞吐量 | 延迟 |
|------|--------|------|
| 文本生成 | 50 tokens/s | < 100ms first token |
| 音频合成 | 10 MB/s | < 500ms first chunk |
| 日志流 | 1,000 lines/s | < 10ms |

### 资源使用

| 场景 | CPU | 内存 | 说明 |
|------|-----|------|------|
| 空闲 | 1% | 64MB | 无请求时 |
| 轻负载 | 10% | 128MB | 10 QPS |
| 中负载 | 50% | 512MB | 100 QPS |
| 重负载 | 100% | 1GB | 1000 QPS |

### 扩展性

| 资源 | 限制 | 说明 |
|------|------|------|
| 最大并发请求 | 10,000 | 可配置 |
| 最大模型数 | 无限制 | 受存储限制 |
| 最大工作流步骤 | 100 | 可配置 |
| 事件保留时间 | 30 天 | 可配置 |

---

## 核心收益

| 方面 | 改进前 (ASMS) | 改进后 (AIMA) |
|------|--------------|---------------|
| **接口数量** | 20+ 种混乱接口 | 4 种标准化接口 |
| **代码重复** | API/MCP/CLI 三份逻辑 | 共享原子单元 |
| **可测试性** | 需要 Mock 大量依赖 | 每个原子单元独立测试 |
| **可编排性** | 无 | Workflow DSL |
| **文档化** | 分散在各处 | 自动从 Schema 生成 |
| **版本控制** | v1/v2 混乱 | 原子单元独立版本 |
| **扩展性** | 修改核心代码 | 注册新原子单元 |

---

## 迁移路径

### Phase 1: 核心框架 ✅
- 定义原子单元接口
- 实现注册表和 Schema
- 构建 Gateway 核心

### Phase 2: 核心领域 ✅
- 迁移 device / model / engine / inference 原子单元

### Phase 3: 适配器 ✅
- 实现 HTTP / MCP / gRPC / CLI 适配器

### Phase 4: 其他领域 ✅
- 迁移 resource / service / app / alert / remote 原子单元

### Phase 5: 编排层 ✅
- 实现 Workflow 引擎和 Pipeline 模板

### Phase 6: 硬件最佳实践 Catalog (NEW)
- Recipe 类型定义和 Store
- 内置 Recipe YAML (RTX 4090, A100, M 系列, CPU)
- 硬件匹配算法
- `catalog.create_recipe`, `catalog.match`, `catalog.list`, `catalog.get`
- Registry Provider (DockerHub + 国内镜像)
- `catalog.apply_recipe` 一键部署

### Phase 7: AI Agent Skills (NEW)
- Skill 类型定义和加载器
- 内置 Skills (setup-llm, troubleshoot-gpu 等)
- SkillRegistry 和 System Prompt 构建
- `skill.add`, `skill.list`, `skill.search` 等原子单元

### Phase 8: AI Agent Operator (NEW)
- LLM Client 接口和适配器 (Anthropic/OpenAI/Ollama)
- Agent 对话循环 (MCP Tool → LLM Tool 桥接)
- 会话管理
- `agent.chat`, `agent.status` 原子单元
- CLI `aima agent chat` / `aima agent ask`

---

## 领域总览

| # | 领域 | Commands | Queries | Resources | 说明 |
|---|------|----------|---------|-----------|------|
| 1 | device | 2 | 3 | 3 动态 | 硬件设备管理 |
| 2 | model | 5 | 4 | 2 | 模型管理 |
| 3 | engine | 4 | 3 | 2 | 推理引擎管理 |
| 4 | inference | 9 (流式) | 2 | 1 | 推理服务 |
| 5 | resource | 3 | 4 | 3 | 资源管理 |
| 6 | service | 5 | 3 | 2 | 模型服务化 |
| 7 | app | 4 | 4 | 2 | Docker 应用 |
| 8 | pipeline | 4 | 4 | 2 | 管道编排 |
| 9 | alert | 5 | 3 | 2 | 告警管理 |
| 10 | remote | 3 (流式) | 2 | 2 | 远程操作 |
| 11 | **catalog** | **3** | **4** | **2** | **硬件最佳实践 ★ NEW** |
| 12 | **agent** | **2** | **2** | **2** | **AI 代理操作 ★ NEW** |
| 13 | **skill** | **4** | **3** | **2** | **技能知识库 ★ NEW** |
| | **合计** | **53** | **41** | **27** | **13 个领域** |
