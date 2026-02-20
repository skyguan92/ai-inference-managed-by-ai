package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultOllamaBaseURL = "http://localhost:11434"

// OllamaClient implements LLMClient for a local Ollama instance.
type OllamaClient struct {
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
// baseURL defaults to http://localhost:11434 if empty.
func NewOllamaClient(model, baseURL string) *OllamaClient {
	if model == "" {
		model = "llama3.2"
	}
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &OllamaClient{
		model:      model,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *OllamaClient) Name() string      { return "ollama" }
func (c *OllamaClient) ModelName() string { return c.model }

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaToolCall struct {
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

type ollamaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
	PromptEvalCount int   `json:"prompt_eval_count,omitempty"`
	EvalCount       int   `json:"eval_count,omitempty"`
}

func (c *OllamaClient) Chat(ctx context.Context, messages []Message, tools []ToolDef, opts ChatOptions) (*ChatResponse, error) {
	apiMessages := make([]ollamaMessage, 0, len(messages))
	for _, m := range messages {
		apiMessages = append(apiMessages, messageToOllama(m))
	}

	req := ollamaRequest{
		Model:    c.model,
		Messages: apiMessages,
		Stream:   false,
	}

	if opts.Temperature > 0 || opts.TopP > 0 {
		req.Options = map[string]any{}
		if opts.Temperature > 0 {
			req.Options["temperature"] = opts.Temperature
		}
		if opts.TopP > 0 {
			req.Options["top_p"] = opts.TopP
		}
	}

	for _, t := range tools {
		tool := ollamaTool{Type: "function"}
		tool.Function.Name = t.Name
		tool.Function.Description = t.Description
		tool.Function.Parameters = t.InputSchema
		req.Tools = append(req.Tools, tool)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request to Ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp ollamaResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", apiResp.Error)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(data))
	}

	return ollamaResponseToChatResponse(&apiResp), nil
}

func messageToOllama(m Message) ollamaMessage {
	msg := ollamaMessage{
		Role:    m.Role,
		Content: m.Content,
	}

	// Ollama uses "tool" role for tool results (same as OpenAI).
	if len(m.ToolCalls) > 0 {
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, ollamaToolCall{
				Function: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				}{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
	}

	return msg
}

func ollamaResponseToChatResponse(r *ollamaResponse) *ChatResponse {
	resp := &ChatResponse{
		Usage: Usage{
			InputTokens:  r.PromptEvalCount,
			OutputTokens: r.EvalCount,
		},
	}

	var toolCalls []ToolCall
	for i, tc := range r.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        fmt.Sprintf("call_%d", i),
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	resp.Message = Message{
		Role:      "assistant",
		Content:   r.Message.Content,
		ToolCalls: toolCalls,
	}
	resp.ToolCalls = toolCalls
	return resp
}
