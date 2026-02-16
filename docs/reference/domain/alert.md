# Alert Domain

告警管理领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/alert/` | `pkg/fleet/alert.go`, `alert_channel.go` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `alert.create_rule` | `{name, condition, severity, channels?, cooldown?}` | `{rule_id}` | 创建规则 |
| `alert.update_rule` | `{rule_id, name?, condition?, enabled?}` | `{success}` | 更新规则 |
| `alert.delete_rule` | `{rule_id}` | `{success}` | 删除规则 |
| `alert.acknowledge` | `{alert_id}` | `{success}` | 确认告警 |
| `alert.resolve` | `{alert_id}` | `{success}` | 解决告警 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `alert.list_rules` | `{enabled_only?}` | `{rules: []}` | 列出规则 |
| `alert.history` | `{rule_id?, status?, severity?, limit?}` | `{alerts: []}` | 告警历史 |
| `alert.active` | `{}` | `{alerts: []}` | 活动告警 |

## 核心结构

```go
type Alert struct {
    ID          string
    DeviceID    string
    RuleID      string
    RuleName    string
    Severity    AlertSeverity     // info, warning, critical
    Status      AlertStatus       // firing, acknowledged, resolved
    Message     string
    Metrics     map[string]any
    TriggeredAt time.Time
}

type NotificationChannel struct {
    ID        string
    Name      string
    Type      ChannelType       // webhook, email, slack, wechat, sms
    Config    map[string]string
    Enabled   bool
}
```

## 通知渠道

| 渠道 | 实现 | 说明 |
|------|------|------|
| Webhook | ✅ | 带 HMAC 签名 |
| Email | ✅ | SMTP |
| Slack | ✅ | Incoming Webhook |
| WeChat | ✅ | 企业微信 |
| SMS | 🔧 | 预留 |

## 实现文件

```
pkg/fleet/
├── alert.go               # 告警管理器与通知发送器
└── alert_channel.go       # 告警通道管理
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `alert.create_rule` | ✅ | `fleet/alert.go` CreateRule() |
| `alert.update_rule` | ✅ | `fleet/alert.go` UpdateRule() |
| `alert.delete_rule` | ✅ | `fleet/alert.go` DeleteRule() |
| `alert.acknowledge` | ✅ | `fleet/alert.go` Acknowledge() |
| `alert.resolve` | ✅ | `fleet/alert.go` Resolve() |
| `alert.list_rules` | ✅ | `fleet/alert.go` |
| `alert.history` | ✅ | `fleet/alert.go` |
| `alert.active` | ✅ | `fleet/alert.go` |
