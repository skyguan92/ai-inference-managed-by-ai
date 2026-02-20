package skill

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddedEvent(t *testing.T) {
	sk := &Skill{ID: "test-id", Name: "Test Skill", Source: SourceUser}
	e := NewAddedEvent(sk)

	assert.Equal(t, EventTypeAdded, e.Type())
	assert.Equal(t, "skill", e.Domain())
	assert.NotZero(t, e.Timestamp())
	assert.NotEmpty(t, e.CorrelationID())

	payload := e.Payload().(map[string]any)
	assert.Equal(t, "test-id", payload["skill_id"])
	assert.Equal(t, "Test Skill", payload["name"])
	assert.Equal(t, SourceUser, payload["source"])
}

func TestRemovedEvent(t *testing.T) {
	e := NewRemovedEvent("skill-42")

	assert.Equal(t, EventTypeRemoved, e.Type())
	assert.Equal(t, "skill", e.Domain())
	assert.NotZero(t, e.Timestamp())
	assert.NotEmpty(t, e.CorrelationID())

	payload := e.Payload().(map[string]any)
	assert.Equal(t, "skill-42", payload["skill_id"])
}

func TestEnabledEvent(t *testing.T) {
	e := NewEnabledEvent("skill-42")

	assert.Equal(t, EventTypeEnabled, e.Type())
	assert.Equal(t, "skill", e.Domain())
	assert.NotEmpty(t, e.CorrelationID())

	payload := e.Payload().(map[string]any)
	assert.Equal(t, "skill-42", payload["skill_id"])
}

func TestDisabledEvent(t *testing.T) {
	e := NewDisabledEvent("skill-42")

	assert.Equal(t, EventTypeDisabled, e.Type())
	assert.Equal(t, "skill", e.Domain())
	assert.NotEmpty(t, e.CorrelationID())

	payload := e.Payload().(map[string]any)
	assert.Equal(t, "skill-42", payload["skill_id"])
}
