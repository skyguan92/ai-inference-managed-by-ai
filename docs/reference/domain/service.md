# Service Domain

模型服务化领域（长期运行的服务实例）。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/service/` | `pkg/service/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `service.create` | `{model_id, resource_class?, replicas?, persistent?}` | `{service_id}` | 创建服务 |
| `service.delete` | `{service_id}` | `{success}` | 删除服务 |
| `service.scale` | `{service_id, replicas}` | `{success}` | 扩缩容 |
| `service.start` | `{service_id}` | `{success}` | 启动服务 |
| `service.stop` | `{service_id, force?}` | `{success}` | 停止服务 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `service.get` | `{service_id}` | `{id, model_id, status, replicas, endpoints, metrics}` | 服务详情 |
| `service.list` | `{status?, model_id?}` | `{services: []}` | 列出服务 |
| `service.recommend` | `{model_id, hint?}` | `{resource_class, replicas, expected_throughput}` | 推荐配置 |

## 核心结构

```go
type ModelService struct {
    ID        string
    Name      string
    ModelID   string
    Status    ServiceStatus
    Config    *ServiceConfig
    Runtime   *ServiceRuntime
    Endpoint  *ServiceEndpoint
    Resources *ServiceResources
    Stats     *ServiceStats
}

type ModelServiceManager struct {
    db               *sql.DB
    store            ServiceRepository
    modelRepo        interface{ Get(id string) (*model.Model, error) }
    engineMgr        *engine.Manager
    resourceMgr      *resource.Manager
    halMgr           *hal.Manager
    lifecycle        *LifecycleManager
    resourceTable    *ResourceTable
    engineSelector   *EngineSelector
    hardwareSelector *HardwareSelector
    telemetry        *TelemetryCollector
}
```

## 实现文件

```
pkg/service/
├── manager.go             # 服务管理器
├── model.go               # 模型服务定义
├── lifecycle.go           # 生命周期管理
├── resource_table.go      # 资源查表
├── engine_selector.go     # 引擎选择器
├── hardware_selector.go   # 硬件选择器
├── telemetry.go           # 遥测收集
└── optimizer.go           # 服务优化
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `service.create` | ✅ | `service/manager.go` Create() |
| `service.delete` | ✅ | `service/manager.go` Delete() |
| `service.start` | ✅ | `service/lifecycle.go` Start() |
| `service.stop` | ✅ | `service/lifecycle.go` Stop() |
| `service.get` | ✅ | `service/manager.go` Get() |
| `service.list` | ✅ | `service/manager.go` List() |
| `service.recommend` | ✅ | `service/optimizer.go` |
| `service.scale` | ⚠️ | 需新增 |
