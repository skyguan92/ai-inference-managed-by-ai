//go:build ignore

// Package main 演示如何订阅和处理事件
//
// 运行方式:
//   go run examples/event_subscription.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// 自定义事件类型
type ModelEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func (e *ModelEvent) Type() string            { return e.eventType }
func (e *ModelEvent) Domain() string          { return e.domain }
func (e *ModelEvent) Payload() any            { return e.payload }
func (e *ModelEvent) Timestamp() time.Time    { return e.timestamp }
func (e *ModelEvent) CorrelationID() string   { return e.correlationID }

// NewModelEvent 创建模型事件
func NewModelEvent(eventType string, payload any, correlationID string) *ModelEvent {
	return &ModelEvent{
		eventType:     eventType,
		domain:        "model",
		payload:       payload,
		timestamp:     time.Now(),
		correlationID: correlationID,
	}
}

func main() {
	fmt.Println("=== AIMA 事件订阅示例 ===")
	fmt.Println()

	// 1. 创建事件总线
	fmt.Println("1. 创建事件总线...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := eventbus.NewInMemoryEventBus()
	fmt.Println("   事件总线创建成功")
	fmt.Println()

	// 2. 订阅所有事件
	fmt.Println("2. 订阅所有事件...")
	allEventsSub := bus.Subscribe(ctx, "")
	go func() {
		for event := range allEventsSub {
			fmt.Printf("   [所有事件] 收到: %s.%s (correlation_id: %s)\n",
				event.Domain(), event.Type(), event.CorrelationID())
		}
	}()
	fmt.Println("   已订阅所有事件")
	fmt.Println()

	// 3. 订阅特定领域事件 - model
	fmt.Println("3. 订阅 model 领域事件...")
	modelEventsSub := bus.Subscribe(ctx, "model")
	go func() {
		for event := range modelEventsSub {
			payload, _ := event.Payload().(map[string]any)
			fmt.Printf("   [Model事件] %s: model=%s\n",
				event.Type(), payload["model_id"])
		}
	}()
	fmt.Println("   已订阅 model 事件")
	fmt.Println()

	// 4. 订阅特定类型事件
	fmt.Println("4. 订阅 model.pull_progress 事件...")
	progressEventsSub := bus.Subscribe(ctx, "model.pull_progress")
	go func() {
		for event := range progressEventsSub {
			payload := event.Payload().(map[string]any)
			fmt.Printf("   [进度] model=%s progress=%.1f%%\n",
				payload["model_id"], payload["progress"])
		}
	}()
	fmt.Println("   已订阅 progress 事件")
	fmt.Println()

	// 5. 发布事件
	fmt.Println("5. 发布事件...")
	time.Sleep(100 * time.Millisecond) // 等待订阅者就绪

	// 发布模型创建事件
	event1 := NewModelEvent("model.created", map[string]any{
		"model_id": "llama3.2:latest",
		"name":     "llama3.2",
		"size":     3825393664,
	}, "trace_001")
	_ = bus.Publish(ctx, event1)
	fmt.Println("   发布: model.created")

	// 发布拉取进度事件
	for i := 0; i <= 100; i += 25 {
		progressEvent := NewModelEvent("model.pull_progress", map[string]any{
			"model_id": "llama3.2:latest",
			"progress": float64(i),
			"status":   "downloading",
		}, "trace_001")
		_ = bus.Publish(ctx, progressEvent)
		time.Sleep(100 * time.Millisecond)
	}

	// 发布完成事件
	event2 := NewModelEvent("model.verified", map[string]any{
		"model_id": "llama3.2:latest",
		"valid":    true,
	}, "trace_001")
	_ = bus.Publish(ctx, event2)
	fmt.Println("   发布: model.verified")
	fmt.Println()

	// 6. 发布推理事件
	fmt.Println("6. 发布推理事件...")
	inferenceEvents := []struct {
		typ     string
		payload map[string]any
	}{
		{
			"inference.request_started",
			map[string]any{
				"request_id": "req_001",
				"model":      "llama3.2",
				"type":       "chat",
			},
		},
		{
			"inference.request_completed",
			map[string]any{
				"request_id": "req_001",
				"duration":   2345,
				"tokens":     150,
			},
		},
	}

	for _, e := range inferenceEvents {
		event := &inferenceEvent{
			eventType:     e.typ,
			domain:        "inference",
			payload:       e.payload,
			timestamp:     time.Now(),
			correlationID: "trace_002",
		}
		_ = bus.Publish(ctx, event)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println("   推理事件发布完成")
	fmt.Println()

	// 7. 使用事件处理器
	fmt.Println("7. 使用事件处理器...")
	handler := &ModelEventHandler{}
	bus.SubscribeWithHandler(ctx, "model", handler)
	fmt.Println("   已注册 ModelEventHandler")

	// 发布事件触发处理器
	_ = bus.Publish(ctx, NewModelEvent("model.created", map[string]any{
		"model_id": "mistral:latest",
		"name":     "mistral",
	}, "trace_003"))
	time.Sleep(100 * time.Millisecond)
	fmt.Println()

	// 8. 事件持久化示例
	fmt.Println("8. 事件持久化...")
	persistentBus, err := eventbus.NewPersistentEventBus(
		"/tmp/aima_events.db",
		eventbus.WithRetention(7*24*time.Hour),
	eventbus.WithMaxEvents(10000),
	)
	if err != nil {
		log.Printf("   创建持久化事件总线失败: %v", err)
	} else {
		fmt.Println("   持久化事件总线创建成功")
		persistentBus.Publish(ctx, NewModelEvent("model.created", map[string]any{
			"model_id": "test:latest",
		}, "trace_004"))
		fmt.Println("   事件已持久化")
	}
	fmt.Println()

	// 等待事件处理
	time.Sleep(500 * time.Millisecond)

	fmt.Println("=== 示例执行完成 ===")
}

// inferenceEvent 推理事件
type inferenceEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func (e *inferenceEvent) Type() string          { return e.eventType }
func (e *inferenceEvent) Domain() string        { return e.domain }
func (e *inferenceEvent) Payload() any          { return e.payload }
func (e *inferenceEvent) Timestamp() time.Time  { return e.timestamp }
func (e *inferenceEvent) CorrelationID() string { return e.correlationID }

// ModelEventHandler 模型事件处理器
type ModelEventHandler struct{}

func (h *ModelEventHandler) Handle(ctx context.Context, event unit.Event) error {
	fmt.Printf("   [Handler] 处理事件: %s\n", event.Type())

	switch event.Type() {
	case "model.created":
		payload := event.Payload().(map[string]any)
		fmt.Printf("      -> 模型 %s 已创建\n", payload["model_id"])
		// 可以在这里发送通知、更新缓存等

	case "model.deleted":
		payload := event.Payload().(map[string]any)
		fmt.Printf("      -> 模型 %s 已删除\n", payload["model_id"])
	}

	return nil
}
