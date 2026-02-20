package catalog

// createTestRecipe builds a minimal Recipe for use in tests.
func createTestRecipe(id, name, gpuVendor string) *Recipe {
	return &Recipe{
		ID:          id,
		Name:        name,
		Description: "Test recipe",
		Version:     "1.0.0",
		Profile: HardwareProfile{
			GPUVendor: gpuVendor,
			GPUModel:  "Test GPU",
			VRAMMinGB: 16,
			OS:        "linux",
		},
		Engine: RecipeEngine{
			Type:  "ollama",
			Image: "ollama/ollama:latest",
		},
		Models: []RecipeModel{
			{Name: "test-model", Source: "ollama", Repo: "llama3", Type: "llm"},
		},
		Verified: true,
		Tags:     []string{"test"},
	}
}
