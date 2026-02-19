# AIMA 推理服务测试方案

## 1. LLM 服务测试 (Qwen3-Coder-Next-FP8)

### 1.1 代码生成测试
```json
{
  "model": "qwen3-coder-next-fp8",
  "messages": [
    {"role": "user", "content": "写一个 Python 函数，计算斐波那契数列的第 n 项"}
  ]
}
```

### 1.2 代码解释测试
```json
{
  "model": "qwen3-coder-next-fp8",
  "messages": [
    {"role": "user", "content": "解释这段代码：def quicksort(arr): return arr if len(arr) <= 1 else quicksort([x for x in arr[1:] if x < arr[0]]) + [arr[0]] + quicksort([x for x in arr[1:] if x >= arr[0]])"}
  ]
}
```

### 1.3 中文对话测试
```json
{
  "model": "qwen3-coder-next-fp8",
  "messages": [
    {"role": "user", "content": "你好，请介绍一下你自己"}
  ]
}
```

### 1.4 数学推理测试
```json
{
  "model": "qwen3-coder-next-fp8",
  "messages": [
    {"role": "user", "content": "解方程：2x + 5 = 15"}
  ]
}
```

## 2. ASR 服务测试 (SenseVoice)

### 2.1 语音转文字（使用参考音频）
```bash
# 测试 ASR 健康检查
curl http://localhost:8001/

# 使用已有的 reference.wav 测试
curl -X POST http://localhost:8001/asr \
  -F "audio=@/mnt/data/models/Qwen3-TTS-0.6B/reference.wav"
```

## 3. TTS 服务测试 (Qwen3-TTS)

### 3.1 文字转语音
```bash
# 测试 TTS 健康检查
curl http://localhost:8002/health

# 简单文本合成
curl -X POST http://localhost:8002/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "你好，这是一个测试", "voice": "default"}'
```

### 3.2 长文本合成
```bash
curl -X POST http://localhost:8002/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "人工智能正在改变我们的生活方式，从智能家居到自动驾驶，AI 技术无处不在。", "voice": "default"}'
```

## 4. 端到端 Pipeline 测试

### 4.1 语音 → 文字 → LLM → 语音
```bash
# 1. ASR: 语音转文字
# 2. LLM: 处理文字
# 3. TTS: 语音输出
```

## 5. 性能测试

### 5.1 并发测试
```bash
# 同时向三个服务发送请求
```

### 5.2 响应时间测试
```bash
# 测量每个服务的首 token 延迟
```
