package inference

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	Content           string `json:"content"`
	FinishReason      string `json:"finish_reason"`
	Usage             Usage  `json:"usage"`
	Model             string `json:"model,omitempty"`
	ID                string `json:"id,omitempty"`
	Created           int64  `json:"created,omitempty"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

type CompletionResponse struct {
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
	Usage        Usage  `json:"usage"`
}

type EmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Usage      Usage       `json:"usage"`
}

type TranscriptionSegment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type TranscriptionResponse struct {
	Text     string                 `json:"text"`
	Segments []TranscriptionSegment `json:"segments"`
	Language string                 `json:"language"`
	Duration float64                `json:"duration,omitempty"`
	Usage    Usage                  `json:"usage,omitempty"`
}

type AudioResponse struct {
	Audio    []byte  `json:"audio"`
	Format   string  `json:"format"`
	Duration float64 `json:"duration"`
}

type GeneratedImage struct {
	Data   []byte `json:"data,omitempty"`
	URL    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
}

type ImageGenerationResponse struct {
	Images []GeneratedImage `json:"images"`
	Format string           `json:"format"`
}

type VideoGenerationResponse struct {
	Video    []byte  `json:"video"`
	Format   string  `json:"format"`
	Duration float64 `json:"duration"`
}

type RerankResult struct {
	Document string  `json:"document"`
	Score    float64 `json:"score"`
	Index    int     `json:"index"`
}

type RerankResponse struct {
	Results []RerankResult `json:"results"`
	Usage   Usage          `json:"usage"`
}

type BBox [4]float64

type Detection struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	BBox       BBox    `json:"bbox"`
}

type DetectionResponse struct {
	Detections []Detection `json:"detections"`
	Model      string      `json:"model,omitempty"`
	Usage      Usage       `json:"usage,omitempty"`
}

type Voice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Language    string `json:"language,omitempty"`
	Gender      string `json:"gender,omitempty"`
	Description string `json:"description,omitempty"`
}

type InferenceModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Provider    string   `json:"provider,omitempty"`
	Description string   `json:"description,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Modalities  []string `json:"modalities,omitempty"`
}

// ChatStreamChunk represents a single chunk in a streaming chat response
type ChatStreamChunk struct {
	ID                string `json:"id,omitempty"`
	Model             string `json:"model,omitempty"`
	Created           int64  `json:"created,omitempty"`
	Content           string `json:"content"`
	FinishReason      string `json:"finish_reason,omitempty"`
	Usage             *Usage `json:"usage,omitempty"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// CompleteStreamChunk represents a single chunk in a streaming completion response
type CompleteStreamChunk struct {
	ID           string `json:"id,omitempty"`
	Model        string `json:"model,omitempty"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason,omitempty"`
	Usage        *Usage `json:"usage,omitempty"`
}
