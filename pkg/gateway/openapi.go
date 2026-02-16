package gateway

import (
	"encoding/json"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

const (
	OpenAPIVersion = "3.0.0"
	APIVersion     = "2.0.0"
	APITitle       = "AIMA API"
	APIDescription = "AI-Managed AI Inference Infrastructure API"
)

type OpenAPISpec struct {
	OpenAPI    string                            `json:"openapi"`
	Info       OpenAPIInfo                       `json:"info"`
	Paths      map[string]map[string]OpenAPIPath `json:"paths"`
	Components *OpenAPIComponents                `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type OpenAPIPath struct {
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter         `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
}

type OpenAPIParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"`
	Required    bool          `json:"required"`
	Description string        `json:"description,omitempty"`
	Schema      OpenAPISchema `json:"schema"`
}

type OpenAPIRequestBody struct {
	Required bool                        `json:"required"`
	Content  map[string]OpenAPIMediaType `json:"content"`
}

type OpenAPIMediaType struct {
	Schema OpenAPISchema `json:"schema"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

type OpenAPIComponents struct {
	Schemas map[string]OpenAPISchema `json:"schemas,omitempty"`
}

type OpenAPISchema struct {
	Type        string                   `json:"type,omitempty"`
	Properties  map[string]OpenAPISchema `json:"properties,omitempty"`
	Items       *OpenAPISchema           `json:"items,omitempty"`
	Required    []string                 `json:"required,omitempty"`
	Description string                   `json:"description,omitempty"`
	Enum        []any                    `json:"enum,omitempty"`
	Default     any                      `json:"default,omitempty"`
	Example     any                      `json:"example,omitempty"`
	Ref         string                   `json:"$ref,omitempty"`
}

func GenerateOpenAPI(registry *unit.Registry) []byte {
	spec := &OpenAPISpec{
		OpenAPI: OpenAPIVersion,
		Info: OpenAPIInfo{
			Title:       APITitle,
			Description: APIDescription,
			Version:     APIVersion,
		},
		Paths: make(map[string]map[string]OpenAPIPath),
		Components: &OpenAPIComponents{
			Schemas: make(map[string]OpenAPISchema),
		},
	}

	spec.addExecuteEndpoint()
	spec.addUnitEndpoints(registry)
	spec.addHealthEndpoint()
	spec.addCommonSchemas()

	data, _ := json.MarshalIndent(spec, "", "  ")
	return data
}

func (s *OpenAPISpec) addExecuteEndpoint() {
	s.Paths["/api/v2/execute"] = map[string]OpenAPIPath{
		"post": {
			Summary:     "Execute atomic unit",
			Description: "Unified endpoint to execute commands, queries, or access resources",
			OperationID: "execute",
			Tags:        []string{"execute"},
			RequestBody: &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: OpenAPISchema{
							Ref: "#/components/schemas/ExecuteRequest",
						},
					},
				},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successful response",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: OpenAPISchema{
								Ref: "#/components/schemas/ExecuteResponse",
							},
						},
					},
				},
				"400": {
					Description: "Invalid request",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: OpenAPISchema{
								Ref: "#/components/schemas/ErrorResponse",
							},
						},
					},
				},
				"404": {
					Description: "Unit not found",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: OpenAPISchema{
								Ref: "#/components/schemas/ErrorResponse",
							},
						},
					},
				},
				"500": {
					Description: "Internal error",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: OpenAPISchema{
								Ref: "#/components/schemas/ErrorResponse",
							},
						},
					},
				},
			},
		},
	}
}

func (s *OpenAPISpec) addUnitEndpoints(registry *unit.Registry) {
	domainPaths := make(map[string][]OpenAPIPath)

	for _, cmd := range registry.ListCommands() {
		path := unitToPath(cmd.Domain(), cmd.Name())
		if _, exists := domainPaths[path]; !exists {
			domainPaths[path] = []OpenAPIPath{}
		}

		method := httpMethodForCommand(cmd.Name())
		domainPaths[path] = append(domainPaths[path], OpenAPIPath{
			Summary:     cmd.Description(),
			Description: cmd.Description(),
			OperationID: sanitizeOperationID(cmd.Name()),
			Tags:        []string{cmd.Domain()},
			RequestBody: schemaToRequestBody(cmd.InputSchema()),
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successful response",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: schemaToOpenAPI(cmd.OutputSchema()),
						},
					},
				},
			},
		})
		_ = method
	}

	for _, q := range registry.ListQueries() {
		path := unitToPath(q.Domain(), q.Name())
		if _, exists := domainPaths[path]; !exists {
			domainPaths[path] = []OpenAPIPath{}
		}

		domainPaths[path] = append(domainPaths[path], OpenAPIPath{
			Summary:     q.Description(),
			Description: q.Description(),
			OperationID: sanitizeOperationID(q.Name()),
			Tags:        []string{q.Domain()},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Successful response",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: schemaToOpenAPI(q.OutputSchema()),
						},
					},
				},
			},
		})
	}

	for path, methods := range domainPaths {
		s.Paths[path] = make(map[string]OpenAPIPath)
		for i, p := range methods {
			method := "post"
			if len(methods) > 1 && i == 0 {
				method = "get"
			}
			s.Paths[path][method] = p
		}
	}
}

func (s *OpenAPISpec) addHealthEndpoint() {
	s.Paths["/health"] = map[string]OpenAPIPath{
		"get": {
			Summary:     "Health check",
			Description: "Returns server health status",
			OperationID: "health",
			Tags:        []string{"system"},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Server is healthy",
					Content: map[string]OpenAPIMediaType{
						"application/json": {
							Schema: OpenAPISchema{
								Type: "object",
								Properties: map[string]OpenAPISchema{
									"status": {Type: "string", Example: "healthy"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *OpenAPISpec) addCommonSchemas() {
	s.Components.Schemas["ExecuteRequest"] = OpenAPISchema{
		Type:     "object",
		Required: []string{"type", "unit"},
		Properties: map[string]OpenAPISchema{
			"type": {
				Type:        "string",
				Description: "Type of operation",
				Enum:        []any{"command", "query", "resource", "workflow"},
			},
			"unit": {
				Type:        "string",
				Description: "Atomic unit name (e.g., model.pull, inference.chat)",
			},
			"input": {
				Type:        "object",
				Description: "Input parameters for the unit",
			},
			"options": {
				Ref: "#/components/schemas/RequestOptions",
			},
		},
	}

	s.Components.Schemas["RequestOptions"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"timeout": {
				Type:        "integer",
				Description: "Timeout in milliseconds",
			},
			"async": {
				Type:        "boolean",
				Description: "Execute asynchronously",
			},
			"trace_id": {
				Type:        "string",
				Description: "Trace ID for distributed tracing",
			},
		},
	}

	s.Components.Schemas["ExecuteResponse"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"success": {
				Type: "boolean",
			},
			"data": {
				Description: "Response data",
			},
			"error": {
				Ref: "#/components/schemas/ErrorInfo",
			},
			"meta": {
				Ref: "#/components/schemas/ResponseMeta",
			},
		},
	}

	s.Components.Schemas["ErrorInfo"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"code": {
				Type:        "string",
				Description: "Error code",
			},
			"message": {
				Type:        "string",
				Description: "Error message",
			},
			"details": {
				Description: "Additional error details",
			},
		},
	}

	s.Components.Schemas["ResponseMeta"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"request_id": {
				Type: "string",
			},
			"duration_ms": {
				Type: "integer",
			},
			"trace_id": {
				Type: "string",
			},
			"pagination": {
				Ref: "#/components/schemas/Pagination",
			},
		},
	}

	s.Components.Schemas["Pagination"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"page": {
				Type: "integer",
			},
			"per_page": {
				Type: "integer",
			},
			"total": {
				Type: "integer",
			},
		},
	}

	s.Components.Schemas["ErrorResponse"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"success": {
				Type:    "boolean",
				Example: false,
			},
			"error": {
				Ref: "#/components/schemas/ErrorInfo",
			},
			"meta": {
				Ref: "#/components/schemas/ResponseMeta",
			},
		},
	}
}

func unitToPath(domain, name string) string {
	action := name
	if idx := indexOf(name, "."); idx >= 0 {
		action = name[idx+1:]
	}
	return "/api/v2/" + domain + "/" + action
}

func httpMethodForCommand(name string) string {
	action := name
	if idx := indexOf(name, "."); idx >= 0 {
		action = name[idx+1:]
	}

	switch action {
	case "create", "pull", "import", "install", "start", "stop", "restart", "delete", "allocate", "release", "run", "chat", "complete", "embed":
		return "post"
	default:
		return "get"
	}
}

func sanitizeOperationID(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '.' {
			result = append(result, '_')
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}

func schemaToRequestBody(s unit.Schema) *OpenAPIRequestBody {
	return &OpenAPIRequestBody{
		Required: true,
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: schemaToOpenAPI(s),
			},
		},
	}
}

func schemaToOpenAPI(s unit.Schema) OpenAPISchema {
	result := OpenAPISchema{
		Type:        s.Type,
		Description: s.Description,
	}

	if len(s.Properties) > 0 {
		result.Properties = make(map[string]OpenAPISchema)
		for name, field := range s.Properties {
			result.Properties[name] = OpenAPISchema{
				Type:        field.Type,
				Description: field.Description,
			}
		}
	}

	if s.Items != nil {
		result.Items = &OpenAPISchema{
			Type: s.Items.Type,
		}
	}

	if len(s.Required) > 0 {
		result.Required = s.Required
	}

	if len(s.Enum) > 0 {
		result.Enum = s.Enum
	}

	if s.Default != nil {
		result.Default = s.Default
	}

	return result
}

func indexOf(s string, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
