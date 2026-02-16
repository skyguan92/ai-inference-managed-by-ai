# 领域映射表

本文档详细列出 AIMA 架构设计中每个原子单元与 ASMS 现有实现的对应关系。

---

## 映射状态说明

| 状态 | 说明 |
|------|------|
| ✅ 已有 | ASMS 中已有对应实现，可直接封装 |
| ⚠️ 需新增 | ASMS 中没有，需要新增 |
| 🔧 需完善 | ASMS 中有部分实现，需要完善 |
| ❌ 不适用 | 原子单元不适用于该领域 |

---

## 1. Device Domain

**ASMS 源码**: `pkg/hal/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `device.detect` | `pkg/hal/v2/manager.go` DiscoverDevices() | ✅ | 检测硬件设备 |
| `device.set_power_limit` | 无 | ⚠️ | 需新增功耗限制功能 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `device.info` | `pkg/hal/interfaces.go` Device 接口 | ✅ | 设备信息 |
| `device.metrics` | `pkg/hal/cache.go` Metrics() | ✅ | 实时指标 |
| `device.health` | `pkg/hal/interfaces.go` HealthStatus() | ✅ | 健康检查 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://device/{id}/info` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://device/{id}/metrics` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://device/{id}/health` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `device.detected` | 无 | ⚠️ | 需新增事件发布 |
| `device.health_changed` | 无 | ⚠️ | 需新增事件发布 |
| `device.metrics_alert` | 无 | ⚠️ | 需新增事件发布 |

---

## 2. Model Domain

**ASMS 源码**: `pkg/model/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `model.create` | `pkg/model/manager.go` Create() | ✅ | 创建模型记录 |
| `model.delete` | `pkg/model/manager.go` Delete() | ✅ | 删除模型 |
| `model.pull` | `pkg/model/downloader/`, `pkg/model/v2/download/` | ✅ | 从源拉取模型 |
| `model.import` | `pkg/model/manager.go` ImportLocal() | ✅ | 导入本地模型 |
| `model.verify` | 无 | ⚠️ | 需新增完整性验证 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `model.get` | `pkg/model/manager.go` Get() | ✅ | 获取模型详情 |
| `model.list` | `pkg/model/manager.go` List() | ✅ | 列出模型 |
| `model.search` | `pkg/model/v2/search/` | ✅ | 搜索模型 |
| `model.estimate_resources` | `pkg/engine/adapters/*.go` EstimateMemory() | ✅ | 预估资源需求 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://model/{id}` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://models/registry` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://models/compatibility` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `model.created` | 无 | ⚠️ | 需新增事件发布 |
| `model.deleted` | 无 | ⚠️ | 需新增事件发布 |
| `model.pull_progress` | 部分有 | 🔧 | 下载进度事件 |
| `model.verified` | 无 | ⚠️ | 需新增事件发布 |

---

## 3. Engine Domain

**ASMS 源码**: `pkg/engine/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `engine.start` | `pkg/engine/manager.go` Start() | ✅ | 启动引擎 |
| `engine.stop` | `pkg/engine/manager.go` Stop() | ✅ | 停止引擎 |
| `engine.restart` | 组合调用 | 🔧 | 可从 start/stop 组合 |
| `engine.install` | `pkg/engine/adapters/*.go` Install() | ✅ | 安装引擎 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `engine.get` | `pkg/engine/manager.go` GetProcess() | ✅ | 获取引擎信息 |
| `engine.list` | `pkg/engine/manager.go` ListProcesses() | ✅ | 列出引擎 |
| `engine.features` | 接口定义中 | 🔧 | 需提取特性信息 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://engine/{name}` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://engines/status` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `engine.started` | 无 | ⚠️ | 需新增事件发布 |
| `engine.stopped` | 无 | ⚠️ | 需新增事件发布 |
| `engine.error` | 无 | ⚠️ | 需新增事件发布 |
| `engine.health_changed` | 无 | ⚠️ | 需新增事件发布 |

### 已实现的适配器

| 适配器 | 文件 | 支持的模型类型 |
|--------|------|----------------|
| Ollama | `adapters/ollama.go` | LLM |
| vLLM | `adapters/vllm.go` | LLM (高性能) |
| SGLang | `adapters/sglang.go` | LLM (高吞吐) |
| Whisper | `adapters/whisper.go` | ASR |
| TTS | `adapters/tts.go` | TTS |
| Diffusion | `adapters/diffusion.go` | ImageGen |
| Transformers | `adapters/transformers.go` | 通用 |
| HuggingFace | `adapters/huggingface.go` | 多模态 |
| Video | `adapters/video.go` | VideoGen |
| Rerank | `adapters/rerank.go` | Rerank |

---

## 4. Inference Domain

**ASMS 源码**: `pkg/engine/adapters/`, `pkg/service/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `inference.chat` | `pkg/engine/adapters/*.go` ChatCompletion() | ✅ | 聊天补全 |
| `inference.complete` | `pkg/engine/adapters/*.go` Completion() | ✅ | 文本补全 |
| `inference.embed` | `pkg/engine/adapters/*.go` Embed() | ✅ | 文本嵌入 |
| `inference.transcribe` | `pkg/engine/adapters/*.go` Transcribe() | ✅ | 语音转文字 |
| `inference.synthesize` | `pkg/engine/adapters/*.go` Synthesize() | ✅ | 文字转语音 |
| `inference.generate_image` | `pkg/engine/adapters/*.go` GenerateImage() | ✅ | 图像生成 |
| `inference.generate_video` | `pkg/engine/adapters/*.go` GenerateVideo() | ✅ | 视频生成 |
| `inference.rerank` | `pkg/engine/adapters/*.go` Rerank() | ✅ | 重排序 |
| `inference.detect` | `pkg/engine/adapters/*.go` Detect() | ✅ | 目标检测 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `inference.models` | `pkg/service/model.go` | ✅ | 列出可用模型 |
| `inference.voices` | TTS 适配器中 | 🔧 | 需提取为独立查询 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://inference/models` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `inference.request_started` | 无 | ⚠️ | 需新增事件发布 |
| `inference.request_completed` | 无 | ⚠️ | 需新增事件发布 |
| `inference.request_failed` | 无 | ⚠️ | 需新增事件发布 |

---

## 5. Resource Domain

**ASMS 源码**: `pkg/resource/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `resource.allocate` | `pkg/resource/manager.go` Allocate() | ✅ | 分配资源 |
| `resource.release` | `pkg/resource/manager.go` Release() | ✅ | 释放资源 |
| `resource.update_slot` | `pkg/resource/manager.go` | 🔧 | 需完善更新功能 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `resource.status` | `pkg/resource/manager.go` Status() | ✅ | 资源状态 |
| `resource.budget` | `pkg/resource/manager.go` MemoryBudget | ✅ | 资源预算 |
| `resource.allocations` | `pkg/resource/manager.go` ListSlots() | ✅ | 分配列表 |
| `resource.can_allocate` | `pkg/resource/manager.go` CanAllocate() | ✅ | 检查是否可分配 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://resource/status` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://resource/budget` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://resource/allocations` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `resource.allocated` | 无 | ⚠️ | 需新增事件发布 |
| `resource.released` | 无 | ⚠️ | 需新增事件发布 |
| `resource.pressure_warning` | 无 | ⚠️ | 需新增事件发布 |
| `resource.preemption` | 无 | ⚠️ | 需新增事件发布 |

---

## 6. Service Domain

**ASMS 源码**: `pkg/service/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `service.create` | `pkg/service/manager.go` Create() | ✅ | 创建服务 |
| `service.delete` | `pkg/service/manager.go` Delete() | ✅ | 删除服务 |
| `service.scale` | 无 | ⚠️ | 需新增扩缩容功能 |
| `service.start` | `pkg/service/lifecycle.go` Start() | ✅ | 启动服务 |
| `service.stop` | `pkg/service/lifecycle.go` Stop() | ✅ | 停止服务 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `service.get` | `pkg/service/manager.go` Get() | ✅ | 获取服务详情 |
| `service.list` | `pkg/service/manager.go` List() | ✅ | 列出服务 |
| `service.recommend` | `pkg/service/optimizer.go` | ✅ | 推荐配置 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://service/{id}` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://services` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `service.created` | 无 | ⚠️ | 需新增事件发布 |
| `service.scaled` | 无 | ⚠️ | 需新增事件发布 |
| `service.failed` | 无 | ⚠️ | 需新增事件发布 |

---

## 7. App Domain

**ASMS 源码**: `pkg/app/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `app.install` | `pkg/app/manager.go` Install() | ✅ | 安装应用 |
| `app.uninstall` | `pkg/app/manager.go` Uninstall() | ✅ | 卸载应用 |
| `app.start` | `pkg/app/manager.go` Start() | ✅ | 启动应用 |
| `app.stop` | `pkg/app/manager.go` Stop() | ✅ | 停止应用 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `app.get` | `pkg/app/manager.go` Get() | ✅ | 获取应用详情 |
| `app.list` | `pkg/app/manager.go` List() | ✅ | 列出应用 |
| `app.logs` | `pkg/app/docker.go` Logs() | ✅ | 获取日志 |
| `app.templates` | `pkg/app/templates.go` | ✅ | 列出模板 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://app/{id}` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://apps/templates` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `app.installed` | 无 | ⚠️ | 需新增事件发布 |
| `app.started` | 无 | ⚠️ | 需新增事件发布 |
| `app.stopped` | 无 | ⚠️ | 需新增事件发布 |
| `app.oom_detected` | 无 | ⚠️ | 需新增事件发布 |

---

## 8. Pipeline Domain

**ASMS 源码**: `pkg/pipeline/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `pipeline.create` | `pkg/pipeline/engine.go` | ✅ | 创建管道 |
| `pipeline.delete` | `pkg/pipeline/engine.go` | ✅ | 删除管道 |
| `pipeline.run` | `pkg/pipeline/engine.go` Execute() | ✅ | 运行管道 |
| `pipeline.cancel` | 无 | ⚠️ | 需新增取消功能 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `pipeline.get` | `pkg/pipeline/engine.go` | ✅ | 获取管道详情 |
| `pipeline.list` | `pkg/pipeline/engine.go` | ✅ | 列出管道 |
| `pipeline.status` | `pkg/pipeline/engine.go` | ✅ | 获取运行状态 |
| `pipeline.validate` | `pkg/pipeline/validator.go` | ✅ | 验证管道定义 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://pipeline/{id}` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://pipelines` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `pipeline.started` | 无 | ⚠️ | 需新增事件发布 |
| `pipeline.step_completed` | 无 | ⚠️ | 需新增事件发布 |
| `pipeline.completed` | 无 | ⚠️ | 需新增事件发布 |
| `pipeline.failed` | 无 | ⚠️ | 需新增事件发布 |

### 预定义管道

| 管道类型 | 步骤 |
|----------|------|
| voice-assistant | ASR → LLM → TTS |
| rag | Embed → Search → LLM |
| vision-chat | Image → VLM → LLM |
| content-gen | LLM → ImageGen |
| detect-describe | YOLO → LLM |
| video-stream-analysis | 提取帧 → VLM 分析 |

---

## 9. Alert Domain

**ASMS 源码**: `pkg/fleet/alert.go`, `alert_channel.go`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `alert.create_rule` | `pkg/fleet/alert.go` CreateRule() | ✅ | 创建告警规则 |
| `alert.update_rule` | `pkg/fleet/alert.go` UpdateRule() | ✅ | 更新规则 |
| `alert.delete_rule` | `pkg/fleet/alert.go` DeleteRule() | ✅ | 删除规则 |
| `alert.acknowledge` | `pkg/fleet/alert.go` Acknowledge() | ✅ | 确认告警 |
| `alert.resolve` | `pkg/fleet/alert.go` Resolve() | ✅ | 解决告警 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `alert.list_rules` | `pkg/fleet/alert.go` | ✅ | 列出规则 |
| `alert.history` | `pkg/fleet/alert.go` | ✅ | 告警历史 |
| `alert.active` | `pkg/fleet/alert.go` | ✅ | 活动告警 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://alerts/rules` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://alerts/active` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `alert.triggered` | 无 | ⚠️ | 需新增事件发布 |
| `alert.acknowledged` | 无 | ⚠️ | 需新增事件发布 |
| `alert.resolved` | 无 | ⚠️ | 需新增事件发布 |

### 通知渠道

| 渠道 | 实现状态 |
|------|----------|
| Webhook | ✅ (带 HMAC 签名) |
| Email | ✅ (SMTP) |
| Slack | ✅ |
| WeChat | ✅ (企业微信) |
| SMS | 🔧 (预留) |

---

## 10. Remote Domain

**ASMS 源码**: `pkg/remote/`

### Commands

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `remote.enable` | `pkg/remote/manager.go` Enable() | ✅ | 启用远程访问 |
| `remote.disable` | `pkg/remote/manager.go` Disable() | ✅ | 禁用远程访问 |
| `remote.exec` | `pkg/remote/manager.go` SandboxExec() | ✅ | 执行远程命令 |

### Queries

| AIMA 原子单元 | ASMS 实现 | 状态 | 备注 |
|---------------|-----------|------|------|
| `remote.status` | `pkg/remote/manager.go` Status() | ✅ | 远程状态 |
| `remote.audit` | `pkg/remote/manager.go` AuditLog() | ✅ | 审计日志 |

### Resources

| URI | ASMS 实现 | 状态 | 备注 |
|-----|-----------|------|------|
| `asms://remote/status` | 无 | ⚠️ | 需新增 Resource 接口 |
| `asms://remote/audit` | 无 | ⚠️ | 需新增 Resource 接口 |

### Events

| 事件类型 | ASMS 实现 | 状态 | 备注 |
|----------|-----------|------|------|
| `remote.enabled` | 无 | ⚠️ | 需新增事件发布 |
| `remote.disabled` | 无 | ⚠️ | 需新增事件发布 |
| `remote.command_executed` | 无 | ⚠️ | 需新增事件发布 |

### 隧道支持

| 提供者 | 实现文件 |
|--------|----------|
| FRP | `tunnel_frp.go` |
| Cloudflare Tunnel | `tunnel_cloudflare.go` |

---

## 统计汇总

### 按状态统计

| 状态 | 数量 | 占比 |
|------|------|------|
| ✅ 已有 | 68 | 71% |
| ⚠️ 需新增 | 24 | 25% |
| 🔧 需完善 | 4 | 4% |

### 按领域统计

| 领域 | 已有 | 需新增 | 需完善 |
|------|------|--------|--------|
| Device | 4 | 4 | 0 |
| Model | 8 | 4 | 1 |
| Engine | 7 | 4 | 1 |
| Inference | 10 | 3 | 1 |
| Resource | 7 | 4 | 0 |
| Service | 7 | 2 | 0 |
| App | 8 | 4 | 0 |
| Pipeline | 7 | 2 | 0 |
| Alert | 8 | 3 | 0 |
| Remote | 5 | 3 | 0 |

### 关键差距

1. **Resource 接口**: 所有 20 个 Resource URI 都需要新建
2. **Event 发布**: 大部分事件发布需要新增
3. **新增功能**: 约 24 个原子单元需要从头实现
4. **完善功能**: 4 个功能需要补充
