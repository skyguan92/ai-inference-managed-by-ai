//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/catalog"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/device"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/remote"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/skill"
	"github.com/stretchr/testify/require"
)

// TestEnv wraps a fully-wired gateway with all domain stores and mock providers.
type TestEnv struct {
	Gateway  *gateway.Gateway
	Registry *unit.Registry

	// Stores (for seeding / inspecting state)
	ModelStore    *model.MemoryStore
	ServiceStore  *service.MemoryStore
	EngineStore   *engine.MemoryStore
	ResourceStore *resource.MemoryStore
	AppStore      *app.MemoryStore
	PipelineStore *pipeline.MemoryStore
	AlertStore    *alert.MemoryStore
	RemoteStore   *remote.MemoryStore
	CatalogStore  *catalog.MemoryStore
	SkillStore    *skill.MemoryStore

	// Providers (for controlling mock behavior)
	ModelProvider     *model.MockProvider
	ServiceProvider   *service.MockProvider
	DeviceProvider    *device.MockProvider
	EngineProvider    *engine.MockProvider
	InferenceProvider *inference.MockProvider
	ResourceProvider  *resource.MockProvider
	AppProvider       *app.MockProvider
	RemoteProvider    *remote.MockProvider
}

func newTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	env := &TestEnv{
		// Stores
		ModelStore:    model.NewMemoryStore(),
		ServiceStore:  service.NewMemoryStore(),
		EngineStore:   engine.NewMemoryStore(),
		ResourceStore: resource.NewMemoryStore(),
		AppStore:      app.NewMemoryStore(),
		PipelineStore: pipeline.NewMemoryStore(),
		AlertStore:    alert.NewMemoryStore(),
		RemoteStore:   remote.NewMemoryStore(),
		CatalogStore:  catalog.NewMemoryStore(),
		SkillStore:    skill.NewMemoryStore(),

		// Providers
		ModelProvider:     model.NewMockProvider(),
		ServiceProvider:   &service.MockProvider{},
		DeviceProvider:    device.NewMockProvider(),
		EngineProvider:    &engine.MockProvider{},
		InferenceProvider: inference.NewMockProvider(),
		ResourceProvider:  &resource.MockProvider{},
		AppProvider:       &app.MockProvider{},
		RemoteProvider:    &remote.MockProvider{},
	}

	env.Registry = unit.NewRegistry()
	err := registry.RegisterAll(env.Registry,
		registry.WithStores(registry.Stores{
			ModelStore:    env.ModelStore,
			EngineStore:   env.EngineStore,
			ResourceStore: env.ResourceStore,
			ServiceStore:  env.ServiceStore,
			AppStore:      env.AppStore,
			PipelineStore: env.PipelineStore,
			AlertStore:    env.AlertStore,
			RemoteStore:   env.RemoteStore,
			CatalogStore:  env.CatalogStore,
			SkillStore:    env.SkillStore,
		}),
		registry.WithProviders(registry.Providers{
			ModelProvider:     env.ModelProvider,
			EngineProvider:    env.EngineProvider,
			DeviceProvider:    env.DeviceProvider,
			InferenceProvider: env.InferenceProvider,
			ResourceProvider:  env.ResourceProvider,
			ServiceProvider:   env.ServiceProvider,
			AppProvider:       env.AppProvider,
			RemoteProvider:    env.RemoteProvider,
		}),
	)
	require.NoError(t, err, "RegisterAll should succeed")

	env.Gateway = gateway.NewGateway(env.Registry, gateway.WithTimeout(5*time.Minute))
	return env
}

// command sends a command through the gateway.
func (e *TestEnv) command(ctx context.Context, unitName string, input map[string]any) *gateway.Response {
	return e.Gateway.Handle(ctx, &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  unitName,
		Input: input,
	})
}

// query sends a query through the gateway.
func (e *TestEnv) query(ctx context.Context, unitName string, input map[string]any) *gateway.Response {
	return e.Gateway.Handle(ctx, &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  unitName,
		Input: input,
	})
}

// seedModel injects a model directly into the MemoryStore (bypasses command pipeline).
func (e *TestEnv) seedModel(t *testing.T, id, name string, modelType model.ModelType, path string) {
	t.Helper()
	now := time.Now().Unix()
	err := e.ModelStore.Create(context.Background(), &model.Model{
		ID:        id,
		Name:      name,
		Type:      modelType,
		Format:    model.FormatSafetensors,
		Status:    model.StatusReady,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err, "seedModel should succeed")
}

// seedService injects a service directly into the MemoryStore.
func (e *TestEnv) seedService(t *testing.T, id, modelID string, status service.ServiceStatus) {
	t.Helper()
	now := time.Now().Unix()
	err := e.ServiceStore.Create(context.Background(), &service.ModelService{
		ID:            id,
		Name:          "service-" + id,
		ModelID:       modelID,
		Status:        status,
		Replicas:      1,
		ResourceClass: service.ResourceClassMedium,
		Endpoints:     []string{"http://localhost:8080"},
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	require.NoError(t, err, "seedService should succeed")
}

// requireSuccess asserts that the gateway response is successful.
func requireSuccess(t *testing.T, resp *gateway.Response, msg string) {
	t.Helper()
	if !resp.Success {
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
			if resp.Error.Details != nil {
				errMsg += " | details: " + toString(resp.Error.Details)
			}
		}
		t.Fatalf("%s: expected success but got error: %s", msg, errMsg)
	}
}

// requireFailure asserts that the gateway response is a failure.
func requireFailure(t *testing.T, resp *gateway.Response, msg string) {
	t.Helper()
	if resp.Success {
		t.Fatalf("%s: expected failure but got success with data: %v", msg, resp.Data)
	}
}

// dataMap extracts the response data as map[string]any.
func dataMap(t *testing.T, resp *gateway.Response) map[string]any {
	t.Helper()
	m, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any data, got %T: %v", resp.Data, resp.Data)
	}
	return m
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
