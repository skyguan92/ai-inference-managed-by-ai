# [2026-02-20] 使用 AIMA CLI 部署 GLM-4.7-Flash + ASR + TTS 服务测试

## 元信息
- 开始时间: 2026-02-20 01:50
- 完成时间: 2026-02-20 02:15
- 实现模型: GLM-5
- 审查模型: (待审查)

## 任务概述
- **目标**: 使用 aima CLI 启动 GLM-4.7-Flash 到 GPU，ASR 和 TTS 模型到 CPU，测试完整语音对话链路
- **范围**: CLI 构建服务创建/启动, 服务部署, 端到端测试
- **优先级**: P0

## 测试结果

### 服务部署状态

| 服务 | 模型 | 设备 | 状态 | 端口 |
|------|------|------|------|------|
| LLM | GLM-4.7-Flash | GPU (GB10) | ✅ 运行中 | 8000 |
| ASR | SenseVoiceSmall | CPU | ✅ 运行中 | 8001 |
| TTS | Qwen3-TTS-0.6B | CPU | ⚠️ Placeholder | 8002 |

### AIMA CLI 命令使用

```bash
# 1. 构建并启动 aima 服务
go build -o ~/go/bin/aima ./cmd/aima
aima start --port 9090

# 2. 创建模型记录
aima model create glm-4.7-flash --type llm --format safetensors --path /mnt/data/models/GLM-4.7-Flash
aima model create sensevoice-small --type asr --format safetensors --path /mnt/data/models/SenseVoiceSmall
aima model create qwen3-tts --type tts --format safetensors --path /mnt/data/models/Qwen3-TTS-0.6B

# 3. 创建服务
aima service create glm-flash --model model-3705aa79 --device gpu --port 8000
aima service create asr-sensevoice --model model-ad65a233 --device cpu --port 8001
aima service create tts-qwen3 --model model-466ca5b4 --device cpu --port 8002

# 4. 启动服务
aima service start svc-vllm-model-3705aa79 --wait --timeout 600
aima service start svc-whisper-model-ad65a233 --wait --timeout 180
aima service start svc-tts-model-466ca5b4 --wait --timeout 180
```

## 发现的 Bug

### Bug #1: vLLM 镜像优先级不合理 ✅ 已修复

**问题描述**:
- `getDockerImages("vllm")` 将 `aima-qwen3-omni-server:latest` 设为最高优先级
- 该镜像专为 Qwen3-Omni 设计，不支持 GLM 模型

**影响**: GLM-4.7-Flash 加载失败，报错 `'Qwen3OmniMoeTalkerTextConfig' object has no attribute 'shared_expert_intermediate_size'`

**修复**:
```go
// pkg/infra/provider/hybrid_engine_provider.go
case "vllm":
    // Priority: GB10 compatible (general) > Qwen3-Omni specific > official image
    return []string{
        "zhiwen-vllm:0128",              // GB10 compatible - supports most models (priority 1)
        "aima-vllm-qwen3-omni:latest",   // Qwen3-Omni vLLM specific (priority 2)
        "aima-qwen3-omni-server:latest", // Qwen3-Omni FastAPI server (priority 3)
        "vllm/vllm-openai:v0.15.0",      // Official image (fallback)
    }
```

### Bug #2: 服务状态不同步 ⚠️ 待修复

**问题描述**:
- 服务状态在数据库中显示为 "running"，但容器被外部删除后状态未更新
- 导致无法使用 `aima service start` 重新启动服务

**影响**: 需要先手动执行 `aima service stop --force` 才能重新启动

**建议修复**: 
1. 添加定期状态同步机制
2. 在启动前检查容器是否实际存在

### Bug #3: 服务启动超时后状态未正确设置 ⚠️ 待修复

**问题描述**:
- 大模型加载需要较长时间（GLM-4.7-Flash 约 2 分钟）
- 健康检查超时后服务状态未正确更新

**影响**: 服务列表显示状态与实际情况不符

**建议修复**:
1. 增加默认健康检查超时时间
2. 在健康检查失败但容器仍在运行时，将状态设为 "starting" 而非失败

### Bug #4: TTS 模型挂载路径错误 ✅ 已修复

**问题描述**:
- AIMA 将模型挂载到 `/models` (复数)
- `qujing-qwen3-tts:latest` 镜像期望模型在 `/model` (单数)

**影响**: TTS 服务无法找到模型文件，回退到 placeholder 模式

**修复**:
```go
// pkg/infra/provider/hybrid_engine_provider.go
// Determine mount path based on image type
mountPath := "/models"
if strings.Contains(image, "qujing-glm-asr-nano") || strings.Contains(image, "qujing-qwen3-tts") {
    mountPath = "/model" // ASR and TTS images expect /model
}
```

### Bug #5: TTS Docker 镜像是 Placeholder ⚠️ 需要更新镜像

**问题描述**:
- `qujing-qwen3-tts:latest` 镜像中 `USE_PLACEHOLDER = True` 硬编码
- 没有实际的 Qwen3-TTS 模型加载代码

**影响**: TTS 服务只能生成测试用的正弦波音频，无法进行真正的语音合成

**建议修复**: 更新 TTS Docker 镜像，实现真正的 Qwen3-TTS 模型加载

## 修改的文件

1. `pkg/infra/provider/hybrid_engine_provider.go`
   - 修复 vLLM 镜像优先级顺序
   - 修复 TTS 模型挂载路径

## 测试验证

### LLM 测试 (GLM-4.7-Flash on GPU)
```bash
curl -s http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "/models", "messages": [{"role": "user", "content": "你好"}], "max_tokens": 50}'
# 返回: 正常的对话响应
```

### ASR 测试 (SenseVoice on CPU)
```bash
curl -s http://localhost:8001/health
# 返回: {"status": "healthy", "model": "SenseVoiceSmall", "device": "cpu", "loaded": true}
```

### TTS 测试 (Qwen3-TTS on CPU - Placeholder)
```bash
curl -s http://localhost:8002/health
# 返回: {"status": "healthy", "model": "Qwen3-TTS-0.6B (PLACEHOLDER)", "placeholder_mode": true}
```

## 后续任务

- [ ] 修复 Bug #2: 服务状态同步机制
- [ ] 修复 Bug #3: 大模型启动超时处理
- [ ] 更新 TTS Docker 镜像以支持真正的 Qwen3-TTS 模型
- [ ] 提交代码审查请求

## 提交信息

- **Commit**: (待提交)
- **Message**: fix(provider): correct vLLM image priority and TTS mount path

---

*测试由 AIMA CLI 完成*
*最后更新: 2026-02-20*
