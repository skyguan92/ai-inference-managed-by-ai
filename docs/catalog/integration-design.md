# AIMA 资产管理集成设计

## 概述

本文档描述如何将 `catalog/` 目录中的资产与 AIMA CLI 集成，实现资产的自动发现和管理。

## 目标

1. **自动发现** - AIMA CLI 自动扫描 catalog 目录
2. **快速部署** - 一键部署验证过的模型组合
3. **状态追踪** - 追踪资产的验证和使用状态

## CLI 命令设计

### 查看资产

```bash
# 列出所有可用模型
aima catalog models

# 列出所有可用引擎
aima catalog engines

# 列出所有硬件配置
aima catalog profiles

# 查看特定资产详情
aima catalog show models/llm/glm-4.7-flash
```

### 快速部署

```bash
# 使用预定义配置部署
aima catalog deploy nvidia-jetson-thor-gb10

# 这会自动：
# 1. 检查所需模型是否存在
# 2. 创建模型记录（如果不存在）
# 3. 创建并启动服务
```

### 验证资产

```bash
# 验证特定模型
aima catalog verify models/llm/glm-4.7-flash

# 验证整个配置
aima catalog verify profiles/nvidia-jetson-thor-gb10
```

## 目录结构映射

```
catalog/
├── models/
│   ├── llm/
│   │   └── glm-4.7-flash.yaml    -> aima model list --type llm
│   ├── asr/
│   │   └── sensevoice-small.yaml -> aima model list --type asr
│   └── tts/
│       └── qwen3-tts-0.6b.yaml   -> aima model list --type tts
├── engines/
│   ├── vllm/
│   │   └── zhiwen-vllm.yaml      -> aima engine list --type vllm
│   └── ...
└── profiles/
    └── nvidia-jetson-thor-gb10.yaml -> aima profile list
```

## 代码集成点

### 1. pkg/cli/catalog.go (新增)

```go
package cli

import (
    "github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
    Use:   "catalog",
    Short: "Manage model and engine assets",
}

var catalogModelsCmd = &cobra.Command{
    Use:   "models",
    Short: "List available models from catalog",
    Run:   listCatalogModels,
}

var catalogEnginesCmd = &cobra.Command{
    Use:   "engines",
    Short: "List available engines from catalog",
    Run:   listCatalogEngines,
}

var catalogProfilesCmd = &cobra.Command{
    Use:   "profiles",
    Short: "List available hardware profiles",
    Run:   listCatalogProfiles,
}
```

### 2. pkg/catalog/loader.go (新增)

```go
package catalog

import (
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

type Catalog struct {
    Models   map[string]*ModelAsset
    Engines  map[string]*EngineAsset
    Profiles map[string]*ProfileAsset
}

func LoadCatalog(root string) (*Catalog, error) {
    c := &Catalog{
        Models:   make(map[string]*ModelAsset),
        Engines:  make(map[string]*EngineAsset),
        Profiles: make(map[string]*ProfileAsset),
    }
    
    // Load models
    modelsDir := filepath.Join(root, "models")
    filepath.Walk(modelsDir, func(path string, info os.FileInfo, err error) error {
        if filepath.Ext(path) == ".yaml" {
            // Load and parse YAML
        }
        return nil
    })
    
    return c, nil
}
```

### 3. 配置文件关联

在 `configs/` 目录下添加引用：

```yaml
# configs/catalog.yaml
catalog:
  path: ./catalog
  auto_discover: true
  cache_ttl: 1h
```

## 资产文件格式

### 模型资产 (models/*.yaml)

```yaml
name: GLM-4.7-Flash
type: llm
vendor: Zhipu AI
validation:
  nvidia_gb10:
    status: verified
    date: "2026-02-20"
    engine: zhiwen-vllm:0128
spec:
  parameters: "62B"
  format: safetensors
  size_gb: 60
engines:
  - name: vllm
    recommended: zhiwen-vllm:0128
```

### 引擎资产 (engines/*.yaml)

```yaml
name: zhiwen-vllm
type: vllm
image:
  name: zhiwen-vllm
  tag: "0128"
verified_models:
  - name: GLM-4.7-Flash
    status: verified
```

### 配置资产 (profiles/*.yaml)

```yaml
name: NVIDIA Jetson Thor GB10
recommended_services:
  llm:
    model: GLM-4.7-Flash
    engine: zhiwen-vllm:0128
    device: gpu
  asr:
    model: SenseVoiceSmall
    engine: qujing-glm-asr-nano:latest
    device: cpu
```

## 使用流程

### 1. 初始化

```bash
# 初始化 catalog
aima catalog init

# 这会创建默认目录结构
```

### 2. 添加资产

```bash
# 添加模型资产
aima catalog add model ./my-model.yaml

# 添加引擎资产
aima catalog add engine ./my-engine.yaml
```

### 3. 部署

```bash
# 使用配置文件部署
aima catalog deploy nvidia-jetson-thor-gb10

# 部署单个服务
aima catalog deploy-service llm glm-4.7-flash
```

## 与现有功能集成

### 与 aima model 集成

```bash
# 从 catalog 创建模型
aima model create --from-catalog models/llm/glm-4.7-flash

# 等价于
aima model create glm-4.7-flash \
  --type llm \
  --format safetensors \
  --path /mnt/data/models/GLM-4.7-Flash
```

### 与 aima service 集成

```bash
# 从 catalog 创建服务
aima service create --from-catalog models/llm/glm-4.7-flash \
  --device gpu \
  --port 8000
```

## 后续扩展

1. **远程仓库** - 支持从远程 Git 仓库同步 catalog
2. **版本控制** - 支持资产版本管理
3. **共享机制** - 支持资产共享和协作
4. **自动测试** - 自动化资产验证流程
