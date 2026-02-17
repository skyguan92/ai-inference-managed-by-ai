# [2026-02-17] 实现 Event 发布机制

## 元信息
- 开始时间: 2026-02-17
- 完成时间: 2026-02-17
- 实现模型: kimi-for-coding/k2.5
- 审查模型: 待审查

## 任务概述
- **目标**: 为所有领域的 Command 和 Query 实现事件发布功能，将原子单元执行与事件总线连接
- **范围**: 所有领域 (model, engine, device, inference, resource, service, app, pipeline, alert, remote)
- **优先级**: P0

## 设计决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 事件结构 | 统一 ExecutionEvent | 简化事件处理，包含完整上下文 |
| 事件类型 | started, completed, failed | 覆盖完整执行生命周期 |
| 实现方式 | ExecutionContext 辅助类 | 简化事件发布代码，减少重复 |
| 向后兼容 | 保留原构造函数 | 不影响现有代码 |

## 实现摘要

### 新增文件
- `pkg/unit/events.go` - 通用事件类型和 ExecutionContext

### 修改文件 (按领域)

#### model 领域
- `pkg/unit/model/commands.go`
  - CreateCommand: 添加 events 字段和 WithEvents 构造函数
  - DeleteCommand: 添加 events 字段和 WithEvents 构造函数
  - PullCommand: 添加事件发布
  - ImportCommand: 添加事件发布
  - VerifyCommand: 添加事件发布
  
- `pkg/unit/model/queries.go`
  - GetQuery: 添加事件发布
  - ListQuery: 添加事件发布
  - SearchQuery: 添加事件发布
  - EstimateResourcesQuery: 添加事件发布

#### engine 领域
- `pkg/unit/engine/commands.go`
  - StartCommand: 添加事件发布
  - StopCommand: 添加事件发布
  - RestartCommand: 添加事件发布
  - InstallCommand: 添加事件发布
  
- `pkg/unit/engine/queries.go`
  - GetQuery: 添加事件发布
  - ListQuery: 添加事件发布
  - FeaturesQuery: 添加事件发布

#### device 领域
- `pkg/unit/device/commands.go`
  - DetectCommand: 添加事件发布
  - SetPowerLimitCommand: 添加事件发布
  
- `pkg/unit/device/queries.go`
  - InfoQuery: 添加事件发布
  - MetricsQuery: 添加事件发布
  - HealthQuery: 添加事件发布

#### inference 领域
- `pkg/unit/inference/commands.go`
  - ChatCommand: 添加事件发布
  - CompleteCommand: 添加事件发布
  - EmbedCommand: 添加事件发布
  - TranscribeCommand: 添加事件发布
  - SynthesizeCommand: 添加事件发布
  - GenerateImageCommand: 添加事件发布
  - GenerateVideoCommand: 添加事件发布
  - RerankCommand: 添加事件发布
  - DetectCommand: 添加事件发布
  
- `pkg/unit/inference/queries.go`
  - ModelsQuery: 添加事件发布
  - VoicesQuery: 添加事件发布

#### resource 领域
- `pkg/unit/resource/commands.go`
  - AllocateCommand: 添加事件发布
  - ReleaseCommand: 添加事件发布
  - UpdateSlotCommand: 添加事件发布
  
- `pkg/unit/resource/queries.go`
  - StatusQuery: 添加事件发布
  - BudgetQuery: 添加事件发布
  - AllocationsQuery: 添加事件发布
  - CanAllocateQuery: 添加事件发布

#### service 领域
- `pkg/unit/service/commands.go`
  - CreateCommand: 添加事件发布
  - DeleteCommand: 添加事件发布
  - ScaleCommand: 添加事件发布
  - StartCommand: 添加事件发布
  - StopCommand: 添加事件发布
  
- `pkg/unit/service/queries.go`
  - GetQuery: 添加事件发布
  - ListQuery: 添加事件发布
  - RecommendQuery: 添加事件发布

#### app 领域
- `pkg/unit/app/commands.go`
  - InstallCommand: 添加事件发布
  - UninstallCommand: 添加事件发布
  - StartCommand: 添加事件发布
  - StopCommand: 添加事件发布
  
- `pkg/unit/app/queries.go`
  - GetQuery: 添加事件发布
  - ListQuery: 添加事件发布
  - LogsQuery: 添加事件发布
  - TemplatesQuery: 添加事件发布

#### pipeline 领域
- `pkg/unit/pipeline/commands.go`
  - CreateCommand: 添加事件发布
  - DeleteCommand: 添加事件发布
  - RunCommand: 添加事件发布
  - CancelCommand: 添加事件发布
  
- `pkg/unit/pipeline/queries.go`
  - GetQuery: 添加事件发布
  - ListQuery: 添加事件发布
  - StatusQuery: 添加事件发布
  - ValidateQuery: 添加事件发布

#### alert 领域
- `pkg/unit/alert/commands.go`
  - CreateRuleCommand: 添加事件发布
  - UpdateRuleCommand: 添加事件发布
  - DeleteRuleCommand: 添加事件发布
  - AcknowledgeCommand: 添加事件发布
  - ResolveCommand: 添加事件发布
  
- `pkg/unit/alert/queries.go`
  - ListRulesQuery: 添加事件发布
  - HistoryQuery: 添加事件发布
  - ActiveQuery: 添加事件发布

#### remote 领域
- `pkg/unit/remote/commands.go`
  - EnableCommand: 添加事件发布
  - DisableCommand: 添加事件发布
  - ExecCommand: 添加事件发布
  
- `pkg/unit/remote/queries.go`
  - StatusQuery: 添加事件发布
  - AuditQuery: 添加事件发布

## 事件载荷结构

```go
type ExecutionEvent struct {
    EventType     string    `json:"event_type"`     // execution_started, execution_completed, execution_failed
    Domain        string    `json:"domain"`         // model, engine, device, etc.
    UnitName      string    `json:"unit_name"`      // command/query name
    Input         any       `json:"input,omitempty"`
    Output        any       `json:"output,omitempty"`
    Error         string    `json:"error,omitempty"`
    Timestamp     time.Time `json:"timestamp"`
    CorrelationID string    `json:"correlation_id"`
    DurationMs    int64     `json:"duration_ms,omitempty"`
}
```

## 使用示例

```go
// 创建带事件发布的 Command
cmd := model.NewCreateCommandWithEvents(store, eventPublisher)

// 或使用默认构造函数（不发布事件）
cmd := model.NewCreateCommand(store)

// Execute 方法会自动发布事件
result, err := cmd.Execute(ctx, input)
```

## 验收标准检查

- [x] 所有 Command 都有事件发布
- [x] 所有 Query 都有事件发布
- [x] 事件包含完整的上下文信息 (unit_name, input, output/error, timestamp, correlation_id)
- [ ] 单元测试验证事件发布

## 注意事项

1. 向后兼容：保留了原有的构造函数，不传 eventPublisher 时不会发布事件
2. 错误处理：事件发布失败不会中断 Command/Query 的执行
3. 性能：使用 NoopEventPublisher 避免 nil 检查开销
4. 关联 ID：每次执行生成唯一的 correlation_id，用于追踪请求链路

## 提交信息

```
feat(unit): implement event publishing for all commands and queries

- Add ExecutionEvent and ExecutionContext in pkg/unit/events.go
- Add EventPublisher support to all Command structs
- Add EventPublisher support to all Query structs
- Publish started, completed, failed events in Execute methods
- Maintain backward compatibility with existing constructors

Refs: Event publishing implementation
Log: .ai/development-log/2026-02-17-event-publishing-implementation.md
```
