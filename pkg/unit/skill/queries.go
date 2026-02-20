package skill

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

// ListQuery implements skill.list.
type ListQuery struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewListQuery(store SkillStore) *ListQuery {
	return &ListQuery{store: store}
}

func NewListQueryWithEvents(store SkillStore, events unit.EventPublisher) *ListQuery {
	return &ListQuery{store: store, events: events}
}

func (q *ListQuery) Name() string        { return "skill.list" }
func (q *ListQuery) Domain() string      { return "skill" }
func (q *ListQuery) Description() string { return "List skills" }

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"category": {
				Name: "category",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by category",
					Enum:        []any{CategorySetup, CategoryTroubleshoot, CategoryOptimize, CategoryManage},
				},
			},
			"source": {
				Name: "source",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by source",
					Enum:        []any{SourceBuiltin, SourceUser, SourceCommunity},
				},
			},
			"enabled_only": {
				Name: "enabled_only",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Only return enabled skills",
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(1000),
				},
			},
			"offset": {
				Name: "offset",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Offset for pagination",
					Min:         ptrs.Float64(0),
				},
			},
		},
	}
}

func (q *ListQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skills": {
				Name: "skills",
				Schema: unit.Schema{
					Type:  "array",
					Items: skillItemSchema(),
				},
			},
			"total": {Name: "total", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (q *ListQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"category": "setup", "enabled_only": true},
			Output:      map[string]any{"skills": []map[string]any{{"id": "setup-llm", "name": "Deploy LLM", "category": "setup"}}, "total": 1},
			Description: "List enabled setup skills",
		},
	}
}

func (q *ListQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	inputMap, _ := input.(map[string]any)

	filter := SkillFilter{
		Category:    getString(inputMap, "category"),
		Source:      getString(inputMap, "source"),
		EnabledOnly: getBool(inputMap, "enabled_only"),
		Limit:       getInt(inputMap, "limit"),
		Offset:      getInt(inputMap, "offset"),
	}

	skills, total, err := q.store.List(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list skills: %w", err)
	}

	output := map[string]any{
		"skills": skillsToMaps(skills),
		"total":  total,
	}
	ec.PublishCompleted(output)
	return output, nil
}

// GetQuery implements skill.get.
type GetQuery struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewGetQuery(store SkillStore) *GetQuery {
	return &GetQuery{store: store}
}

func NewGetQueryWithEvents(store SkillStore, events unit.EventPublisher) *GetQuery {
	return &GetQuery{store: store, events: events}
}

func (q *GetQuery) Name() string        { return "skill.get" }
func (q *GetQuery) Domain() string      { return "skill" }
func (q *GetQuery) Description() string { return "Get skill details" }

func (q *GetQuery) InputSchema() unit.Schema {
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

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skill": {
				Name:   "skill",
				Schema: *skillItemSchema(),
			},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"skill_id": "setup-llm"},
			Output:      map[string]any{"skill": map[string]any{"id": "setup-llm", "name": "Deploy LLM", "category": "setup"}},
			Description: "Get a specific skill",
		},
	}
}

func (q *GetQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
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

	sk, err := q.store.Get(ctx, skillID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get skill: %w", err)
	}

	output := map[string]any{"skill": skillToMap(sk)}
	ec.PublishCompleted(output)
	return output, nil
}

// SearchQuery implements skill.search.
type SearchQuery struct {
	store  SkillStore
	events unit.EventPublisher
}

func NewSearchQuery(store SkillStore) *SearchQuery {
	return &SearchQuery{store: store}
}

func NewSearchQueryWithEvents(store SkillStore, events unit.EventPublisher) *SearchQuery {
	return &SearchQuery{store: store, events: events}
}

func (q *SearchQuery) Name() string        { return "skill.search" }
func (q *SearchQuery) Domain() string      { return "skill" }
func (q *SearchQuery) Description() string { return "Search skills by text query" }

func (q *SearchQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"query": {
				Name: "query",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Search query (matches name, description, content)",
				},
			},
			"category": {
				Name: "category",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by category",
					Enum:        []any{CategorySetup, CategoryTroubleshoot, CategoryOptimize, CategoryManage},
				},
			},
		},
		Required: []string{"query"},
	}
}

func (q *SearchQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"skills": {
				Name: "skills",
				Schema: unit.Schema{
					Type:  "array",
					Items: skillItemSchema(),
				},
			},
		},
	}
}

func (q *SearchQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"query": "gpu", "category": "troubleshoot"},
			Output:      map[string]any{"skills": []map[string]any{{"id": "troubleshoot-gpu", "name": "GPU Troubleshooting"}}},
			Description: "Search for GPU-related troubleshooting skills",
		},
	}
}

func (q *SearchQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	query, _ := inputMap["query"].(string)
	if query == "" {
		ec.PublishFailed(ErrInvalidInput)
		return nil, ErrInvalidInput
	}

	category := getString(inputMap, "category")

	skills, err := q.store.Search(ctx, query, category)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("search skills: %w", err)
	}

	output := map[string]any{"skills": skillsToMaps(skills)}
	ec.PublishCompleted(output)
	return output, nil
}

// helpers

func skillItemSchema() *unit.Schema {
	return &unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":        {Name: "name", Schema: unit.Schema{Type: "string"}},
			"category":    {Name: "category", Schema: unit.Schema{Type: "string"}},
			"description": {Name: "description", Schema: unit.Schema{Type: "string"}},
			"priority":    {Name: "priority", Schema: unit.Schema{Type: "number"}},
			"enabled":     {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
			"source":      {Name: "source", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func skillToMap(sk *Skill) map[string]any {
	return map[string]any{
		"id":          sk.ID,
		"name":        sk.Name,
		"category":    sk.Category,
		"description": sk.Description,
		"trigger":     sk.Trigger,
		"content":     sk.Content,
		"priority":    sk.Priority,
		"enabled":     sk.Enabled,
		"source":      sk.Source,
	}
}

func skillsToMaps(skills []Skill) []map[string]any {
	result := make([]map[string]any, len(skills))
	for i := range skills {
		result[i] = skillToMap(&skills[i])
	}
	return result
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
