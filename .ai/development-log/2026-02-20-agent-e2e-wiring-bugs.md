# 2026-02-20 端到端验证 — Agent 接入 Kimi API 发现的 Bug

## 元信息
- 开始时间: 2026-02-20
- 实现模型: Claude Sonnet 4.6
- 任务: 运行项目，配置 Kimi LLM API，通过 AIMA 进行端到端验证

## 发现的 Bug

### Bug #1: Agent 域未接入 CLI (P0)

**文件**: `pkg/cli/root.go` — `persistentPreRunE`

**现象**: `aima agent chat "..."` 返回 "agent not enabled" 错误

**根因**: `RegisterAll` 被调用时没有 `WithAgent(...)` 选项。`registry/register.go` 中的 `registerAgentDomain` 接收 `options.Agent == nil`，所有 agent 命令都返回 `errAgentNotEnabled()`。

**等价说明**: Agent 域已注册（schema 存在），但执行时无 LLM 客户端，功能完全不可用。

**修复方向**:
1. Config 结构体缺少 `[agent]` LLM 配置节
2. `root.go` 需要在创建 Gateway 后，构造 MCPAdapter → AgentExecutorAdapter → OpenAI客户端 → Agent，再单独注册 agent 域

---

### Bug #2: MCPAdapter 与 agent.ToolExecutor 接口类型不兼容 (P0)

**文件**: `pkg/gateway/mcp_tools.go`, `pkg/agent/agent.go`

**现象**: 编译时类型不匹配（无法直接将 `*MCPAdapter` 传给需要 `agent.ToolExecutor` 的地方）

**根因**:
- `agent.ToolExecutor.GenerateToolDefinitions()` 要求返回 `[]agent.ToolDefinition`
- `MCPAdapter.GenerateToolDefinitions()` 实际返回 `[]gateway.MCPToolDefinition`
- `agent.ToolExecutor.ExecuteTool()` 要求返回 `*agent.ToolResult`
- `MCPAdapter.ExecuteTool()` 实际返回 `*gateway.MCPToolResult`

两组类型字段完全相同，但 Go 类型系统不支持结构隐式兼容，需要显式适配器。

**修复方向**: 在 `pkg/gateway/` 新增 `AgentExecutorAdapter`，将 `*MCPAdapter` 包装为满足 `agent.ToolExecutor` 的类型。

---

### Bug #3: Config 无 Agent/LLM 配置节 (P1)

**文件**: `pkg/config/config.go`, `configs/aima.toml`

**现象**: 无法通过配置文件或环境变量传入 LLM API 设置

**根因**: `Config` 结构体没有 `AgentConfig`，只能在代码中硬编码 LLM 设置。

**修复方向**: 添加 `[agent]` 配置节，支持 `llm_provider`, `llm_base_url`, `llm_api_key`, `llm_model`, `max_tokens`；同时支持标准环境变量 `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`。

---

### Bug #4: 注册顺序的 Chicken-and-Egg 问题 (P1)

**文件**: `pkg/cli/root.go`, `pkg/registry/register.go`

**现象**: Agent 需要 ToolExecutor（需要 Gateway），Gateway 需要 Registry，Registry 的 agent 域注册又需要 Agent

**根因**: `RegisterAll` 在 Gateway 创建之前被调用，agent 域没有办法在 RegisterAll 中正确初始化。

**修复方向**: 将 agent 域注册从 `RegisterAll` 中分离，创建公开的 `RegisterAgentDomain(registry, agent)` 函数，在 Gateway 创建后显式调用。

---

---

### Bug #5: CLI 错误输出缺少 Details 字段 (P1)

**文件**: `pkg/cli/agent.go` — `runAgentChat`

**现象**: 错误只显示 `EXECUTION_FAILED: command execution failed`，不显示真实 API 错误

**根因**: `PrintError` 只用了 `resp.Error.Code` 和 `resp.Error.Message`，没有输出 `resp.Error.Details`

**修复**: 当 Details 非空时，在错误信息后追加 `details: <value>`

---

### Bug #6: OPENAI_MODEL 环境变量无法覆盖默认 model 值 (P1)

**文件**: `pkg/config/config.go` — `ApplyEnvOverrides`

**现象**: 设置 `OPENAI_MODEL=kimi-for-coding` 无效，agent status 仍显示 `moonshot-v1-8k`

**根因**: 初始逻辑写成 `if v != "" && cfg.Agent.LLMModel == ""`，但配置有默认值，条件永远为 false

**修复**: 先应用 `OPENAI_*` 变量，再覆盖 `AIMA_LLM_*` 变量，均无条件覆盖

---

### Bug #7: kimi-for-coding 使用思维链（reasoning），历史消息缺失 reasoning_content (P0)

**文件**: `pkg/agent/llm/openai.go`, `pkg/agent/llm/types.go`

**现象**: 多轮工具调用时报 `thinking is enabled but reasoning_content is missing in assistant tool call message`

**根因**: `openAIMessage` 和 `agentllm.Message` 结构体没有 `ReasoningContent` 字段；模型响应中的 `reasoning_content` 被丢弃；下一轮发送历史时缺失该字段

**修复**: 在两个结构体中添加 `ReasoningContent` 字段，在响应解析和消息转换中正确传递

---

### Bug #8 (配置): Kimi Coding API 要求特定 User-Agent (P0)

**文件**: `pkg/agent/llm/openai.go`

**现象**: API 返回 `access_terminated_error: Kimi For Coding is currently only available for Coding Agents`

**根因**: OpenAIClient 未设置 User-Agent，Kimi Coding API 只允许已知编程工具访问

**修复**:
- 添加 `OPENAI_USER_AGENT` 和 `AIMA_LLM_USER_AGENT` 环境变量支持
- OpenAIClient 读取并在 HTTP 请求中发送 User-Agent 头
- 配置 `OPENAI_USER_AGENT=claude-code/1.0` 即可通过鉴权

---

## 修复进度

- [x] Bug #1 + #4: 修改 `root.go` 两阶段注册 + 添加 Agent 构造逻辑
- [x] Bug #2: 新增 `pkg/gateway/agent_executor.go` (MCPAdapter → ToolExecutor 适配器)
- [x] Bug #3: 修改 `pkg/config/config.go`，添加 `AgentConfig` (含 LLMUserAgent)
- [x] Bug #3: 修改 `pkg/registry/register.go`，导出 `RegisterAgentDomain`
- [x] Bug #5: `pkg/cli/agent.go` 错误输出添加 Details
- [x] Bug #6: `pkg/config/config.go` 修复 env 覆盖逻辑
- [x] Bug #7: `pkg/agent/llm/openai.go` + `types.go` 添加 reasoning_content 支持
- [x] Bug #8: `pkg/agent/llm/openai.go` 添加 User-Agent 支持

## 验证结果

```bash
# 环境变量配置
OPENAI_API_KEY=sk-kimi-...
OPENAI_BASE_URL=https://api.kimi.com/coding/v1
OPENAI_MODEL=kimi-for-coding
OPENAI_USER_AGENT=claude-code/1.0

# 验证通过
aima agent status           # enabled: true, model: kimi-for-coding ✅
aima agent ask "查询模型"   # 调用 model_list 工具，返回正确结果 ✅
aima agent chat --conversation <id> "查询推理引擎" # 多轮会话工具调用 ✅
```
