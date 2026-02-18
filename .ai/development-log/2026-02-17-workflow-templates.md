# 2026-02-17 Workflow 预定义模板

## 元信息
- 开始时间: 2026-02-17
- 完成时间: 2026-02-17
- 实现模型: kimi-for-coding/k2.5
- 审查模型: - (自检)

## 任务概述
- **目标**: 添加更多 Workflow DSL 模板，支持常见 AI 工作流场景
- **范围**: pkg/workflow/templates/
- **优先级**: P1

## 设计决策
| 决策点 | 选择 | 理由 |
|--------|------|------|
| 模板格式 | YAML | 与现有 voice_assistant.yaml 保持一致 |
| 模板加载 | Go embed.FS | 编译时嵌入，无需运行时文件系统访问 |
| 保留原有模板 | rag_pipeline.yaml | 向后兼容，同时添加新 rag.yaml |

## 实现摘要

### 新增/修改文件

#### 模板文件 (pkg/workflow/templates/)
1. **rag.yaml** - 新的 RAG 问答流程模板
   - 3 个步骤: embed -> search -> chat
   - 支持向量检索和上下文回答

2. **batch_inference.yaml** - 批量推理模板
   - 2 个步骤: load_data -> batch_process
   - 支持批量数据处理

3. **multimodal_chat.yaml** - 多模态对话模板
   - 2 个步骤: analyze_image -> chat
   - 支持图像理解和对话

4. **video_analysis.yaml** - 视频分析模板
   - 3 个步骤: extract_frames -> analyze_frames -> summarize
   - 支持视频帧提取和摘要生成

#### 代码文件
5. **pkg/workflow/templates.go** - 模板加载器
   - 使用 `//go:embed` 嵌入模板文件
   - 提供 `LoadPredefinedTemplate()`, `ListPredefinedTemplates()`, `IsPredefinedTemplate()` 函数

6. **pkg/workflow/templates_test.go** - 模板测试
   - 每个模板的加载测试
   - 模板列表功能测试
   - 所有预定义模板的有效性验证

### 关键代码

```go
//go:embed templates/*.yaml
var templateFS embed.FS

var predefinedTemplates = map[string]string{
    "voice_assistant": "templates/voice_assistant.yaml",
    "rag":             "templates/rag.yaml",
    "batch_inference": "templates/batch_inference.yaml",
    "multimodal_chat": "templates/multimodal_chat.yaml",
    "video_analysis":  "templates/video_analysis.yaml",
    "rag_pipeline":    "templates/rag_pipeline.yaml",
}
```

## 测试结果

```bash
$ /usr/local/go/bin/go test -v ./pkg/workflow/...
=== RUN   TestLoadPredefinedTemplate
--- PASS: TestLoadPredefinedTemplate (0.00s)
=== RUN   TestListPredefinedTemplates
--- PASS: TestListPredefinedTemplates (0.00s)
=== RUN   TestIsPredefinedTemplate
--- PASS: TestIsPredefinedTemplate (0.00s)
=== RUN   TestAllPredefinedTemplatesValid
--- PASS: TestAllPredefinedTemplatesValid (0.00s)
PASS
ok      github.com/jguan/ai-inference-managed-by-ai/pkg/workflow    0.157s
```

### 测试覆盖
- 6 个预定义模板的加载验证
- 模板结构完整性检查 (steps, output)
- 模板解析有效性验证

## 遇到的问题
1. **问题**: templates.go 中使用了 `tvar` 而非 `var`
   **解决**: 修正语法错误，确保 Go embed 语法正确

## 模板清单

| 模板名称 | 描述 | 步骤数 | 适用场景 |
|----------|------|--------|----------|
| voice_assistant | 语音助手流程 | 3 | ASR → LLM → TTS |
| rag | RAG 问答 | 3 | 向量检索问答 |
| rag_pipeline | RAG 完整流程 | 4 | 带重排序的 RAG |
| batch_inference | 批量推理 | 2 | 批量数据处理 |
| multimodal_chat | 多模态对话 | 2 | 图像+文本理解 |
| video_analysis | 视频分析 | 3 | 视频摘要生成 |

## 后续任务
- [ ] 添加更多业务场景模板（如：agent 工作流、代码生成等）
- [ ] 实现模板参数验证
- [ ] 添加模板文档说明

## 提交信息
- **Branch**: AiIMA-kimi
- **Files Changed**:
  - pkg/workflow/templates/rag.yaml (新增)
  - pkg/workflow/templates/batch_inference.yaml (修改)
  - pkg/workflow/templates/multimodal_chat.yaml (修改)
  - pkg/workflow/templates/video_analysis.yaml (修改)
  - pkg/workflow/templates.go (新增)
  - pkg/workflow/templates_test.go (新增)
