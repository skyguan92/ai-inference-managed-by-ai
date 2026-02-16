# AIMA 项目文档

> ai-inference-managed-by-ai - 让 AI 管理的 AI 推理基础设施

## 文档导航

### 核心文档

| 文档 | 说明 | 路径 |
|------|------|------|
| [架构设计](./ARCHITECTURE.md) | 新系统架构设计文档 | `docs/ARCHITECTURE.md` |
| [项目概览](./reference/overview.md) | 项目整体介绍和目标 | `docs/reference/overview.md` |

### 迁移参考

| 文档 | 说明 | 路径 |
|------|------|------|
| [迁移分析报告](./reference/migration/analysis.md) | ASMS → AIMA 完整迁移分析 | `docs/reference/migration/analysis.md` |
| [领域映射表](./reference/migration/domain-mapping.md) | 10 个核心领域的详细映射 | `docs/reference/migration/domain-mapping.md` |
| [迁移优先级](./reference/migration/priority.md) | 迁移阶段和工作量评估 | `docs/reference/migration/priority.md` |

### 领域详情

| 领域 | 说明 | 路径 |
|------|------|------|
| [Device](./reference/domain/device.md) | 硬件设备管理 | `docs/reference/domain/device.md` |
| [Model](./reference/domain/model.md) | 模型管理 | `docs/reference/domain/model.md` |
| [Engine](./reference/domain/engine.md) | 推理引擎管理 | `docs/reference/domain/engine.md` |
| [Inference](./reference/domain/inference.md) | 推理服务 | `docs/reference/domain/inference.md` |
| [Resource](./reference/domain/resource.md) | 资源管理 | `docs/reference/domain/resource.md` |
| [Service](./reference/domain/service.md) | 服务管理 | `docs/reference/domain/service.md` |
| [App](./reference/domain/app.md) | 应用管理 | `docs/reference/domain/app.md` |
| [Pipeline](./reference/domain/pipeline.md) | 管道编排 | `docs/reference/domain/pipeline.md` |
| [Alert](./reference/domain/alert.md) | 告警管理 | `docs/reference/domain/alert.md` |
| [Remote](./reference/domain/remote.md) | 远程访问 | `docs/reference/domain/remote.md` |

### 原项目参考

| 文档 | 说明 | 路径 |
|------|------|------|
| [ASMS 项目参考](./reference/asms/README.md) | 原项目结构和模块说明 | `docs/reference/asms/README.md` |

---

## 快速开始

### 对于 AI Coding Agent

1. **理解架构**: 先阅读 [架构设计](./ARCHITECTURE.md) 了解四种原子单元接口
2. **了解现状**: 阅读 [迁移分析报告](./reference/migration/analysis.md) 了解已有代码
3. **查看映射**: 使用 [领域映射表](./reference/migration/domain-mapping.md) 确定具体实现位置
4. **开始开发**: 按 [迁移优先级](./reference/migration/priority.md) 顺序进行

### 关键概念

```
┌─────────────────────────────────────────────────────────────────────┐
│                      四种原子单元接口 (核心)                          │
├─────────────────────────────────────────────────────────────────────┤
│  Command  - 有副作用的操作 (create, delete, start, stop...)         │
│  Query    - 无副作用的查询 (get, list, search, status...)           │
│  Event    - 异步事件 (created, deleted, started, error...)          │
│  Resource - 可寻址资源 (asms://device/{id}, asms://model/{id}...)   │
└─────────────────────────────────────────────────────────────────────┘
```

### 命名规范

```
{domain}.{action}

示例:
- model.pull      # 从源拉取模型
- model.list      # 列出模型
- inference.chat  # 聊天补全
- engine.start    # 启动引擎
- resource.allocate # 分配资源
```

---

## 项目状态

- [x] 目录结构创建
- [x] Go 模块初始化
- [x] 架构文档编写
- [x] 迁移分析完成
- [ ] 核心框架实现
- [ ] 领域原子单元实现
- [ ] Gateway 实现
- [ ] 适配器实现
