# AIMA Coder 规则

## 首次使用说明

**重要**: 首次切换到此模式时，请手动选择模型 `minimax/minimax-m2.5`
Kilo 会记住此选择（Sticky Models 机制）

## 开发工作流

### 1. 任务开始前
```bash
# 阅读规范
cat AGENTS.md
cat docs/ARCHITECTURE.md
cat docs/reference/domain/{相关领域}.md
```

### 2. 实现代码
- 遵循 AGENTS.md 中的代码规范
- 接口定义放文件顶部
- 实现紧随其后

### 3. 编写测试
- 核心模块：先写测试
- 非核心模块：后补测试

### 4. 完成后验证
```bash
go fmt ./...
go vet ./...
go test ./...
```

## 代码模板

### Command 实现
```go
// pkg/unit/{domain}/commands.go

type PullCommand struct {
    // fields
}

func (c *PullCommand) Name() string { return "model.pull" }
func (c *PullCommand) Domain() string { return "model" }
func (c *PullCommand) InputSchema() Schema { /* ... */ }
func (c *PullCommand) OutputSchema() Schema { /* ... */ }
func (c *PullCommand) Execute(ctx context.Context, input any) (any, error) {
    // implementation
}
```

### Query 实现
```go
// pkg/unit/{domain}/queries.go

type GetQuery struct {
    // fields
}

func (q *GetQuery) Name() string { return "model.get" }
// ... 类似 Command
```

## 输出格式

完成后报告：
```markdown
## 实现完成

### 新增/修改文件
- `pkg/unit/model/commands.go` - 实现 model.pull
- `pkg/unit/model/commands_test.go` - 测试文件

### 测试结果
```
$ go test ./pkg/unit/model/... -cover
ok  coverage: 75%
```

### 关键实现点
- [列出关键实现]
```
