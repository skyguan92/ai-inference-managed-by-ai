# AIMA 硬件适配经验记录

## 记录信息
- **日期**: 2026-02-19
- **机器**: NVIDIA Jetson Thor / GB10
- **架构**: ARM64 (aarch64)
- **统一内存**: 128GB
- **GPU**: NVIDIA GB10 (sm_121a)

---

## 问题背景

### 硬件特性
| 特性 | 详情 |
|------|------|
| **架构** | ARM64 (aarch64) |
| **GPU** | NVIDIA GB10 (代号 sm_121a) |
| **内存** | 128GB 统一内存 (CPU/GPU 共享) |
| **CUDA** | 13.1+ |
| **驱动** | 570+ |

### 遇到的问题
1. **GPU 架构不支持**: vLLM 官方镜像 (v0.15.0) 不支持 GB10 的 `sm_121a` 架构
2. **Triton 编译错误**: `ptxas fatal: Value 'sm_121a' is not defined for option 'gpu-name'`
3. **内存管理**: 统一内存架构需要特殊处理

---

## 解决方案

### 1. vLLM 镜像选择

| 镜像 | 版本 | 支持情况 | 说明 |
|------|------|----------|------|
| `vllm/vllm-openai:v0.15.0` | 0.15.0 | ❌ 不支持 | 官方镜像，不支持 GB10 |
| `zhiwen-vllm:0128` | 自定义 | ✅ 支持 | GB10 兼容镜像，CUDA 13.1 |

**推荐**: 使用 `zhiwen-vllm:0128` 镜像

**镜像特点**:
- CUDA 13.1
- PyTorch 2.10.0
- FlashInfer 0.6.1
- NCCL 2.29.2
- 使用 `/opt/nvidia/nvidia_entrypoint.sh` 入口

### 2. 启动参数配置

```yaml
vllm:
  image: "zhiwen-vllm:0128"
  command:
    - "vllm"
    - "serve"
    - "/models"
    - "--port", "8000"
    - "--gpu-memory-utilization", "0.75"
    - "--max-model-len", "8192"  # 限制上下文长度
```

### 3. 资源限制策略

统一内存系统需要谨慎分配：

```yaml
resource_limits:
  vllm:
    memory: "0"          # 不限制，让 vLLM 自行管理
    gpu_memory: "80g"    # 限制 80GB 统一内存
    gpu: true
    
  asr:
    memory: "4g"         # 限制 4GB
    cpu: 2.0             # 限制 2 核 CPU
    gpu: false           # 强制使用 CPU
    
  tts:
    memory: "4g"
    cpu: 2.0
    gpu: false
```

### 4. ASR/TTS 强制 CPU 运行

为避免与 vLLM 竞争 GPU 资源：
- ASR: 使用 CPU (4GB 内存, 2 核)
- TTS: 使用 CPU (4GB 内存, 2 核)

---

## 适配代码修改

### 文件: `pkg/infra/provider/hybrid_engine_provider.go`

```go
// 1. 镜像选择优先级
func (p *HybridEngineProvider) getDockerImages(name, version string) []string {
    switch name {
    case "vllm":
        return []string{
            "zhiwen-vllm:0128",                 // GB10 兼容镜像
            "vllm/vllm-openai:v0.15.0",         // 官方镜像
        }
    }
}

// 2. 启动命令适配
func (p *HybridEngineProvider) buildDockerCommand(engineType string, image string, config map[string]any, port int) []string {
    switch engineType {
    case "vllm":
        if strings.Contains(image, "zhiwen-vllm") {
            // GB10 兼容镜像使用 nvidia_entrypoint.sh
            return []string{
                "vllm", "serve", "/models",
                "--port", strconv.Itoa(port),
                "--gpu-memory-utilization", "0.75",
                "--max-model-len", "8192",
            }
        }
        // 官方镜像默认命令
        return []string{
            "--model", "/models",
            "--port", strconv.Itoa(port),
            "--gpu-memory-utilization", "0.75",
        }
    }
}
```

---

## 经验总结

### 新硬件适配流程

```
1. 识别硬件架构
   └── 检查 GPU 型号和架构 (sm_xx)
   └── 确认 CUDA 版本要求

2. 测试官方镜像
   └── 尝试官方镜像启动
   └── 记录错误信息
   └── 判断是否支持

3. 寻找替代方案
   └── 搜索社区定制镜像
   └── 检查镜像 CUDA/PyTorch 版本
   └── 验证镜像兼容性

4. 修改启动配置
   └── 适配镜像入口点
   └── 调整启动参数
   └── 配置资源限制

5. 验证功能
   └── 健康检查
   └── 基础功能测试
   └── 性能基准测试
```

### 关键检查点

| 检查项 | 命令 | 预期结果 |
|--------|------|----------|
| GPU 架构 | `nvidia-smi` | 识别 GPU 型号 |
| CUDA 版本 | `nvcc --version` | 与镜像匹配 |
| 镜像兼容性 | `docker run --rm image nvidia-smi` | 正常显示 |
| Triton 支持 | 查看启动日志 | 无 PTXAS 错误 |

### 常见错误及解决

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `sm_xx is not defined` | GPU 架构不支持 | 使用更新的镜像 |
| `PTXAS error` | Triton 编译失败 | 禁用 Triton 或更新 CUDA |
| `CUDA out of memory` | 统一内存不足 | 降低 gpu_memory_utilization |
| `Engine core initialization failed` | 内核编译失败 | 检查驱动版本 |

---

## 后续建议

### 1. 镜像管理
- 建立硬件-镜像映射表
- 维护兼容性测试列表
- 记录已知工作配置

### 2. 自动检测
```go
// 检测 GPU 架构并推荐镜像
gpuArch := detectGPUArch()
recommendedImage := getRecommendedImage(gpuArch)
```

### 3. 社区贡献
- 向 vLLM 社区反馈 GB10 支持需求
- 分享适配经验
- 参与镜像维护

---

## 参考资料

- [NVIDIA Jetson Thor Documentation](https://developer.nvidia.com/jetson-thor)
- [vLLM GitHub Issues - GPU Support](https://github.com/vllm-project/vllm/issues)
- [CUDA GPU Architecture List](https://developer.nvidia.com/cuda-gpus)
- [Triton Compiler Issues](https://github.com/openai/triton/issues)

---

**记录人**: AIMA Assistant
**状态**: ✅ 已解决
**适用版本**: aima >= 0.1.0
