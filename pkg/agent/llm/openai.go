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

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIClient implements LLMClient for any OpenAI-compatible API.
type OpenAIClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI-compatible client.
// baseURL defaults to the official OpenAI endpoint if empty.
// The API key is read from OPENAI_API_KEY if apiKey is empty.
func NewOpenAIClient(model, apiKey, baseURL string) *OpenAIClient {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIClient{
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *OpenAIClient) Name() string      { return "openai" }
func (c *OpenAIClient) ModelName() string { return c.model }

// openAIMessage is the wire format for OpenAI chat messages.
type openAIMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content"` // string or nil
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	Name       string         `json:"name,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
	MaxTokens int            `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
	TopP        float64      `json:"top_p,omitempty"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message, tools []ToolDef, opts ChatOptions) (*ChatResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	apiMessages := make([]openAIMessage, 0, len(messages))
	for _, m := range messages {
		apiMessages = append(apiMessages, messageToOpenAI(m))
	}

	req := openAIRequest{
		Model:       c.model,
		Messages:    apiMessages,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		TopP:        opts.TopP,
	}

	for _, t := range tools {
		tool := openAITool{Type: "function"}
		tool.Function.Name = t.Name
		tool.Function.Description = t.Description
		tool.Function.Parameters = t.InputSchema
		req.Tools = append(req.Tools, tool)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("openai API error [%s]: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai API status %d: %s", resp.StatusCode, string(data))
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai API returned no choices")
	}

	return openAIResponseToChatResponse(&apiResp), nil
}

func messageToOpenAI(m Message) openAIMessage {
	msg := openAIMessage{
		Role:    m.Role,
		Content: m.Content,
	}

	if m.Role == "tool" {
		msg.Role = "tool"
		msg.ToolCallID = m.ToolCallID
	}

	if len(m.ToolCalls) > 0 {
		for _, tc := range m.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			msg.ToolCalls = append(msg.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return msg
}

func openAIResponseToChatResponse(r *openAIResponse) *ChatResponse {
	choice := r.Choices[0]
	resp := &ChatResponse{
		Usage: Usage{
			InputTokens:  r.Usage.PromptTokens,
			OutputTokens: r.Usage.CompletionTokens,
		},
	}

	content := ""
	if s, ok := choice.Message.Content.(string); ok {
		content = s
	}

	var toolCalls []ToolCall
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	resp.Message = Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	}
	resp.ToolCalls = toolCalls
	return resp
}
