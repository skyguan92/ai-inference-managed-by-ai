# 迁移优先级和工作量评估

本文档定义了从 ASMS 迁移到 AIMA 的开发阶段和工作量评估。

---

## 迁移原则

1. **先框架后业务**: 先建立核心框架，再迁移业务逻辑
2. **先核心后边缘**: 优先迁移最常用的核心领域
3. **增量迁移**: 保持系统可运行，逐步替换
4. **测试驱动**: 每个原子单元都有对应的测试

---

## 阶段划分

```
Week 1-2   Phase 1: 核心框架
Week 3-6   Phase 2: 核心领域 (Device, Model, Engine, Inference)
Week 7-8   Phase 3: 适配器 (HTTP, MCP, CLI)
Week 9-10  Phase 4: 其他领域 (Resource, Service, App, Alert, Remote)
Week 11-12 Phase 5: 编排层 (Workflow DSL, Pipeline 升级)
Week 13    Phase 6: 完善和文档
```

---

## Phase 1: 核心框架 (1-2 周)

### 目标

建立原子单元的核心抽象和基础设施。

### 交付物

| 组件 | 文件 | 工作量 | 说明 |
|------|------|--------|------|
| 接口定义 | `pkg/unit/types.go` | 1 天 | Command/Query/Event/Resource 接口 |
| Schema 定义 | `pkg/unit/schema.go` | 1 天 | 输入输出 Schema 和验证 |
| Registry | `pkg/unit/registry.go` | 2 天 | 单元注册表和查找 |
| 执行上下文 | `pkg/unit/context.go` | 0.5 天 | 执行上下文和元数据 |
| Gateway 核心 | `pkg/gateway/gateway.go` | 2 天 | 统一请求处理 |
| 错误处理 | `pkg/gateway/errors.go` | 0.5 天 | 统一错误格式 |
| EventBus | 从 ASMS 复用 | - | 直接复制 |

### 关键代码

```go
// pkg/unit/types.go

type Command interface {
    Name() string
    Domain() string
    InputSchema() Schema
    OutputSchema() Schema
    Execute(ctx context.Context, input any) (output any, err error)
    Description() string
    Examples() []Example
}

type Query interface {
    Name() string
    Domain() string
    InputSchema() Schema
    OutputSchema() Schema
    Execute(ctx context.Context, input any) (output any, err error)
    Description() string
    Examples() []Example
}

type Event interface {
    Type() string
    Domain() string
    Payload() any
    Timestamp() time.Time
    CorrelationID() string
}

type Resource interface {
    URI() string
    Domain() string
    Schema() Schema
    Get(ctx context.Context) (any, error)
    Watch(ctx context.Context) (<-chan ResourceUpdate, error)
}
```

### 验收标准

- [ ] 四种接口类型定义完整
- [ ] Registry 可以注册和查找单元
- [ ] Gateway 可以处理基础请求
- [ ] Schema 验证正常工作
- [ ] 单元测试覆盖率 > 80%

---

## Phase 2: 核心领域 (3-4 周)

### 2.1 Device Domain (3 天)

**ASMS 源码**: `pkg/hal/`

| 原子单元 | 工作量 | 优先级 |
|----------|--------|--------|
| `device.detect` | 0.5 天 | P0 |
| `device.info` | 0.5 天 | P0 |
| `device.metrics` | 0.5 天 | P0 |
| `device.health` | 0.5 天 | P1 |
| Events | 1 天 | P2 |

### 2.2 Model Domain (5 天)

**ASMS 源码**: `pkg/model/`

| 原子单元 | 工作量 | 优先级 |
|----------|--------|--------|
| `model.create` | 0.5 天 | P0 |
| `model.delete` | 0.5 天 | P0 |
| `model.pull` | 1 天 | P0 |
| `model.get` | 0.5 天 | P0 |
| `model.list` | 0.5 天 | P0 |
| `model.search` | 1 天 | P1 |
| `model.import` | 0.5 天 | P1 |
| `model.estimate_resources` | 0.5 天 | P1 |

### 2.3 Engine Domain (5 天)

**ASMS 源码**: `pkg/engine/`

| 原子单元 | 工作量 | 优先级 |
|----------|--------|--------|
| `engine.start` | 1 天 | P0 |
| `engine.stop` | 0.5 天 | P0 |
| `engine.get` | 0.5 天 | P0 |
| `engine.list` | 0.5 天 | P0 |
| `engine.install` | 1 天 | P1 |
| 适配器封装 | 1.5 天 | P1 |

### 2.4 Inference Domain (6 天)

**ASMS 源码**: `pkg/engine/adapters/`

| 原子单元 | 工作量 | 优先级 |
|----------|--------|--------|
| `inference.chat` | 1 天 | P0 |
| `inference.complete` | 0.5 天 | P0 |
| `inference.embed` | 0.5 天 | P1 |
| `inference.transcribe` | 1 天 | P1 |
| `inference.synthesize` | 1 天 | P1 |
| `inference.generate_image` | 1 天 | P2 |
| `inference.rerank` | 0.5 天 | P2 |
| `inference.detect` | 0.5 天 | P2 |

---

## Phase 3: 适配器 (2 周)

### 3.1 HTTP 适配器 (4 天)

**ASMS 源码**: `pkg/api/`

| 任务 | 工作量 |
|------|--------|
| 请求路由 | 1 天 |
| 响应格式化 | 1 天 |
| 中间件 (Auth, Rate Limit) | 1 天 |
| OpenAPI 文档生成 | 1 天 |

### 3.2 MCP 适配器 (4 天)

**ASMS 源码**: `pkg/mcp/`

| 任务 | 工作量 |
|------|--------|
| Tool 定义生成 | 1 天 |
| Tool 执行处理 | 1 天 |
| Resource 订阅 | 1 天 |
| 会话管理 | 1 天 |

### 3.3 CLI 适配器 (2 天)

**ASMS 源码**: `pkg/cli/`

| 任务 | 工作量 |
|------|--------|
| 命令注册 | 0.5 天 |
| 参数解析 | 0.5 天 |
| 输出格式化 | 0.5 天 |
| 自动补全 | 0.5 天 |

---

## Phase 4: 其他领域 (2 周)

### 4.1 Resource Domain (2 天)

**ASMS 源码**: `pkg/resource/`

### 4.2 Service Domain (2 天)

**ASMS 源码**: `pkg/service/`

### 4.3 App Domain (2 天)

**ASMS 源码**: `pkg/app/`

### 4.4 Pipeline Domain (3 天)

**ASMS 源码**: `pkg/pipeline/`

### 4.5 Alert Domain (2 天)

**ASMS 源码**: `pkg/fleet/alert.go`

### 4.6 Remote Domain (1 天)

**ASMS 源码**: `pkg/remote/`

---

## Phase 5: 编排层 (2 周)

### 5.1 Workflow DSL (5 天)

| 任务 | 工作量 |
|------|--------|
| DSL 解析 | 1 天 |
| DAG 验证 | 1 天 |
| 步骤执行器 | 2 天 |
| 变量解析 | 1 天 |

### 5.2 Pipeline 升级 (3 天)

| 任务 | 工作量 |
|------|--------|
| 现有 Pipeline 迁移 | 2 天 |
| 新增功能 (cancel, retry) | 1 天 |

### 5.3 预构建模板 (2 天)

| 模板 | 工作量 |
|------|--------|
| voice_assistant.yaml | 0.5 天 |
| rag_pipeline.yaml | 0.5 天 |
| batch_inference.yaml | 0.5 天 |
| multimodal_chat.yaml | 0.5 天 |

---

## Phase 6: 完善和文档 (1 周)

### 6.1 测试完善 (3 天)

- 补充单元测试
- 集成测试
- 端到端测试

### 6.2 文档完善 (2 天)

- API 文档
- 架构图更新
- 使用示例

### 6.3 清理和优化 (2 天)

- 清理遗留代码
- 性能优化
- 代码审查

---

## 风险和依赖

### 风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 接口设计变更 | 高 | 尽早确定接口，减少变更 |
| ASMS 代码依赖 | 中 | 逐步迁移，保持兼容 |
| 测试覆盖不足 | 中 | 测试驱动开发 |
| 性能退化 | 低 | 性能基准测试 |

### 依赖

| 依赖 | 来源 | 状态 |
|------|------|------|
| Go 1.22+ | 环境 | ✅ 已安装 |
| SQLite | 存储 | ✅ ASMS 已有 |
| Docker SDK | 容器 | ✅ ASMS 已有 |
| ASMS 源码 | 参考 | ✅ 可访问 |

---

## 里程碑

| 里程碑 | 时间 | 交付物 |
|--------|------|--------|
| M1: 框架就绪 | Week 2 | 核心接口和 Registry |
| M2: 核心领域完成 | Week 6 | Device/Model/Engine/Inference 原子单元 |
| M3: 适配器完成 | Week 8 | HTTP/MCP/CLI 适配器 |
| M4: 全部领域完成 | Week 10 | 所有 10 个领域原子单元 |
| M5: 编排层完成 | Week 12 | Workflow DSL 和 Pipeline |
| M6: 发布就绪 | Week 13 | 完整文档和测试 |

---

## 资源需求

| 角色 | 人数 | 周数 |
|------|------|------|
| 后端开发 | 1-2 | 13 |
| 测试工程师 | 1 | 4 (兼职) |
| 技术文档 | 1 | 2 (兼职) |

---

## 下一步行动

1. **立即开始**: Phase 1 核心框架
2. **并行准备**: 阅读 ASMS 源码，理解现有实现
3. **持续进行**: 补充测试用例
