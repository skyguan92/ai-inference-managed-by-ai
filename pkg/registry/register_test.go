package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/device"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/remote"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

func TestRegisterAll(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAll(registry)
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	if len(registry.ListCommands()) == 0 {
		t.Error("Expected commands to be registered, got 0")
	}

	if len(registry.ListQueries()) == 0 {
		t.Error("Expected queries to be registered, got 0")
	}
}

func TestRegisterAllWithDefaults(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAllWithDefaults(registry)
	if err != nil {
		t.Fatalf("RegisterAllWithDefaults() error = %v", err)
	}

	tests := []struct {
		name     string
		unitName string
		wantCmd  bool
		wantQry  bool
	}{
		{"model.create command", "model.create", true, false},
		{"model.list query", "model.list", false, true},
		{"device.detect command", "device.detect", true, false},
		{"device.info query", "device.info", false, true},
		{"engine.start command", "engine.start", true, false},
		{"engine.list query", "engine.list", false, true},
		{"inference.chat command", "inference.chat", true, false},
		{"inference.models query", "inference.models", false, true},
		{"resource.allocate command", "resource.allocate", true, false},
		{"resource.status query", "resource.status", false, true},
		{"service.create command", "service.create", true, false},
		{"service.list query", "service.list", false, true},
		{"app.install command", "app.install", true, false},
		{"app.list query", "app.list", false, true},
		{"pipeline.create command", "pipeline.create", true, false},
		{"pipeline.list query", "pipeline.list", false, true},
		{"alert.create_rule command", "alert.create_rule", true, false},
		{"alert.list_rules query", "alert.list_rules", false, true},
		{"remote.enable command", "remote.enable", true, false},
		{"remote.status query", "remote.status", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantCmd {
				cmd := registry.GetCommand(tt.unitName)
				if cmd == nil {
					t.Errorf("Expected command %s to be registered", tt.unitName)
				}
			}
			if tt.wantQry {
				qry := registry.GetQuery(tt.unitName)
				if qry == nil {
					t.Errorf("Expected query %s to be registered", tt.unitName)
				}
			}
		})
	}
}

func TestRegisterAllWithStores(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAll(registry, WithStores(Stores{}))
	if err != nil {
		t.Fatalf("RegisterAll() with empty stores error = %v", err)
	}

	if registry.CommandCount() == 0 {
		t.Error("Expected commands to be registered with empty stores")
	}
}

func TestCommandCounts(t *testing.T) {
	registry := unit.NewRegistry()

	err := RegisterAll(registry)
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	cmdCount := registry.CommandCount()
	if cmdCount < 40 {
		t.Errorf("Expected at least 40 commands, got %d", cmdCount)
	}

	qryCount := registry.QueryCount()
	if qryCount < 20 {
		t.Errorf("Expected at least 20 queries, got %d", qryCount)
	}
}

func TestWithProviders(t *testing.T) {
	registry := unit.NewRegistry()
	providers := Providers{}
	err := RegisterAll(registry, WithProviders(providers))
	if err != nil {
		t.Fatalf("RegisterAll() with providers error = %v", err)
	}
}

func TestWithModelStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := model.NewMemoryStore()
	err := RegisterAll(registry, WithModelStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with model store error = %v", err)
	}
}

func TestWithEngineStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	err := RegisterAll(registry, WithEngineStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with engine store error = %v", err)
	}
}

func TestWithResourceStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := resource.NewMemoryStore()
	err := RegisterAll(registry, WithResourceStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with resource store error = %v", err)
	}
}

func TestWithServiceStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := service.NewMemoryStore()
	err := RegisterAll(registry, WithServiceStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with service store error = %v", err)
	}
}

func TestWithAppStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := app.NewMemoryStore()
	err := RegisterAll(registry, WithAppStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with app store error = %v", err)
	}
}

func TestWithPipelineStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := pipeline.NewMemoryStore()
	err := RegisterAll(registry, WithPipelineStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with pipeline store error = %v", err)
	}
}

func TestWithAlertStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := alert.NewMemoryStore()
	err := RegisterAll(registry, WithAlertStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with alert store error = %v", err)
	}
}

func TestWithRemoteStore(t *testing.T) {
	registry := unit.NewRegistry()
	store := remote.NewMemoryStore()
	err := RegisterAll(registry, WithRemoteStore(store))
	if err != nil {
		t.Fatalf("RegisterAll() with remote store error = %v", err)
	}
}

func TestWithModelProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &model.MockProvider{}
	err := RegisterAll(registry, WithModelProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with model provider error = %v", err)
	}
}

func TestWithEngineProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &engine.MockProvider{}
	err := RegisterAll(registry, WithEngineProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with engine provider error = %v", err)
	}
}

type mockDeviceProvider struct{}

func (m *mockDeviceProvider) Detect(ctx context.Context) ([]device.DeviceInfo, error) {
	return nil, nil
}
func (m *mockDeviceProvider) GetDevice(ctx context.Context, deviceID string) (*device.DeviceInfo, error) {
	return nil, nil
}
func (m *mockDeviceProvider) GetMetrics(ctx context.Context, deviceID string) (*device.DeviceMetrics, error) {
	return nil, nil
}
func (m *mockDeviceProvider) GetHealth(ctx context.Context, deviceID string) (*device.DeviceHealth, error) {
	return nil, nil
}
func (m *mockDeviceProvider) SetPowerLimit(ctx context.Context, deviceID string, limitWatts float64) error {
	return nil
}

func TestWithDeviceProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &mockDeviceProvider{}
	err := RegisterAll(registry, WithDeviceProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with device provider error = %v", err)
	}
}

func TestWithInferenceProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := inference.NewMockProvider()
	err := RegisterAll(registry, WithInferenceProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with inference provider error = %v", err)
	}
}

func TestWithResourceProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &resource.MockProvider{}
	err := RegisterAll(registry, WithResourceProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with resource provider error = %v", err)
	}
	if registry.GetQuery("resource.can_allocate") == nil {
		t.Error("Expected resource.can_allocate query with provider")
	}
}

func TestWithServiceProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &service.MockProvider{}
	err := RegisterAll(registry, WithServiceProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with service provider error = %v", err)
	}
	if registry.GetQuery("service.recommend") == nil {
		t.Error("Expected service.recommend query with provider")
	}
}

func TestWithAppProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &app.MockProvider{}
	err := RegisterAll(registry, WithAppProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with app provider error = %v", err)
	}
	if registry.GetQuery("app.templates") == nil {
		t.Error("Expected app.templates query with provider")
	}
}

func TestWithRemoteProvider(t *testing.T) {
	registry := unit.NewRegistry()
	provider := &remote.MockProvider{}
	err := RegisterAll(registry, WithRemoteProvider(provider))
	if err != nil {
		t.Fatalf("RegisterAll() with remote provider error = %v", err)
	}
}

func TestCreateStepExecutor_Command(t *testing.T) {
	registry := unit.NewRegistry()
	executor := createStepExecutor(registry)

	ctx := context.Background()
	_, err := executor(ctx, "nonexistent.command", nil)
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
}

func TestCreateStepExecutor_WithCommand(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testCommand{name: "test.cmd", output: map[string]any{"key": "value"}})
	executor := createStepExecutor(registry)

	ctx := context.Background()
	result, err := executor(ctx, "test.cmd", map[string]any{"input": "data"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Expected key=value, got %v", result)
	}
}

func TestCreateStepExecutor_WithCommandNonMapOutput(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testCommand{name: "test.cmd2", output: "string_result"})
	executor := createStepExecutor(registry)

	ctx := context.Background()
	result, err := executor(ctx, "test.cmd2", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result["result"] != "string_result" {
		t.Errorf("Expected result=string_result, got %v", result)
	}
}

func TestCreateStepExecutor_WithCommandError(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testCommand{name: "test.err_cmd", err: errors.New("cmd error")})
	executor := createStepExecutor(registry)

	ctx := context.Background()
	_, err := executor(ctx, "test.err_cmd", nil)
	if err == nil || err.Error() != "cmd error" {
		t.Errorf("Expected cmd error, got %v", err)
	}
}

func TestCreateStepExecutor_WithQuery(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testQuery{name: "test.query", output: map[string]any{"qkey": "qvalue"}})
	executor := createStepExecutor(registry)

	ctx := context.Background()
	result, err := executor(ctx, "test.query", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result["qkey"] != "qvalue" {
		t.Errorf("Expected qkey=qvalue, got %v", result)
	}
}

func TestCreateStepExecutor_WithQueryNonMapOutput(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testQuery{name: "test.query2", output: 42})
	executor := createStepExecutor(registry)

	ctx := context.Background()
	result, err := executor(ctx, "test.query2", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result["result"] != 42 {
		t.Errorf("Expected result=42, got %v", result)
	}
}

func TestCreateStepExecutor_WithQueryError(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testQuery{name: "test.err_query", err: errors.New("query error")})
	executor := createStepExecutor(registry)

	ctx := context.Background()
	_, err := executor(ctx, "test.err_query", nil)
	if err == nil || err.Error() != "query error" {
		t.Errorf("Expected query error, got %v", err)
	}
}

type testCommand struct {
	name   string
	output any
	err    error
}

func (c *testCommand) Name() string              { return c.name }
func (c *testCommand) Domain() string            { return "test" }
func (c *testCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (c *testCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (c *testCommand) Description() string       { return "test command" }
func (c *testCommand) Examples() []unit.Example  { return nil }
func (c *testCommand) Execute(ctx context.Context, input any) (any, error) {
	return c.output, c.err
}

type testQuery struct {
	name   string
	output any
	err    error
}

func (q *testQuery) Name() string              { return q.name }
func (q *testQuery) Domain() string            { return "test" }
func (q *testQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (q *testQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (q *testQuery) Description() string       { return "test query" }
func (q *testQuery) Examples() []unit.Example  { return nil }
func (q *testQuery) Execute(ctx context.Context, input any) (any, error) {
	return q.output, q.err
}

func TestRegisterAll_ComboOptions(t *testing.T) {
	registry := unit.NewRegistry()
	err := RegisterAll(registry,
		WithModelStore(model.NewMemoryStore()),
		WithModelProvider(&model.MockProvider{}),
		WithEngineStore(engine.NewMemoryStore()),
		WithEngineProvider(&engine.MockProvider{}),
		WithDeviceProvider(&mockDeviceProvider{}),
		WithInferenceProvider(inference.NewMockProvider()),
		WithResourceStore(resource.NewMemoryStore()),
		WithResourceProvider(&resource.MockProvider{}),
		WithServiceStore(service.NewMemoryStore()),
		WithServiceProvider(&service.MockProvider{}),
		WithAppStore(app.NewMemoryStore()),
		WithAppProvider(&app.MockProvider{}),
		WithPipelineStore(pipeline.NewMemoryStore()),
		WithAlertStore(alert.NewMemoryStore()),
		WithRemoteStore(remote.NewMemoryStore()),
		WithRemoteProvider(&remote.MockProvider{}),
	)
	if err != nil {
		t.Fatalf("RegisterAll() with all options error = %v", err)
	}
}

func TestRegisterAll_DuplicateError(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testCommand{name: "model.create"})
	err := RegisterAll(registry)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestRegisterAll_WithStoresAndProviders(t *testing.T) {
	registry := unit.NewRegistry()
	stores := Stores{
		ModelStore:    model.NewMemoryStore(),
		EngineStore:   engine.NewMemoryStore(),
		ResourceStore: resource.NewMemoryStore(),
		ServiceStore:  service.NewMemoryStore(),
		AppStore:      app.NewMemoryStore(),
		PipelineStore: pipeline.NewMemoryStore(),
		AlertStore:    alert.NewMemoryStore(),
		RemoteStore:   remote.NewMemoryStore(),
	}
	providers := Providers{
		ModelProvider:     &model.MockProvider{},
		EngineProvider:    &engine.MockProvider{},
		DeviceProvider:    &mockDeviceProvider{},
		InferenceProvider: inference.NewMockProvider(),
		ResourceProvider:  &resource.MockProvider{},
		ServiceProvider:   &service.MockProvider{},
		AppProvider:       &app.MockProvider{},
		RemoteProvider:    &remote.MockProvider{},
	}
	err := RegisterAll(registry, WithStores(stores), WithProviders(providers))
	if err != nil {
		t.Fatalf("RegisterAll() with stores and providers error = %v", err)
	}
}

func TestRegisterAll_ErrorPaths(t *testing.T) {
	t.Run("model domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "model.create"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate model.create")
		}
	})

	t.Run("device domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "device.detect"})
		_ = RegisterAll(registry, WithModelStore(model.NewMemoryStore()))
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate device.detect")
		}
	})

	t.Run("engine domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "engine.start"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate engine.start")
		}
	})

	t.Run("inference domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "inference.chat"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate inference.chat")
		}
	})

	t.Run("resource domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "resource.allocate"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate resource.allocate")
		}
	})

	t.Run("service domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "service.create"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate service.create")
		}
	})

	t.Run("app domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "app.install"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate app.install")
		}
	})

	t.Run("pipeline domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "pipeline.create"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate pipeline.create")
		}
	})

	t.Run("alert domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "alert.create_rule"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate alert.create_rule")
		}
	})

	t.Run("remote domain error", func(t *testing.T) {
		registry := unit.NewRegistry()
		_ = registry.RegisterCommand(&testCommand{name: "remote.enable"})
		err := RegisterAll(registry)
		if err == nil {
			t.Error("Expected error for duplicate remote.enable")
		}
	})
}
