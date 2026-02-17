// Package main 演示如何使用流式推理
//
// 运行方式:
//   go run examples/streaming_example.go
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func main() {
	fmt.Println("=== AIMA 流式推理示例 ===")
	fmt.Println()

	// 1. 初始化
	fmt.Println("1. 初始化...")
	r := unit.NewRegistry()
	if err := registry.RegisterAllWithDefaults(r); err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	gw := gateway.NewGateway(r, gateway.WithTimeout(5*time.Minute))
	fmt.Println("   初始化完成")
	fmt.Println()

	// 2. 检查 Command 是否支持流式
	fmt.Println("2. 检查 inference.chat 是否支持流式...")
	cmd := r.GetCommand("inference.chat")
	if cmd == nil {
		log.Fatal("   inference.chat 命令不存在")
	}

	streamingCmd, ok := cmd.(unit.StreamingCommand)
	if !ok {
		log.Println("   该命令不支持流式接口")
	} else {
		fmt.Printf("   支持流式: %v\n", streamingCmd.SupportsStreaming())
	}
	fmt.Println()

	// 3. 使用 Gateway 进行流式调用
	fmt.Println("3. 使用 Gateway 进行流式推理...")
	fmt.Println("   请求内容: \"请讲一个关于人工智能的短故事\"")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stream, err := gw.HandleStream(ctx, &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "inference.chat",
		Input: map[string]any{
			"model": "llama3.2",
			"messages": []map[string]string{
				{"role": "user", "content": "请讲一个关于人工智能的短故事"},
			},
			"stream":      true,
			"temperature": 0.7,
			"max_tokens":  200,
		},
		Options: gateway.RequestOptions{
			Timeout: 2 * time.Minute,
		},
	})

	if err != nil {
		log.Fatalf("   流式调用失败: %v", err)
	}

	// 读取流式响应
	fmt.Println("   流式输出:")
	fmt.Println("   " + strings.Repeat("-", 50))
	fullContent := ""
	chunkCount := 0

	for chunk := range stream {
		if chunk.Error != nil {
			fmt.Printf("\n   错误: %s - %s\n", chunk.Error.Code, chunk.Error.Message)
			break
		}

		if chunk.Done {
			fmt.Println("\n   " + strings.Repeat("-", 50))
			fmt.Println("   [流结束]")
			if chunk.Metadata != nil {
				metadata, _ := json.Marshal(chunk.Metadata)
				fmt.Printf("   元数据: %s\n", string(metadata))
			}
			break
		}

		if chunk.Data != nil {
			content, ok := chunk.Data.(string)
			if ok {
				fmt.Print(content)
				fullContent += content
				chunkCount++
			}
		}
	}

	fmt.Printf("\n   总字符数: %d, 块数: %d\n", len(fullContent), chunkCount)
	fmt.Println()

	// 4. 演示 HTTP SSE 流式请求
	fmt.Println("4. HTTP SSE 流式请求示例...")
	fmt.Println("   curl 命令示例:")
	fmt.Println(`   curl -X POST http://localhost:9090/api/v2/stream \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -H "Accept: text/event-stream" \`)
	fmt.Println(`     -d '{`)
	fmt.Println(`       "type": "command",`)
	fmt.Println(`       "unit": "inference.chat",`)
	fmt.Println(`       "input": {`)
	fmt.Println(`         "model": "llama3.2",`)
	fmt.Println(`         "messages": [{"role": "user", "content": "Hello"}],`)
	fmt.Println(`         "stream": true`)
	fmt.Println(`       }`)
	fmt.Println(`     }'`)
	fmt.Println()

	// 5. 演示 JavaScript 客户端
	fmt.Println("5. JavaScript 客户端示例:")
	jsExample := `
const eventSource = new EventSource('/api/v2/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    type: 'command',
    unit: 'inference.chat',
    input: {
      model: 'llama3.2',
      messages: [{ role: 'user', content: 'Hello' }],
      stream: true
    }
  })
});

eventSource.onmessage = (event) => {
  const chunk = JSON.parse(event.data);
  if (chunk.done) {
    eventSource.close();
    console.log('流结束');
  } else {
    process.stdout.write(chunk.data);
  }
};

eventSource.onerror = (error) => {
  console.error('流错误:', error);
  eventSource.close();
};
`
	fmt.Println(jsExample)
	fmt.Println()

	// 6. 其他流式场景
	fmt.Println("6. 其他流式场景:")
	scenarios := []struct {
		name string
		unit string
		desc string
	}{
		{
			name: "文本补全",
			unit: "inference.complete",
			desc: "流式文本生成",
		},
		{
			name: "语音合成",
			unit: "inference.synthesize",
			desc: "流式音频生成",
		},
		{
			name: "模型拉取",
			unit: "model.pull",
			desc: "进度流",
		},
		{
			name: "日志查看",
			unit: "app.logs",
			desc: "实时日志流",
		},
	}

	for _, s := range scenarios {
		fmt.Printf("   - %s (%s): %s\n", s.name, s.unit, s.desc)
	}
	fmt.Println()

	fmt.Println("=== 示例执行完成 ===")
}

// StreamClient HTTP 流式客户端示例
type StreamClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewStreamClient(baseURL string) *StreamClient {
	return &StreamClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 0, // 流式请求不设置超时
		},
	}
}

// StreamChat 流式聊天
func (c *StreamClient) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	// 构建请求
	body, _ := json.Marshal(map[string]any{
		"type": "command",
		"unit": "inference.chat",
		"input": map[string]any{
			"model":    req.Model,
			"messages": req.Messages,
			"stream":   true,
		},
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/api/v2/stream",
		strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// 解析 SSE 流
	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err == io.EOF {
				return
			}
			if err != nil {
				chunks <- StreamChunk{Error: err}
				return
			}

			// 解析 SSE 数据行
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			select {
			case chunks <- chunk:
			case <-ctx.Done():
				return
			}

			if chunk.Done {
				return
			}
		}
	}()

	return chunks, nil
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model    string
	Messages []map[string]string
}

// StreamChunk 流式块
type StreamChunk struct {
	Data     string `json:"data"`
	Metadata any    `json:"metadata"`
	Done     bool   `json:"done"`
	Error    error  `json:"-"`
}
