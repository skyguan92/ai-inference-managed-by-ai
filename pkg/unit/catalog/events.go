package catalog

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeRecipeCreated = "catalog.recipe_created"
	EventTypeRecipeMatched = "catalog.recipe_matched"
	EventTypeRecipeApplied = "catalog.recipe_applied"
)

// RecipeCreatedEvent is emitted when a new recipe is created.
type RecipeCreatedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRecipeCreatedEvent(recipe *Recipe) *RecipeCreatedEvent {
	return &RecipeCreatedEvent{
		eventType: EventTypeRecipeCreated,
		domain:    "catalog",
		payload: map[string]any{
			"recipe_id": recipe.ID,
			"name":      recipe.Name,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RecipeCreatedEvent) Type() string          { return e.eventType }
func (e *RecipeCreatedEvent) Domain() string        { return e.domain }
func (e *RecipeCreatedEvent) Payload() any          { return e.payload }
func (e *RecipeCreatedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RecipeCreatedEvent) CorrelationID() string { return e.correlationID }

// RecipeMatchedEvent is emitted when a recipe is matched to hardware.
type RecipeMatchedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRecipeMatchedEvent(recipeID string, profile HardwareProfile, score int) *RecipeMatchedEvent {
	return &RecipeMatchedEvent{
		eventType: EventTypeRecipeMatched,
		domain:    "catalog",
		payload: map[string]any{
			"recipe_id":        recipeID,
			"hardware_profile": profile,
			"score":            score,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RecipeMatchedEvent) Type() string          { return e.eventType }
func (e *RecipeMatchedEvent) Domain() string        { return e.domain }
func (e *RecipeMatchedEvent) Payload() any          { return e.payload }
func (e *RecipeMatchedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RecipeMatchedEvent) CorrelationID() string { return e.correlationID }

// RecipeAppliedEvent is emitted when a recipe has been deployed.
type RecipeAppliedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewRecipeAppliedEvent(recipeID string, engineReady bool, modelsReady []ModelStatus) *RecipeAppliedEvent {
	return &RecipeAppliedEvent{
		eventType: EventTypeRecipeApplied,
		domain:    "catalog",
		payload: map[string]any{
			"recipe_id":    recipeID,
			"engine_ready": engineReady,
			"models_ready": modelsReady,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *RecipeAppliedEvent) Type() string          { return e.eventType }
func (e *RecipeAppliedEvent) Domain() string        { return e.domain }
func (e *RecipeAppliedEvent) Payload() any          { return e.payload }
func (e *RecipeAppliedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *RecipeAppliedEvent) CorrelationID() string { return e.correlationID }
