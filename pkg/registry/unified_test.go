package registry

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/remote"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

func TestNewUnifiedRegistry(t *testing.T) {
	registry := NewUnifiedRegistry()

	if registry == nil {
		t.Fatal("Expected NewUnifiedRegistry to return non-nil registry")
	}

	if registry.registry == nil {
		t.Error("Expected internal registry to be initialized")
	}

	if registry.options == nil {
		t.Error("Expected options to be initialized")
	}
}

func TestUnifiedRegistry_RegisterAll(t *testing.T) {
	registry := NewUnifiedRegistry()

	err := registry.RegisterAll()
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	if registry.Registry().CommandCount() == 0 {
		t.Error("Expected commands to be registered")
	}

	if registry.Registry().QueryCount() == 0 {
		t.Error("Expected queries to be registered")
	}
}

func TestRegisterModelDomain(t *testing.T) {
	registry := NewUnifiedRegistry()

	err := registry.RegisterModelDomain()
	if err != nil {
		t.Fatalf("RegisterModelDomain() error = %v", err)
	}

	reg := registry.Registry()

	tests := []struct {
		name     string
		unitName string
		wantCmd  bool
		wantQry  bool
	}{
		{"model.create command", "model.create", true, false},
		{"model.delete command", "model.delete", true, false},
		{"model.pull command", "model.pull", true, false},
		{"model.import command", "model.import", true, false},
		{"model.verify command", "model.verify", true, false},
		{"model.get query", "model.get", false, true},
		{"model.list query", "model.list", false, true},
		{"model.search query", "model.search", false, true},
		{"model.estimate_resources query", "model.estimate_resources", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantCmd {
				cmd := reg.GetCommand(tt.unitName)
				if cmd == nil {
					t.Errorf("Expected command %s to be registered", tt.unitName)
				}
			}
			if tt.wantQry {
				qry := reg.GetQuery(tt.unitName)
				if qry == nil {
					t.Errorf("Expected query %s to be registered", tt.unitName)
				}
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := NewUnifiedRegistry()

	innerReg := registry.Registry()
	if innerReg == nil {
		t.Fatal("Expected Registry() to return non-nil inner registry")
	}

	if _, ok := interface{}(innerReg).(*unit.Registry); !ok {
		t.Error("Expected returned registry to be of type *unit.Registry")
	}
}

func TestUnifiedRegistry_AllDomains(t *testing.T) {
	registry := NewUnifiedRegistry()

	if err := registry.RegisterModelDomain(); err != nil {
		t.Fatalf("RegisterModelDomain() error = %v", err)
	}

	if err := registry.RegisterDeviceDomain(); err != nil {
		t.Fatalf("RegisterDeviceDomain() error = %v", err)
	}

	if err := registry.RegisterEngineDomain(); err != nil {
		t.Fatalf("RegisterEngineDomain() error = %v", err)
	}

	if err := registry.RegisterInferenceDomain(); err != nil {
		t.Fatalf("RegisterInferenceDomain() error = %v", err)
	}

	if err := registry.RegisterResourceDomain(); err != nil {
		t.Fatalf("RegisterResourceDomain() error = %v", err)
	}

	if err := registry.RegisterServiceDomain(); err != nil {
		t.Fatalf("RegisterServiceDomain() error = %v", err)
	}

	if err := registry.RegisterAppDomain(); err != nil {
		t.Fatalf("RegisterAppDomain() error = %v", err)
	}

	if err := registry.RegisterPipelineDomain(); err != nil {
		t.Fatalf("RegisterPipelineDomain() error = %v", err)
	}

	if err := registry.RegisterAlertDomain(); err != nil {
		t.Fatalf("RegisterAlertDomain() error = %v", err)
	}

	if err := registry.RegisterRemoteDomain(); err != nil {
		t.Fatalf("RegisterRemoteDomain() error = %v", err)
	}

	reg := registry.Registry()
	if reg.CommandCount() < 40 {
		t.Errorf("Expected at least 40 commands, got %d", reg.CommandCount())
	}

	if reg.QueryCount() < 20 {
		t.Errorf("Expected at least 20 queries, got %d", reg.QueryCount())
	}
}

func TestUnifiedRegistry_WithStores(t *testing.T) {
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

	registry := NewUnifiedRegistry().WithStores(stores)

	if registry.options.Stores.ModelStore == nil {
		t.Error("Expected ModelStore to be set")
	}

	if err := registry.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}
}

func TestUnifiedRegistry_WithProviders(t *testing.T) {
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

	registry := NewUnifiedRegistry().WithProviders(providers)

	if registry.options.Providers.ModelProvider == nil {
		t.Error("Expected ModelProvider to be set")
	}

	if err := registry.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	reg := registry.Registry()
	if reg.GetQuery("resource.can_allocate") == nil {
		t.Error("Expected resource.can_allocate query with provider")
	}

	if reg.GetQuery("service.recommend") == nil {
		t.Error("Expected service.recommend query with provider")
	}

	if reg.GetQuery("app.templates") == nil {
		t.Error("Expected app.templates query with provider")
	}
}

func TestUnifiedRegistry_WithStoresAndProviders(t *testing.T) {
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

	registry := NewUnifiedRegistry().WithStores(stores).WithProviders(providers)

	if err := registry.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	reg := registry.Registry()
	if reg.CommandCount() == 0 {
		t.Error("Expected commands to be registered")
	}

	if reg.QueryCount() == 0 {
		t.Error("Expected queries to be registered")
	}
}

func TestUnifiedRegistry_DuplicateRegistration(t *testing.T) {
	registry := NewUnifiedRegistry()

	if err := registry.RegisterAll(); err != nil {
		t.Fatalf("First RegisterAll() error = %v", err)
	}

	err := registry.RegisterAll()
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestUnifiedRegistry_IndividualDomainErrors(t *testing.T) {
	t.Run("model domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "model.create"})

		err := registry.RegisterModelDomain()
		if err == nil {
			t.Error("Expected error for duplicate model.create")
		}
	})

	t.Run("device domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "device.detect"})

		err := registry.RegisterDeviceDomain()
		if err == nil {
			t.Error("Expected error for duplicate device.detect")
		}
	})

	t.Run("engine domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "engine.start"})

		err := registry.RegisterEngineDomain()
		if err == nil {
			t.Error("Expected error for duplicate engine.start")
		}
	})

	t.Run("inference domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "inference.chat"})

		err := registry.RegisterInferenceDomain()
		if err == nil {
			t.Error("Expected error for duplicate inference.chat")
		}
	})

	t.Run("resource domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "resource.allocate"})

		err := registry.RegisterResourceDomain()
		if err == nil {
			t.Error("Expected error for duplicate resource.allocate")
		}
	})

	t.Run("service domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "service.create"})

		err := registry.RegisterServiceDomain()
		if err == nil {
			t.Error("Expected error for duplicate service.create")
		}
	})

	t.Run("app domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "app.install"})

		err := registry.RegisterAppDomain()
		if err == nil {
			t.Error("Expected error for duplicate app.install")
		}
	})

	t.Run("pipeline domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "pipeline.create"})

		err := registry.RegisterPipelineDomain()
		if err == nil {
			t.Error("Expected error for duplicate pipeline.create")
		}
	})

	t.Run("alert domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "alert.create_rule"})

		err := registry.RegisterAlertDomain()
		if err == nil {
			t.Error("Expected error for duplicate alert.create_rule")
		}
	})

	t.Run("remote domain duplicate", func(t *testing.T) {
		registry := NewUnifiedRegistry()
		_ = registry.Registry().RegisterCommand(&testCommand{name: "remote.enable"})

		err := registry.RegisterRemoteDomain()
		if err == nil {
			t.Error("Expected error for duplicate remote.enable")
		}
	})
}

func TestUnifiedRegistry_ChainMethods(t *testing.T) {
	registry := NewUnifiedRegistry().
		WithStores(Stores{ModelStore: model.NewMemoryStore()}).
		WithProviders(Providers{ModelProvider: &model.MockProvider{}})

	if err := registry.RegisterModelDomain(); err != nil {
		t.Fatalf("RegisterModelDomain() error = %v", err)
	}

	reg := registry.Registry()
	if reg.GetCommand("model.create") == nil {
		t.Error("Expected model.create command to be registered")
	}
}
