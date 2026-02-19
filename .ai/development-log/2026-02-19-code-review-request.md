# Code Review Request: End-to-End Inference Services Implementation

## 元信息
- **提交时间**: 2026-02-19
- **实现模型**: Kimi (k2p5)
- **请求审查模型**: Claude / GLM-5
- **分支**: AiIMA-kimi
- **修复时间**: 2026-02-19

## 功能概述

本次提交实现了完整的端到端 AI 推理基础设施，包括：

1. **HybridEngineProvider** - 混合引擎提供者，支持 Docker 和原生进程模式
2. **资源管理** - 内存/CPU/GPU 限制和分配
3. **健康检查** - 带重试机制的服务启动验证
4. **硬件适配** - NVIDIA Jetson Thor GB10 GPU 兼容性支持
5. **SQLite 持久化** - 模型和服务状态持久化

## 修改文件列表

### 核心实现（新增）
- `pkg/infra/provider/hybrid_engine_provider.go` (794 lines)
- `pkg/infra/docker/simple_client.go` (160 lines)

### 修改的文件
- `pkg/infra/docker/mock.go` - 添加 Memory/CPU 字段到 ContainerOptions
- `pkg/cli/root.go` - CLI 命令集成
- `pkg/infra/provider/huggingface/provider.go` - HuggingFace 提供者增强
- `pkg/unit/events.go` - 事件系统更新
- `pkg/unit/service/queries.go` - 服务查询更新
- `pkg/unit/service/types.go` - 类型定义更新
- `go.mod/go.sum` - 依赖更新

### 其他新增文件
- `pkg/infra/store/` - SQLite 存储实现
- `pkg/infra/provider/vllm/` - vLLM 特定适配
- `docs/reference/hardware/nvidia-gb10-adaptation.md` - 硬件适配文档
- `test/` - 测试脚本和报告

## 修复记录

### 已修复问题

#### 1. ✅ 并发安全问题 [已修复]
- **修改**: 添加 `sync.RWMutex` 保护所有 map 操作
- **位置**: `hybrid_engine_provider.go:44-45`, `325-327`, `416-430`, `433-457`
- **验证**: `go vet` 通过

#### 2. ✅ HTTP Body 未关闭 [已修复]
- **修改**: 使用匿名函数 + defer 确保 body 关闭
- **位置**: `hybrid_engine_provider.go:349-361`
- **验证**: 资源泄漏风险消除

#### 3. ✅ 硬编码配置 [已修复]
- **修改**: 支持环境变量覆盖默认配置
- **环境变量**:
  - `AIMA_VLLM_MEMORY`, `AIMA_VLLM_CPU`, `AIMA_VLLM_GPU`
  - `AIMA_WHISPER_MEMORY`, `AIMA_WHISPER_CPU`, `AIMA_WHISPER_GPU`
  - `AIMA_ASR_MEMORY`, `AIMA_ASR_CPU`, `AIMA_ASR_GPU`
  - `AIMA_TTS_MEMORY`, `AIMA_TTS_CPU`, `AIMA_TTS_GPU`
- **位置**: `hybrid_engine_provider.go:72-115`

#### 4. ✅ 命令注入风险 [已修复]
- **修改**: 添加 `validateModelPath()` 函数验证路径
- **检查项**:
  - 禁止 shell 元字符: `;`, `&`, `|`, `` ` ``, `$`, `(`, `)`, `<`, `>`, `\`, `'`, `"`
  - 禁止目录遍历: `..`
  - 必须是绝对路径: 以 `/` 开头
- **位置**: `hybrid_engine_provider.go:291-294`, `585-600`

#### 5. ✅ 僵尸进程风险 [已修复]
- **修改**: 添加 goroutine 等待进程结束并清理 map
- **位置**: `hybrid_engine_provider.go:416-430`
- **行为**: 进程退出后自动从 `nativeProcesses` 中删除

## 关键实现细节

### 1. 资源限制配置（现在可配置）
```go
// 默认资源限制（可被环境变量覆盖）
"vllm":   {GPU: true, Memory: "0", CPU: 0}
"whisper": {GPU: false, Memory: "4g", CPU: 2.0}
"tts":    {GPU: false, Memory: "4g", CPU: 2.0}

// 覆盖示例
export AIMA_VLLM_MEMORY="16g"
export AIMA_VLLM_GPU="false"
```

### 2. 并发安全
```go
type HybridEngineProvider struct {
    // ... other fields ...
    mu sync.RWMutex
}

// 写操作
p.mu.Lock()
p.containers[engineType] = containerID
p.mu.Unlock()

// 读操作
p.mu.RLock()
containerID, exists := p.containers[name]
p.mu.RUnlock()
```

### 3. Docker 镜像优先级（GB10 兼容）
```go
"vllm": []string{
    "zhiwen-vllm:0128",         // GB10 兼容镜像（优先）
    "vllm/vllm-openai:v0.15.0", // 官方镜像（回退）
}
```

## 测试验证

| 服务 | 模型 | 状态 | 测试结果 |
|------|------|------|----------|
| LLM | Qwen3-Coder-Next (FP8) | ✅ 运行中 | 代码生成、数学推理正确 |
| ASR | SenseVoice-Small | ✅ 运行中 | 语音转录准确 |
| TTS | Qwen3-TTS-0.6B | ✅ 运行中 | 语音合成清晰 |

### 代码质量检查
- [x] `go fmt` - 格式化通过
- [x] `go vet` - 静态检查通过

## 审查关注点

1. **并发安全** - ✅ 已添加 mutex 保护
2. **资源泄漏** - ✅ HTTP body 和进程都已正确处理
3. **配置灵活性** - ✅ 支持环境变量覆盖
4. **安全性** - ✅ 路径验证防止注入
5. **兼容性** - GB10 特定代码不影响其他平台

## 建议改进（可选）

1. **日志记录**: 后续使用结构化日志替代 `fmt.Printf`
2. **指标收集**: 后续添加 Prometheus 指标暴露
3. **配置外置**: 后续将配置移到 YAML/JSON 文件

## 最终结论

- [x] 代码符合架构设计
- [x] 接口实现完整
- [x] 错误处理正确
- [x] 并发安全已解决
- [x] 安全漏洞已修复
- [x] 命名规范一致
- [x] `go vet` 通过

**状态**: 修复完成，建议合并
