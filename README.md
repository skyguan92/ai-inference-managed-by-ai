# ai-inference-managed-by-ai (AIMA)

> 让 AI 管理的 AI 推理基础设施

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 项目概述

**AIMA** 是一个面向 AI Agent 和机器用户的 AI 推理基础设施管理平台。它提供统一的原子化接口来管理 AI 模型、推理引擎、硬件设备和应用部署。

### 核心设计理念

- **一切皆原子服务** - 所有功能拆分为最小可组合单元（Command/Query/Event/Resource）
- **统一入口** - HTTP/MCP/gRPC/CLI 共享同一套原子单元
- **AI First** - 接口设计优先考虑 AI Agent 可理解性
- **可编排性** - 高级功能通过 Workflow DSL 编排原子单元实现

## 快速开始

### 安装

```bash
go install github.com/jguan/ai-inference-managed-by-ai/cmd/aima@latest
```

### 启动服务

```bash
# 启动 HTTP 和 MCP 服务
aima start

# 查看帮助
aima --help
```

### 执行命令

```bash
# 列出模型
aima exec model.list

# 拉取模型
aima exec model.pull --source ollama --repo llama3

# 聊天推理
aima exec inference.chat --model llama3 --message "Hello, world!"

# 查看设备状态
aima exec device.info

# 查看资源状态
aima exec resource.status
```

## HTTP API 示例

```bash
# 执行命令
curl -X POST http://localhost:9090/api/v2/execute \
  -H "Content-Type: application/json" \
  -d '{
    "type": "command",
    "unit": "model.pull",
    "input": {
      "source": "ollama",
      "repo": "llama3.2"
    }
  }'

# 执行查询
curl -X POST http://localhost:9090/api/v2/execute \
  -H "Content-Type: application/json" \
  -d '{
    "type": "query",
    "unit": "model.list",
    "input": {
      "limit": 10
    }
  }'

# 运行工作流
curl -X POST http://localhost:9090/api/v2/workflow/rag/run \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "query": "什么是 AI 推理？"
    }
  }'
```

## 功能特性

| 领域 | 功能 |
|------|------|
| **Device** | 硬件设备检测、监控、功耗管理 |
| **Model** | 模型拉取、导入、验证、搜索 |
| **Engine** | 推理引擎管理（Ollama、vLLM、TensorRT-LLM） |
| **Inference** | 聊天、补全、嵌入、语音、图像、视频生成 |
| **Resource** | 资源分配、预算管理、压力监控 |
| **Service** | 模型服务化、自动扩缩容 |
| **App** | Docker 应用部署（ComfyUI、OpenWebUI 等） |
| **Pipeline** | 工作流编排、预构建模板 |
| **Alert** | 告警规则、通知渠道 |
| **Remote** | 远程访问、安全隧道 |

## 文档

- [架构设计](docs/ARCHITECTURE.md) - 系统架构和接口设计
- [API 文档](docs/api.md) - HTTP/MCP/gRPC API 详细说明
- [开发指南](docs/development.md) - 如何扩展和贡献
- [迁移分析](docs/reference/migration/analysis.md) - 从 ASMS 迁移
- [领域设计](docs/reference/domain/) - 各领域的详细设计

## 项目结构

```
ai-inference-managed-by-ai/
├── cmd/aima/              # CLI 入口
├── pkg/
│   ├── unit/              # 原子单元 (核心)
│   ├── service/           # 服务层
│   ├── workflow/          # 编排层
│   ├── gateway/           # 统一入口 (HTTP/MCP/gRPC)
│   └── infra/             # 基础设施
├── examples/              # 使用示例
├── configs/               # 配置文件
└── docs/                  # 文档
```

## 示例代码

查看 [examples/](examples/) 目录获取完整示例：

- [basic_usage.go](examples/basic_usage.go) - 基础使用
- [custom_command.go](examples/custom_command.go) - 自定义 Command
- [pipeline_example.go](examples/pipeline_example.go) - 工作流编排
- [event_subscription.go](examples/event_subscription.go) - 事件订阅
- [streaming_example.go](examples/streaming_example.go) - 流式推理

## 开发

本项目 100% 由 AI Coding Agent 开发，详见 [AGENTS.md](./AGENTS.md)。

### 构建

```bash
# 构建二进制
go build -o aima ./cmd/aima

# 运行测试
go test ./...

# 代码格式化
go fmt ./...
```

## 部署目标

- **主要平台**: NVIDIA DGX Spark (GB10 SoC, 128GB 统一内存)
- **次要平台**: NVIDIA Jetson, RTX 显卡, 通用 Linux/Windows/macOS
- **部署方式**: 单二进制文件，零外部依赖

## License

MIT License - 详见 [LICENSE](LICENSE) 文件
