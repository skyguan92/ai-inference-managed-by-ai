# AIMA 资产目录索引

## 概述

本目录包含 AIMA 项目验证的模型、引擎和硬件配置资产。这些资产经过实际测试验证，可直接用于生产部署。

## 目录结构

```
catalog/
├── models/           # 模型资产
│   ├── llm/          # 大语言模型
│   │   └── glm-4.7-flash.yaml
│   ├── asr/          # 语音识别模型
│   │   └── sensevoice-small.yaml
│   └── tts/          # 语音合成模型
│       └── qwen3-tts-0.6b.yaml
├── engines/          # 推理引擎
│   ├── vllm/
│   │   └── vllm-0.14.0-cu131-gb10.yaml
│   ├── asr/
│   │   └── funasr-sensevoice-cpu.yaml
│   └── tts/
│       └── qwen-tts-cpu.yaml
└── profiles/         # 硬件配置
    └── nvidia-jetson-thor-gb10.yaml
```

## 模型资产

| 模型 | 类型 | 参数量 | 验证状态 | 最佳引擎 |
|------|------|--------|----------|----------|
| GLM-4.7-Flash | LLM | 62B | ✅ 已验证 | vllm-0.14.0-cu131-gb10 |
| SenseVoiceSmall | ASR | 230M | ✅ 已验证 | funasr-sensevoice-cpu |
| Qwen3-TTS-0.6B | TTS | 0.6B | ✅ 已验证 | qwen-tts-cpu |

## 引擎资产

| 引擎名称 | 类型 | 版本 | Docker 镜像 | 验证状态 |
|----------|------|------|-------------|----------|
| vllm-0.14.0-cu131-gb10 | vLLM | 0.14.0 (CUDA 13.1) | zhiwen-vllm:0128 | ✅ 已验证 |
| funasr-sensevoice-cpu | ASR | FunASR 1.0 | qujing-glm-asr-nano:latest | ✅ 已验证 |
| qwen-tts-cpu | TTS | qwen-tts 0.1.1 | qujing-qwen3-tts-real:latest | ✅ 已验证 |

## 硬件配置

| 配置 | 类型 | CPU | GPU | 验证状态 |
|------|------|-----|-----|----------|
| NVIDIA Jetson Thor GB10 | Edge | 20核 ARM | 128GB 统一内存 | ✅ 已验证 |

## 快速开始

### 使用 AIMA CLI 部署

```bash
# 1. 查看可用模型
aima model list

# 2. 创建模型
aima model create glm-4.7-flash --type llm --path /mnt/data/models/GLM-4.7-Flash

# 3. 创建服务
aima service create glm-flash --model <model-id> --device gpu --port 8000

# 4. 启动服务
aima service start svc-vllm-<id> --wait --timeout 600
```

### 直接使用 Docker

```bash
# LLM (GPU)
docker run -d --gpus all -p 8000:8000 \
  -v /mnt/data/models/GLM-4.7-Flash:/models \
  zhiwen-vllm:0128 --model /models

# ASR (CPU)
docker run -d -p 8001:8000 --memory 4g \
  -v /mnt/data/models/SenseVoiceSmall:/model \
  qujing-glm-asr-nano:latest

# TTS (CPU)
docker run -d -p 8002:8002 --memory 4g \
  -v /mnt/data/models/Qwen3-TTS-0.6B:/model \
  qujing-qwen3-tts-real:latest
```

## 贡献指南

### 添加新模型

1. 在 `catalog/models/<type>/` 创建 YAML 文件
2. 按照模板填写模型信息
3. 运行验证测试
4. 更新索引

### 添加新引擎

1. 在 `catalog/engines/<type>/` 创建 YAML 文件
2. 提供完整的构建和使用说明
3. 验证与至少一个模型的兼容性
4. 更新索引

## 验证状态说明

| 状态 | 说明 |
|------|------|
| ✅ verified | 已在指定平台上验证通过 |
| ⚠️ partial | 部分功能验证通过 |
| ❌ failed | 验证失败 |
| ⏳ pending | 待验证 |

## 文件格式规范

所有资产文件使用 YAML 格式，包含以下必要字段：

```yaml
name: <资产名称>
type: <类型>
vendor: <厂商>
validation:
  <平台>:
    status: <验证状态>
    date: <验证日期>
```

## 相关文档

- [模型资产格式](./docs/catalog/model-format.md)
- [引擎资产格式](./docs/catalog/engine-format.md)
- [配置文件格式](./docs/catalog/profile-format.md)
