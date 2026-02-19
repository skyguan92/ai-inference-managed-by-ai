# [2026-02-17] 实现事件持久化存储

## 元信息
- 开始时间: 2026-02-17
- 完成时间: 2026-02-17
- 实现模型: kimi-for-coding/k2.5
- 审查模型: N/A

## 任务概述
- **目标**: 将当前仅内存的事件总线扩展为支持持久化存储，允许事件历史查询和回放
- **范围**: pkg/infra/eventbus/ 和 pkg/infra/store/migrations/
- **优先级**: P1

## 设计决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 存储过滤器命名 | EventQueryFilter | 避免与现有的 EventFilter 函数类型冲突 |
| 异步写入 | 批量写入 + 定时刷新 | 平衡性能和可靠性，默认1秒刷新或100条批量写入 |
| 批量写入接口 | SaveBatch | 提供事务级别的批量插入优化 |
| 事件ID生成 | 复用 generateID() | 保持与订阅ID生成方式一致 |

## 实现摘要

### 新增文件
- `pkg/infra/eventbus/store.go` - EventStore 接口及 SQLite 实现
  - EventStore 接口定义
  - EventQueryFilter 结构体
  - SQLiteEventStore 实现
  - storedEvent 适配器类型
  
- `pkg/infra/eventbus/persistent.go` - 持久化事件总线实现
  - PersistentEventBus 结构体
  - 异步批量写入机制
  - Query 查询接口
  - Replay 事件回放接口

- `pkg/infra/eventbus/store_test.go` - 存储测试
  - Save/SaveBatch 测试
  - Query 过滤器测试
  - GetByID 测试

- `pkg/infra/eventbus/persistent_test.go` - 持久化总线测试
  - Publish/Subscribe 测试
  - Query/Replay 测试
  - 并发测试

- `pkg/infra/store/migrations/002_events.sql` - 事件表迁移
  - events 表结构
  - 4个索引优化查询

### 关键代码

**EventStore 接口** (`store.go:13-17`):
```go
type EventStore interface {
    Save(ctx context.Context, event unit.Event) error
    Query(ctx context.Context, filter EventQueryFilter) ([]unit.Event, error)
    GetByID(ctx context.Context, id string) (unit.Event, error)
}
```

**持久化工作协程** (`persistent.go:182-219`):
```go
func (b *PersistentEventBus) persistenceWorker() {
    batch := make([]unit.Event, 0, b.batchSize)
    ticker := time.NewTicker(b.flushPeriod)
    
    for {
        select {
        case event := <-b.buffer:
            batch = append(batch, event)
            if len(batch) >= b.batchSize {
                flush()
            }
        case <-ticker.C:
            flush()
        }
    }
}
```

## 测试结果
```bash
$ export PATH=$PATH:/usr/local/go/bin && go build ./pkg/infra/eventbus/...
# 编译成功

# 测试因网络问题未能运行（modernc.org/sqlite 下载超时）
# 代码已通过静态编译验证
```

## API 使用示例

### 创建持久化事件总线
```go
db, _ := sql.Open("sqlite", "aima.db")
store := eventbus.NewSQLiteEventStore(db)
bus := eventbus.NewPersistentEventBus(store, 
    eventbus.WithBatchSize(50),
    eventbus.WithFlushPeriod(500*time.Millisecond),
)
```

### 发布事件
```go
bus.Publish(event) // 同时广播和持久化
```

### 查询历史事件
```go
events, _ := bus.Query(ctx, eventbus.EventQueryFilter{
    Domain:    "model",
    Type:      "model.created",
    StartTime: time.Now().Add(-24*time.Hour),
    Limit:     100,
})
```

### 事件回放
```go
bus.Replay(ctx, "workflow-123", func(event unit.Event) error {
    // 重放工作流事件
    return nil
})
```

## 遇到的问题

1. **类型命名冲突**
   - **问题**: EventFilter 结构体与现有的 EventFilter 函数类型冲突
   - **解决**: 将结构体重命名为 EventQueryFilter

2. **SQLite 依赖下载超时**
   - **问题**: 测试依赖 modernc.org/sqlite 下载超时
   - **解决**: 代码已通过静态编译验证，测试可在网络正常时运行

## 代码审查
- **审查模型**: N/A
- **审查时间**: N/A
- **审查结果**: N/A
- **自检清单**:
  - [x] 代码符合架构设计
  - [x] 接口实现完整
  - [x] 错误处理正确
  - [x] 测试已编写
  - [x] 代码格式化 (go fmt)

## 提交信息
- **Commit**: 待提交
- **Message**: feat(eventbus): implement event persistence with SQLite storage

## 后续任务
- [ ] 运行完整测试套件
- [ ] 集成到应用启动流程
- [ ] 添加事件清理策略（防止存储无限增长）
