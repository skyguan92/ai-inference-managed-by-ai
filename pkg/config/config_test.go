package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.General.DataDir == "" {
		t.Error("General.DataDir should not be empty")
	}
	if cfg.API.ListenAddr != "127.0.0.1:9090" {
		t.Errorf("API.ListenAddr = %q, want %q", cfg.API.ListenAddr, "127.0.0.1:9090")
	}
	if cfg.Model.DefaultSource != "ollama" {
		t.Errorf("Model.DefaultSource = %q, want %q", cfg.Model.DefaultSource, "ollama")
	}
	if cfg.Engine.AutoStart != true {
		t.Error("Engine.AutoStart should be true")
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "info")
	}
}

func TestLoadFromFile(t *testing.T) {
	content := `
[general]
data_dir = "/custom/data"

[api]
listen_addr = "0.0.0.0:8080"

[model]
default_source = "huggingface"
`

	tmpFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_ = tmpFile.Close()

	cfg, err := LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.General.DataDir != "/custom/data" {
		t.Errorf("General.DataDir = %q, want %q", cfg.General.DataDir, "/custom/data")
	}
	if cfg.API.ListenAddr != "0.0.0.0:8080" {
		t.Errorf("API.ListenAddr = %q, want %q", cfg.API.ListenAddr, "0.0.0.0:8080")
	}
	if cfg.Model.DefaultSource != "huggingface" {
		t.Errorf("Model.DefaultSource = %q, want %q", cfg.Model.DefaultSource, "huggingface")
	}
}

func TestLoadFromFile_ExpandHome(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	content := `
[general]
data_dir = "~/test-data"

[model]
storage_dir = "~/test-models"
`
	tmpFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_ = tmpFile.Close()

	cfg, err := LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	expectedDataDir := filepath.Join(homeDir, "test-data")
	if cfg.General.DataDir != expectedDataDir {
		t.Errorf("General.DataDir = %q, want %q", cfg.General.DataDir, expectedDataDir)
	}

	expectedStorageDir := filepath.Join(homeDir, "test-models")
	if cfg.Model.StorageDir != expectedStorageDir {
		t.Errorf("Model.StorageDir = %q, want %q", cfg.Model.StorageDir, expectedStorageDir)
	}
}

func TestLoadFromFile_NotExist(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/config.toml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid inference_pool_pct",
			modify: func(c *Config) {
				c.Resource.InferencePoolPct = 1.5
			},
			wantErr: true,
		},
		{
			name: "invalid container_pool_pct",
			modify: func(c *Config) {
				c.Resource.ContainerPoolPct = -0.1
			},
			wantErr: true,
		},
		{
			name: "pools exceed 100%",
			modify: func(c *Config) {
				c.Resource.InferencePoolPct = 0.7
				c.Resource.ContainerPoolPct = 0.5
			},
			wantErr: true,
		},
		{
			name: "invalid pressure_threshold",
			modify: func(c *Config) {
				c.Resource.PressureThreshold = 1.5
			},
			wantErr: true,
		},
		{
			name: "invalid max_concurrent_steps",
			modify: func(c *Config) {
				c.Workflow.MaxConcurrentSteps = 0
			},
			wantErr: true,
		},
		{
			name: "invalid logging level",
			modify: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid logging format",
			modify: func(c *Config) {
				c.Logging.Format = "invalid"
			},
			wantErr: true,
		},
		{
			name: "negative rate limit",
			modify: func(c *Config) {
				c.Security.RateLimitPerMin = -1
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := Default()

	_ = os.Setenv("AIMA_DATA_DIR", "/env-data")
	_ = os.Setenv("AIMA_API_LISTEN", "0.0.0.0:3000")
	_ = os.Setenv("AIMA_LOG_LEVEL", "debug")
	_ = os.Setenv("AIMA_REMOTE_ENABLED", "true")
	defer func() {
		_ = os.Unsetenv("AIMA_DATA_DIR")
		_ = os.Unsetenv("AIMA_API_LISTEN")
		_ = os.Unsetenv("AIMA_LOG_LEVEL")
		_ = os.Unsetenv("AIMA_REMOTE_ENABLED")
	}()

	ApplyEnvOverrides(cfg)

	if cfg.General.DataDir != "/env-data" {
		t.Errorf("General.DataDir = %q, want %q", cfg.General.DataDir, "/env-data")
	}
	if cfg.API.ListenAddr != "0.0.0.0:3000" {
		t.Errorf("API.ListenAddr = %q, want %q", cfg.API.ListenAddr, "0.0.0.0:3000")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Remote.Enabled != true {
		t.Error("Remote.Enabled should be true")
	}
}

func TestApplyEnvOverrides_BooleanValues(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			cfg := Default()
			cfg.Remote.Enabled = false

			_ = os.Setenv("AIMA_REMOTE_ENABLED", tt.value)
			defer func() { _ = os.Unsetenv("AIMA_REMOTE_ENABLED") }()

			ApplyEnvOverrides(cfg)

			if cfg.Remote.Enabled != tt.expected {
				t.Errorf("Remote.Enabled = %v, want %v", cfg.Remote.Enabled, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(homeDir, "test")},
		{"~/", homeDir},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := expandPath(tt.input)
			if err != nil {
				t.Fatalf("expandPath(%q) error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("with config file", func(t *testing.T) {
		content := `
[general]
data_dir = "/test-data"

[model]
default_source = "modelscope"
`
		tmpFile, err := os.CreateTemp("", "config-*.toml")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
		_ = tmpFile.Close()

		cfg, err := Load(tmpFile.Name())
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if cfg.General.DataDir != "/test-data" {
			t.Errorf("General.DataDir = %q, want %q", cfg.General.DataDir, "/test-data")
		}
		if cfg.Model.DefaultSource != "modelscope" {
			t.Errorf("Model.DefaultSource = %q, want %q", cfg.Model.DefaultSource, "modelscope")
		}
	})

	t.Run("without config file", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if cfg.API.ListenAddr != "127.0.0.1:9090" {
			t.Errorf("API.ListenAddr = %q, want default", cfg.API.ListenAddr)
		}
	})

	t.Run("with env overrides", func(t *testing.T) {
		_ = os.Setenv("AIMA_OLLAMA_ADDR", "remote:11434")
		defer func() { _ = os.Unsetenv("AIMA_OLLAMA_ADDR") }()

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if cfg.Engine.OllamaAddr != "remote:11434" {
			t.Errorf("Engine.OllamaAddr = %q, want %q", cfg.Engine.OllamaAddr, "remote:11434")
		}
	})
}

func TestPostProcess_DurationParsing(t *testing.T) {
	content := `
[gateway]
request_timeout = "60s"

[workflow]
step_timeout = "10m"

[alert]
check_interval = "30s"
`
	tmpFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_ = tmpFile.Close()

	cfg, err := LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.Gateway.RequestTimeoutD.Seconds() != 60 {
		t.Errorf("Gateway.RequestTimeoutD = %v, want 60s", cfg.Gateway.RequestTimeoutD)
	}
	if cfg.Workflow.StepTimeoutD.Minutes() != 10 {
		t.Errorf("Workflow.StepTimeoutD = %v, want 10m", cfg.Workflow.StepTimeoutD)
	}
	if cfg.Alert.CheckIntervalD.Seconds() != 30 {
		t.Errorf("Alert.CheckIntervalD = %v, want 30s", cfg.Alert.CheckIntervalD)
	}
}
