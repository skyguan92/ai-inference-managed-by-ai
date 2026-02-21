package catalog

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectRoot returns the absolute path to the project root (two levels above
// this test file's directory: pkg/unit/catalog → pkg/unit → pkg → project root).
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// file is …/pkg/unit/catalog/engine_asset_loader_test.go
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

func TestLoadEngineAsset_vllm(t *testing.T) {
	path := filepath.Join(projectRoot(), "catalog", "engines", "vllm", "vllm-0.14.0-cu131-gb10.yaml")
	asset, err := LoadEngineAsset(path)
	require.NoError(t, err)

	assert.Equal(t, "vllm-0.14.0-cu131-gb10", asset.Name)
	assert.Equal(t, "vllm", asset.Type)
	assert.Equal(t, "zhiwen-vllm:0128", asset.ImageFullName)
	assert.Contains(t, asset.AlternativeNames, "docker.1ms.run/scitrera/dgx-spark-vllm:0.14.0-t5")
	assert.True(t, asset.GPURequired)
	assert.Equal(t, 8, asset.CPUCoresMin)
	assert.Equal(t, "32GB", asset.MemoryMin)
	assert.Equal(t, "/health", asset.HealthCheckPath)
	assert.Equal(t, "5m", asset.HealthCheckTimeout)
	// vLLM default_args don't include --port, so DefaultPort should be 0
	assert.Equal(t, 0, asset.DefaultPort)
	assert.Contains(t, asset.DefaultArgs, "--gpu-memory-utilization")
	assert.Contains(t, asset.DefaultArgs, "--trust-remote-code")
	// BaseCommand should be populated from startup.command
	assert.Equal(t, []string{"vllm", "serve", "/models"}, asset.BaseCommand)
}

func TestLoadEngineAsset_asr(t *testing.T) {
	path := filepath.Join(projectRoot(), "catalog", "engines", "asr", "funasr-sensevoice-cpu.yaml")
	asset, err := LoadEngineAsset(path)
	require.NoError(t, err)

	assert.Equal(t, "funasr-sensevoice-cpu", asset.Name)
	assert.Equal(t, "asr", asset.Type)
	assert.Equal(t, "qujing-glm-asr-nano:latest", asset.ImageFullName)
	assert.False(t, asset.GPURequired)
	assert.Equal(t, 2, asset.CPUCoresMin)
	assert.Equal(t, "4GB", asset.MemoryMin)
	assert.Equal(t, "/health", asset.HealthCheckPath)
	assert.Equal(t, "60s", asset.HealthCheckTimeout)
	assert.Equal(t, 0, asset.DefaultPort) // no --port in default_args
}

func TestLoadEngineAsset_tts(t *testing.T) {
	path := filepath.Join(projectRoot(), "catalog", "engines", "tts", "qwen-tts-cpu.yaml")
	asset, err := LoadEngineAsset(path)
	require.NoError(t, err)

	assert.Equal(t, "qwen-tts-cpu", asset.Name)
	assert.Equal(t, "tts", asset.Type)
	assert.Equal(t, "qujing-qwen3-tts-real:latest", asset.ImageFullName)
	assert.False(t, asset.GPURequired)
	assert.Equal(t, 2, asset.CPUCoresMin)
	assert.Equal(t, "4GB", asset.MemoryMin)
	assert.Equal(t, "/health", asset.HealthCheckPath)
	assert.Equal(t, "60s", asset.HealthCheckTimeout)
	// TTS default_args include --port 8002
	assert.Equal(t, 8002, asset.DefaultPort)
}

func TestLoadEngineAssets_allEngines(t *testing.T) {
	dir := filepath.Join(projectRoot(), "catalog", "engines")
	assets, err := LoadEngineAssets(dir)
	require.NoError(t, err)

	assert.Contains(t, assets, "vllm")
	assert.Contains(t, assets, "asr")
	assert.Contains(t, assets, "tts")

	vllm := assets["vllm"]
	assert.Equal(t, "zhiwen-vllm:0128", vllm.ImageFullName)

	asr := assets["asr"]
	assert.Equal(t, "qujing-glm-asr-nano:latest", asr.ImageFullName)

	tts := assets["tts"]
	assert.Equal(t, "qujing-qwen3-tts-real:latest", tts.ImageFullName)
}

func TestLoadEngineAssets_missingDir(t *testing.T) {
	_, err := LoadEngineAssets("/nonexistent/path/to/engines")
	assert.Error(t, err)
}

func TestEngineAsset_ToRecipeEngine(t *testing.T) {
	asset := EngineAsset{
		Type:             "vllm",
		ImageFullName:    "zhiwen-vllm:0128",
		AlternativeNames: []string{"fallback:latest"},
	}
	re := asset.ToRecipeEngine()
	assert.Equal(t, "vllm", re.Type)
	assert.Equal(t, "zhiwen-vllm:0128", re.Image)
	assert.Equal(t, []string{"fallback:latest"}, re.FallbackImages)
}

func TestParseDefaultPort(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected int
	}{
		{"port present", []string{"--model", "/model", "--port", "8002"}, 8002},
		{"no port flag", []string{"--model", "/model", "--device", "cpu"}, 0},
		{"port at end", []string{"--port"}, 0},
		{"empty args", []string{}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseDefaultPort(tc.args))
		})
	}
}

func TestStripMarkdownHeaders(t *testing.T) {
	input := `# Title
## Section
name: foo
type: bar
## Another Section
key: value
`
	result := string(stripMarkdownHeaders([]byte(input)))
	assert.NotContains(t, result, "# Title")
	assert.NotContains(t, result, "## Section")
	assert.Contains(t, result, "name: foo")
	assert.Contains(t, result, "type: bar")
	assert.Contains(t, result, "key: value")
}
