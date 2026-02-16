# AIMA Orchestrator 模式详细规则

## Sticky Models 机制

Kilo Code 支持 **Sticky Models**：每个模式会记住上次使用的模型。

### 首次使用配置

在使用 Orchestrator 委派任务前，需要先配置各模式的默认模型：

| 模式 | 切换方法 | 选择模型 |
|------|----------|----------|
| aima-coder | 切换到 💻 AIMA Coder | minimax/minimax-m2.5 |
| aima-reviewer | 切换到 🔍 AIMA Reviewer | glm-5 |
| aima-writer | 切换到 📝 AIMA Writer | kimi-for-coding/k2.5 |
| aima-architect | 切换到 🏗️ AIMA Architect | glm-5 |

**重要**: 首次使用每个模式时，必须手动切换并选择对应模型。之后 Kilo 会自动记住。

## 可用模型列表

| 模型 ID | 用途 | 特点 |
|---------|------|------|
| `glm-5` | 复杂推理、架构设计、代码审查 | 最强推理能力 |
| `minimax/minimax-m2.5` | 代码实现、快速探索 | 高性价比 |
| `kimi-for-coding/k2.5` | 文档编写、测试设计 | 语言表达优秀 |

## 任务-模式映射决策树

```
任务开始
    │
    ├─ 是否涉及架构决策？
    │   └─ 是 → aima-architect (glm-5)
    │
    ├─ 是否涉及代码审查？
    │   └─ 是 → aima-reviewer (glm-5)
    │
    ├─ 是否是代码实现？
    │   └─ 是 → aima-coder (minimax-m2.5)
    │
    ├─ 是否是文档/测试？
    │   └─ 是 → aima-writer (kimi-k2.5)
    │
    └─ 是否是快速探索？
        └─ 是 → explore (内置模式)
```

## Task tool 使用

### 基本调用

```
Task tool 参数:
- description: 简短任务描述（3-5词）
- prompt: 详细任务描述和上下文
- subagent_type: "general" 或 "explore"
```

### 委派流程

1. **分解任务** - 确定需要哪种类型的子任务
2. **选择模式** - 根据任务类型选择合适的子模式
3. **准备上下文** - 整理子任务需要的背景信息
4. **委派执行** - 调用 Task tool
5. **审查结果** - 验证输出是否符合预期

## 任务分解原则

### SMART 原则

每个子任务必须满足：
- **S**pecific - 具体明确
- **M**easurable - 可衡量
- **A**chievable - 可实现
- **R**elevant - 与目标相关
- **T**ime-bound - 有时间限制

### 任务粒度

- 单个子任务预计时间：15分钟 - 2小时
- 如果超过2小时，继续拆分
- 如果少于15分钟，考虑合并

## 委派模板

### 代码实现任务 → aima-coder

```markdown
**任务**: 实现 [功能名称]

**上下文**:
- 项目路径: /home/qujing/projects/ai-inference-managed-by-ai
- 相关文件:
  - pkg/unit/types.go (接口定义)
  - docs/ARCHITECTURE.md (架构文档)
- 依赖: [已完成的依赖任务]

**要求**:
1. 遵循 AGENTS.md 中的代码规范
2. 实现完整的接口方法
3. 编写对应的单元测试

**期望输出**:
- 新增/修改的文件列表
- 测试运行结果
- 实现摘要
```

### 代码审查任务 → aima-reviewer

```markdown
**任务**: 审查 [功能名称] 的代码

**上下文**:
- 待审查文件:
  - [文件路径1]
  - [文件路径2]
- 审查标准:
  - 架构符合性 (docs/ARCHITECTURE.md)
  - 代码规范 (AGENTS.md)
  - 测试覆盖

**审查清单**:
- [ ] 代码符合架构设计
- [ ] 接口实现完整
- [ ] 错误处理正确
- [ ] 测试覆盖充分
- [ ] 命名规范一致
- [ ] 无安全漏洞

**期望输出**:
- 审查结果: 通过/需修改/退回
- 具体意见列表
```

### 文档/测试任务 → aima-writer

```markdown
**任务**: 编写 [文档名称] / 设计 [测试用例]

**上下文**:
- 文档类型: API文档/用户指南/开发日志/测试用例
- 目标读者: [读者群体]
- 相关代码: [代码路径]

**要求**:
1. 结构清晰，易于理解
2. 包含代码示例
3. 符合项目文档风格

**期望输出**:
- 文件路径
- 内容摘要
```

### 架构设计任务 → aima-architect

```markdown
**任务**: 设计 [架构名称]

**背景**: [设计背景]

**需求**:
- 功能需求: [...]
- 非功能需求: [...]

**约束**: [约束条件]

**期望输出**:
- 设计文档
- 候选方案
- 推荐方案及理由
```

## 进度跟踪

### 状态定义

| 状态 | 说明 | 可转换到 |
|------|------|----------|
| pending | 等待开始 | in_progress, cancelled |
| in_progress | 正在执行 | completed, blocked, pending |
| completed | 已完成 | - |
| blocked | 被阻塞 | pending, in_progress |
| cancelled | 已取消 | - |

### 使用 TodoWrite

```json
{
  "todos": [
    {"id": "task-001", "content": "实现 model.pull", "status": "in_progress", "priority": "high"},
    {"id": "task-002", "content": "审查代码", "status": "pending", "priority": "high"}
  ]
}
```

### 日报格式

```markdown
# 📅 开发日报 - YYYY-MM-DD

## 今日完成
- [x] Task-001: [任务名称]

## 今日进行中
- [ ] Task-003: [任务名称] (50%)

## 今日阻塞
- Task-004: [任务名称]
  - 原因: [原因]
  - 预计解除: [时间]

## 明日计划
1. [任务]
2. [任务]
```

## 异常处理

### Subagent 失败

1. **第一次失败** - 分析原因，调整任务描述，重新委派
2. **第二次失败** - 考虑更换模式，或进一步拆分任务
3. **第三次失败** - 标记为 blocked，等待人工介入

### 上下文膨胀

当会话上下文超过 60%：
1. 立即使用 /compact 压缩
2. 将详细讨论移到 subagent
3. 只保留关键决策在主会话

## 质量检查点

### 功能完成后

- [ ] 代码已通过 go vet
- [ ] 代码已通过 go fmt
- [ ] 测试覆盖率达标
- [ ] 代码审查已通过
- [ ] 开发日志已创建
- [ ] Git 已提交

### 里程碑完成后

- [ ] 所有功能测试通过
- [ ] 集成测试通过
- [ ] 文档已更新
- [ ] 无已知 bug
- [ ] 代码审查全部通过
