// Package main 演示如何使用 Workflow 编排原子单元
//
// 运行方式:
//   go run examples/pipeline_example.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/workflow"
)

func main() {
	fmt.Println("=== AIMA Pipeline/Workflow 示例 ===")
	fmt.Println()

	// 1. 初始化
	fmt.Println("1. 初始化 Registry...")
	r := unit.NewRegistry()
	if err := registry.RegisterAllWithDefaults(r); err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	fmt.Println("   Registry 初始化完成")
	fmt.Println()

	// 2. 创建工作流引擎
	fmt.Println("2. 创建工作流引擎...")
	wfEngine := workflow.NewWorkflowEngine(r, nil) // nil store for demo
	fmt.Println("   工作流引擎创建成功")
	fmt.Println()

	// 3. 创建带工作流引擎的 Gateway
	fmt.Println("3. 创建 Gateway...")
	gw := gateway.NewGateway(r, gateway.WithWorkflowEngine(wfEngine))
	fmt.Println("   Gateway 创建成功")
	fmt.Println()

	// 4. 定义简单工作流
	fmt.Println("4. 定义并执行简单工作流...")
	simpleWorkflow := &workflow.WorkflowDef{
		ID:          "simple_demo",
		Name:        "简单演示",
		Description: "演示基本工作流",
		Config: map[string]any{
			"timeout": "1m",
		},
		Steps: []workflow.StepDef{
			{
				ID:   "step1",
				Type: "custom.hello",
				Input: map[string]any{
					"name":     "${input.name}",
					"language": "zh",
				},
			},
		},
		Output: map[string]any{
			"result": "${steps.step1.output.greeting}",
		},
	}

	// 注册工作流
	if err := wfEngine.RegisterWorkflow(context.Background(), simpleWorkflow); err != nil {
		log.Printf("   注册工作流失败: %v", err)
	} else {
		fmt.Println("   工作流注册成功")
	}

	// 执行工作流
	resp := gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeWorkflow,
		Unit: "simple_demo",
		Input: map[string]any{
			"name": "AIMA",
		},
	})
	printResponse(resp)
	fmt.Println()

	// 5. 定义复杂工作流 - RAG Pipeline
	fmt.Println("5. 定义 RAG Pipeline...")
	ragWorkflow := &workflow.WorkflowDef{
		ID:          "demo_rag",
		Name:        "RAG 演示",
		Description: "检索增强生成演示",
		Config: map[string]any{
			"llm_model": "llama3.2",
			"timeout":   "5m",
		},
		Steps: []workflow.StepDef{
			{
				ID:   "retrieve",
				Type: "query", // 假设有一个知识库查询
				Input: map[string]any{
					"query": "${input.question}",
					"top_k": 3,
				},
				OnFailure: "continue",
			},
			{
				ID:   "generate",
				Type: "inference.chat",
				Input: map[string]any{
					"model": "${config.llm_model}",
					"messages": []map[string]string{
						{
							"role":    "system",
							"content": "你是一个有帮助的助手。使用以下上下文回答问题：${steps.retrieve.output.context}",
						},
						{
							"role":    "user",
							"content": "${input.question}",
						},
					},
				},
			},
		},
		Output: map[string]any{
			"answer":   "${steps.generate.output.content}",
			"context":  "${steps.retrieve.output.context}",
			"sources":  "${steps.retrieve.output.sources}",
		},
	}

	// 注册 RAG 工作流
	if err := wfEngine.RegisterWorkflow(context.Background(), ragWorkflow); err != nil {
		log.Printf("   注册 RAG 工作流失败: %v", err)
	} else {
		fmt.Println("   RAG 工作流注册成功")
	}

	// 执行 RAG 工作流
	fmt.Println("   执行 RAG 工作流...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeWorkflow,
		Unit: "demo_rag",
		Input: map[string]any{
			"question": "什么是 AIMA?",
		},
	})
	printResponse(resp)
	fmt.Println()

	// 6. 显示 YAML 格式的工作流定义
	fmt.Println("6. YAML 格式工作流定义示例:")
	yamlWorkflow := `
name: voice_assistant
description: 语音输入 → ASR → LLM → TTS → 音频输出

config:
  llm_model: "llama3.2"
  tts_model: "tts-1"
  voice: "alloy"

steps:
  - id: transcribe
    type: inference.transcribe
    input:
      model: "whisper-large-v3"
      audio: "${input.audio}"
    output: text
  
  - id: chat
    type: inference.chat
    input:
      model: "${config.llm_model}"
      messages:
        - role: user
          content: "${steps.transcribe.text}"
    output: response
  
  - id: synthesize
    type: inference.synthesize
    input:
      model: "${config.tts_model}"
      text: "${steps.chat.response.content}"
      voice: "${config.voice}"
    output: audio

output:
  text: "${steps.transcribe.text}"
  response: "${steps.chat.response.content}"
  audio: "${steps.synthesize.audio}"
`
	fmt.Println(yamlWorkflow)

	// 7. 工作流模板示例
	fmt.Println("7. 使用预构建模板:")
	templateNames := []string{
		"rag",              // RAG 问答
		"voice_assistant",  // 语音助手
		"batch_inference",  // 批量推理
		"multimodal_chat",  // 多模态对话
		"video_analysis",   // 视频分析
	}
	for _, name := range templateNames {
		fmt.Printf("   - %s\n", name)
	}
	fmt.Println()

	fmt.Println("=== 示例执行完成 ===")
}

func printResponse(resp *gateway.Response) {
	data, _ := json.MarshalIndent(resp, "   ", "  ")
	fmt.Printf("   响应:\n%s\n", string(data))
}
