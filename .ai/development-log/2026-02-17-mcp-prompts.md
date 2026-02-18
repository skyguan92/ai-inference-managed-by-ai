# 2026-02-17 MCP Prompts 实现

## 元信息
- **开始时间**: 2026-02-17 10:40
- **完成时间**: 2026-02-17 11:00
- **实现模型**: kimi-for-coding/k2.5

## 任务概述
- **目标**: 为 MCP 适配器添加实用的 Prompts，帮助 AI Agent 更好地使用 AIMA 功能
- **范围**: pkg/gateway/mcp_prompts.go, pkg/gateway/mcp_prompts_test.go, pkg/gateway/mcp_adapter.go
- **优先级**: P1

## 设计决策
| 决策点 | 选择 | 理由 |
|--------|------|------|
| Prompt 结构 | 新增 Template 字段 | 用于存储提示模板内容 |
| 参数替换 | 简单字符串替换 | 不需要引入 text/template，避免过度设计 |
| 条件块 | {{if .arg}}...{{end}} 格式 | 模拟 Go 模板语法，AI 易于理解 |
| Prompt 列表返回 | 不暴露 Template 字段 | 遵循 MCP 协议，列表只返回元数据 |

## 实现摘要

### 新增/修改文件
- `pkg/gateway/mcp_prompts.go` - 扩充实现，添加 5 个预定义 Prompts
- `pkg/gateway/mcp_prompts_test.go` - 新增测试文件
- `pkg/gateway/mcp_adapter.go` - 添加 `prompts/get` 路由
- `pkg/unit/errors.go` - 修复编译错误（3 处 `errors.Is` 误用）

### 5 个预定义 Prompts
1. **model_management** - 帮助 AI Agent 管理模型
2. **inference_assistant** - 帮助 AI Agent 执行推理
3. **resource_optimizer** - 帮助 AI Agent 优化资源使用
4. **troubleshooting** - 帮助 AI Agent 排查问题（支持 issue_type 参数）
5. **pipeline_builder** - 帮助 AI Agent 构建工作流（支持 pipeline_type 参数）

### 关键代码

```go
// Prompt 结构
type MCPPrompt struct {
    Name        string              `json:"name"`
    Description string              `json:"description,omitempty"`
    Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
    Template    string              `json:"-"` // 不序列化
}

// 处理获取 Prompt 请求
func (a *MCPAdapter) handlePromptsGet(ctx context.Context, req *MCPRequest) *MCPResponse {
    // 参数解析 → 查找 Prompt → 渲染模板 → 返回结果
}

// 参数替换和条件块处理
func (a *MCPAdapter) renderPrompt(prompt *MCPPrompt, args map[string]string) (string, error) {
    // 支持简单替换和条件块
}
```

## 测试结果

```bash
$ go test ./pkg/gateway/... -run TestMCPAdapter.*Prompt -v
=== RUN   TestMCPAdapter_GetPrompts
--- PASS: TestMCPAdapter_GetPrompts (0.00s)
=== RUN   TestMCPAdapter_handlePromptsList
--- PASS: TestMCPAdapter_handlePromptsList (0.00s)
=== RUN   TestMCPAdapter_handlePromptsGet
--- PASS: TestMCPAdapter_handlePromptsGet (0.00s)
=== RUN   TestMCPAdapter_handlePromptsGet_WithArgs
--- PASS: TestMCPAdapter_handlePromptsGet_WithArgs (0.00s)
=== RUN   TestMCPAdapter_renderPrompt
--- PASS: TestMCPAdapter_renderPrompt (0.00s)
=== RUN   TestMCPAdapter_findPrompt
--- PASS: TestMCPAdapter_findPrompt (0.00s)
=== RUN   TestMCPAdapter_ListPrompts
--- PASS: TestMCPAdapter_ListPrompts (0.00s)
PASS
```

所有 7 个测试函数，共 18 个测试用例全部通过。

## 遇到的问题

### 1. pkg/unit/errors.go 编译错误
**问题**: `errors.Is(err, ErrCodeNotFound)` 等 3 处代码使用错误
**原因**: `ErrCodeNotFound` 是 `ErrorCode` 类型（string），不是 `error` 类型
**解决**: 改为 `errors.Is(err, ErrNotFound)`，使用 error 类型的变量

### 2. 函数名冲突
**问题**: `contains` 函数在 streaming_test.go 中已定义
**解决**: 重命名为 `promptContains` 和 `promptContainsHelper`

### 3. 条件块渲染逻辑
**问题**: 当参数存在时，需要保留条件块内容但移除标记
**解决**: 新增 `removeConditionalMarkers` 函数，在参数存在时只移除 `{{if .arg}}` 和 `{{end}}` 标记

## 验收标准
- [x] 5 个预定义 Prompts
- [x] Prompts 可动态获取
- [x] 参数替换支持
- [x] 单元测试

## 提交信息
```
feat(mcp): add 5 predefined prompts for AI Agent assistance

- Add model_management, inference_assistant, resource_optimizer,
  troubleshooting, pipeline_builder prompts
- Support argument substitution and conditional blocks
- Add comprehensive unit tests
- Fix errors.go compilation issues

Refs: .ai/development-log/2026-02-17-mcp-prompts.md
```

## 后续任务
- [ ] 根据实际使用反馈优化 Prompt 内容
- [ ] 添加更多领域特定的 Prompts
- [ ] 支持动态加载自定义 Prompts
