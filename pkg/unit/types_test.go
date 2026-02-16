package unit

import (
	"context"
	"testing"
	"time"
)

func TestSchemaFields(t *testing.T) {
	s := Schema{
		Type:        "object",
		Title:       "Test Schema",
		Description: "A test schema",
		Required:    []string{"id", "name"},
		Optional:    []string{"description"},
		Properties: map[string]Field{
			"id": {
				Name:   "id",
				Schema: Schema{Type: "string"},
			},
			"name": {
				Name:   "name",
				Schema: Schema{Type: "string", MinLength: ptr(1), MaxLength: ptr(100)},
			},
		},
	}

	if s.Type != "object" {
		t.Errorf("expected Type to be 'object', got %s", s.Type)
	}
	if len(s.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(s.Required))
	}
	if s.Properties["id"].Type != "string" {
		t.Errorf("expected id property type to be 'string', got %s", s.Properties["id"].Type)
	}
}

func TestSchemaWithValidation(t *testing.T) {
	s := Schema{
		Type:     "number",
		Min:      ptr(0.0),
		Max:      ptr(100.0),
		Enum:     []any{"small", "medium", "large"},
		Default:  "medium",
		Examples: []any{"small"},
	}

	if *s.Min != 0.0 {
		t.Errorf("expected Min to be 0.0, got %f", *s.Min)
	}
	if *s.Max != 100.0 {
		t.Errorf("expected Max to be 100.0, got %f", *s.Max)
	}
	if len(s.Enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(s.Enum))
	}
}

func TestSchemaArrayWithItems(t *testing.T) {
	s := Schema{
		Type: "array",
		Items: &Schema{
			Type: "string",
		},
	}

	if s.Type != "array" {
		t.Errorf("expected Type to be 'array', got %s", s.Type)
	}
	if s.Items == nil {
		t.Error("expected Items to be non-nil")
	}
	if s.Items.Type != "string" {
		t.Errorf("expected Items.Type to be 'string', got %s", s.Items.Type)
	}
}

func TestExample(t *testing.T) {
	ex := Example{
		Input:       map[string]any{"name": "test"},
		Output:      map[string]any{"id": "123"},
		Description: "Create a test item",
	}

	if ex.Input.(map[string]any)["name"] != "test" {
		t.Error("expected Input name to be 'test'")
	}
	if ex.Output.(map[string]any)["id"] != "123" {
		t.Error("expected Output id to be '123'")
	}
}

func TestResourceUpdate(t *testing.T) {
	now := time.Now()
	update := ResourceUpdate{
		URI:       "asms://model/123",
		Timestamp: now,
		Operation: "created",
		Data:      map[string]any{"name": "llama3"},
	}

	if update.URI != "asms://model/123" {
		t.Errorf("expected URI 'asms://model/123', got %s", update.URI)
	}
	if update.Operation != "created" {
		t.Errorf("expected Operation 'created', got %s", update.Operation)
	}
	if !update.Timestamp.Equal(now) {
		t.Error("expected Timestamp to match")
	}
}

func TestResourceUpdateWithError(t *testing.T) {
	testErr := assertError("test error")
	update := ResourceUpdate{
		URI:       "asms://model/123",
		Operation: "error",
		Error:     testErr,
	}

	if update.Error == nil {
		t.Error("expected Error to be non-nil")
	}
	if update.Error.Error() != "test error" {
		t.Errorf("expected error message 'test error', got %s", update.Error.Error())
	}
}

func TestCommandInterface(t *testing.T) {
	cmd := &mockCommand{
		name:        "model.pull",
		domain:      "model",
		description: "Pull a model from source",
	}

	_ = Command(cmd)

	if cmd.Name() != "model.pull" {
		t.Errorf("expected Name 'model.pull', got %s", cmd.Name())
	}
	if cmd.Domain() != "model" {
		t.Errorf("expected Domain 'model', got %s", cmd.Domain())
	}
	if cmd.Description() != "Pull a model from source" {
		t.Errorf("expected Description 'Pull a model from source', got %s", cmd.Description())
	}
}

func TestQueryInterface(t *testing.T) {
	q := &mockQuery{
		name:        "model.list",
		domain:      "model",
		description: "List all models",
	}

	_ = Query(q)

	if q.Name() != "model.list" {
		t.Errorf("expected Name 'model.list', got %s", q.Name())
	}
	if q.Domain() != "model" {
		t.Errorf("expected Domain 'model', got %s", q.Domain())
	}
}

func TestEventInterface(t *testing.T) {
	now := time.Now()
	evt := &mockEvent{
		eventType:     "model.created",
		domain:        "model",
		payload:       map[string]any{"model_id": "123"},
		timestamp:     now,
		correlationID: "corr-123",
	}

	_ = Event(evt)

	if evt.Type() != "model.created" {
		t.Errorf("expected Type 'model.created', got %s", evt.Type())
	}
	if evt.Domain() != "model" {
		t.Errorf("expected Domain 'model', got %s", evt.Domain())
	}
	if evt.CorrelationID() != "corr-123" {
		t.Errorf("expected CorrelationID 'corr-123', got %s", evt.CorrelationID())
	}
	if !evt.Timestamp().Equal(now) {
		t.Error("expected Timestamp to match")
	}
}

func TestResourceInterface(t *testing.T) {
	res := &mockResource{
		uri:    "asms://model/123",
		domain: "model",
	}

	_ = Resource(res)

	if res.URI() != "asms://model/123" {
		t.Errorf("expected URI 'asms://model/123', got %s", res.URI())
	}
	if res.Domain() != "model" {
		t.Errorf("expected Domain 'model', got %s", res.Domain())
	}
}

func TestCommandExecute(t *testing.T) {
	cmd := &mockCommand{
		name:    "test.command",
		domain:  "test",
		execute: func(ctx context.Context, input any) (any, error) { return "result", nil },
	}

	ctx := context.Background()
	result, err := cmd.Execute(ctx, map[string]any{"key": "value"})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "result" {
		t.Errorf("expected result 'result', got %v", result)
	}
}

func TestQueryExecute(t *testing.T) {
	q := &mockQuery{
		name:    "test.query",
		domain:  "test",
		execute: func(ctx context.Context, input any) (any, error) { return []string{"a", "b"}, nil },
	}

	ctx := context.Background()
	result, err := q.Execute(ctx, nil)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(result.([]string)) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.([]string)))
	}
}

func TestResourceGet(t *testing.T) {
	res := &mockResource{
		uri:    "asms://test/123",
		domain: "test",
		get: func(ctx context.Context) (any, error) {
			return map[string]any{"id": "123", "status": "ready"}, nil
		},
	}

	ctx := context.Background()
	data, err := res.Get(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	result := data.(map[string]any)
	if result["status"] != "ready" {
		t.Errorf("expected status 'ready', got %v", result["status"])
	}
}

func TestResourceWatch(t *testing.T) {
	ch := make(chan ResourceUpdate, 1)
	res := &mockResource{
		uri:    "asms://test/123",
		domain: "test",
		watch: func(ctx context.Context) (<-chan ResourceUpdate, error) {
			ch <- ResourceUpdate{URI: "asms://test/123", Operation: "update"}
			return ch, nil
		},
	}

	ctx := context.Background()
	updates, err := res.Watch(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	update := <-updates
	if update.Operation != "update" {
		t.Errorf("expected Operation 'update', got %s", update.Operation)
	}
}

func TestEventPayload(t *testing.T) {
	payload := map[string]any{
		"model_id": "llama3",
		"status":   "ready",
	}
	evt := &mockEvent{
		eventType: "model.ready",
		domain:    "model",
		payload:   payload,
	}

	result := evt.Payload().(map[string]any)
	if result["model_id"] != "llama3" {
		t.Errorf("expected model_id 'llama3', got %v", result["model_id"])
	}
}

func TestCommandSchemas(t *testing.T) {
	inputSchema := Schema{Type: "object", Required: []string{"source", "repo"}}
	outputSchema := Schema{Type: "object", Required: []string{"model_id"}}

	cmd := &mockCommand{
		name:         "model.pull",
		domain:       "model",
		inputSchema:  inputSchema,
		outputSchema: outputSchema,
	}

	if cmd.InputSchema().Type != "object" {
		t.Errorf("expected InputSchema Type 'object', got %s", cmd.InputSchema().Type)
	}
	if cmd.OutputSchema().Type != "object" {
		t.Errorf("expected OutputSchema Type 'object', got %s", cmd.OutputSchema().Type)
	}
}

func TestQuerySchemas(t *testing.T) {
	inputSchema := Schema{Type: "object", Optional: []string{"type", "status"}}
	outputSchema := Schema{Type: "object"}

	q := &mockQuery{
		name:         "model.list",
		domain:       "model",
		inputSchema:  inputSchema,
		outputSchema: outputSchema,
	}

	if q.InputSchema().Type != "object" {
		t.Errorf("expected InputSchema Type 'object', got %s", q.InputSchema().Type)
	}
	if len(q.InputSchema().Optional) != 2 {
		t.Errorf("expected 2 optional fields, got %d", len(q.InputSchema().Optional))
	}
}

func TestCommandExamples(t *testing.T) {
	examples := []Example{
		{
			Input:       map[string]any{"source": "ollama", "repo": "llama3"},
			Output:      map[string]any{"model_id": "llama3:latest"},
			Description: "Pull llama3 from Ollama",
		},
	}

	cmd := &mockCommand{
		name:     "model.pull",
		domain:   "model",
		examples: examples,
	}

	if len(cmd.Examples()) != 1 {
		t.Errorf("expected 1 example, got %d", len(cmd.Examples()))
	}
	if cmd.Examples()[0].Description != "Pull llama3 from Ollama" {
		t.Errorf("unexpected example description: %s", cmd.Examples()[0].Description)
	}
}

func TestResourceSchema(t *testing.T) {
	schema := Schema{
		Type:  "object",
		Title: "Model Resource",
		Properties: map[string]Field{
			"id":   {Name: "id", Schema: Schema{Type: "string"}},
			"name": {Name: "name", Schema: Schema{Type: "string"}},
		},
	}

	res := &mockResource{
		uri:    "asms://model/123",
		domain: "model",
		schema: schema,
	}

	if res.Schema().Title != "Model Resource" {
		t.Errorf("expected Schema Title 'Model Resource', got %s", res.Schema().Title)
	}
	if len(res.Schema().Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(res.Schema().Properties))
	}
}

type mockCommand struct {
	name         string
	domain       string
	inputSchema  Schema
	outputSchema Schema
	description  string
	examples     []Example
	execute      func(ctx context.Context, input any) (any, error)
}

func (m *mockCommand) Name() string         { return m.name }
func (m *mockCommand) Domain() string       { return m.domain }
func (m *mockCommand) InputSchema() Schema  { return m.inputSchema }
func (m *mockCommand) OutputSchema() Schema { return m.outputSchema }
func (m *mockCommand) Description() string  { return m.description }
func (m *mockCommand) Examples() []Example  { return m.examples }
func (m *mockCommand) Execute(ctx context.Context, input any) (any, error) {
	if m.execute != nil {
		return m.execute(ctx, input)
	}
	return nil, nil
}

type mockQuery struct {
	name         string
	domain       string
	inputSchema  Schema
	outputSchema Schema
	description  string
	examples     []Example
	execute      func(ctx context.Context, input any) (any, error)
}

func (m *mockQuery) Name() string         { return m.name }
func (m *mockQuery) Domain() string       { return m.domain }
func (m *mockQuery) InputSchema() Schema  { return m.inputSchema }
func (m *mockQuery) OutputSchema() Schema { return m.outputSchema }
func (m *mockQuery) Description() string  { return m.description }
func (m *mockQuery) Examples() []Example  { return m.examples }
func (m *mockQuery) Execute(ctx context.Context, input any) (any, error) {
	if m.execute != nil {
		return m.execute(ctx, input)
	}
	return nil, nil
}

type mockEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func (m *mockEvent) Type() string          { return m.eventType }
func (m *mockEvent) Domain() string        { return m.domain }
func (m *mockEvent) Payload() any          { return m.payload }
func (m *mockEvent) Timestamp() time.Time  { return m.timestamp }
func (m *mockEvent) CorrelationID() string { return m.correlationID }

type mockResource struct {
	uri    string
	domain string
	schema Schema
	get    func(ctx context.Context) (any, error)
	watch  func(ctx context.Context) (<-chan ResourceUpdate, error)
}

func (m *mockResource) URI() string    { return m.uri }
func (m *mockResource) Domain() string { return m.domain }
func (m *mockResource) Schema() Schema { return m.schema }
func (m *mockResource) Get(ctx context.Context) (any, error) {
	if m.get != nil {
		return m.get(ctx)
	}
	return nil, nil
}
func (m *mockResource) Watch(ctx context.Context) (<-chan ResourceUpdate, error) {
	if m.watch != nil {
		return m.watch(ctx)
	}
	return nil, nil
}

func ptr[T any](v T) *T {
	return &v
}

type assertError string

func (e assertError) Error() string { return string(e) }
