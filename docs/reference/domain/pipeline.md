# Pipeline Domain

管道编排领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/pipeline/` | `pkg/pipeline/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `pipeline.create` | `{name, steps, config?}` | `{pipeline_id}` | 创建管道 |
| `pipeline.delete` | `{pipeline_id}` | `{success}` | 删除管道 |
| `pipeline.run` | `{pipeline_id, input, async?}` | `{run_id, status}` | 运行管道 |
| `pipeline.cancel` | `{run_id}` | `{success}` | 取消运行 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `pipeline.get` | `{pipeline_id}` | `{id, name, steps, status, config}` | 管道详情 |
| `pipeline.list` | `{}` | `{pipelines: []}` | 列出管道 |
| `pipeline.status` | `{run_id}` | `{status, step_results, error?}` | 运行状态 |
| `pipeline.validate` | `{steps}` | `{valid, issues: []}` | 验证定义 |

## 预定义管道

| 管道 | 步骤 | 说明 |
|------|------|------|
| voice-assistant | ASR → LLM → TTS | 语音助手 |
| rag | Embed → Search → LLM | RAG 问答 |
| vision-chat | Image → VLM → LLM | 视觉对话 |
| content-gen | LLM → ImageGen | 内容生成 |
| detect-describe | YOLO → LLM | 检测描述 |
| video-stream-analysis | 提取帧 → VLM | 视频分析 |

## 核心结构

```go
type Pipeline struct {
    ID          string
    Name        string
    Type        PipelineType
    DisplayName string
    Description string
    Steps       []PipelineStep
    Connections []Connection
    Status      PipelineStatus
}

type PipelineStep struct {
    Name       string
    Type       string          // model type
    Model      string
    Config     map[string]any
    Streaming  bool
    Condition  string          // 条件表达式
    DependsOn  []string        // 依赖步骤
    Retry      *RetryConfig
    Compensate string          // 失败补偿
}
```

## 实现文件

```
pkg/pipeline/
├── types.go               # 管道类型
├── engine.go              # 管道执行引擎
├── template.go            # 管道模板
├── template_loader.go     # 模板加载器
├── validator.go           # 管道验证
├── condition.go           # 条件执行
├── retry.go               # 重试机制
├── cache.go               # 结果缓存
├── dependency_checker.go  # 依赖检查
└── metrics.go             # 管道指标
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `pipeline.create` | ✅ | `pipeline/engine.go` |
| `pipeline.delete` | ✅ | `pipeline/engine.go` |
| `pipeline.run` | ✅ | `pipeline/engine.go` Execute() |
| `pipeline.get` | ✅ | `pipeline/engine.go` |
| `pipeline.list` | ✅ | `pipeline/engine.go` |
| `pipeline.status` | ✅ | `pipeline/engine.go` |
| `pipeline.validate` | ✅ | `pipeline/validator.go` |
| `pipeline.cancel` | ⚠️ | 需新增 |
