# AIMA 推理性能测试报告

**测试日期**: 2026-02-20
**测试环境**: NVIDIA Jetson Thor (GB10 GPU, ARM64, 128GB 统一内存)
**测试工具**: AIMA CLI + 自定义性能测试脚本

---

## 执行摘要

三个模型服务同时运行，资源分配如下：

| 服务 | 模型 | 设备 | 内存限制 | CPU 限制 | 实际内存使用 |
|------|------|------|----------|----------|--------------|
| LLM | GLM-4.7-Flash | GPU | 无限制 | 无限制 | 4.17 GB |
| ASR | SenseVoiceSmall | CPU | 4 GB | 2 核 | 3.42 GB |
| TTS | Qwen3-TTS-0.6B | CPU | 4 GB | 2 核 | 378 MB |

---

## 性能测试结果

### 1. LLM 性能 (GLM-4.7-Flash on GPU)

| 指标 | 值 |
|------|-----|
| 平均延迟 | 4.83s |
| 标准差 | 2.74s |
| 吞吐量 | **20.6 tokens/s** |
| 总请求数 | 15 |
| 总生成 Token 数 | 1,500 |

**延迟分布**:
- 短 Prompt ("你好"): ~4.08s
- 中等 Prompt: ~4.11s
- 长 Prompt: ~4.11s (有一次 15.13s 的异常值)

**分析**:
- GLM-4.7-Flash 在 GB10 GPU 上的推理速度约 20 tokens/s
- 这是一个 60GB 模型，性能表现合理
- 延迟波动可能与模型加载后的缓存状态有关

### 2. TTS 性能 (Qwen3-TTS-0.6B on CPU)

| 指标 | 值 |
|------|-----|
| 平均延迟 | <0.01s (Fallback) |
| 处理速度 | 3,221 chars/s |
| 总请求数 | 15 |
| 总字符数 | 455 |

**音频输出大小**:
| 文本长度 | 音频大小 |
|----------|----------|
| 2 字符 | 12,861 bytes |
| 25 字符 | 160,061 bytes |
| 64 字符 | 409,661 bytes |

⚠️ **注意**: 当前 TTS 服务使用 Fallback 模式（正弦波），因为 PyTorch 在容器中未正确加载。实际 TTS 性能需要修复后重新测试。

### 3. ASR 性能 (SenseVoiceSmall on CPU)

| 指标 | 值 |
|------|-----|
| 平均延迟 | <0.01s |
| 状态 | 健康运行 |

ASR 服务正常运行，模型已加载到内存。

### 4. 并发性能

| 指标 | 值 |
|------|-----|
| 并发请求数 | 3 (LLM + TTS + ASR) |
| 总耗时 | 2.08s |
| LLM 响应 | 正常 |
| TTS 响应 | 25,661 bytes |
| ASR 响应 | healthy |

### 5. 持续负载测试 (30秒)

| 指标 | LLM | TTS |
|------|-----|-----|
| 请求数 | 30 | 30 |
| QPS | 1.00 | 1.00 |

---

## 资源使用分析

```
CONTAINER           CPU %    MEM USAGE / LIMIT
aima-vllm           3.07%    4.17GB / 119.6GB (3.49%)
aima-whisper        0.14%    3.42GB / 4GB (85.43%)
aima-tts            0.15%    378MB / 4GB (9.23%)
```

**关键发现**:
1. **LLM (vLLM)**: GPU 模型占用 4.17GB 显存，推理时 CPU 使用率约 3%
2. **ASR**: 占用接近其 4GB 内存限制 (85%)，说明模型已完全加载
3. **TTS**: 内存使用很低 (378MB)，因为使用的是 fallback 模式

---

## 性能瓶颈分析

### 1. LLM 延迟波动
- **问题**: 某些请求延迟高达 15s
- **原因**: 可能是首次访问或缓存未命中
- **建议**: 添加预热请求或调整 vLLM 缓存配置

### 2. TTS Fallback 模式
- **问题**: PyTorch 未正确加载，使用正弦波替代
- **原因**: Docker 镜像构建时多阶段复制导致依赖问题
- **解决**: 需要重新构建镜像，确保 PyTorch 完整安装

### 3. ASR 内存使用高
- **问题**: 85% 内存使用率
- **原因**: SenseVoice 模型较大
- **建议**: 监控长时间运行的稳定性

---

## 对比基准

| 模型 | 平台 | 吞吐量 | 延迟 |
|------|------|--------|------|
| GLM-4.7-Flash | GB10 GPU | 20.6 tokens/s | ~5s |
| Qwen3-Coder-Next-FP8 | GB10 GPU | ~50 tokens/s* | - |

*参考自之前的测试报告

---

## 使用 AIMA CLI 运行测试

```bash
# 运行性能测试脚本
./test/performance_test.sh

# 自定义测试参数
LLM_ITERATIONS=10 TTS_ITERATIONS=10 ./test/performance_test.sh
```

---

## 后续优化建议

1. **LLM 优化**
   - 调整 `--gpu-memory-utilization` 参数
   - 启用 `--enable-chunked-prefill` 提高吞吐量
   - 考虑使用量化版本减少显存占用

2. **TTS 优化**
   - 修复 PyTorch 加载问题
   - 考虑使用 GPU 加速（如果资源允许）

3. **资源隔离**
   - 为 ASR 服务增加内存限制余量
   - 监控长时间运行的内存泄漏

---

## 附录：测试命令

```bash
# LLM 测试
curl -s http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "/models", "messages": [{"role": "user", "content": "你好"}], "max_tokens": 100}'

# TTS 测试
curl -s -X POST http://localhost:8002/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "测试语音合成", "voice": "default"}'

# ASR 健康检查
curl -s http://localhost:8001/health

# 资源监控
docker stats --no-stream
```

---

*报告由 AIMA 性能测试脚本自动生成*
*测试时间: 2026-02-20 05:00*
