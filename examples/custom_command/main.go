//go:build ignore

// Package main 演示如何创建和注册自定义 Command
//
// 运行方式:
//   go run examples/custom_command.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// CustomCommand 是一个自定义 Command 示例
// 它演示了如何实现 Command 接口的所有方法
type CustomCommand struct {
	name        string
	description string
}

// NewCustomCommand 创建自定义 Command 实例
func NewCustomCommand() *CustomCommand {
	return &CustomCommand{
		name:        "custom.hello",
		description: "一个简单的问候命令",
	}
}

// Name 返回 Command 名称
func (c *CustomCommand) Name() string {
	return c.name
}

// Domain 返回所属领域
func (c *CustomCommand) Domain() string {
	return "custom"
}

// Description 返回描述
func (c *CustomCommand) Description() string {
	return c.description
}

// InputSchema 定义输入参数结构
func (c *CustomCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Title:       "HelloInput",
		Description: "问候命令的输入参数",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "要问候的名字",
					MinLength:   intPtr(1),
					MaxLength:   intPtr(50),
				},
			},
			"language": {
				Name: "language",
				Schema: unit.Schema{
					Type:        "string",
					Description: "语言 (en/zh/es)",
					Enum:        []any{"en", "zh", "es"},
					Default:     "en",
				},
			},
		},
		Required: []string{"name"},
		Optional: []string{"language"},
	}
}

// OutputSchema 定义输出结构
func (c *CustomCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Title:       "HelloOutput",
		Description: "问候命令的输出",
		Properties: map[string]unit.Field{
			"greeting": {
				Name: "greeting",
				Schema: unit.Schema{
					Type:        "string",
					Description: "问候语",
				},
			},
			"timestamp": {
				Name: "timestamp",
				Schema: unit.Schema{
					Type:        "string",
					Description: "执行时间戳",
					Format:      "date-time",
				},
			},
			"request_id": {
				Name: "request_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "请求 ID",
				},
			},
		},
	}
}

// Examples 提供使用示例
func (c *CustomCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"name":     "World",
				"language": "en",
			},
			Output: map[string]any{
				"greeting":  "Hello, World!",
				"timestamp": "2026-02-17T10:00:00Z",
				"request_id": "req_123",
			},
			Description: "英语问候",
		},
		{
			Input: map[string]any{
				"name":     "世界",
				"language": "zh",
			},
			Output: map[string]any{
				"greeting":  "你好, 世界!",
				"timestamp": "2026-02-17T10:00:00Z",
				"request_id": "req_124",
			},
			Description: "中文问候",
		},
	}
}

// Execute 执行命令
func (c *CustomCommand) Execute(ctx context.Context, input any) (any, error) {
	// 解析输入
	params, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any, got %T", input)
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required and must be a non-empty string")
	}

	language := "en"
	if v, ok := params["language"]; ok {
		if s, ok := v.(string); ok {
			language = s
		}
	}

	// 生成问候语
	var greeting string
	switch language {
	case "zh":
		greeting = fmt.Sprintf("你好, %s!", name)
	case "es":
		greeting = fmt.Sprintf("¡Hola, %s!", name)
	default: // en
		greeting = fmt.Sprintf("Hello, %s!", name)
	}

	// 获取请求 ID（如果有）
	requestID := "unknown"
	if ctx != nil {
		// 这里可以添加从 context 获取 request_id 的逻辑
		requestID = generateRequestID()
	}

	return map[string]any{
		"greeting":   greeting,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"request_id": requestID,
	}, nil
}

// intPtr 辅助函数
func intPtr(i int) *int {
	return &i
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func main() {
	fmt.Println("=== AIMA 自定义 Command 示例 ===")
	fmt.Println()

	// 1. 创建 Registry
	fmt.Println("1. 创建 Registry...")
	r := unit.NewRegistry()
	fmt.Println("   Registry 创建成功")
	fmt.Println()

	// 2. 注册自定义 Command
	fmt.Println("2. 注册自定义 Command...")
	customCmd := NewCustomCommand()
	_ = r.RegisterCommand(customCmd)
	fmt.Printf("   已注册 Command: %s\n", customCmd.Name())
	fmt.Println()

	// 3. 创建 Gateway
	fmt.Println("3. 创建 Gateway...")
	gw := gateway.NewGateway(r, gateway.WithTimeout(10*time.Second))
	fmt.Println("   Gateway 创建成功")
	fmt.Println()

	// 4. 测试自定义 Command - 英语
	fmt.Println("4. 测试自定义 Command (英语)...")
	resp := gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "custom.hello",
		Input: map[string]any{
			"name":     "AIMA",
			"language": "en",
		},
	})
	printResponse(resp)
	fmt.Println()

	// 5. 测试自定义 Command - 中文
	fmt.Println("5. 测试自定义 Command (中文)...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "custom.hello",
		Input: map[string]any{
			"name":     "世界",
			"language": "zh",
		},
	})
	printResponse(resp)
	fmt.Println()

	// 6. 测试验证错误
	fmt.Println("6. 测试验证错误 (缺少 name)...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "custom.hello",
		Input: map[string]any{
			"language": "en",
		},
	})
	printResponse(resp)
	fmt.Println()

	// 7. 显示 Schema
	fmt.Println("7. 显示 Command Schema...")
	fmt.Printf("   输入 Schema: %+v\n", customCmd.InputSchema())
	fmt.Printf("   输出 Schema: %+v\n", customCmd.OutputSchema())
	fmt.Println()

	// 8. 显示示例
	fmt.Println("8. 显示示例...")
	for i, example := range customCmd.Examples() {
		fmt.Printf("   示例 %d: %s\n", i+1, example.Description)
		fmt.Printf("     输入: %v\n", example.Input)
		fmt.Printf("     输出: %v\n", example.Output)
	}
	fmt.Println()

	fmt.Println("=== 示例执行完成 ===")
}

func printResponse(resp *gateway.Response) {
	if resp.Success {
		fmt.Printf("   ✓ 成功: %+v\n", resp.Data)
	} else {
		fmt.Printf("   ✗ 失败: %s - %s\n", resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			fmt.Printf("     详情: %v\n", resp.Error.Details)
		}
	}
}
