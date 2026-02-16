# App Domain

Docker 应用管理领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/app/` | `pkg/app/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `app.install` | `{template, name?, config?}` | `{app_id}` | 安装应用 |
| `app.uninstall` | `{app_id, remove_data?}` | `{success}` | 卸载应用 |
| `app.start` | `{app_id}` | `{success}` | 启动应用 |
| `app.stop` | `{app_id, timeout?}` | `{success}` | 停止应用 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `app.get` | `{app_id}` | `{id, name, template, status, ports, volumes, metrics}` | 应用详情 |
| `app.list` | `{status?}` | `{apps: []}` | 列出应用 |
| `app.logs` | `{app_id, tail?, since?}` | `{logs: []}` | 获取日志 |
| `app.templates` | `{category?}` | `{templates: []}` | 列出模板 |

## 应用分类

```go
type AppCategory string

const (
    AppCategoryAIChat     AppCategory = "ai-chat"
    AppCategoryDevTool    AppCategory = "dev-tool"
    AppCategoryAIWorkflow AppCategory = "ai-workflow"
    AppCategoryMonitoring AppCategory = "monitoring"
    AppCategoryStorage    AppCategory = "storage"
    AppCategoryCustom     AppCategory = "custom"
)
```

## 实现文件

```
pkg/app/
├── types.go               # 应用类型
├── manager.go             # 应用管理器
├── templates.go           # 应用模板
├── docker.go              # Docker 集成
└── docker_real.go         # 真实 Docker 操作
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `app.install` | ✅ | `app/manager.go` Install() |
| `app.uninstall` | ✅ | `app/manager.go` Uninstall() |
| `app.start` | ✅ | `app/manager.go` Start() |
| `app.stop` | ✅ | `app/manager.go` Stop() |
| `app.get` | ✅ | `app/manager.go` Get() |
| `app.list` | ✅ | `app/manager.go` List() |
| `app.logs` | ✅ | `app/docker.go` Logs() |
| `app.templates` | ✅ | `app/templates.go` |
