# 项目概览

## 项目定位

**ai-inference-managed-by-ai (AIMA)** 是一个面向 AI Agent 和机器用户的 AI 推理基础设施管理平台。

## 核心设计理念

### 1. 一切皆原子服务

所有功能拆分为最小可组合单元，实现以下四种接口之一：

| 接口类型 | 说明 | 特点 |
|----------|------|------|
| **Command** | 有副作用的操作 | 创建、删除、启动、停止 |
| **Query** | 无副作用的查询 | 获取、列表、搜索、状态 |
| **Event** | 异步事件通知 | 状态变化、告警、进度 |
| **Resource** | 可寻址资源 | URI 访问、Watch 订阅 |

### 2. 统一入口

HTTP、MCP、CLI 三种访问方式共享同一套原子单元：

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│   HTTP  │     │   MCP   │     │   CLI   │
└────┬────┘     └────┬────┘     └────┬────┘
     │               │               │
     └───────────────┼───────────────┘
                     │
              ┌──────▼──────┐
              │   Gateway   │
              └──────┬──────┘
                     │
              ┌──────▼──────┐
              │  Registry   │
              └──────┬──────┘
                     │
     ┌───────────────┼───────────────┐
     │               │               │
┌────▼────┐    ┌────▼────┐    ┌────▼────┐
│ Command │    │  Query  │    │Resource │
└─────────┘    └─────────┘    └─────────┘
```

### 3. AI First

接口设计优先考虑 AI Agent 的可理解性：

- Schema 驱动：每个原子单元都有输入/输出 Schema
- 自描述：包含描述、示例、文档
- 机器可读：JSON Schema 格式

### 4. 可编排性

高级功能通过编排原子单元实现：

```yaml
# Pipeline DSL 示例
name: voice_assistant
steps:
  - id: transcribe
    type: inference.transcribe
    input: { model: "whisper", audio: "${input.audio}" }
  
  - id: chat
    type: inference.chat
    input: { model: "llama3.2", messages: [...] }
  
  - id: synthesize
    type: inference.synthesize
    input: { model: "tts-1", text: "${steps.chat.response}" }
```

---

## 架构分层

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Orchestration Layer                            │
│          Pipelines · Workflows · Pre-built Flows · DSL              │
├─────────────────────────────────────────────────────────────────────┤
│                      Service Layer                                  │
│    ModelService · InferenceService · DeviceService · AppService    │
├─────────────────────────────────────────────────────────────────────┤
│                   Atomic Unit Layer (核心)                          │
│     Command · Query · Event · Resource (4 种接口类型)               │
├─────────────────────────────────────────────────────────────────────┤
│                   Infrastructure Layer                              │
│      HAL · Store · EventBus · Docker · Network · Crypto            │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 技术栈

| 层级 | 技术选型 |
|------|----------|
| 语言 | Go 1.22+ |
| 存储 | SQLite (WAL 模式) |
| 容器 | Docker SDK |
| 事件 | 内存 EventBus |
| 协议 | HTTP/1.1, MCP, gRPC (可选) |
| 配置 | TOML |

---

## 部署目标

| 平台 | 优先级 | 特点 |
|------|--------|------|
| NVIDIA DGX Spark (GB10) | 主要 | 128GB 统一内存 |
| NVIDIA Jetson | 次要 | 边缘计算 |
| RTX 显卡 | 次要 | 消费级 GPU |
| 通用 Linux/Windows/macOS | 兼容 | CPU 推理 |

---

## 与 ASMS 的关系

AIMA 是 ASMS 项目的架构升级版本：

| 维度 | ASMS | AIMA |
|------|------|------|
| 架构 | 领域驱动 | 原子单元 + 编排 |
| 接口 | 20+ 混乱接口 | 4 种标准化接口 |
| 入口 | HTTP/MCP/CLI 分离 | 统一 Gateway |
| 代码复用 | API/MCP/CLI 三份逻辑 | 共享原子单元 |

**迁移策略**: 保留 ASMS 的核心实现，重构为原子单元接口，统一入口层。

---

## 开发路线

详见 [迁移优先级](./migration/priority.md)

1. **Phase 1**: 核心框架 (1-2 周)
2. **Phase 2**: 核心领域 (3-4 周)
3. **Phase 3**: 适配器 (2 周)
4. **Phase 4**: 其他领域 (2 周)
5. **Phase 5**: 编排层 (2 周)
6. **Phase 6**: 完善和文档 (1 周)
