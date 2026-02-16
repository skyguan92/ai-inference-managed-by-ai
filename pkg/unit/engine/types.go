package engine

type EngineType string

const (
	EngineTypeOllama       EngineType = "ollama"
	EngineTypeVLLM         EngineType = "vllm"
	EngineTypeSGLang       EngineType = "sglang"
	EngineTypeWhisper      EngineType = "whisper"
	EngineTypeTTS          EngineType = "tts"
	EngineTypeDiffusion    EngineType = "diffusion"
	EngineTypeTransformers EngineType = "transformers"
	EngineTypeHuggingFace  EngineType = "huggingface"
	EngineTypeVideo        EngineType = "video"
	EngineTypeRerank       EngineType = "rerank"
)

type EngineStatus string

const (
	EngineStatusStopped    EngineStatus = "stopped"
	EngineStatusStarting   EngineStatus = "starting"
	EngineStatusRunning    EngineStatus = "running"
	EngineStatusStopping   EngineStatus = "stopping"
	EngineStatusError      EngineStatus = "error"
	EngineStatusInstalling EngineStatus = "installing"
)

type Engine struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         EngineType     `json:"type"`
	Status       EngineStatus   `json:"status"`
	Version      string         `json:"version,omitempty"`
	Path         string         `json:"path,omitempty"`
	ProcessID    string         `json:"process_id,omitempty"`
	Models       []string       `json:"models,omitempty"`
	Capabilities []string       `json:"capabilities,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
	CreatedAt    int64          `json:"created_at"`
	UpdatedAt    int64          `json:"updated_at"`
}

type EngineFeatures struct {
	SupportsStreaming    bool `json:"supports_streaming"`
	SupportsBatch        bool `json:"supports_batch"`
	SupportsMultimodal   bool `json:"supports_multimodal"`
	SupportsTools        bool `json:"supports_tools"`
	SupportsEmbedding    bool `json:"supports_embedding"`
	MaxConcurrent        int  `json:"max_concurrent"`
	MaxContextLength     int  `json:"max_context_length"`
	MaxBatchSize         int  `json:"max_batch_size"`
	SupportsGPULayers    bool `json:"supports_gpu_layers"`
	SupportsQuantization bool `json:"supports_quantization"`
}

type EngineFilter struct {
	Type   EngineType
	Status EngineStatus
	Limit  int
	Offset int
}

type InstallResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
}

type StartResult struct {
	ProcessID string       `json:"process_id"`
	Status    EngineStatus `json:"status"`
}

type StopResult struct {
	Success bool `json:"success"`
}

type RestartResult struct {
	ProcessID string       `json:"process_id"`
	Status    EngineStatus `json:"status"`
}
