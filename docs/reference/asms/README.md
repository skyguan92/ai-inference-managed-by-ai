# ASMS 项目参考

本文件夹包含对原 ASMS 项目的参考文档，用于指导 AIMA 的迁移开发。

## 源码位置

```
/home/qujing/projects/asms
```

## 项目结构

```
asms/
├── cmd/                    # 命令行入口
│   ├── asms-agent/         # Agent 守护进程
│   ├── asms-ctl/           # CLI 工具
│   └── perf/               # 性能测试工具
│
├── pkg/                    # 核心包
│   ├── hal/                # 硬件抽象层
│   ├── model/              # 模型管理
│   ├── engine/             # 推理引擎
│   ├── service/            # 服务管理
│   ├── resource/           # 资源管理
│   ├── app/                # 应用管理
│   ├── pipeline/           # 管道编排
│   ├── remote/             # 远程访问
│   ├── fleet/              # 集群管理
│   ├── runtime/            # 运行时
│   ├── store/              # 存储
│   ├── api/                # HTTP API
│   ├── mcp/                # MCP 协议
│   ├── cli/                # CLI 命令
│   ├── config/             # 配置管理
│   ├── eventbus/           # 事件总线
│   ├── monitor/            # 监控
│   └── ...
│
├── configs/                # 配置文件
├── docs/                   # 文档
├── templates/              # 应用模板
├── pipelines/              # 管道定义
└── tests/                  # 测试
```

## 核心模块速查

### HAL (Hardware Abstraction Layer)

```go
// pkg/hal/interfaces.go
type Device interface {
    ID() string
    Name() string
    Vendor() string
    TotalMemory() uint64
    AvailableMemory() uint64
    Utilization() (float64, error)
    Temperature() (float64, error)
    PowerUsage() (float64, error)
    HealthStatus() (HealthStatus, error)
}
```

### Model Manager

```go
// pkg/model/manager.go
type Manager struct {
    db         *sql.DB
    downloader *downloader.Manager
    compat     *CompatibilityChecker
}

func (m *Manager) Create(model *Model) error
func (m *Manager) Get(id string) (*Model, error)
func (m *Manager) List(filter ListFilter) ([]*Model, error)
func (m *Manager) Delete(id string) error
```

### Engine Manager

```go
// pkg/engine/manager.go
type Manager struct {
    adapters   map[string]EngineAdapter
    processes  map[string]*EngineProcess
    router     *Router
    hal        *hal.Manager
}

func (m *Manager) Start(ctx context.Context, name string, config EngineConfig) (*EngineProcess, error)
func (m *Manager) Stop(ctx context.Context, process *EngineProcess) error
func (m *Manager) GetProcess(id string) (*EngineProcess, bool)
```

### Engine Adapter

```go
// pkg/engine/adapters/ollama.go
type OllamaAdapter struct {
    client *http.Client
    addr   string
}

func (a *OllamaAdapter) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
func (a *OllamaAdapter) ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
```

## 重要文件列表

### 接口定义

| 文件 | 说明 |
|------|------|
| `pkg/hal/interfaces.go` | 设备接口定义 |
| `pkg/engine/adapters/interfaces.go` | 引擎适配器接口 |
| `pkg/model/types.go` | 模型类型定义 |
| `pkg/service/model.go` | 服务类型定义 |
| `pkg/pipeline/types.go` | 管道类型定义 |

### 核心实现

| 文件 | 说明 |
|------|------|
| `pkg/hal/v2/manager.go` | HAL 管理器 |
| `pkg/model/manager.go` | 模型管理器 |
| `pkg/engine/manager.go` | 引擎管理器 |
| `pkg/service/manager.go` | 服务管理器 |
| `pkg/resource/manager.go` | 资源管理器 |
| `pkg/app/manager.go` | 应用管理器 |
| `pkg/pipeline/engine.go` | 管道引擎 |
| `pkg/remote/manager.go` | 远程管理器 |
| `pkg/fleet/alert.go` | 告警管理 |
| `pkg/store/db.go` | 数据库 |

### API Handlers

| 文件 | 说明 |
|------|------|
| `pkg/api/handlers/device.go` | 设备 API |
| `pkg/api/handlers/model.go` | 模型 API |
| `pkg/api/handlers/engine.go` | 引擎 API |
| `pkg/api/handlers/inference.go` | 推理 API |
| `pkg/api/handlers/service.go` | 服务 API |
| `pkg/api/handlers/app.go` | 应用 API |
| `pkg/api/handlers/pipeline.go` | 管道 API |
| `pkg/api/handlers/remote.go` | 远程 API |
| `pkg/api/handlers/alert.go` | 告警 API |

### MCP Tools

| 文件 | 说明 |
|------|------|
| `pkg/mcp/tools.go` | 工具框架 |
| `pkg/mcp/model_tools.go` | 模型工具 |
| `pkg/mcp/runtime_tools.go` | 运行时工具 |
| `pkg/mcp/alerts.go` | 告警工具 |
| `pkg/mcp/resources.go` | 资源订阅 |

## 复用策略

### 直接复用

以下模块可以直接复制或少量修改后复用：

- `pkg/store/` - SQLite 存储
- `pkg/eventbus/` - 事件总线
- `pkg/config/` - 配置管理
- `pkg/logging/` - 日志

### 封装复用

以下模块需要封装为原子单元接口：

- `pkg/hal/` → `pkg/unit/device/`
- `pkg/model/` → `pkg/unit/model/`
- `pkg/engine/` → `pkg/unit/engine/` + `pkg/unit/inference/`
- `pkg/resource/` → `pkg/unit/resource/`
- `pkg/service/` → `pkg/unit/service/`
- `pkg/app/` → `pkg/unit/app/`
- `pkg/pipeline/` → `pkg/unit/pipeline/`
- `pkg/fleet/alert.go` → `pkg/unit/alert/`
- `pkg/remote/` → `pkg/unit/remote/`

### 重构替换

以下模块需要重构或替换：

- `pkg/api/` → `pkg/gateway/http_adapter.go`
- `pkg/mcp/` → `pkg/gateway/mcp_adapter.go`
- `pkg/cli/` → `pkg/gateway/cli_adapter.go`

## 关键依赖

```go
// go.mod 中的主要依赖
require (
    github.com/docker/docker v24.0.0
    github.com/mattn/go-sqlite3 v1.14.0
    github.com/spf13/cobra v1.7.0
    github.com/spf13/viper v1.16.0
    go.uber.org/zap v1.24.0
)
```

## 测试覆盖率

当前 ASMS 测试覆盖率约 60%，主要测试文件：

- `pkg/engine/*_test.go`
- `pkg/model/*_test.go`
- `pkg/service/*_test.go`
- `pkg/pipeline/*_test.go`
- `pkg/mcp/*_test.go`

## 已知问题

1. **Resource 接口缺失**: ASMS 没有实现 Resource URI 访问模式
2. **Event 发布不完整**: 大部分操作没有发布事件
3. **API/MCP/CLI 重复逻辑**: 三套入口有大量重复代码
4. **Schema 不统一**: 各接口的输入输出格式不一致

## 迁移注意事项

1. **保持兼容性**: 迁移过程中保持 API 兼容
2. **增量迁移**: 每次迁移一个领域，确保测试通过
3. **文档同步**: 迁移时更新相关文档
4. **性能基准**: 迁移前后进行性能对比
