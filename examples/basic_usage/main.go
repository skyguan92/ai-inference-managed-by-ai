//go:build ignore

// Package main 演示 AIMA 的基础使用方法
//
// 运行方式:
//   go run examples/basic_usage.go
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
)

func main() {
	fmt.Println("=== AIMA 基础使用示例 ===")
	fmt.Println()

	// 1. 创建 Registry 并注册所有原子单元
	fmt.Println("1. 初始化 Registry...")
	r := unit.NewRegistry()
	if err := registry.RegisterAllWithDefaults(r); err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	fmt.Printf("   已注册 %d 个 Commands, %d 个 Queries, %d 个 Resources\n",
		len(r.ListCommands()), len(r.ListQueries()), len(r.ListResources()))
	fmt.Println()

	// 2. 创建 Gateway
	fmt.Println("2. 创建 Gateway...")
	gw := gateway.NewGateway(r, gateway.WithTimeout(30*time.Second))
	fmt.Println("   Gateway 创建成功")
	fmt.Println()

	// 3. 执行 Command - 拉取模型
	fmt.Println("3. 执行 model.pull 命令...")
	resp := gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "model.pull",
		Input: map[string]any{
			"source": "ollama",
			"repo":   "llama3.2",
		},
		Options: gateway.RequestOptions{
			Timeout: 10 * time.Minute,
		},
	})

	printResponse(resp)
	fmt.Println()

	// 4. 执行 Query - 列出模型
	fmt.Println("4. 执行 model.list 查询...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "model.list",
		Input: map[string]any{
			"limit": 10,
		},
	})

	printResponse(resp)
	fmt.Println()

	// 5. 执行 Query - 获取设备信息
	fmt.Println("5. 执行 device.info 查询...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "device.info",
		Input: map[string]any{},
	})

	printResponse(resp)
	fmt.Println()

	// 6. 执行 Resource Get
	fmt.Println("6. 获取资源 asms://resource/status...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeResource,
		Unit: "asms://resource/status",
	})

	printResponse(resp)
	fmt.Println()

	// 7. 执行推理
	fmt.Println("7. 执行 inference.chat...")
	resp = gw.Handle(context.Background(), &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "inference.chat",
		Input: map[string]any{
			"model": "llama3.2",
			"messages": []map[string]string{
				{"role": "user", "content": "Hello!"},
			},
			"temperature": 0.7,
			"max_tokens":  100,
		},
	})

	printResponse(resp)
	fmt.Println()

	fmt.Println("=== 示例执行完成 ===")
}

// printResponse 打印响应结果
func printResponse(resp *gateway.Response) {
	data, _ := json.MarshalIndent(resp, "   ", "  ")
	fmt.Printf("   响应:\n%s\n", string(data))
}
