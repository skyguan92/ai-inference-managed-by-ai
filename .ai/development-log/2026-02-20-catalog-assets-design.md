# [2026-02-20] 资产沉淀：模型和引擎资产目录设计

## 元信息
- 开始时间: 2026-02-20 06:30
- 完成时间: 2026-02-20 06:50
- 实现模型: GLM-5
- 审查模型: (待审查)

## 任务概述
- **目标**: 将测试验证的模型和引擎沉淀为项目资产，设计合理的目录架构
- **范围**: 目录设计、资产文件、文档、集成设计
- **优先级**: P1

## 设计决策

### 1. 目录结构

选择了三层结构：

```
catalog/
├── models/       # 模型资产（按类型分类）
│   ├── llm/
│   ├── asr/
│   └── tts/
├── engines/      # 引擎资产（按类型分类）
│   ├── vllm/
│   ├── asr/
│   └── tts/
└── profiles/     # 硬件配置（组合配置）
```

**理由**:
- 清晰的关注点分离
- 便于扩展新的模型/引擎类型
- profiles 支持一键部署

### 2. 资产文件格式

选择 YAML 格式：

**理由**:
- 人类可读性强
- 支持复杂嵌套结构
- 与 Kubernetes/Helm 生态一致
- 便于 CI/CD 集成

### 3. 验证状态追踪

每个资产都包含 `validation` 字段：

```yaml
validation:
  nvidia_gb10:
    status: verified
    date: "2026-02-20"
    engine: zhiwen-vllm:0128
    notes: "GB10 GPU 上稳定运行"
```

**理由**:
- 追踪验证历史
- 明确平台兼容性
- 便于用户选择

## 新增文件

### 模型资产

| 文件 | 描述 |
|------|------|
| `catalog/models/llm/glm-4.7-flash.yaml` | GLM-4.7-Flash 模型资产 |
| `catalog/models/asr/sensevoice-small.yaml` | SenseVoiceSmall 模型资产 |
| `catalog/models/tts/qwen3-tts-0.6b.yaml` | Qwen3-TTS-0.6B 模型资产 |

### 引擎资产

| 文件 | 描述 |
|------|------|
| `catalog/engines/vllm/zhiwen-vllm.yaml` | zhiwen-vllm:0128 引擎资产 |
| `catalog/engines/asr/qujing-glm-asr-nano.yaml` | ASR 引擎资产 |
| `catalog/engines/tts/qujing-qwen3-tts-real.yaml` | TTS 引擎资产 |

### 配置资产

| 文件 | 描述 |
|------|------|
| `catalog/profiles/nvidia-jetson-thor-gb10.yaml` | NVIDIA Jetson Thor GB10 配置 |

### 文档

| 文件 | 描述 |
|------|------|
| `catalog/README.md` | 资产目录索引 |
| `docs/catalog/integration-design.md` | AIMA CLI 集成设计 |

## 验证资产

| 资产 | 平台 | 状态 |
|------|------|------|
| GLM-4.7-Flash | NVIDIA GB10 GPU | ✅ verified |
| SenseVoiceSmall | NVIDIA GB10 CPU | ✅ verified |
| Qwen3-TTS-0.6B | NVIDIA GB10 CPU | ✅ verified |
| zhiwen-vllm:0128 | NVIDIA GB10 | ✅ verified |
| qujing-glm-asr-nano:latest | arm64 | ✅ verified |
| qujing-qwen3-tts-real:latest | arm64 | ✅ verified |

## 与 AIMA 集成设计

设计了 CLI 命令：

```bash
# 查看资产
aima catalog models
aima catalog engines
aima catalog profiles

# 部署配置
aima catalog deploy nvidia-jetson-thor-gb10

# 验证资产
aima catalog verify models/llm/glm-4.7-flash
```

## 后续任务

- [ ] 实现 `pkg/cli/catalog.go` 命令
- [ ] 实现 `pkg/catalog/loader.go` 加载器
- [ ] 添加资产缓存机制
- [ ] 支持远程仓库同步
- [ ] 添加资产验证测试

## 提交信息

- **Commit**: (待提交)
- **Message**: feat(catalog): add model and engine assets catalog

## 参考资料

- 测试报告: `test/performance_report.md`
- TTS 修复日志: `.ai/development-log/2026-02-20-bug-fixes-and-real-tts.md`
- 架构文档: `docs/ARCHITECTURE.md`

---

*由 AIMA CLI 和 GLM-5 完成*
*最后更新: 2026-02-20*
