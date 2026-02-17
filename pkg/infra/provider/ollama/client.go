package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream"`
}

type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

type GenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type GenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Model              string       `json:"model"`
	CreatedAt          string       `json:"created_at"`
	Message            *ChatMessage `json:"message,omitempty"`
	Done               bool         `json:"done"`
	TotalDuration      int64        `json:"total_duration,omitempty"`
	LoadDuration       int64        `json:"load_duration,omitempty"`
	PromptEvalCount    int          `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64        `json:"prompt_eval_duration,omitempty"`
	EvalCount          int          `json:"eval_count,omitempty"`
	EvalDuration       int64        `json:"eval_duration,omitempty"`
}

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

type ModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	Details    struct {
		Format            string `json:"format"`
		Family            string `json:"family"`
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
	} `json:"details"`
}

type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

type ShowModelRequest struct {
	Name string `json:"name"`
}

type ShowModelResponse struct {
	License    string `json:"license,omitempty"`
	Modelfile  string `json:"modelfile,omitempty"`
	Parameters string `json:"parameters,omitempty"`
	Template   string `json:"template,omitempty"`
	Details    struct {
		Format            string `json:"format"`
		Family            string `json:"family"`
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
	} `json:"details,omitempty"`
	ModelInfo map[string]any `json:"model_info,omitempty"`
}

type DeleteModelRequest struct {
	Name string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, reqBody, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	url := c.baseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if reqBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	respData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(respData, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("ollama error: %s", errResp.Error)
		}
		return fmt.Errorf("ollama error: status %d, body: %s", httpResp.StatusCode, string(respData))
	}

	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) doStreamingRequest(ctx context.Context, method, path string, reqBody any, handler func(line []byte) error) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respData, _ := io.ReadAll(httpResp.Body)
		var errResp ErrorResponse
		if json.Unmarshal(respData, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("ollama error: %s", errResp.Error)
		}
		return fmt.Errorf("ollama error: status %d", httpResp.StatusCode)
	}

	decoder := json.NewDecoder(httpResp.Body)
	for {
		var line json.RawMessage
		if err := decoder.Decode(&line); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode line: %w", err)
		}
		if err := handler(line); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) Pull(ctx context.Context, req *PullRequest, handler func(*PullResponse) error) error {
	if handler != nil {
		return c.doStreamingRequest(ctx, http.MethodPost, "/api/pull", req, func(line []byte) error {
			var resp PullResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				return fmt.Errorf("unmarshal pull response: %w", err)
			}
			return handler(&resp)
		})
	}

	req.Stream = false
	var resp PullResponse
	return c.doRequest(ctx, http.MethodPost, "/api/pull", req, &resp)
}

func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	req.Stream = false
	var resp GenerateResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/generate", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	var resp ChatResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/chat", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	var resp EmbeddingResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/embeddings", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	var resp ListModelsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/tags", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ShowModel(ctx context.Context, req *ShowModelRequest) (*ShowModelResponse, error) {
	var resp ShowModelResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/show", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DeleteModel(ctx context.Context, req *DeleteModelRequest) error {
	return c.doRequest(ctx, http.MethodDelete, "/api/delete", req, nil)
}

func (c *Client) IsRunning(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
