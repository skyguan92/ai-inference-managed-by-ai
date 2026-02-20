package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
	defaultMaxTokens    = 4096
)

// AnthropicClient implements LLMClient using Anthropic's Messages API via raw HTTP.
type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAnthropicClient creates a new Anthropic client.
// The API key is read from ANTHROPIC_API_KEY if apiKey is empty.
func NewAnthropicClient(model, apiKey string) *AnthropicClient {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	return &AnthropicClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

func (c *AnthropicClient) Name() string      { return "anthropic" }
func (c *AnthropicClient) ModelName() string { return c.model }

// anthropicMessage is the wire format for Anthropic API messages.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []anthropicContentBlock
}

type anthropicContentBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []anthropicContentBlock `json:"content"`
	StopReason   string                  `json:"stop_reason"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []Message, tools []ToolDef, opts ChatOptions) (*ChatResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	// Separate system prompt from conversation messages.
	var system string
	var apiMessages []anthropicMessage
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		apiMessages = append(apiMessages, messagesToAnthropic(m))
	}

	req := anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  apiMessages,
	}

	for _, t := range tools {
		req.Tools = append(req.Tools, anthropicTool(t))
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("anthropic API error [%s]: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API status %d: %s", resp.StatusCode, string(data))
	}

	return anthropicResponseToChatResponse(&apiResp), nil
}

// messagesToAnthropic converts a Message to Anthropic wire format.
func messagesToAnthropic(m Message) anthropicMessage {
	switch m.Role {
	case "tool":
		// Tool results use a content block array.
		return anthropicMessage{
			Role: "user",
			Content: []anthropicContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				},
			},
		}
	case "assistant":
		if len(m.ToolCalls) == 0 {
			return anthropicMessage{Role: "assistant", Content: m.Content}
		}
		// Assistant message with tool calls uses content block array.
		blocks := []anthropicContentBlock{}
		if m.Content != "" {
			blocks = append(blocks, anthropicContentBlock{Type: "text", Text: m.Content})
		}
		for _, tc := range m.ToolCalls {
			blocks = append(blocks, anthropicContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Arguments,
			})
		}
		return anthropicMessage{Role: "assistant", Content: blocks}
	default:
		return anthropicMessage{Role: m.Role, Content: m.Content}
	}
}

// anthropicResponseToChatResponse converts an Anthropic API response to our ChatResponse.
func anthropicResponseToChatResponse(r *anthropicResponse) *ChatResponse {
	resp := &ChatResponse{
		Usage: Usage{
			InputTokens:  r.Usage.InputTokens,
			OutputTokens: r.Usage.OutputTokens,
		},
	}

	var textContent string
	var toolCalls []ToolCall

	for _, block := range r.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	resp.Message = Message{
		Role:      "assistant",
		Content:   textContent,
		ToolCalls: toolCalls,
	}
	resp.ToolCalls = toolCalls
	return resp
}
