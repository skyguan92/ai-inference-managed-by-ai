package catalog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecipeCreatedEvent(t *testing.T) {
	recipe := createTestRecipe("r1", "Test Recipe", "NVIDIA")
	event := NewRecipeCreatedEvent(recipe)

	assert.Equal(t, EventTypeRecipeCreated, event.Type())
	assert.Equal(t, "catalog", event.Domain())
	assert.NotEmpty(t, event.CorrelationID())
	assert.WithinDuration(t, time.Now(), event.Timestamp(), time.Second)

	payload, ok := event.Payload().(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "r1", payload["recipe_id"])
	assert.Equal(t, "Test Recipe", payload["name"])
}

func TestRecipeMatchedEvent(t *testing.T) {
	profile := HardwareProfile{GPUVendor: "NVIDIA", GPUModel: "RTX 4090"}
	event := NewRecipeMatchedEvent("r1", profile, 85)

	assert.Equal(t, EventTypeRecipeMatched, event.Type())
	assert.Equal(t, "catalog", event.Domain())
	assert.NotEmpty(t, event.CorrelationID())

	payload, ok := event.Payload().(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "r1", payload["recipe_id"])
	assert.Equal(t, 85, payload["score"])
}

func TestRecipeAppliedEvent(t *testing.T) {
	modelsReady := []ModelStatus{
		{Name: "Llama3", Ready: true},
		{Name: "Qwen2", Ready: false},
	}
	event := NewRecipeAppliedEvent("r1", true, modelsReady)

	assert.Equal(t, EventTypeRecipeApplied, event.Type())
	assert.Equal(t, "catalog", event.Domain())
	assert.NotEmpty(t, event.CorrelationID())

	payload, ok := event.Payload().(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "r1", payload["recipe_id"])
	assert.Equal(t, true, payload["engine_ready"])
	assert.Len(t, payload["models_ready"], 2)
}
