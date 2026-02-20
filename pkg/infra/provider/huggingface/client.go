package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		baseURL: "https://huggingface.co",
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

func NewClientWithBaseURL(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

type ModelInfo struct {
	ID           string            `json:"_id"`
	ModelID      string            `json:"id"`
	Author       string            `json:"author"`
	SHA          string            `json:"sha"`
	LastModified string            `json:"lastModified"`
	Private      bool              `json:"private"`
	Disabled     bool              `json:"disabled"`
	Gated        bool              `json:"gated"`
	PipelineTag  string            `json:"pipeline_tag"`
	Tags         []string          `json:"tags"`
	Downloads    int               `json:"downloads"`
	Likes        int               `json:"likes"`
	LibraryName  string            `json:"library_name"`
	CreatedAt    string            `json:"createdAt"`
	ModelIndex   []ModelIndexEntry `json:"model_index,omitempty"`
	Siblings     []Sibling         `json:"siblings"`
	Config       map[string]any    `json:"config,omitempty"`
	CardData     map[string]any    `json:"cardData,omitempty"`
	Safetensors  *SafetensorsInfo  `json:"safetensors,omitempty"`
}

type ModelIndexEntry struct {
	Name    string             `json:"name"`
	Results []ModelIndexResult `json:"results"`
}

type ModelIndexResult struct {
	Task    `json:"task"`
	Dataset `json:"dataset"`
	Metric  `json:"metric"`
}

type Task struct {
	Type string `json:"type"`
}

type Dataset struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type Metric struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type Sibling struct {
	Rfilename string   `json:"rfilename"`
	BlobID    string   `json:"blobId,omitempty"`
	Size      int64    `json:"size,omitempty"`
	LFS       *LFSInfo `json:"lfs,omitempty"`
}

type LFSInfo struct {
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Pointer string `json:"pointer,omitempty"`
}

type SafetensorsInfo struct {
	Total      int64            `json:"total"`
	Parameters map[string]int64 `json:"parameters,omitempty"`
}

type SearchResponse struct {
	Items         []ModelInfo `json:"items"`
	TotalItems    int         `json:"totalItems"`
	NumItemsTotal int         `json:"numItemsTotal"`
	Search        string      `json:"search"`
	Limit         int         `json:"limit"`
	Offset        int         `json:"offset"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
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

	fullURL := c.baseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if reqBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
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
		if json.Unmarshal(respData, &errResp) == nil {
			if errResp.Error != "" {
				return fmt.Errorf("huggingface error: %s", errResp.Error)
			}
			if errResp.Message != "" {
				return fmt.Errorf("huggingface error: %s", errResp.Message)
			}
		}
		return fmt.Errorf("huggingface error: status %d, body: %s", httpResp.StatusCode, string(respData))
	}

	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) GetModelInfo(ctx context.Context, repoID string) (*ModelInfo, error) {
	var info ModelInfo
	apiPath := "/api/models/" + repoID
	if err := c.doRequest(ctx, http.MethodGet, apiPath, nil, &info); err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}
	return &info, nil
}

func (c *Client) SearchModels(ctx context.Context, query string, filter map[string]string, limit, offset int) (*SearchResponse, error) {
	params := url.Values{}
	if query != "" {
		params.Set("search", query)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	for k, v := range filter {
		params.Add("filter", k+":"+v)
	}

	apiPath := "/api/models?" + params.Encode()
	var resp SearchResponse
	if err := c.doRequest(ctx, http.MethodGet, apiPath, nil, &resp); err != nil {
		return nil, fmt.Errorf("search models: %w", err)
	}
	return &resp, nil
}

func (c *Client) DownloadFile(ctx context.Context, repoID, filename, revision string, progressFn func(downloaded, total int64)) (io.ReadCloser, int64, error) {
	if revision == "" {
		revision = "main"
	}

	downloadURL := fmt.Sprintf("%s/%s/resolve/%s/%s", c.baseURL, repoID, revision, filename)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		defer httpResp.Body.Close()
		respData, _ := io.ReadAll(httpResp.Body)
		return nil, 0, fmt.Errorf("download file: status %d, body: %s", httpResp.StatusCode, string(respData))
	}

	totalSize, _ := strconv.ParseInt(httpResp.Header.Get("Content-Length"), 10, 64)

	if progressFn != nil && totalSize > 0 {
		return &progressReader{
			reader:     httpResp.Body,
			total:      totalSize,
			progressFn: progressFn,
		}, totalSize, nil
	}

	return httpResp.Body, totalSize, nil
}

type progressReader struct {
	reader     io.ReadCloser
	read       int64
	total      int64
	progressFn func(downloaded, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.progressFn != nil {
		pr.progressFn(pr.read, pr.total)
	}
	return n, err
}

func (pr *progressReader) Close() error {
	return pr.reader.Close()
}

func (c *Client) GetFileURL(repoID, filename, revision string) string {
	if revision == "" {
		revision = "main"
	}
	return fmt.Sprintf("%s/%s/resolve/%s/%s", c.baseURL, repoID, revision, filename)
}

func (c *Client) GetLFSPointer(ctx context.Context, repoID, filename, _ string) (string, int64, error) {
	info, err := c.GetModelInfo(ctx, repoID)
	if err != nil {
		return "", 0, err
	}

	for _, sibling := range info.Siblings {
		if sibling.Rfilename == filename && sibling.LFS != nil {
			return sibling.LFS.SHA256, sibling.LFS.Size, nil
		}
	}

	return "", 0, fmt.Errorf("file %s not found or not LFS", filename)
}

func GetModelFormat(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".gguf"):
		return "gguf"
	case strings.HasSuffix(filename, ".safetensors"):
		return "safetensors"
	case strings.HasSuffix(filename, ".onnx"):
		return "onnx"
	case strings.HasSuffix(filename, ".engine") || strings.HasSuffix(filename, ".plan"):
		return "tensorrt"
	case strings.HasSuffix(filename, ".bin") || strings.HasSuffix(filename, ".pt") || strings.HasSuffix(filename, ".pth"):
		return "pytorch"
	default:
		return ""
	}
}

func DetectModelType(info *ModelInfo) string {
	if info.PipelineTag != "" {
		switch info.PipelineTag {
		case "text-generation", "text2text-generation":
			return "llm"
		case "image-text-to-text", "visual-question-answering":
			return "vlm"
		case "automatic-speech-recognition":
			return "asr"
		case "text-to-speech":
			return "tts"
		case "feature-extraction", "sentence-similarity":
			return "embedding"
		case "text-to-image", "image-to-image":
			return "diffusion"
		case "text-to-video", "image-to-video":
			return "video_gen"
		case "object-detection", "image-segmentation":
			return "detection"
		case "text-ranking", "reranking":
			return "rerank"
		}
	}

	for _, tag := range info.Tags {
		switch tag {
		case "text-generation", "causal-lm", "causal-language-model":
			return "llm"
		case "vision-language", "vlm", "visual-language":
			return "vlm"
		case "speech-recognition", "asr":
			return "asr"
		case "speech-synthesis", "tts":
			return "tts"
		case "sentence-embeddings", "embeddings":
			return "embedding"
		case "diffusion", "stable-diffusion":
			return "diffusion"
		}
	}

	return "llm"
}
