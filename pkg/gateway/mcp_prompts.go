package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type MCPPrompt struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
	Template    string              `json:"-"`
}

type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type MCPPromptsListResult struct {
	Prompts []MCPPrompt `json:"prompts"`
}

type MCPPromptGetParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type MCPPromptMessage struct {
	Role    string `json:"role"`
	Content struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type MCPPromptGetResult struct {
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages"`
}

var (
	modelManagementPrompt = MCPPrompt{
		Name:        "model_management",
		Description: "帮助 AI Agent 管理模型",
		Template: `你是一名模型管理助手。帮助用户管理 AI 模型。

可用操作:
- 拉取模型: model.pull
- 列出模型: model.list
- 导入本地模型: model.import
- 删除模型: model.delete

请根据用户需求选择合适的操作。`,
	}

	inferenceAssistantPrompt = MCPPrompt{
		Name:        "inference_assistant",
		Description: "帮助 AI Agent 执行推理",
		Template: `你是一名推理助手。帮助用户执行 AI 推理任务。

支持的推理类型:
- 聊天: inference.chat
- 文本补全: inference.complete
- 嵌入: inference.embed
- 语音识别: inference.transcribe
- 语音合成: inference.synthesize
- 图像生成: inference.generate_image
- 视频生成: inference.generate_video

请先检查可用模型，然后执行推理。`,
	}

	resourceOptimizerPrompt = MCPPrompt{
		Name:        "resource_optimizer",
		Description: "帮助 AI Agent 优化资源使用",
		Template: `你是一名资源优化助手。帮助用户优化 GPU/内存使用。

常用操作:
- 查看资源状态: resource.status
- 分配资源槽: resource.allocate
- 释放资源: resource.release
- 查看分配: resource.allocations

当资源紧张时，建议释放未使用的资源。`,
	}

	troubleshootingPrompt = MCPPrompt{
		Name:        "troubleshooting",
		Description: "帮助 AI Agent 排查问题",
		Arguments: []MCPPromptArgument{
			{Name: "issue_type", Description: "问题类型", Required: false},
		},
		Template: `你是一名故障排查助手。帮助用户诊断和解决系统问题。

排查步骤:
1. 检查设备健康: device.health
2. 查看引擎状态: engine.get
3. 检查告警: alert.active
4. 查看资源压力: resource.status

{{if .issue_type}}当前关注问题类型: {{.issue_type}}
{{end}}
根据发现的问题提供解决方案。`,
	}

	pipelineBuilderPrompt = MCPPrompt{
		Name:        "pipeline_builder",
		Description: "帮助 AI Agent 构建工作流",
		Arguments: []MCPPromptArgument{
			{Name: "pipeline_type", Description: "管道类型 (voice_assistant/rag/multimodal_chat/video_analysis)", Required: false},
		},
		Template: `你是一名工作流构建助手。帮助用户构建 AI 处理管道。

预定义管道:
- voice_assistant: 语音助手 (ASR→LLM→TTS)
- rag: RAG 问答 (Embed→Search→LLM)
- multimodal_chat: 多模态对话
- video_analysis: 视频分析

{{if .pipeline_type}}当前选择管道: {{.pipeline_type}}
{{end}}
也可以创建自定义管道。`,
	}
)

func (a *MCPAdapter) handlePromptsList(ctx context.Context, req *MCPRequest) *MCPResponse {
	prompts := a.GetPrompts()
	listPrompts := make([]MCPPrompt, len(prompts))
	for i, p := range prompts {
		listPrompts[i] = MCPPrompt{
			Name:        p.Name,
			Description: p.Description,
			Arguments:   p.Arguments,
		}
	}
	result := &MCPPromptsListResult{
		Prompts: listPrompts,
	}
	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) handlePromptsGet(ctx context.Context, req *MCPRequest) *MCPResponse {
	var params MCPPromptGetParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "invalid prompts/get params: "+err.Error())
		}
	}

	if params.Name == "" {
		return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "prompt name is required")
	}

	prompt, err := a.findPrompt(params.Name)
	if err != nil {
		return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, err.Error())
	}

	content, err := a.renderPrompt(prompt, params.Arguments)
	if err != nil {
		return a.errorResponse(req.ID, MCPErrorCodeInternalError, "failed to render prompt: "+err.Error())
	}

	result := &MCPPromptGetResult{
		Description: prompt.Description,
		Messages: []MCPPromptMessage{{
			Role: "assistant",
			Content: struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				Type: "text",
				Text: content,
			},
		}},
	}

	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) GetPrompts() []MCPPrompt {
	return []MCPPrompt{
		modelManagementPrompt,
		inferenceAssistantPrompt,
		resourceOptimizerPrompt,
		troubleshootingPrompt,
		pipelineBuilderPrompt,
	}
}

func (a *MCPAdapter) findPrompt(name string) (*MCPPrompt, error) {
	for _, p := range a.GetPrompts() {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("prompt not found: %s", name)
}

func (a *MCPAdapter) renderPrompt(prompt *MCPPrompt, args map[string]string) (string, error) {
	content := prompt.Template

	for _, arg := range prompt.Arguments {
		placeholder := "{{." + arg.Name + "}}"
		value, exists := args[arg.Name]

		if arg.Required && !exists {
			return "", fmt.Errorf("required argument missing: %s", arg.Name)
		}

		if exists {
			content = strings.ReplaceAll(content, placeholder, value)
			content = removeConditionalMarkers(content, arg.Name)
		} else {
			content = removeConditionalBlock(content, arg.Name)
		}
	}

	content = cleanTemplate(content)

	return content, nil
}

func removeConditionalMarkers(content, argName string) string {
	startMarker := "{{if ." + argName + "}}"
	endMarker := "{{end}}"

	content = strings.ReplaceAll(content, startMarker, "")

	for {
		endIdx := strings.Index(content, endMarker)
		if endIdx == -1 {
			break
		}
		content = content[:endIdx] + content[endIdx+len(endMarker):]
	}

	return content
}

func removeConditionalBlock(content, argName string) string {
	startMarker := "{{if ." + argName + "}}"
	endMarker := "{{end}}"

	for {
		startIdx := strings.Index(content, startMarker)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(content[startIdx:], endMarker)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + len(endMarker)

		content = content[:startIdx] + content[endIdx:]
	}

	return content
}

func cleanTemplate(content string) string {
	content = strings.ReplaceAll(content, "{{end}}", "")
	content = strings.TrimSpace(content)

	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return content
}

func (a *MCPAdapter) ListPrompts() []MCPPrompt {
	return a.GetPrompts()
}
