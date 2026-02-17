# 开发指南

本指南介绍如何为 AIMA 添加新的功能单元和扩展现有功能。

## 目录

- [快速开始](#快速开始)
- [添加新的 Command](#添加新的-command)
- [添加新的 Query](#添加新的-query)
- [添加新的 Resource](#添加新的-resource)
- [添加新的事件](#添加新的事件)
- [测试规范](#测试规范)
- [最佳实践](#最佳实践)

---

## 快速开始

### 环境准备

```bash
# 克隆仓库
git clone https://github.com/jguan/ai-inference-managed-by-ai
cd ai-inference-managed-by-ai

# 安装依赖
go mod download

# 运行测试
go test ./...
```

### 项目结构

```
pkg/
├── unit/              # 原子单元
│   ├── types.go       # 核心接口定义
│   ├── registry.go    # 注册表
│   ├── schema.go      # Schema 定义
│   └── {domain}/      # 领域实现
│       ├── commands.go
│       ├── queries.go
│       ├── resources.go
│       └── events.go
├── service/           # 服务层
├── workflow/          # 编排层
├── gateway/           # 统一入口
└── infra/             # 基础设施
```

---

## 添加新的 Command

Command 是有副作用的操作。以下是创建 Command 的完整步骤。

### 1. 定义 Command 结构

```go
// pkg/unit/mydomain/commands.go
package mydomain

import (
    "context"
    "fmt"
    
    "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// MyCommand 实现具体的业务逻辑
type MyCommand struct {
    store  MyStore
    client MyClient
}

// NewMyCommand 创建 Command 实例
func NewMyCommand(store MyStore, client MyClient) *MyCommand {
    return &MyCommand{
        store:  store,
        client: client,
    }
}
```

### 2. 实现 Command 接口

```go
// Name 返回 Command 名称（格式：domain.action）
func (c *MyCommand) Name() string {
    return "mydomain.myaction"
}

// Domain 返回所属领域
func (c *MyCommand) Domain() string {
    return "mydomain"
}

// Description 返回描述
func (c *MyCommand) Description() string {
    return "执行我的操作"
}

// InputSchema 定义输入参数
func (c *MyCommand) InputSchema() unit.Schema {
    return unit.Schema{
        Type: "object",
        Properties: map[string]unit.Field{
            "param1": {
                Name: "param1",
                Schema: unit.Schema{
                    Type:        "string",
                    Description: "参数1",
                    Required:    []string{"param1"},
                },
            },
            "param2": {
                Name: "param2",
                Schema: unit.Schema{
                    Type:        "integer",
                    Description: "参数2",
                    Default:     10,
                    Min:         float64Ptr(1),
                    Max:         float64Ptr(100),
                },
            },
        },
        Required: []string{"param1"},
        Optional: []string{"param2"},
    }
}

// OutputSchema 定义输出结构
func (c *MyCommand) OutputSchema() unit.Schema {
    return unit.Schema{
        Type: "object",
        Properties: map[string]unit.Field{
            "result": {
                Name: "result",
                Schema: unit.Schema{
                    Type:        "string",
                    Description: "操作结果",
                },
            },
            "id": {
                Name: "id",
                Schema: unit.Schema{
                    Type:        "string",
                    Description: "生成的 ID",
                },
            },
        },
    }
}

// Examples 提供使用示例
func (c *MyCommand) Examples() []unit.Example {
    return []unit.Example{
        {
            Input: map[string]any{
                "param1": "value1",
                "param2": 20,
            },
            Output: map[string]any{
                "result": "success",
                "id":     "generated-id-123",
            },
            Description: "基本用法",
        },
    }
}
```

### 3. 实现 Execute 方法

```go
// Execute 执行命令
func (c *MyCommand) Execute(ctx context.Context, input any) (any, error) {
    // 解析输入
    params, ok := input.(map[string]any)
    if !ok {
        return nil, fmt.Errorf("invalid input type: expected map[string]any")
    }
    
    param1, ok := params["param1"].(string)
    if !ok || param1 == "" {
        return nil, fmt.Errorf("param1 is required")
    }
    
    param2 := 10
    if v, ok := params["param2"]; ok {
        if vi, ok := v.(float64); ok {
            param2 = int(vi)
        }
    }
    
    // 执行业务逻辑
    result, err := c.doSomething(ctx, param1, param2)
    if err != nil {
        return nil, fmt.Errorf("myaction failed: %w", err)
    }
    
    return map[string]any{
        "result": "success",
        "id":     result.ID,
    }, nil
}
```

### 4. 注册到 Registry

```go
// pkg/unit/mydomain/register.go
package mydomain

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

func Register(registry *unit.Registry, store MyStore, client MyClient) {
    // 注册 Command
    registry.RegisterCommand(NewMyCommand(store, client))
    
    // 注册其他单元...
}
```

---

## 添加新的 Query

Query 是无副作用的查询操作。实现方式与 Command 类似，但语义上保证无副作用。

```go
// pkg/unit/mydomain/queries.go
type MyQuery struct {
    store MyStore
}

func NewMyQuery(store MyStore) *MyQuery {
    return &MyQuery{store: store}
}

func (q *MyQuery) Name() string { return "mydomain.myquery" }
func (q *MyQuery) Domain() string { return "mydomain" }
func (q *MyQuery) Description() string { return "查询数据" }

func (q *MyQuery) InputSchema() unit.Schema {
    return unit.Schema{
        Type: "object",
        Properties: map[string]unit.Field{
            "id": {
                Name: "id",
                Schema: unit.Schema{
                    Type:        "string",
                    Description: "查询 ID",
                    Required:    []string{"id"},
                },
            },
        },
    }
}

func (q *MyQuery) OutputSchema() unit.Schema {
    return unit.Schema{
        Type: "object",
        Properties: map[string]unit.Field{
            "data": {
                Name: "data",
                Schema: unit.Schema{
                    Type:        "object",
                    Description: "查询结果",
                },
            },
        },
    }
}

func (q *MyQuery) Examples() []unit.Example {
    return []unit.Example{
        {
            Input:  map[string]any{"id": "test-id"},
            Output: map[string]any{"data": map[string]any{"name": "Test"}},
        },
    }
}

func (q *MyQuery) Execute(ctx context.Context, input any) (any, error) {
    params := input.(map[string]any)
    id := params["id"].(string)
    
    data, err := q.store.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    
    return map[string]any{"data": data}, nil
}
```

---

## 添加新的 Resource

Resource 是可寻址的状态资源，支持实时订阅。

```go
// pkg/unit/mydomain/resources.go
type MyResource struct {
    uri   string
    store MyStore
}

func NewMyResource(uri string, store MyStore) *MyResource {
    return &MyResource{uri: uri, store: store}
}

func (r *MyResource) URI() string { return r.uri }
func (r *MyResource) Domain() string { return "mydomain" }

func (r *MyResource) Schema() unit.Schema {
    return unit.Schema{
        Type:        "object",
        Description: "我的资源",
        Properties: map[string]unit.Field{
            "status": {
                Name: "status",
                Schema: unit.Schema{
                    Type: "string",
                    Enum: []any{"pending", "running", "completed"},
                },
            },
        },
    }
}

// Get 获取当前状态
func (r *MyResource) Get(ctx context.Context) (any, error) {
    // 从 URI 解析 ID
    id := extractIDFromURI(r.uri)
    return r.store.GetStatus(ctx, id)
}

// Watch 订阅资源变更
func (r *MyResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
    updates := make(chan unit.ResourceUpdate, 10)
    
    go func() {
        defer close(updates)
        
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case t := <-ticker.C:
                data, err := r.Get(ctx)
                if err != nil {
                    updates <- unit.ResourceUpdate{
                        URI:       r.uri,
                        Timestamp: t,
                        Operation: "error",
                        Error:     err,
                    }
                    continue
                }
                
                updates <- unit.ResourceUpdate{
                    URI:       r.uri,
                    Timestamp: t,
                    Operation: "update",
                    Data:      data,
                }
            }
        }
    }()
    
    return updates, nil
}
```

### Resource Factory（动态资源）

对于动态 URI 模式（如 `asms://mydomain/{id}`），使用 ResourceFactory：

```go
type MyResourceFactory struct {
    store MyStore
}

func (f *MyResourceFactory) Pattern() string {
    return "asms://mydomain/*"
}

func (f *MyResourceFactory) CanCreate(uri string) bool {
    return strings.HasPrefix(uri, "asms://mydomain/")
}

func (f *MyResourceFactory) Create(uri string) (unit.Resource, error) {
    return NewMyResource(uri, f.store), nil
}

// 注册
registry.RegisterResourceFactory(&MyResourceFactory{store: store})
```

---

## 添加新的事件

```go
// pkg/unit/mydomain/events.go
type MyEvent struct {
    eventType     string
    domain        string
    payload       any
    timestamp     time.Time
    correlationID string
}

func NewMyEvent(payload any, correlationID string) *MyEvent {
    return &MyEvent{
        eventType:     "mydomain.myevent",
        domain:        "mydomain",
        payload:       payload,
        timestamp:     time.Now(),
        correlationID: correlationID,
    }
}

func (e *MyEvent) Type() string        { return e.eventType }
func (e *MyEvent) Domain() string      { return e.domain }
func (e *MyEvent) Payload() any        { return e.payload }
func (e *MyEvent) Timestamp() time.Time { return e.timestamp }
func (e *MyEvent) CorrelationID() string { return e.correlationID }

// 在 Command 中发布事件
func (c *MyCommand) Execute(ctx context.Context, input any) (any, error) {
    // ... 执行逻辑
    
    // 发布事件
    if bus := unit.GetEventBus(ctx); bus != nil {
        event := NewMyEvent(result, unit.GetTraceID(ctx))
        bus.Publish(ctx, event)
    }
    
    return result, nil
}
```

---

## 测试规范

### 测试文件命名

```
pkg/unit/mydomain/commands.go    -> commands_test.go
pkg/unit/mydomain/queries.go     -> queries_test.go
pkg/unit/mydomain/resources.go   -> resources_test.go
```

### Command 测试

```go
// pkg/unit/mydomain/commands_test.go
package mydomain

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock Store
type MockStore struct {
    mock.Mock
}

func (m *MockStore) Get(ctx context.Context, id string) (any, error) {
    args := m.Called(ctx, id)
    return args.Get(0), args.Error(1)
}

func TestMyCommand_Execute(t *testing.T) {
    tests := []struct {
        name      string
        input     any
        mockSetup func(*MockStore)
        want      any
        wantErr   bool
    }{
        {
            name:  "success",
            input: map[string]any{"param1": "test", "param2": 20},
            mockSetup: func(m *MockStore) {
                m.On("Get", mock.Anything, "test").Return(&Result{ID: "123"}, nil)
            },
            want:    map[string]any{"result": "success", "id": "123"},
            wantErr: false,
        },
        {
            name:    "invalid input type",
            input:   "invalid",
            wantErr: true,
        },
        {
            name:    "missing required param",
            input:   map[string]any{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            store := new(MockStore)
            if tt.mockSetup != nil {
                tt.mockSetup(store)
            }
            
            cmd := NewMyCommand(store, nil)
            got, err := cmd.Execute(context.Background(), tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
            store.AssertExpectations(t)
        })
    }
}

func TestMyCommand_Schema(t *testing.T) {
    cmd := NewMyCommand(nil, nil)
    
    schema := cmd.InputSchema()
    assert.Equal(t, "object", schema.Type)
    assert.Contains(t, schema.Required, "param1")
    
    output := cmd.OutputSchema()
    assert.Equal(t, "object", output.Type)
    assert.Contains(t, output.Properties, "result")
}
```

### 覆盖率要求

| 模块类型 | 最低覆盖率 |
|----------|-----------|
| 核心框架 | 80% |
| 原子单元 | 70% |
| 服务层 | 60% |
| 适配器 | 50% |

---

## 最佳实践

### 1. 错误处理

```go
// 使用带上下文的错误
return nil, fmt.Errorf("myaction failed for %s: %w", id, err)

// 定义领域错误
var (
    ErrNotFound     = errors.New("not found")
    ErrInvalidInput = errors.New("invalid input")
)

// 在 Execute 中转换错误
if errors.Is(err, ErrNotFound) {
    return nil, NewNotFoundError(id)
}
```

### 2. 上下文使用

```go
func (c *MyCommand) Execute(ctx context.Context, input any) (any, error) {
    // 获取请求元信息
    requestID := unit.GetRequestID(ctx)
    traceID := unit.GetTraceID(ctx)
    
    // 设置超时
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // 检查取消
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // ...
}
```

### 3. Schema 设计

```go
// 使用合理的默认值
Default: defaultValue

// 使用验证规则
Min:       float64Ptr(0),
Max:       float64Ptr(100),
Pattern:   "^[a-zA-Z0-9_-]+$",
MinLength: intPtr(1),
MaxLength: intPtr(256),

// 提供示例
Examples: []any{
    map[string]any{"name": "example"},
}
```

### 4. 并发安全

```go
type MyCommand struct {
    mu     sync.RWMutex
    cache  map[string]any
    store  MyStore
}

func (c *MyCommand) Execute(ctx context.Context, input any) (any, error) {
    c.mu.RLock()
    cached, ok := c.cache[key]
    c.mu.RUnlock()
    
    if ok {
        return cached, nil
    }
    
    // ... 计算结果
    
    c.mu.Lock()
    c.cache[key] = result
    c.mu.Unlock()
    
    return result, nil
}
```

### 5. 日志记录

```go
// 使用结构化日志
log.Printf("[MyCommand] Execute started: request_id=%s param1=%s", 
    unit.GetRequestID(ctx), param1)

// 记录错误
log.Printf("[MyCommand] Execute failed: request_id=%s error=%v", 
    unit.GetRequestID(ctx), err)
```

---

## 贡献指南

1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/my-feature`)
3. 编写代码和测试
4. 运行测试 (`go test ./...`)
5. 格式化代码 (`go fmt ./...`)
6. 提交更改 (`git commit -am 'Add my feature'`)
7. 推送到分支 (`git push origin feature/my-feature`)
8. 创建 Pull Request

---

## 参考

- [架构设计](ARCHITECTURE.md)
- [API 文档](api.md)
- [领域设计](reference/domain/)
