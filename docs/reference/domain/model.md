# Model Domain

模型管理领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/model/` | `pkg/model/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `model.create` | `{name, type, source?, format?, path?}` | `{model_id}` | 创建模型记录 |
| `model.delete` | `{model_id, force?}` | `{success}` | 删除模型 |
| `model.pull` | `{source, repo, tag?, mirror?}` | `{model_id, status}` | 从源拉取 |
| `model.import` | `{path, name?, type?, auto_detect?}` | `{model_id}` | 导入本地模型 |
| `model.verify` | `{model_id, checksum?}` | `{valid, issues: []}` | 验证完整性 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `model.get` | `{model_id}` | `{id, name, type, format, status, size, requirements}` | 模型详情 |
| `model.list` | `{type?, status?, format?, limit?, offset?}` | `{items: [], total}` | 列出模型 |
| `model.search` | `{query, source?, type?, limit?}` | `{results: []}` | 搜索模型 |
| `model.estimate_resources` | `{model_id}` | `{memory_min, memory_recommended, gpu_type}` | 预估资源 |

## 模型类型

```go
type ModelType string

const (
    ModelTypeLLM       ModelType = "llm"        // 大语言模型
    ModelTypeVLM       ModelType = "vlm"        // 视觉语言模型
    ModelTypeASR       ModelType = "asr"        // 语音识别
    ModelTypeTTS       ModelType = "tts"        // 语音合成
    ModelTypeEmbedding ModelType = "embedding"  // 文本嵌入
    ModelTypeDiffusion ModelType = "diffusion"  // 图像生成
    ModelTypeVideoGen  ModelType = "video_gen"  // 视频生成
    ModelTypeDetection ModelType = "detection"  // 目标检测
    ModelTypeRerank    ModelType = "rerank"     // 重排序
)
```

## 模型格式

```go
type ModelFormat string

const (
    FormatGGUF        ModelFormat = "gguf"
    FormatSafetensors ModelFormat = "safetensors"
    FormatONNX        ModelFormat = "onnx"
    FormatTensorRT    ModelFormat = "tensorrt"
    FormatPyTorch     ModelFormat = "pytorch"
)
```

## 下载源

| 源 | 实现文件 |
|----|----------|
| Ollama | `pkg/model/downloader/ollama.go` |
| HuggingFace | `pkg/model/downloader/huggingface.go` |
| ModelScope | `pkg/model/downloader/modelscope.go` |

## 实现文件

```
pkg/model/
├── types.go               # 模型类型
├── manager.go             # 模型管理器
├── compatibility.go       # 兼容性检查
├── downloader/            # 下载器
│   ├── ollama.go
│   ├── huggingface.go
│   └── modelscope.go
└── v2/
    ├── types.go
    ├── service/           # 服务注册
    ├── search/            # 搜索聚合
    └── download/          # 下载管理
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `model.create` | ✅ | `model/manager.go` Create() |
| `model.delete` | ✅ | `model/manager.go` Delete() |
| `model.pull` | ✅ | `model/downloader/`, `model/v2/download/` |
| `model.import` | ✅ | `model/manager.go` ImportLocal() |
| `model.get` | ✅ | `model/manager.go` Get() |
| `model.list` | ✅ | `model/manager.go` List() |
| `model.search` | ✅ | `model/v2/search/` |
| `model.estimate_resources` | ✅ | `engine/adapters/*.go` EstimateMemory() |
| `model.verify` | ⚠️ | 需新增 |
