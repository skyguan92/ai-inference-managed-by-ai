package catalog

// HardwareProfile describes a specific hardware configuration.
type HardwareProfile struct {
	GPUVendor  string   `json:"gpu_vendor" yaml:"gpu_vendor"`
	GPUModel   string   `json:"gpu_model" yaml:"gpu_model"`
	GPUArch    string   `json:"gpu_arch" yaml:"gpu_arch"`
	VRAMMinGB  int      `json:"vram_min_gb" yaml:"vram_min_gb"`
	CPUArch    string   `json:"cpu_arch" yaml:"cpu_arch"`
	OS         string   `json:"os" yaml:"os"`
	UnifiedMem bool     `json:"unified_memory" yaml:"unified_memory"`
	Tags       []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// RecipeEngine describes the inference engine image configuration.
type RecipeEngine struct {
	Type           string         `json:"type" yaml:"type"`
	Image          string         `json:"image" yaml:"image"`
	FallbackImages []string       `json:"fallback_images,omitempty" yaml:"fallback_images,omitempty"`
	Config         map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
}

// RecipeModel describes model download configuration.
type RecipeModel struct {
	Name           string `json:"name" yaml:"name"`
	Source         string `json:"source" yaml:"source"`
	Repo           string `json:"repo" yaml:"repo"`
	Tag            string `json:"tag,omitempty" yaml:"tag,omitempty"`
	Type           string `json:"type" yaml:"type"`
	Format         string `json:"format,omitempty" yaml:"format,omitempty"`
	Mirror         string `json:"mirror,omitempty" yaml:"mirror,omitempty"`
	MemoryRequired int64  `json:"memory_required,omitempty" yaml:"memory_required,omitempty"`
}

// ResourceLimits describes resource limit settings for a recipe.
type ResourceLimits struct {
	GPUMemoryUtilization float64 `json:"gpu_memory_utilization" yaml:"gpu_memory_utilization"`
	MaxModelLen          int     `json:"max_model_len" yaml:"max_model_len"`
	TensorParallel       int     `json:"tensor_parallel" yaml:"tensor_parallel"`
}

// RecipeStatus represents the local readiness status of recipe artifacts.
type RecipeStatus struct {
	EngineReady  bool          `json:"engine_ready"`
	ModelsReady  []ModelStatus `json:"models_ready"`
}

// ModelStatus represents the local readiness status of a single model.
type ModelStatus struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}

// Recipe maps hardware to a validated engine+model+config combination.
type Recipe struct {
	ID             string          `json:"id" yaml:"id"`
	Name           string          `json:"name" yaml:"name"`
	Description    string          `json:"description" yaml:"description"`
	Version        string          `json:"version" yaml:"version"`
	Author         string          `json:"author,omitempty" yaml:"author,omitempty"`
	Profile        HardwareProfile `json:"profile" yaml:"profile"`
	Engine         RecipeEngine    `json:"engine" yaml:"engine"`
	Models         []RecipeModel   `json:"models" yaml:"models"`
	ResourceLimits ResourceLimits  `json:"resource_limits" yaml:"resource_limits"`
	Verified       bool            `json:"verified" yaml:"verified"`
	Tags           []string        `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// MatchResult pairs a Recipe with its computed hardware match score.
type MatchResult struct {
	Recipe Recipe `json:"recipe"`
	Score  int    `json:"score"`
}

// RecipeFilter holds filtering parameters for listing recipes.
type RecipeFilter struct {
	Tags         []string
	GPUVendor    string
	VerifiedOnly bool
	Limit        int
	Offset       int
}
