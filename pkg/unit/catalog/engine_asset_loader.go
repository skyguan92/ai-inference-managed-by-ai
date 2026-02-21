package catalog

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// EngineAsset represents a parsed engine asset YAML file.
type EngineAsset struct {
	Name               string   // e.g. "vllm-0.14.0-cu131-gb10"
	Type               string   // e.g. "vllm", "asr", "tts"
	ImageFullName      string   // e.g. "zhiwen-vllm:0128"
	AlternativeNames   []string // fallback images
	BaseCommand        []string // startup.command (e.g. ["vllm", "serve", "/models"])
	DefaultArgs        []string // startup.default_args
	HealthCheckPath    string   // startup.health_check.path
	HealthCheckTimeout string   // startup.health_check.timeout
	DefaultPort        int      // derived from startup.default_args "--port" value
	GPURequired        bool     // requirements.gpu.required
	MemoryMin          string   // requirements.cpu.memory_min
	CPUCoresMin        int      // requirements.cpu.cores_min
}

// engineAssetYAML mirrors the YAML structure for unmarshalling.
type engineAssetYAML struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
	Image struct {
		FullName         string   `yaml:"full_name"`
		AlternativeNames []string `yaml:"alternative_names"`
	} `yaml:"image"`
	Requirements struct {
		GPU struct {
			Required bool `yaml:"required"`
		} `yaml:"gpu"`
		CPU struct {
			CoresMin  int    `yaml:"cores_min"`
			MemoryMin string `yaml:"memory_min"`
		} `yaml:"cpu"`
	} `yaml:"requirements"`
	Startup struct {
		Command     []string `yaml:"command"`
		DefaultArgs []string `yaml:"default_args"`
		HealthCheck struct {
			Path    string `yaml:"path"`
			Timeout string `yaml:"timeout"`
		} `yaml:"health_check"`
	} `yaml:"startup"`
}

// stripMarkdownHeaders removes lines starting with '#' from YAML content so
// that yaml.v3 can parse the remainder cleanly.
func stripMarkdownHeaders(content []byte) []byte {
	var out strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

// parseDefaultPort scans a list of CLI args for "--port <value>" and returns
// the integer value, or 0 if not present.
func parseDefaultPort(args []string) int {
	for i, arg := range args {
		if arg == "--port" && i+1 < len(args) {
			if p, err := strconv.Atoi(args[i+1]); err == nil {
				return p
			}
		}
	}
	return 0
}

// parseEngineAssetBytes parses raw YAML bytes into an EngineAsset.
func parseEngineAssetBytes(raw []byte) (EngineAsset, error) {
	cleaned := stripMarkdownHeaders(raw)

	var y engineAssetYAML
	if err := yaml.Unmarshal(cleaned, &y); err != nil {
		return EngineAsset{}, err
	}

	return EngineAsset{
		Name:               y.Name,
		Type:               y.Type,
		ImageFullName:      y.Image.FullName,
		AlternativeNames:   y.Image.AlternativeNames,
		BaseCommand:        y.Startup.Command,
		DefaultArgs:        y.Startup.DefaultArgs,
		HealthCheckPath:    y.Startup.HealthCheck.Path,
		HealthCheckTimeout: y.Startup.HealthCheck.Timeout,
		DefaultPort:        parseDefaultPort(y.Startup.DefaultArgs),
		GPURequired:        y.Requirements.GPU.Required,
		MemoryMin:          y.Requirements.CPU.MemoryMin,
		CPUCoresMin:        y.Requirements.CPU.CoresMin,
	}, nil
}

// LoadEngineAsset parses a single engine asset YAML file from the filesystem.
func LoadEngineAsset(path string) (EngineAsset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return EngineAsset{}, err
	}
	return parseEngineAssetBytes(raw)
}

// LoadEngineAssets reads all *.yaml files under dir (recursively), parses each
// into an EngineAsset, and returns a map keyed by engine type (e.g. "vllm").
// If multiple files share the same type, the last one parsed wins.
func LoadEngineAssets(dir string) (map[string]EngineAsset, error) {
	assets := make(map[string]EngineAsset)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}

		asset, parseErr := LoadEngineAsset(path)
		if parseErr != nil {
			// Skip files that cannot be parsed rather than aborting.
			return nil
		}

		if asset.Type != "" {
			assets[asset.Type] = asset
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return assets, nil
}

// LoadEngineAssetsFromFS is like LoadEngineAssets but reads from an fs.FS
// (e.g. an embed.FS). dir is the root directory within the FS to walk.
func LoadEngineAssetsFromFS(fsys fs.FS, dir string) (map[string]EngineAsset, error) {
	assets := make(map[string]EngineAsset)

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}

		raw, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return nil
		}

		asset, parseErr := parseEngineAssetBytes(raw)
		if parseErr != nil {
			return nil
		}

		if asset.Type != "" {
			assets[asset.Type] = asset
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return assets, nil
}

// ToRecipeEngine converts an EngineAsset to the catalog.RecipeEngine type.
func (a EngineAsset) ToRecipeEngine() RecipeEngine {
	return RecipeEngine{
		Type:           a.Type,
		Image:          a.ImageFullName,
		FallbackImages: a.AlternativeNames,
	}
}
