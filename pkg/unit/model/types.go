package model

type ModelType string

const (
	ModelTypeLLM       ModelType = "llm"
	ModelTypeVLM       ModelType = "vlm"
	ModelTypeASR       ModelType = "asr"
	ModelTypeTTS       ModelType = "tts"
	ModelTypeEmbedding ModelType = "embedding"
	ModelTypeDiffusion ModelType = "diffusion"
	ModelTypeVideoGen  ModelType = "video_gen"
	ModelTypeDetection ModelType = "detection"
	ModelTypeRerank    ModelType = "rerank"
)

type ModelFormat string

const (
	FormatGGUF        ModelFormat = "gguf"
	FormatSafetensors ModelFormat = "safetensors"
	FormatONNX        ModelFormat = "onnx"
	FormatTensorRT    ModelFormat = "tensorrt"
	FormatPyTorch     ModelFormat = "pytorch"
)

type ModelStatus string

const (
	StatusPending   ModelStatus = "pending"
	StatusPulling   ModelStatus = "pulling"
	StatusReady     ModelStatus = "ready"
	StatusError     ModelStatus = "error"
	StatusVerifying ModelStatus = "verifying"
)

type Model struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Type         ModelType          `json:"type"`
	Format       ModelFormat        `json:"format"`
	Status       ModelStatus        `json:"status"`
	Source       string             `json:"source,omitempty"`
	Path         string             `json:"path,omitempty"`
	Size         int64              `json:"size,omitempty"`
	Checksum     string             `json:"checksum,omitempty"`
	Requirements *ModelRequirements `json:"requirements,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	CreatedAt    int64              `json:"created_at"`
	UpdatedAt    int64              `json:"updated_at"`
}

type ModelRequirements struct {
	MemoryMin         int64  `json:"memory_min,omitempty"`
	MemoryRecommended int64  `json:"memory_recommended,omitempty"`
	GPUType           string `json:"gpu_type,omitempty"`
	GPUMemory         int64  `json:"gpu_memory,omitempty"`
}

type ModelSearchResult struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        ModelType `json:"type"`
	Source      string    `json:"source"`
	Description string    `json:"description,omitempty"`
	Downloads   int       `json:"downloads,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

type PullProgress struct {
	ModelID    string  `json:"model_id"`
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
	BytesTotal int64   `json:"bytes_total,omitempty"`
	BytesDone  int64   `json:"bytes_done,omitempty"`
	Speed      float64 `json:"speed,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type VerificationResult struct {
	Valid  bool     `json:"valid"`
	Issues []string `json:"issues,omitempty"`
}
