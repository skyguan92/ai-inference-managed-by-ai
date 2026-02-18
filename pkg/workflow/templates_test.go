package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPredefinedTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantErr   bool
		errMsg    string
		checkFunc func(t *testing.T, def *WorkflowDef)
	}{
		{
			name:     "load voice_assistant template",
			template: "voice_assistant",
			wantErr:  false,
			checkFunc: func(t *testing.T, def *WorkflowDef) {
				assert.Equal(t, "voice_assistant", def.Name)
				assert.NotEmpty(t, def.Steps)
				assert.Equal(t, "transcribe", def.Steps[0].ID)
			},
		},
		{
			name:     "load rag template",
			template: "rag",
			wantErr:  false,
			checkFunc: func(t *testing.T, def *WorkflowDef) {
				assert.Equal(t, "rag", def.Name)
				assert.Len(t, def.Steps, 3)
				assert.Equal(t, "embed", def.Steps[0].ID)
				assert.Equal(t, "search", def.Steps[1].ID)
				assert.Equal(t, "chat", def.Steps[2].ID)
				assert.NotNil(t, def.Output["answer"])
				assert.NotNil(t, def.Output["sources"])
			},
		},
		{
			name:     "load batch_inference template",
			template: "batch_inference",
			wantErr:  false,
			checkFunc: func(t *testing.T, def *WorkflowDef) {
				assert.Equal(t, "batch_inference", def.Name)
				assert.Len(t, def.Steps, 2)
				assert.Equal(t, "load_data", def.Steps[0].ID)
				assert.Equal(t, "batch_process", def.Steps[1].ID)
				assert.NotNil(t, def.Output["results"])
			},
		},
		{
			name:     "load multimodal_chat template",
			template: "multimodal_chat",
			wantErr:  false,
			checkFunc: func(t *testing.T, def *WorkflowDef) {
				assert.Equal(t, "multimodal_chat", def.Name)
				assert.Len(t, def.Steps, 2)
				assert.Equal(t, "analyze_image", def.Steps[0].ID)
				assert.Equal(t, "chat", def.Steps[1].ID)
				assert.NotNil(t, def.Output["response"])
			},
		},
		{
			name:     "load video_analysis template",
			template: "video_analysis",
			wantErr:  false,
			checkFunc: func(t *testing.T, def *WorkflowDef) {
				assert.Equal(t, "video_analysis", def.Name)
				assert.Len(t, def.Steps, 3)
				assert.Equal(t, "extract_frames", def.Steps[0].ID)
				assert.Equal(t, "analyze_frames", def.Steps[1].ID)
				assert.Equal(t, "summarize", def.Steps[2].ID)
				assert.NotNil(t, def.Output["summary"])
			},
		},
		{
			name:     "load rag_pipeline template",
			template: "rag_pipeline",
			wantErr:  false,
			checkFunc: func(t *testing.T, def *WorkflowDef) {
				assert.Equal(t, "rag_pipeline", def.Name)
				assert.Len(t, def.Steps, 4)
				assert.NotNil(t, def.Output["answer"])
				assert.NotNil(t, def.Output["sources"])
			},
		},
		{
			name:     "load non-existent template",
			template: "non_existent",
			wantErr:  true,
			errMsg:   "template not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := LoadPredefinedTemplate(tt.template)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, def)
			} else {
				require.NoError(t, err)
				require.NotNil(t, def)
				if tt.checkFunc != nil {
					tt.checkFunc(t, def)
				}
			}
		})
	}
}

func TestListPredefinedTemplates(t *testing.T) {
	templates := ListPredefinedTemplates()

	assert.NotEmpty(t, templates)
	assert.Contains(t, templates, "voice_assistant")
	assert.Contains(t, templates, "rag")
	assert.Contains(t, templates, "batch_inference")
	assert.Contains(t, templates, "multimodal_chat")
	assert.Contains(t, templates, "video_analysis")
	assert.Contains(t, templates, "rag_pipeline")
}

func TestIsPredefinedTemplate(t *testing.T) {
	assert.True(t, IsPredefinedTemplate("voice_assistant"))
	assert.True(t, IsPredefinedTemplate("rag"))
	assert.True(t, IsPredefinedTemplate("batch_inference"))
	assert.True(t, IsPredefinedTemplate("multimodal_chat"))
	assert.True(t, IsPredefinedTemplate("video_analysis"))
	assert.True(t, IsPredefinedTemplate("rag_pipeline"))

	assert.False(t, IsPredefinedTemplate("non_existent"))
	assert.False(t, IsPredefinedTemplate(""))
}

func TestAllPredefinedTemplatesValid(t *testing.T) {
	templates := ListPredefinedTemplates()

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			def, err := LoadPredefinedTemplate(name)
			require.NoError(t, err, "template %s should be valid", name)
			require.NotNil(t, def)
			require.NotEmpty(t, def.Name, "template %s should have a name", name)
			require.NotEmpty(t, def.Steps, "template %s should have steps", name)

			for _, step := range def.Steps {
				require.NotEmpty(t, step.ID, "step in template %s should have an ID", name)
				require.NotEmpty(t, step.Type, "step %s in template %s should have a type", step.ID, name)
			}
		})
	}
}
