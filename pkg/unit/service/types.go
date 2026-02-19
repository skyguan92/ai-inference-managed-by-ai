package service

type ServiceStatus string

const (
	ServiceStatusCreating ServiceStatus = "creating"
	ServiceStatusRunning  ServiceStatus = "running"
	ServiceStatusStopped  ServiceStatus = "stopped"
	ServiceStatusFailed   ServiceStatus = "failed"
)

type ResourceClass string

const (
	ResourceClassSmall  ResourceClass = "small"
	ResourceClassMedium ResourceClass = "medium"
	ResourceClassLarge  ResourceClass = "large"
)

type ModelService struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	ModelID        string          `json:"model_id"`
	Status         ServiceStatus   `json:"status"`
	Replicas       int             `json:"replicas"`
	ResourceClass  ResourceClass   `json:"resource_class"`
	Endpoints      []string        `json:"endpoints"`
	ActiveReplicas int             `json:"active_replicas"`
	Config         map[string]any  `json:"config,omitempty"`
	Metrics        *ServiceMetrics `json:"metrics,omitempty"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
}

type ServiceMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	LatencyP50        float64 `json:"latency_p50"`
	LatencyP99        float64 `json:"latency_p99"`
	TotalRequests     int64   `json:"total_requests"`
	ErrorRate         float64 `json:"error_rate"`
}

type ServiceFilter struct {
	Status  ServiceStatus `json:"status,omitempty"`
	ModelID string        `json:"model_id,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Offset  int           `json:"offset,omitempty"`
}

type CreateResult struct {
	ServiceID string `json:"service_id"`
}

type ScaleResult struct {
	Success bool `json:"success"`
}

type StartResult struct {
	Success bool `json:"success"`
}

type StopResult struct {
	Success bool `json:"success"`
}

type DeleteResult struct {
	Success bool `json:"success"`
}

type Recommendation struct {
	ResourceClass      ResourceClass `json:"resource_class"`
	Replicas           int           `json:"replicas"`
	ExpectedThroughput float64       `json:"expected_throughput"`
	EngineType         string        `json:"engine_type"`         // 推荐引擎类型: vllm, whisper, tts, ollama
	DeviceType         string        `json:"device_type"`         // 推荐设备: gpu, cpu
	Reason             string        `json:"reason"`              // 推荐理由
}
