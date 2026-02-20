package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General  GeneralConfig  `toml:"general"`
	API      APIConfig      `toml:"api"`
	Gateway  GatewayConfig  `toml:"gateway"`
	Resource ResourceConfig `toml:"resource"`
	Model    ModelConfig    `toml:"model"`
	Engine   EngineConfig   `toml:"engine"`
	Workflow WorkflowConfig `toml:"workflow"`
	Alert    AlertConfig    `toml:"alert"`
	Remote   RemoteConfig   `toml:"remote"`
	Security SecurityConfig `toml:"security"`
	Logging  LoggingConfig  `toml:"logging"`
	Agent    AgentConfig    `toml:"agent"`
}

type GeneralConfig struct {
	DataDir  string `toml:"data_dir"`
	Hostname string `toml:"hostname"`
	DeviceID string `toml:"device_id"`
}

type APIConfig struct {
	ListenAddr string `toml:"listen_addr"`
	EnableCORS bool   `toml:"enable_cors"`
	TLSCert    string `toml:"tls_cert"`
	TLSKey     string `toml:"tls_key"`
}

type GatewayConfig struct {
	RequestTimeout  string        `toml:"request_timeout"`
	MaxRequestSize  string        `toml:"max_request_size"`
	EnableTracing   bool          `toml:"enable_tracing"`
	RequestTimeoutD time.Duration `toml:"-"`
}

type ResourceConfig struct {
	SystemReservedMB  int     `toml:"system_reserved_mb"`
	InferencePoolPct  float64 `toml:"inference_pool_pct"`
	ContainerPoolPct  float64 `toml:"container_pool_pct"`
	PressureThreshold float64 `toml:"pressure_threshold"`
}

type ModelConfig struct {
	StorageDir    string `toml:"storage_dir"`
	DefaultSource string `toml:"default_source"`
	MaxCacheGB    int    `toml:"max_cache_gb"`
}

type EngineConfig struct {
	AutoStart  bool   `toml:"auto_start"`
	OllamaAddr string `toml:"ollama_addr"`
}

type WorkflowConfig struct {
	MaxConcurrentSteps int           `toml:"max_concurrent_steps"`
	StepTimeout        string        `toml:"step_timeout"`
	EnableCaching      bool          `toml:"enable_caching"`
	StepTimeoutD       time.Duration `toml:"-"`
}

type AlertConfig struct {
	Enabled        bool          `toml:"enabled"`
	CheckInterval  string        `toml:"check_interval"`
	CheckIntervalD time.Duration `toml:"-"`
}

type RemoteConfig struct {
	Enabled  bool   `toml:"enabled"`
	Provider string `toml:"provider"`
}

type SecurityConfig struct {
	APIKey          string `toml:"api_key"`
	RateLimitPerMin int    `toml:"rate_limit_per_min"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
	File   string `toml:"file"`
}

// AgentConfig holds settings for the AI Agent Operator.
type AgentConfig struct {
	// LLMProvider selects the client implementation: "openai" (default), "anthropic", "ollama".
	LLMProvider string `toml:"llm_provider"`
	// LLMBaseURL overrides the default API base URL (e.g. for OpenAI-compatible endpoints).
	LLMBaseURL string `toml:"llm_base_url"`
	// LLMAPIKey is the API key for the chosen LLM provider.
	LLMAPIKey string `toml:"llm_api_key"`
	// LLMModel is the model identifier to use.
	LLMModel string `toml:"llm_model"`
	// MaxTokens limits the token count for each LLM response.
	MaxTokens int `toml:"max_tokens"`
	// LLMUserAgent sets the HTTP User-Agent for LLM API requests.
	// Some endpoints (e.g. Kimi For Coding) restrict access by User-Agent.
	LLMUserAgent string `toml:"llm_user_agent"`
}

func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".aima")

	return &Config{
		General: GeneralConfig{
			DataDir:  dataDir,
			Hostname: "",
			DeviceID: "",
		},
		API: APIConfig{
			ListenAddr: "127.0.0.1:9090",
			EnableCORS: false,
			TLSCert:    "",
			TLSKey:     "",
		},
		Gateway: GatewayConfig{
			RequestTimeout: "30s",
			MaxRequestSize: "10MB",
			EnableTracing:  false,
		},
		Resource: ResourceConfig{
			SystemReservedMB:  10240,
			InferencePoolPct:  0.6,
			ContainerPoolPct:  0.2,
			PressureThreshold: 0.9,
		},
		Model: ModelConfig{
			StorageDir:    filepath.Join(dataDir, "models"),
			DefaultSource: "ollama",
			MaxCacheGB:    50,
		},
		Engine: EngineConfig{
			AutoStart:  true,
			OllamaAddr: "localhost:11434",
		},
		Workflow: WorkflowConfig{
			MaxConcurrentSteps: 10,
			StepTimeout:        "5m",
			EnableCaching:      true,
		},
		Alert: AlertConfig{
			Enabled:       true,
			CheckInterval: "1m",
		},
		Remote: RemoteConfig{
			Enabled:  false,
			Provider: "frp",
		},
		Security: SecurityConfig{
			APIKey:          "",
			RateLimitPerMin: 120,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			File:   filepath.Join(dataDir, "logs", "aima.log"),
		},
		Agent: AgentConfig{
			LLMProvider: "openai",
			LLMBaseURL:  "",
			LLMAPIKey:   "",
			LLMModel:    "moonshot-v1-8k",
			MaxTokens:   4096,
		},
	}
}

func LoadFromFile(path string) (*Config, error) {
	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("expand path: %w", err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := Default()
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("decode TOML: %w", err)
	}

	if err := cfg.postProcess(); err != nil {
		return nil, fmt.Errorf("post process config: %w", err)
	}

	return cfg, nil
}

func (c *Config) postProcess() error {
	var err error

	if c.Gateway.RequestTimeoutD, err = time.ParseDuration(c.Gateway.RequestTimeout); err != nil {
		return fmt.Errorf("parse gateway.request_timeout: %w", err)
	}

	if c.Workflow.StepTimeoutD, err = time.ParseDuration(c.Workflow.StepTimeout); err != nil {
		return fmt.Errorf("parse workflow.step_timeout: %w", err)
	}

	if c.Alert.CheckIntervalD, err = time.ParseDuration(c.Alert.CheckInterval); err != nil {
		return fmt.Errorf("parse alert.check_interval: %w", err)
	}

	c.General.DataDir, err = expandPath(c.General.DataDir)
	if err != nil {
		return fmt.Errorf("expand general.data_dir: %w", err)
	}

	c.Model.StorageDir, err = expandPath(c.Model.StorageDir)
	if err != nil {
		return fmt.Errorf("expand model.storage_dir: %w", err)
	}

	c.Logging.File, err = expandPath(c.Logging.File)
	if err != nil {
		return fmt.Errorf("expand logging.file: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Resource.InferencePoolPct+c.Resource.ContainerPoolPct > 1.0 {
		return fmt.Errorf("resource pool percentages exceed 100%% (inference=%.2f, container=%.2f)",
			c.Resource.InferencePoolPct, c.Resource.ContainerPoolPct)
	}

	if c.Resource.InferencePoolPct < 0 || c.Resource.InferencePoolPct > 1 {
		return fmt.Errorf("inference_pool_pct must be between 0 and 1, got %.2f", c.Resource.InferencePoolPct)
	}

	if c.Resource.ContainerPoolPct < 0 || c.Resource.ContainerPoolPct > 1 {
		return fmt.Errorf("container_pool_pct must be between 0 and 1, got %.2f", c.Resource.ContainerPoolPct)
	}

	if c.Resource.PressureThreshold < 0 || c.Resource.PressureThreshold > 1 {
		return fmt.Errorf("pressure_threshold must be between 0 and 1, got %.2f", c.Resource.PressureThreshold)
	}

	if c.Workflow.MaxConcurrentSteps < 1 {
		return fmt.Errorf("max_concurrent_steps must be at least 1, got %d", c.Workflow.MaxConcurrentSteps)
	}

	if c.Security.RateLimitPerMin < 0 {
		return fmt.Errorf("rate_limit_per_min cannot be negative, got %d", c.Security.RateLimitPerMin)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.Logging.Level)] {
		return fmt.Errorf("invalid logging level: %s (valid: debug, info, warn, error)", c.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[strings.ToLower(c.Logging.Format)] {
		return fmt.Errorf("invalid logging format: %s (valid: json, text)", c.Logging.Format)
	}

	return nil
}

func ApplyEnvOverrides(cfg *Config) {
	if v := os.Getenv("AIMA_DATA_DIR"); v != "" {
		cfg.General.DataDir = v
	}
	if v := os.Getenv("AIMA_HOSTNAME"); v != "" {
		cfg.General.Hostname = v
	}
	if v := os.Getenv("AIMA_DEVICE_ID"); v != "" {
		cfg.General.DeviceID = v
	}
	if v := os.Getenv("AIMA_API_LISTEN"); v != "" {
		cfg.API.ListenAddr = v
	}
	if v := os.Getenv("AIMA_API_TLS_CERT"); v != "" {
		cfg.API.TLSCert = v
	}
	if v := os.Getenv("AIMA_API_TLS_KEY"); v != "" {
		cfg.API.TLSKey = v
	}
	if v := os.Getenv("AIMA_API_KEY"); v != "" {
		cfg.Security.APIKey = v
	}
	if v := os.Getenv("AIMA_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("AIMA_OLLAMA_ADDR"); v != "" {
		cfg.Engine.OllamaAddr = v
	}
	if v := os.Getenv("AIMA_MODEL_STORAGE_DIR"); v != "" {
		cfg.Model.StorageDir = v
	}
	if v := os.Getenv("AIMA_REMOTE_ENABLED"); v != "" {
		cfg.Remote.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("AIMA_ALERT_ENABLED"); v != "" {
		cfg.Alert.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	// Agent / LLM settings.
	// Standard OPENAI_* vars are applied first, then AIMA_LLM_* vars override them.
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.Agent.LLMAPIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.Agent.LLMBaseURL = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.Agent.LLMModel = v
	}
	if v := os.Getenv("AIMA_LLM_PROVIDER"); v != "" {
		cfg.Agent.LLMProvider = v
	}
	if v := os.Getenv("AIMA_LLM_API_KEY"); v != "" {
		cfg.Agent.LLMAPIKey = v
	}
	if v := os.Getenv("AIMA_LLM_BASE_URL"); v != "" {
		cfg.Agent.LLMBaseURL = v
	}
	if v := os.Getenv("AIMA_LLM_MODEL"); v != "" {
		cfg.Agent.LLMModel = v
	}
	if v := os.Getenv("OPENAI_USER_AGENT"); v != "" {
		cfg.Agent.LLMUserAgent = v
	}
	if v := os.Getenv("AIMA_LLM_USER_AGENT"); v != "" {
		cfg.Agent.LLMUserAgent = v
	}
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}

func Load(configPath string) (*Config, error) {
	var cfg *Config
	var err error

	if configPath != "" {
		cfg, err = LoadFromFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("load config from %s: %w", configPath, err)
		}
	} else {
		cfg = Default()
	}

	ApplyEnvOverrides(cfg)

	if err := cfg.postProcess(); err != nil {
		return nil, fmt.Errorf("post process config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}
