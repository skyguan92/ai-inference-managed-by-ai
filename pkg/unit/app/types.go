package app

type AppStatus string

const (
	AppStatusInstalled AppStatus = "installed"
	AppStatusRunning   AppStatus = "running"
	AppStatusStopped   AppStatus = "stopped"
	AppStatusError     AppStatus = "error"
)

type AppCategory string

const (
	AppCategoryAIChat     AppCategory = "ai-chat"
	AppCategoryDevTool    AppCategory = "dev-tool"
	AppCategoryMonitoring AppCategory = "monitoring"
	AppCategoryCustom     AppCategory = "custom"
)

type App struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Template  string         `json:"template"`
	Status    AppStatus      `json:"status"`
	Ports     []int          `json:"ports,omitempty"`
	Volumes   []string       `json:"volumes,omitempty"`
	Config    map[string]any `json:"config,omitempty"`
	Metrics   *AppMetrics    `json:"metrics,omitempty"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

type AppMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	Uptime      int64   `json:"uptime"`
}

type Template struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Category      AppCategory    `json:"category"`
	Description   string         `json:"description"`
	Image         string         `json:"image"`
	DefaultPorts  []int          `json:"default_ports,omitempty"`
	DefaultConfig map[string]any `json:"default_config,omitempty"`
}

type AppFilter struct {
	Status   AppStatus
	Category AppCategory
	Template string
	Limit    int
	Offset   int
}

type InstallResult struct {
	AppID string `json:"app_id"`
}

type UninstallResult struct {
	Success bool `json:"success"`
}

type StartResult struct {
	Success bool `json:"success"`
}

type StopResult struct {
	Success bool `json:"success"`
}

type LogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
	Level     string `json:"level"`
}
