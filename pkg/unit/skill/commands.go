package skill

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// AddCommand implements skill.add.
type AddCommand struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewAddCommand(store SkillStore) *AddCommand {
	return &AddCommand{store: store}
}

func NewAddCommandWithEvents(store SkillStore, events unit.EventPublisher) *AddCommand {
	return &AddCommand{store: store, events: events}
}

func (c *AddCommand) Name() string        { return "skill.add" }
func (c *AddCommand) Domain() string      { return "skill" }
func (c *AddCommand) Description() string { return "Add a new skill" }

func (c *AddCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"content": {
				Name: "content",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Skill file content (YAML front-matter + Markdown body)",
				},
			},
			"source": {
				Name: "source",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Skill source",
					Enum:        []any{SourceUser, SourceCommunity},
					Default:     SourceUser,
				},
			},
		},
		Required: []string{"content"},
	}
}

func (c *AddCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skill_id": {Name: "skill_id", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (c *AddCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"content": "---\nid: my-skill\nname: My Skill\ncategory: manage\nenabled: true\nsource: user\n---\n\n# My Skill\n\nContent here.",
				"source":  "user",
			},
			Output:      map[string]any{"skill_id": "my-skill"},
			Description: "Add a user skill",
		},
	}
}

func (c *AddCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	content, _ := inputMap["content"].(string)
	if content == "" {
		ec.PublishFailed(ErrInvalidInput)
		return nil, ErrInvalidInput
	}

	sk, err := ParseSkillFile(content)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("parse skill: %w", err)
	}

	// Override source if provided; ensure not builtin
	if source, ok := inputMap["source"].(string); ok && source != "" {
		sk.Source = source
	}
	if sk.Source == "" {
		sk.Source = SourceUser
	}
	if sk.Source == SourceBuiltin {
		err := fmt.Errorf("cannot add skill with builtin source via API")
		ec.PublishFailed(err)
		return nil, err
	}

	// Generate ID if missing
	if sk.ID == "" {
		sk.ID = uuid.New().String()
	}

	if err := c.store.Add(ctx, sk); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("add skill: %w", err)
	}

	if c.events != nil {
		_ = c.events.Publish(NewAddedEvent(sk))
	}

	output := map[string]any{"skill_id": sk.ID}
	ec.PublishCompleted(output)
	return output, nil
}

// RemoveCommand implements skill.remove.
type RemoveCommand struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewRemoveCommand(store SkillStore) *RemoveCommand {
	return &RemoveCommand{store: store}
}

func NewRemoveCommandWithEvents(store SkillStore, events unit.EventPublisher) *RemoveCommand {
	return &RemoveCommand{store: store, events: events}
}

func (c *RemoveCommand) Name() string        { return "skill.remove" }
func (c *RemoveCommand) Domain() string      { return "skill" }
func (c *RemoveCommand) Description() string { return "Remove a user skill (builtin skills cannot be removed)" }

func (c *RemoveCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skill_id": {
				Name: "skill_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Skill identifier",
				},
			},
		},
		Required: []string{"skill_id"},
	}
}

func (c *RemoveCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {Name: "success", Schema: unit.Schema{Type: "boolean"}},
		},
	}
}

func (c *RemoveCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"skill_id": "my-skill"},
			Output:      map[string]any{"success": true},
			Description: "Remove a user skill",
		},
	}
}

func (c *RemoveCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	skillID, _ := inputMap["skill_id"].(string)
	if skillID == "" {
		ec.PublishFailed(ErrInvalidInput)
		return nil, ErrInvalidInput
	}

	if err := c.store.Remove(ctx, skillID); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("remove skill: %w", err)
	}

	if c.events != nil {
		_ = c.events.Publish(NewRemovedEvent(skillID))
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

// EnableCommand implements skill.enable.
type EnableCommand struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewEnableCommand(store SkillStore) *EnableCommand {
	return &EnableCommand{store: store}
}

func NewEnableCommandWithEvents(store SkillStore, events unit.EventPublisher) *EnableCommand {
	return &EnableCommand{store: store, events: events}
}

func (c *EnableCommand) Name() string        { return "skill.enable" }
func (c *EnableCommand) Domain() string      { return "skill" }
func (c *EnableCommand) Description() string { return "Enable a skill" }

func (c *EnableCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skill_id": {
				Name: "skill_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Skill identifier",
				},
			},
		},
		Required: []string{"skill_id"},
	}
}

func (c *EnableCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {Name: "success", Schema: unit.Schema{Type: "boolean"}},
		},
	}
}

func (c *EnableCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"skill_id": "setup-llm"},
			Output:      map[string]any{"success": true},
			Description: "Enable a skill",
		},
	}
}

func (c *EnableCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	skillID, _ := inputMap["skill_id"].(string)
	if skillID == "" {
		ec.PublishFailed(ErrInvalidInput)
		return nil, ErrInvalidInput
	}

	sk, err := c.store.Get(ctx, skillID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get skill: %w", err)
	}

	sk.Enabled = true
	if err := c.store.Update(ctx, sk); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("enable skill: %w", err)
	}

	if c.events != nil {
		_ = c.events.Publish(NewEnabledEvent(skillID))
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

// DisableCommand implements skill.disable.
type DisableCommand struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewDisableCommand(store SkillStore) *DisableCommand {
	return &DisableCommand{store: store}
}

func NewDisableCommandWithEvents(store SkillStore, events unit.EventPublisher) *DisableCommand {
	return &DisableCommand{store: store, events: events}
}

func (c *DisableCommand) Name() string        { return "skill.disable" }
func (c *DisableCommand) Domain() string      { return "skill" }
func (c *DisableCommand) Description() string { return "Disable a skill" }

func (c *DisableCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skill_id": {
				Name: "skill_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Skill identifier",
				},
			},
		},
		Required: []string{"skill_id"},
	}
}

func (c *DisableCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {Name: "success", Schema: unit.Schema{Type: "boolean"}},
		},
	}
}

func (c *DisableCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"skill_id": "setup-llm"},
			Output:      map[string]any{"success": true},
			Description: "Disable a skill",
		},
	}
}

func (c *DisableCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	skillID, _ := inputMap["skill_id"].(string)
	if skillID == "" {
		ec.PublishFailed(ErrInvalidInput)
		return nil, ErrInvalidInput
	}

	sk, err := c.store.Get(ctx, skillID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get skill: %w", err)
	}

	sk.Enabled = false
	if err := c.store.Update(ctx, sk); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("disable skill: %w", err)
	}

	if c.events != nil {
		_ = c.events.Publish(NewDisabledEvent(skillID))
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}
