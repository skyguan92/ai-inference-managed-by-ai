package modelscope

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
		baseURL: "https://api.modelscope.cn",
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
	Code    string       `json:"Code"`
	Message string       `json:"Message"`
	Data    *ModelDetail `json:"Data"`
}

type ModelDetail struct {
	ID            string       `json:"Id"`
	Name          string       `json:"Name"`
	NameCN        string       `json:"NameCN"`
	Org           string       `json:"Org"`
	OriginalName  string       `json:"OriginalName"`
	Version       string       `json:"Version"`
	CreatedAt     string       `json:"CreatedAt"`
	UpdatedAt     string       `json:"UpdatedAt"`
	Downloads     int          `json:"Downloads"`
	Follows       int          `json:"Follows"`
	Stars         int          `json:"Stars"`
	License       string       `json:"License"`
	Shape         string       `json:"Shape"`
	Gigabytes     float64      `json:"Gigabytes"`
	DatasetIDs    []string     `json:"DatasetIds"`
	Tags          []Tag        `json:"Tags"`
	PipelineTag   string       `json:"PipelineTag"`
	ModelType     string       `json:"ModelType"`
	FileSummary   *FileSummary `json:"FileSummary"`
	ModelFileList []ModelFile  `json:"ModelFileList"`
	Snapshots     []Snapshot   `json:"Snapshots"`
}

type Tag struct {
	Name        string `json:"Name"`
	TagType     string `json:"TagType"`
	Category    string `json:"Category"`
	SubCategory string `json:"SubCategory"`
}

type FileSummary struct {
	TotalSize int64 `json:"TotalSize"`
	FileCount int   `json:"FileCount"`
}

type ModelFile struct {
	Name            string   `json:"Name"`
	Composition     []string `json:"Composition"`
	Size            int64    `json:"Size"`
	ExtraMeta       string   `json:"ExtraMeta"`
	AccessID        string   `json:"AccessId"`
	URL             string   `json:"Url"`
	Type            string   `json:"Type"`
	Format          string   `json:"Format"`
	CommitID        string   `json:"CommitId"`
	VersionID       string   `json:"VersionId"`
	DownloadedCount int      `json:"DownloadedCount"`
}

type Snapshot struct {
	Name        string `json:"Name"`
	CommitID    string `json:"CommitId"`
	VersionID   string `json:"VersionId"`
	CreatedTime string `json:"CreatedTime"`
}

type SearchResponse struct {
	Code    string      `json:"Code"`
	Message string      `json:"Message"`
	Data    *SearchData `json:"Data"`
}

type SearchData struct {
	Total    int         `json:"Total"`
	Page     int         `json:"Page"`
	PageSize int         `json:"PageSize"`
	Data     []ModelItem `json:"Data"`
}

type ModelItem struct {
	ID           string  `json:"Id"`
	Name         string  `json:"Name"`
	NameCN       string  `json:"NameCN"`
	Org          string  `json:"Org"`
	OriginalName string  `json:"OriginalName"`
	Version      string  `json:"Version"`
	CreatedAt    string  `json:"CreatedAt"`
	UpdatedAt    string  `json:"UpdatedAt"`
	Downloads    int     `json:"Downloads"`
	Stars        int     `json:"Stars"`
	Gigabytes    float64 `json:"Gigabytes"`
	Tags         []Tag   `json:"Tags"`
	PipelineTag  string  `json:"PipelineTag"`
	ModelType    string  `json:"ModelType"`
	Description  string  `json:"Description"`
}

type ErrorResponse struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
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
	defer func() { _ = httpResp.Body.Close() }()

	respData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(respData, &errResp) == nil {
			if errResp.Message != "" {
				return fmt.Errorf("modelscope error: %s", errResp.Message)
			}
			if errResp.Code != "" && errResp.Code != "0" {
				return fmt.Errorf("modelscope error: code=%s", errResp.Code)
			}
		}
		return fmt.Errorf("modelscope error: status %d, body: %s", httpResp.StatusCode, string(respData))
	}

	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) GetModelInfo(ctx context.Context, modelName string) (*ModelInfo, error) {
	path := "/api/v1/models/" + modelName
	var info ModelInfo
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &info); err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}
	if info.Data == nil {
		return nil, fmt.Errorf("model not found: %s", modelName)
	}
	return &info, nil
}

func (c *Client) SearchModels(ctx context.Context, query string, modelType string, task string, limit, offset int) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("Page", "1")
	if limit > 0 {
		params.Set("PageSize", strconv.Itoa(limit))
	} else {
		params.Set("PageSize", "20")
	}
	if offset > 0 {
		params.Set("Offset", strconv.Itoa(offset))
	}
	if query != "" {
		params.Set("Search", query)
	}
	if modelType != "" {
		params.Set("ModelType", modelType)
	}
	if task != "" {
		params.Set("Task", task)
	}

	path := "/api/v1/models?" + params.Encode()
	var resp SearchResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("search models: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetDownloadURL(ctx context.Context, modelName, fileName, versionID string) (string, error) {
	info, err := c.GetModelInfo(ctx, modelName)
	if err != nil {
		return "", err
	}

	for _, file := range info.Data.ModelFileList {
		if file.Name == fileName {
			if file.URL != "" {
				return file.URL, nil
			}
			return fmt.Sprintf("https://modelscope.cn/models/%s/resolve/%s/%s", modelName, versionID, fileName), nil
		}
	}

	return "", fmt.Errorf("file not found: %s", fileName)
}

func (c *Client) DownloadFile(ctx context.Context, modelName, fileName, versionID string, progressFn func(downloaded, total int64)) (io.ReadCloser, int64, error) {
	downloadURL, err := c.GetDownloadURL(ctx, modelName, fileName, versionID)
	if err != nil {
		return nil, 0, err
	}

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
		defer func() { _ = httpResp.Body.Close() }()
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

func GetModelFormat(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".gguf"):
		return "gguf"
	case strings.HasSuffix(filename, ".safetensors"):
		return "safetensors"
	case strings.HasSuffix(filename, ".onnx"):
		return "onnx"
	case strings.HasSuffix(filename, ".bin"):
		return "bin"
	case strings.HasSuffix(filename, ".pt") || strings.HasSuffix(filename, ".pth"):
		return "pytorch"
	case strings.HasSuffix(filename, ".pdparams"):
		return "paddle"
	default:
		return ""
	}
}

func DetectModelType(pipelineTag, modelType string) string {
	if pipelineTag != "" {
		switch pipelineTag {
		case "text-generation", "text2text-generation", "chat":
			return "llm"
		case "image-text-to-text", "visual-question-answering", "multimodal":
			return "vlm"
		case "automatic-speech-recognition", "speech-recognition", "asr":
			return "asr"
		case "text-to-speech", "speech-synthesis", "tts":
			return "tts"
		case "feature-extraction", "sentence-similarity", "embedding":
			return "embedding"
		case "text-to-image", "image-to-image", "image-generation":
			return "diffusion"
		case "text-to-video", "video-generation":
			return "video_gen"
		case "object-detection", "image-segmentation":
			return "detection"
		case "text-ranking", "rerank", "text-retrieval":
			return "rerank"
		}
	}

	if modelType != "" {
		switch modelType {
		case "LLM", "LanguageModel":
			return "llm"
		case "VLM", "MultiModal":
			return "vlm"
		case "ASR", "SpeechRecognition":
			return "asr"
		case "TTS", "SpeechSynthesis":
			return "tts"
		case "Embedding", "TextEmbedding":
			return "embedding"
		case "Diffusion", "StableDiffusion":
			return "diffusion"
		case "VideoGen":
			return "video_gen"
		case "Detection":
			return "detection"
		case "Rerank":
			return "rerank"
		}
	}

	return "llm"
}
