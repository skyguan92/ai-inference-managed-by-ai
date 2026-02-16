# ai-inference-managed-by-ai

> 让 AI 管理的 AI 推理基础设施

## 项目概述

**AIMA** 是一个面向 AI Agent 和机器用户的 AI 推理基础设施管理平台。

### 核心设计理念

- **一切皆原子服务** - 所有功能拆分为最小可组合单元
- **四种接口类型** - Command、Query、Event、Resource
- **统一入口** - HTTP/MCP/CLI 共享同一套原子单元
- **AI First** - 接口设计优先考虑 AI Agent 可理解性

## 快速开始

```bash
# 查看文档
cd docs && cat README.md

# 了解架构
cat docs/ARCHITECTURE.md

# 查看迁移分析
cat docs/reference/migration/analysis.md
```

## 文档

完整文档位于 `docs/` 目录：

- [文档索引](docs/README.md)
- [架构设计](docs/ARCHITECTURE.md)
- [项目概览](docs/reference/overview.md)
- [迁移分析](docs/reference/migration/analysis.md)
- [领域映射表](docs/reference/migration/domain-mapping.md)

## 项目结构

```
ai-inference-managed-by-ai/
├── cmd/aima/              # 命令行入口
├── pkg/
│   ├── unit/              # 原子单元 (核心)
│   │   ├── device/
│   │   ├── model/
│   │   ├── engine/
│   │   ├── inference/
│   │   ├── resource/
│   │   ├── service/
│   │   ├── app/
│   │   ├── pipeline/
│   │   ├── alert/
│   │   └── remote/
│   ├── service/           # 服务层
│   ├── workflow/          # 编排层
│   ├── gateway/           # 统一入口
│   ├── infra/             # 基础设施
│   ├── config/            # 配置管理
│   └── cli/               # CLI 命令
├── configs/               # 配置文件
└── docs/                  # 文档
```

## 开发指南

本项目 100% 由 AI Coding Agent 开发，详见 **[AGENTS.md](./AGENTS.md)**。

所有 AI Agent 在开发前必须阅读 AGENTS.md 中的规则和指南。

## 开发状态

- [x] 目录结构创建
- [x] Go 模块初始化
- [x] 架构文档编写
- [x] 迁移分析完成
- [x] 开发规则制定 (AGENTS.md)
- [ ] 核心框架实现
- [ ] 领域原子单元实现
- [ ] Gateway 实现
- [ ] 适配器实现

## 关联项目

- **ASMS**: `/home/qujing/projects/asms` - 原 AI 服务管理系统

## License

MIT
