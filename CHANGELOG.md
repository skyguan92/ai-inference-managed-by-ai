# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Core Framework
- **Atomic Units**: Command, Query, Event, Resource 四种标准化接口
- **Registry**: 原子单元注册表，支持动态注册和发现
- **Schema System**: 结构化输入输出定义与验证
- **Error System**: 统一的错误码和处理机制

#### Domains
- **Device Domain**: 硬件设备检测、监控、功耗管理
- **Model Domain**: 模型拉取、导入、验证、搜索
- **Engine Domain**: 推理引擎管理（Ollama、vLLM、TensorRT-LLM）
- **Inference Domain**: 聊天、补全、嵌入、语音、图像、视频生成
- **Resource Domain**: 资源分配、预算管理、压力监控
- **Service Domain**: 模型服务化、自动扩缩容
- **App Domain**: Docker 应用部署
- **Pipeline Domain**: 工作流编排
- **Alert Domain**: 告警管理
- **Remote Domain**: 远程访问

#### Gateway & Adapters
- **HTTP Adapter**: RESTful API 支持
- **MCP Adapter**: Model Context Protocol 支持
- **gRPC Adapter**: 高性能 RPC 支持
- **CLI Adapter**: 命令行工具
- **Streaming Support**: Server-Sent Events 流式响应

#### Infrastructure
- **Event Bus**: 内存和持久化事件总线
- **Event Persistence**: SQLite 事件存储与回放
- **HAL (Hardware Abstraction Layer)**: NVIDIA GPU 支持
- **Store**: SQLite 数据存储与迁移
- **Metrics**: Prometheus 指标收集

#### Workflow Engine
- **DAG Execution**: 有向无环图工作流执行
- **Variable Resolution**: 步骤间变量传递
- **Pre-built Templates**: 预构建工作流模板
  - RAG Pipeline
  - Voice Assistant
  - Batch Inference
  - Multimodal Chat
  - Video Analysis

#### Documentation
- **README**: 项目介绍和快速开始
- **Architecture**: 系统架构详细文档
- **API Documentation**: HTTP/MCP/gRPC API 文档
- **Development Guide**: 开发者指南
- **Examples**: 使用示例代码

### Changed

- N/A (initial release)

### Deprecated

- N/A (initial release)

### Removed

- N/A (initial release)

### Fixed

- N/A (initial release)

### Security

- API Key 认证支持
- 请求速率限制
- 安全上下文传递

---

## [0.1.0] - 2026-02-17

### Added
- Initial release of AIMA (AI Inference Managed by AI)
- Complete atomic unit framework
- All 10 domains implementation
- Gateway with HTTP/MCP/gRPC/CLI adapters
- Workflow engine with pre-built templates
- Event system with persistence
- Comprehensive documentation and examples

---

## Migration Notes

### From ASMS (AI Service Management System)

| ASMS Concept | AIMA Equivalent |
|--------------|-----------------|
| API Endpoint | Command/Query |
| WebSocket Event | Event |
| REST Resource | Resource |
| Service | Service Layer |
| Job | Workflow |

See [migration/analysis.md](docs/reference/migration/analysis.md) for detailed migration guide.

---

## Roadmap

### v0.2.0 (Planned)
- [ ] Web UI Dashboard
- [ ] Distributed deployment support
- [ ] Kubernetes operator
- [ ] More inference engines (llama.cpp, exllama)
- [ ] Model quantization tools

### v0.3.0 (Planned)
- [ ] Multi-node cluster support
- [ ] Advanced scheduling algorithms
- [ ] Model sharding for large models
- [ ] Federated learning support

### v1.0.0 (Planned)
- [ ] Production-ready stability
- [ ] Complete observability stack
- [ ] Enterprise features (RBAC, audit log)
- [ ] Cloud provider integrations
