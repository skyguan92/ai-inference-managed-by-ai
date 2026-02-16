package app

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type GetQuery struct {
	store    AppStore
	provider AppProvider
}

func NewGetQuery(store AppStore, provider AppProvider) *GetQuery {
	return &GetQuery{store: store, provider: provider}
}

func (q *GetQuery) Name() string {
	return "app.get"
}

func (q *GetQuery) Domain() string {
	return "app"
}

func (q *GetQuery) Description() string {
	return "Get detailed information about an application"
}

func (q *GetQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"app_id": {
				Name: "app_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Application ID",
				},
			},
		},
		Required: []string{"app_id"},
	}
}

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"id":       {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":     {Name: "name", Schema: unit.Schema{Type: "string"}},
			"template": {Name: "template", Schema: unit.Schema{Type: "string"}},
			"status":   {Name: "status", Schema: unit.Schema{Type: "string"}},
			"ports":    {Name: "ports", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "number"}}},
			"volumes":  {Name: "volumes", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"metrics": {
				Name: "metrics",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"cpu_usage":    {Name: "cpu_usage", Schema: unit.Schema{Type: "number"}},
						"memory_usage": {Name: "memory_usage", Schema: unit.Schema{Type: "number"}},
						"uptime":       {Name: "uptime", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"app_id": "app-abc123"},
			Output:      map[string]any{"id": "app-abc123", "name": "open-webui", "template": "open-webui", "status": "running", "ports": []int{8080}},
			Description: "Get application details",
		},
	}
}

func (q *GetQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	appID, _ := inputMap["app_id"].(string)
	if appID == "" {
		return nil, ErrInvalidAppID
	}

	app, err := q.store.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	result := map[string]any{
		"id":       app.ID,
		"name":     app.Name,
		"template": app.Template,
		"status":   string(app.Status),
		"ports":    app.Ports,
		"volumes":  app.Volumes,
	}

	if q.provider != nil && app.Status == AppStatusRunning {
		metrics, err := q.provider.GetMetrics(ctx, appID)
		if err == nil {
			result["metrics"] = map[string]any{
				"cpu_usage":    metrics.CPUUsage,
				"memory_usage": metrics.MemoryUsage,
				"uptime":       metrics.Uptime,
			}
		}
	}

	return result, nil
}

type ListQuery struct {
	store AppStore
}

func NewListQuery(store AppStore) *ListQuery {
	return &ListQuery{store: store}
}

func (q *ListQuery) Name() string {
	return "app.list"
}

func (q *ListQuery) Domain() string {
	return "app"
}

func (q *ListQuery) Description() string {
	return "List all applications with optional filtering"
}

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by status",
					Enum:        []any{string(AppStatusInstalled), string(AppStatusRunning), string(AppStatusStopped), string(AppStatusError)},
				},
			},
			"template": {
				Name: "template",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by template",
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrFloat(1),
					Max:         ptrFloat(100),
				},
			},
			"offset": {
				Name: "offset",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Offset for pagination",
					Min:         ptrFloat(0),
				},
			},
		},
	}
}

func (q *ListQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"apps": {
				Name: "apps",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":       {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":     {Name: "name", Schema: unit.Schema{Type: "string"}},
							"template": {Name: "template", Schema: unit.Schema{Type: "string"}},
							"status":   {Name: "status", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (q *ListQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"apps": []map[string]any{{"id": "app-abc123", "name": "open-webui", "template": "open-webui", "status": "running"}}},
			Description: "List all applications",
		},
		{
			Input:       map[string]any{"status": "running"},
			Output:      map[string]any{"apps": []map[string]any{{"id": "app-abc123", "name": "open-webui", "template": "open-webui", "status": "running"}}},
			Description: "List running applications",
		},
	}
}

func (q *ListQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)

	filter := AppFilter{
		Limit:  100,
		Offset: 0,
	}

	if s, ok := inputMap["status"].(string); ok && s != "" {
		filter.Status = AppStatus(s)
	}
	if t, ok := inputMap["template"].(string); ok && t != "" {
		filter.Template = t
	}
	if limit, ok := toInt(inputMap["limit"]); ok && limit > 0 {
		filter.Limit = limit
	}
	if offset, ok := toInt(inputMap["offset"]); ok && offset >= 0 {
		filter.Offset = offset
	}

	apps, _, err := q.store.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	items := make([]map[string]any, len(apps))
	for i, a := range apps {
		items[i] = map[string]any{
			"id":       a.ID,
			"name":     a.Name,
			"template": a.Template,
			"status":   string(a.Status),
		}
	}

	return map[string]any{"apps": items}, nil
}

type LogsQuery struct {
	store    AppStore
	provider AppProvider
}

func NewLogsQuery(store AppStore, provider AppProvider) *LogsQuery {
	return &LogsQuery{store: store, provider: provider}
}

func (q *LogsQuery) Name() string {
	return "app.logs"
}

func (q *LogsQuery) Domain() string {
	return "app"
}

func (q *LogsQuery) Description() string {
	return "Get application logs"
}

func (q *LogsQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"app_id": {
				Name: "app_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Application ID",
				},
			},
			"tail": {
				Name: "tail",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Number of lines to return",
				},
			},
			"since": {
				Name: "since",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Unix timestamp to get logs since",
				},
			},
		},
		Required: []string{"app_id"},
	}
}

func (q *LogsQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"logs": {
				Name: "logs",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"timestamp": {Name: "timestamp", Schema: unit.Schema{Type: "number"}},
							"message":   {Name: "message", Schema: unit.Schema{Type: "string"}},
							"level":     {Name: "level", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (q *LogsQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"app_id": "app-abc123"},
			Output:      map[string]any{"logs": []map[string]any{{"timestamp": 1700000000, "message": "App started", "level": "info"}}},
			Description: "Get application logs",
		},
		{
			Input:       map[string]any{"app_id": "app-abc123", "tail": 100},
			Output:      map[string]any{"logs": []map[string]any{{"timestamp": 1700000000, "message": "App started", "level": "info"}}},
			Description: "Get last 100 log lines",
		},
	}
}

func (q *LogsQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil || q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	appID, _ := inputMap["app_id"].(string)
	if appID == "" {
		return nil, ErrInvalidAppID
	}

	_, err := q.store.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	tail := 100
	if t, ok := toInt(inputMap["tail"]); ok && t > 0 {
		tail = t
	}

	var since int64
	if s, ok := toInt64(inputMap["since"]); ok && s > 0 {
		since = s
	}

	logs, err := q.provider.GetLogs(ctx, appID, tail, since)
	if err != nil {
		return nil, fmt.Errorf("get logs for app %s: %w", appID, err)
	}

	items := make([]map[string]any, len(logs))
	for i, l := range logs {
		items[i] = map[string]any{
			"timestamp": l.Timestamp,
			"message":   l.Message,
			"level":     l.Level,
		}
	}

	return map[string]any{"logs": items}, nil
}

type TemplatesQuery struct {
	provider AppProvider
}

func NewTemplatesQuery(provider AppProvider) *TemplatesQuery {
	return &TemplatesQuery{provider: provider}
}

func (q *TemplatesQuery) Name() string {
	return "app.templates"
}

func (q *TemplatesQuery) Domain() string {
	return "app"
}

func (q *TemplatesQuery) Description() string {
	return "List available application templates"
}

func (q *TemplatesQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"category": {
				Name: "category",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by category",
					Enum:        []any{string(AppCategoryAIChat), string(AppCategoryDevTool), string(AppCategoryMonitoring), string(AppCategoryCustom)},
				},
			},
		},
	}
}

func (q *TemplatesQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"templates": {
				Name: "templates",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":        {Name: "name", Schema: unit.Schema{Type: "string"}},
							"category":    {Name: "category", Schema: unit.Schema{Type: "string"}},
							"description": {Name: "description", Schema: unit.Schema{Type: "string"}},
							"image":       {Name: "image", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (q *TemplatesQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"templates": []map[string]any{{"id": "open-webui", "name": "Open WebUI", "category": "ai-chat", "description": "AI Chat Interface", "image": "ghcr.io/open-webui/open-webui:main"}}},
			Description: "List all templates",
		},
		{
			Input:       map[string]any{"category": "ai-chat"},
			Output:      map[string]any{"templates": []map[string]any{{"id": "open-webui", "name": "Open WebUI", "category": "ai-chat", "description": "AI Chat Interface", "image": "ghcr.io/open-webui/open-webui:main"}}},
			Description: "List AI chat templates",
		},
	}
}

func (q *TemplatesQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)

	var category AppCategory
	if c, ok := inputMap["category"].(string); ok && c != "" {
		category = AppCategory(c)
	}

	templates, err := q.provider.GetTemplates(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("get templates: %w", err)
	}

	items := make([]map[string]any, len(templates))
	for i, t := range templates {
		items[i] = map[string]any{
			"id":          t.ID,
			"name":        t.Name,
			"category":    string(t.Category),
			"description": t.Description,
			"image":       t.Image,
		}
	}

	return map[string]any{"templates": items}, nil
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		return int64(val), true
	case float32:
		return int64(val), true
	default:
		return 0, false
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}
