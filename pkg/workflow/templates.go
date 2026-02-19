package workflow

import (
	"embed"
	"fmt"
)

//go:embed templates/*.yaml
var templateFS embed.FS

var predefinedTemplates = map[string]string{
	"voice_assistant": "templates/voice_assistant.yaml",
	"rag":             "templates/rag.yaml",
	"batch_inference": "templates/batch_inference.yaml",
	"multimodal_chat": "templates/multimodal_chat.yaml",
	"video_analysis":  "templates/video_analysis.yaml",
	"rag_pipeline":    "templates/rag_pipeline.yaml",
}

func LoadPredefinedTemplate(name string) (*WorkflowDef, error) {
	path, exists := predefinedTemplates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	data, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", name, err)
	}

	return ParseYAML(data)
}

func ListPredefinedTemplates() []string {
	names := make([]string, 0, len(predefinedTemplates))
	for name := range predefinedTemplates {
		names = append(names, name)
	}
	return names
}

func IsPredefinedTemplate(name string) bool {
	_, exists := predefinedTemplates[name]
	return exists
}
