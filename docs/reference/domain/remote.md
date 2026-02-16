# Remote Domain

远程访问领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/remote/` | `pkg/remote/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `remote.enable` | `{provider, config?}` | `{tunnel_id, public_url}` | 启用远程访问 |
| `remote.disable` | `{}` | `{success}` | 禁用远程访问 |
| `remote.exec` | `{command, timeout?}` | `{stdout, stderr, exit_code}` | 执行远程命令 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `remote.status` | `{}` | `{enabled, provider, public_url, uptime}` | 远程状态 |
| `remote.audit` | `{since?, limit?}` | `{records: []}` | 审计日志 |

## 核心结构

```go
type TunnelConfig struct {
    Provider    string        // frp, cloudflare, tailscale, wireguard
    FRPServer   string
    FRPToken    string
    CFToken     string
    ExposeSSH   bool
    ExposeAPI   bool
    ExposeMCP   bool
    ExposeWebUI bool
    AllowedIPs  []string
    AuthRequired bool
    SessionTTL  time.Duration
}

type SandboxConfig struct {
    AllowedCommands    []string
    RestrictedCommands []string
    BlockedCommands    []string
    MaxExecTime        time.Duration
    MaxOutputSize      int
    WorkingDir         string
    RunAsUser          string
}
```

## 隧道支持

| 提供者 | 实现文件 | 说明 |
|--------|----------|------|
| FRP | `tunnel_frp.go` | FRP 隧道 |
| Cloudflare | `tunnel_cloudflare.go` | Cloudflare Tunnel |

## 实现文件

```
pkg/remote/
├── types.go               # 远程访问类型
├── manager.go             # 远程管理器
├── tunnel_frp.go          # FRP 隧道实现
└── tunnel_cloudflare.go   # Cloudflare Tunnel 实现
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `remote.enable` | ✅ | `remote/manager.go` Enable() |
| `remote.disable` | ✅ | `remote/manager.go` Disable() |
| `remote.exec` | ✅ | `remote/manager.go` SandboxExec() |
| `remote.status` | ✅ | `remote/manager.go` Status() |
| `remote.audit` | ✅ | `remote/manager.go` AuditLog() |
