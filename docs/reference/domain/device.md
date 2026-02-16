# Device Domain

硬件设备管理领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/device/` | `pkg/hal/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `device.detect` | `{}` | `{devices: [{id, name, vendor, type, memory}]}` | 检测硬件设备 |
| `device.set_power_limit` | `{device_id, limit_watts}` | `{success}` | 设置功耗限制 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `device.info` | `{device_id?}` | `{id, name, vendor, architecture, capabilities, memory}` | 设备信息 |
| `device.metrics` | `{device_id?, history?}` | `{utilization, temperature, power, memory_used, memory_total}` | 实时指标 |
| `device.health` | `{device_id?}` | `{status, issues: []}` | 健康检查 |

### Resources

| URI | 说明 |
|-----|------|
| `asms://device/{id}/info` | 设备信息 |
| `asms://device/{id}/metrics` | 实时指标 |
| `asms://device/{id}/health` | 健康状态 |

### Events

| 类型 | 载荷 | 说明 |
|------|------|------|
| `device.detected` | `{device}` | 检测到新设备 |
| `device.health_changed` | `{device_id, old_status, new_status}` | 健康状态变化 |
| `device.metrics_alert` | `{device_id, metric, value, threshold}` | 指标告警 |

## ASMS 核心接口

```go
// pkg/hal/interfaces.go
type DeviceProvider interface {
    Detect() ([]Device, error)
    Name() string
    Supported() bool
}

type Device interface {
    ID() string
    Name() string
    Vendor() string
    Architecture() string
    Capabilities() DeviceCapabilities
    TotalMemory() uint64
    AvailableMemory() uint64
    Utilization() (float64, error)
    Temperature() (float64, error)
    PowerUsage() (float64, error)
    HealthStatus() (HealthStatus, error)
    Metrics() (DeviceMetrics, error)
}
```

## 实现文件

```
pkg/hal/
├── interfaces.go          # 核心接口
├── cache.go               # 指标缓存
├── v2/
│   ├── interfaces.go      # v2 扩展接口
│   └── manager.go         # HAL 管理器
├── nvidia/
│   └── provider.go        # NVIDIA 提供者
└── generic/
    └── provider.go        # 通用 CPU 提供者
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `device.detect` | ✅ | `hal/v2/manager.go` DiscoverDevices() |
| `device.info` | ✅ | `hal/interfaces.go` Device 接口 |
| `device.metrics` | ✅ | `hal/cache.go` Metrics() |
| `device.health` | ✅ | `hal/interfaces.go` HealthStatus() |
| `device.set_power_limit` | ⚠️ | 需新增 |
| Resource URI | ⚠️ | 需新增 |
| Events | ⚠️ | 需新增 |
