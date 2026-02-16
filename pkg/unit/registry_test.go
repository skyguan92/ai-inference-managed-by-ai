package unit

import (
	"context"
	"sync"
	"testing"
	"time"
)

type regTestCommand struct {
	name        string
	domain      string
	description string
}

func (m *regTestCommand) Name() string         { return m.name }
func (m *regTestCommand) Domain() string       { return m.domain }
func (m *regTestCommand) InputSchema() Schema  { return Schema{} }
func (m *regTestCommand) OutputSchema() Schema { return Schema{} }
func (m *regTestCommand) Execute(ctx context.Context, input any) (any, error) {
	return nil, nil
}
func (m *regTestCommand) Description() string { return m.description }
func (m *regTestCommand) Examples() []Example { return nil }

type regTestQuery struct {
	name   string
	domain string
}

func (m *regTestQuery) Name() string         { return m.name }
func (m *regTestQuery) Domain() string       { return m.domain }
func (m *regTestQuery) InputSchema() Schema  { return Schema{} }
func (m *regTestQuery) OutputSchema() Schema { return Schema{} }
func (m *regTestQuery) Execute(ctx context.Context, input any) (any, error) {
	return nil, nil
}
func (m *regTestQuery) Description() string { return "" }
func (m *regTestQuery) Examples() []Example { return nil }

type regTestResource struct {
	uri    string
	domain string
}

func (m *regTestResource) URI() string    { return m.uri }
func (m *regTestResource) Domain() string { return m.domain }
func (m *regTestResource) Schema() Schema { return Schema{} }
func (m *regTestResource) Get(ctx context.Context) (any, error) {
	return nil, nil
}
func (m *regTestResource) Watch(ctx context.Context) (<-chan ResourceUpdate, error) {
	return nil, nil
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.commands == nil {
		t.Error("commands map not initialized")
	}
	if r.queries == nil {
		t.Error("queries map not initialized")
	}
	if r.resources == nil {
		t.Error("resources map not initialized")
	}
}

func TestRegisterCommand(t *testing.T) {
	r := NewRegistry()
	cmd := &regTestCommand{name: "model.pull", domain: "model"}

	err := r.RegisterCommand(cmd)
	if err != nil {
		t.Errorf("RegisterCommand failed: %v", err)
	}

	if r.CommandCount() != 1 {
		t.Errorf("expected 1 command, got %d", r.CommandCount())
	}

	err = r.RegisterCommand(cmd)
	if err != ErrCommandAlreadyRegistered {
		t.Errorf("expected ErrCommandAlreadyRegistered, got %v", err)
	}

	err = r.RegisterCommand(nil)
	if err != ErrCommandNotFound {
		t.Errorf("expected ErrCommandNotFound for nil, got %v", err)
	}
}

func TestRegisterQuery(t *testing.T) {
	r := NewRegistry()
	q := &regTestQuery{name: "model.list", domain: "model"}

	err := r.RegisterQuery(q)
	if err != nil {
		t.Errorf("RegisterQuery failed: %v", err)
	}

	if r.QueryCount() != 1 {
		t.Errorf("expected 1 query, got %d", r.QueryCount())
	}

	err = r.RegisterQuery(q)
	if err != ErrQueryAlreadyRegistered {
		t.Errorf("expected ErrQueryAlreadyRegistered, got %v", err)
	}

	err = r.RegisterQuery(nil)
	if err != ErrQueryNotFound {
		t.Errorf("expected ErrQueryNotFound for nil, got %v", err)
	}
}

func TestRegisterResource(t *testing.T) {
	r := NewRegistry()
	res := &regTestResource{uri: "asms://model/test", domain: "model"}

	err := r.RegisterResource(res)
	if err != nil {
		t.Errorf("RegisterResource failed: %v", err)
	}

	if r.ResourceCount() != 1 {
		t.Errorf("expected 1 resource, got %d", r.ResourceCount())
	}

	err = r.RegisterResource(res)
	if err != ErrResourceAlreadyRegistered {
		t.Errorf("expected ErrResourceAlreadyRegistered, got %v", err)
	}

	err = r.RegisterResource(nil)
	if err != ErrResourceNotFound {
		t.Errorf("expected ErrResourceNotFound for nil, got %v", err)
	}
}

func TestGetCommand(t *testing.T) {
	r := NewRegistry()
	cmd := &regTestCommand{name: "model.pull", domain: "model"}
	_ = r.RegisterCommand(cmd)

	got := r.GetCommand("model.pull")
	if got == nil {
		t.Error("GetCommand returned nil")
	}
	if got.Name() != "model.pull" {
		t.Errorf("expected name 'model.pull', got '%s'", got.Name())
	}

	got = r.GetCommand("nonexistent")
	if got != nil {
		t.Error("GetCommand should return nil for nonexistent command")
	}
}

func TestGetQuery(t *testing.T) {
	r := NewRegistry()
	q := &regTestQuery{name: "model.list", domain: "model"}
	_ = r.RegisterQuery(q)

	got := r.GetQuery("model.list")
	if got == nil {
		t.Error("GetQuery returned nil")
	}
	if got.Name() != "model.list" {
		t.Errorf("expected name 'model.list', got '%s'", got.Name())
	}

	got = r.GetQuery("nonexistent")
	if got != nil {
		t.Error("GetQuery should return nil for nonexistent query")
	}
}

func TestGetResource(t *testing.T) {
	r := NewRegistry()
	res := &regTestResource{uri: "asms://model/test", domain: "model"}
	_ = r.RegisterResource(res)

	got := r.GetResource("asms://model/test")
	if got == nil {
		t.Error("GetResource returned nil")
	}
	if got.URI() != "asms://model/test" {
		t.Errorf("expected uri 'asms://model/test', got '%s'", got.URI())
	}

	got = r.GetResource("nonexistent")
	if got != nil {
		t.Error("GetResource should return nil for nonexistent resource")
	}
}

func TestListCommands(t *testing.T) {
	r := NewRegistry()

	list := r.ListCommands()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	_ = r.RegisterCommand(&regTestCommand{name: "model.pull", domain: "model"})
	_ = r.RegisterCommand(&regTestCommand{name: "model.delete", domain: "model"})
	_ = r.RegisterCommand(&regTestCommand{name: "engine.start", domain: "engine"})

	list = r.ListCommands()
	if len(list) != 3 {
		t.Errorf("expected 3 commands, got %d", len(list))
	}
}

func TestListQueries(t *testing.T) {
	r := NewRegistry()

	list := r.ListQueries()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	_ = r.RegisterQuery(&regTestQuery{name: "model.list", domain: "model"})
	_ = r.RegisterQuery(&regTestQuery{name: "model.get", domain: "model"})

	list = r.ListQueries()
	if len(list) != 2 {
		t.Errorf("expected 2 queries, got %d", len(list))
	}
}

func TestListResources(t *testing.T) {
	r := NewRegistry()

	list := r.ListResources()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	_ = r.RegisterResource(&regTestResource{uri: "asms://model/test1", domain: "model"})
	_ = r.RegisterResource(&regTestResource{uri: "asms://model/test2", domain: "model"})

	list = r.ListResources()
	if len(list) != 2 {
		t.Errorf("expected 2 resources, got %d", len(list))
	}
}

func TestGet(t *testing.T) {
	r := NewRegistry()

	cmd := &regTestCommand{name: "model.pull", domain: "model"}
	q := &regTestQuery{name: "model.list", domain: "model"}
	res := &regTestResource{uri: "asms://model/test", domain: "model"}

	_ = r.RegisterCommand(cmd)
	_ = r.RegisterQuery(q)
	_ = r.RegisterResource(res)

	got := r.Get("model.pull")
	if got == nil {
		t.Error("Get returned nil for command")
	}
	if _, ok := got.(Command); !ok {
		t.Error("Get did not return a Command")
	}

	got = r.Get("model.list")
	if got == nil {
		t.Error("Get returned nil for query")
	}
	if _, ok := got.(Query); !ok {
		t.Error("Get did not return a Query")
	}

	got = r.Get("asms://model/test")
	if got == nil {
		t.Error("Get returned nil for resource")
	}
	if _, ok := got.(Resource); !ok {
		t.Error("Get did not return a Resource")
	}

	got = r.Get("nonexistent")
	if got != nil {
		t.Error("Get should return nil for nonexistent unit")
	}
}

func TestUnregisterCommand(t *testing.T) {
	r := NewRegistry()
	cmd := &regTestCommand{name: "model.pull", domain: "model"}
	_ = r.RegisterCommand(cmd)

	if !r.UnregisterCommand("model.pull") {
		t.Error("UnregisterCommand should return true for existing command")
	}

	if r.UnregisterCommand("model.pull") {
		t.Error("UnregisterCommand should return false for non-existing command")
	}

	if r.CommandCount() != 0 {
		t.Errorf("expected 0 commands, got %d", r.CommandCount())
	}
}

func TestUnregisterQuery(t *testing.T) {
	r := NewRegistry()
	q := &regTestQuery{name: "model.list", domain: "model"}
	_ = r.RegisterQuery(q)

	if !r.UnregisterQuery("model.list") {
		t.Error("UnregisterQuery should return true for existing query")
	}

	if r.UnregisterQuery("model.list") {
		t.Error("UnregisterQuery should return false for non-existing query")
	}

	if r.QueryCount() != 0 {
		t.Errorf("expected 0 queries, got %d", r.QueryCount())
	}
}

func TestUnregisterResource(t *testing.T) {
	r := NewRegistry()
	res := &regTestResource{uri: "asms://model/test", domain: "model"}
	_ = r.RegisterResource(res)

	if !r.UnregisterResource("asms://model/test") {
		t.Error("UnregisterResource should return true for existing resource")
	}

	if r.UnregisterResource("asms://model/test") {
		t.Error("UnregisterResource should return false for non-existing resource")
	}

	if r.ResourceCount() != 0 {
		t.Errorf("expected 0 resources, got %d", r.ResourceCount())
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func(idx int) {
			defer wg.Done()
			cmd := &regTestCommand{name: "cmd" + string(rune(idx)), domain: "test"}
			_ = r.RegisterCommand(cmd)
		}(i)

		go func(idx int) {
			defer wg.Done()
			q := &regTestQuery{name: "query" + string(rune(idx)), domain: "test"}
			_ = r.RegisterQuery(q)
		}(i)

		go func(idx int) {
			defer wg.Done()
			res := &regTestResource{uri: "uri" + string(rune(idx)), domain: "test"}
			_ = r.RegisterResource(res)
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			_ = r.ListCommands()
		}()

		go func() {
			defer wg.Done()
			_ = r.ListQueries()
		}()

		go func() {
			defer wg.Done()
			_ = r.ListResources()
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("concurrent access test timed out")
	}
}
