package registry

import (
	"context"
	"fmt"

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

type Stores struct {
	ModelStore    model.ModelStore
	EngineStore   engine.EngineStore
	ResourceStore resource.ResourceStore
	ServiceStore  service.ServiceStore
	AppStore      app.AppStore
	PipelineStore pipeline.PipelineStore
	AlertStore    alert.Store
	RemoteStore   remote.RemoteStore
}

type Providers struct {
	ModelProvider     model.ModelProvider
	EngineProvider    engine.EngineProvider
	DeviceProvider    device.DeviceProvider
	InferenceProvider inference.InferenceProvider
	ResourceProvider  resource.ResourceProvider
	ServiceProvider   service.ServiceProvider
	AppProvider       app.AppProvider
	RemoteProvider    remote.RemoteProvider
}

type Options struct {
	Stores    Stores
	Providers Providers
	EventBus  unit.EventPublisher
}

type Option func(*Options)

func WithStores(stores Stores) Option {
	return func(o *Options) {
		o.Stores = stores
	}
}

func WithProviders(providers Providers) Option {
	return func(o *Options) {
		o.Providers = providers
	}
}

func WithEventBus(eventBus unit.EventPublisher) Option {
	return func(o *Options) {
		o.EventBus = eventBus
	}
}

func WithModelStore(s model.ModelStore) Option {
	return func(o *Options) {
		o.Stores.ModelStore = s
	}
}

func WithEngineStore(s engine.EngineStore) Option {
	return func(o *Options) {
		o.Stores.EngineStore = s
	}
}

func WithResourceStore(s resource.ResourceStore) Option {
	return func(o *Options) {
		o.Stores.ResourceStore = s
	}
}

func WithServiceStore(s service.ServiceStore) Option {
	return func(o *Options) {
		o.Stores.ServiceStore = s
	}
}

func WithAppStore(s app.AppStore) Option {
	return func(o *Options) {
		o.Stores.AppStore = s
	}
}

func WithPipelineStore(s pipeline.PipelineStore) Option {
	return func(o *Options) {
		o.Stores.PipelineStore = s
	}
}

func WithAlertStore(s alert.Store) Option {
	return func(o *Options) {
		o.Stores.AlertStore = s
	}
}

func WithRemoteStore(s remote.RemoteStore) Option {
	return func(o *Options) {
		o.Stores.RemoteStore = s
	}
}

func WithModelProvider(p model.ModelProvider) Option {
	return func(o *Options) {
		o.Providers.ModelProvider = p
	}
}

func WithEngineProvider(p engine.EngineProvider) Option {
	return func(o *Options) {
		o.Providers.EngineProvider = p
	}
}

func WithDeviceProvider(p device.DeviceProvider) Option {
	return func(o *Options) {
		o.Providers.DeviceProvider = p
	}
}

func WithInferenceProvider(p inference.InferenceProvider) Option {
	return func(o *Options) {
		o.Providers.InferenceProvider = p
	}
}

func WithResourceProvider(p resource.ResourceProvider) Option {
	return func(o *Options) {
		o.Providers.ResourceProvider = p
	}
}

func WithServiceProvider(p service.ServiceProvider) Option {
	return func(o *Options) {
		o.Providers.ServiceProvider = p
	}
}

func WithAppProvider(p app.AppProvider) Option {
	return func(o *Options) {
		o.Providers.AppProvider = p
	}
}

func WithRemoteProvider(p remote.RemoteProvider) Option {
	return func(o *Options) {
		o.Providers.RemoteProvider = p
	}
}

func RegisterAll(registry *unit.Registry, opts ...Option) error {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	if err := registerModelDomain(registry, options); err != nil {
		return fmt.Errorf("register model domain: %w", err)
	}

	if err := registerDeviceDomain(registry, options); err != nil {
		return fmt.Errorf("register device domain: %w", err)
	}

	if err := registerEngineDomain(registry, options); err != nil {
		return fmt.Errorf("register engine domain: %w", err)
	}

	if err := registerInferenceDomain(registry, options); err != nil {
		return fmt.Errorf("register inference domain: %w", err)
	}

	if err := registerResourceDomain(registry, options); err != nil {
		return fmt.Errorf("register resource domain: %w", err)
	}

	if err := registerServiceDomain(registry, options); err != nil {
		return fmt.Errorf("register service domain: %w", err)
	}

	if err := registerAppDomain(registry, options); err != nil {
		return fmt.Errorf("register app domain: %w", err)
	}

	if err := registerPipelineDomain(registry, options); err != nil {
		return fmt.Errorf("register pipeline domain: %w", err)
	}

	if err := registerAlertDomain(registry, options); err != nil {
		return fmt.Errorf("register alert domain: %w", err)
	}

	if err := registerRemoteDomain(registry, options); err != nil {
		return fmt.Errorf("register remote domain: %w", err)
	}

	return nil
}

func registerModelDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.ModelStore
	provider := options.Providers.ModelProvider

	if store == nil {
		store = model.NewMemoryStore()
	}

	if err := registry.RegisterCommand(model.NewCreateCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(model.NewDeleteCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(model.NewPullCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(model.NewImportCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(model.NewVerifyCommand(store, provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(model.NewGetQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(model.NewListQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(model.NewSearchQuery(provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(model.NewEstimateResourcesQuery(store, provider)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(model.NewModelResourceFactory(store)); err != nil {
		return err
	}

	// Register static resources
	if err := registry.RegisterResource(model.NewCompatibilityResource()); err != nil {
		return err
	}

	return nil
}

func registerDeviceDomain(registry *unit.Registry, options *Options) error {
	provider := options.Providers.DeviceProvider

	if err := registry.RegisterCommand(device.NewDetectCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(device.NewSetPowerLimitCommand(provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(device.NewInfoQuery(provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(device.NewMetricsQuery(provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(device.NewHealthQuery(provider)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(device.NewDeviceResourceFactory(provider)); err != nil {
		return err
	}

	return nil
}

func registerEngineDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.EngineStore
	provider := options.Providers.EngineProvider

	if store == nil {
		store = engine.NewMemoryStore()
	}

	if err := registry.RegisterCommand(engine.NewStartCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(engine.NewStopCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(engine.NewRestartCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(engine.NewInstallCommand(store, provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(engine.NewGetQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(engine.NewListQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(engine.NewFeaturesQuery(store, provider)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(engine.NewEngineResourceFactory(store)); err != nil {
		return err
	}

	return nil
}

func registerInferenceDomain(registry *unit.Registry, options *Options) error {
	provider := options.Providers.InferenceProvider
	events := options.EventBus

	if err := registry.RegisterCommand(inference.NewChatCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewCompleteCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewEmbedCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewTranscribeCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewSynthesizeCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewGenerateImageCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewGenerateVideoCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewRerankCommandWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewDetectCommandWithEvents(provider, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(inference.NewModelsQueryWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(inference.NewVoicesQueryWithEvents(provider, events)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(inference.NewInferenceResourceFactory(provider)); err != nil {
		return err
	}

	return nil
}

func registerResourceDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.ResourceStore
	provider := options.Providers.ResourceProvider
	events := options.EventBus

	if store == nil {
		store = resource.NewMemoryStore()
	}

	if err := registry.RegisterCommand(resource.NewAllocateCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(resource.NewReleaseCommandWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(resource.NewUpdateSlotCommandWithEvents(store, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(resource.NewStatusQueryWithEvents(provider, store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(resource.NewBudgetQueryWithEvents(provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(resource.NewAllocationsQueryWithEvents(store, events)); err != nil {
		return err
	}

	if provider != nil {
		if err := registry.RegisterQuery(resource.NewCanAllocateQueryWithEvents(provider, events)); err != nil {
			return err
		}
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(resource.NewResourceFactory(provider, store)); err != nil {
		return err
	}

	return nil
}

func registerServiceDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.ServiceStore
	provider := options.Providers.ServiceProvider
	events := options.EventBus

	if store == nil {
		store = service.NewMemoryStore()
	}

	if err := registry.RegisterCommand(service.NewCreateCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewDeleteCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewScaleCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewStartCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewStopCommandWithEvents(store, provider, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(service.NewGetQueryWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(service.NewListQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(service.NewStatusQueryWithEvents(store, events)); err != nil {
		return err
	}
	if provider != nil {
		if err := registry.RegisterQuery(service.NewRecommendQueryWithEvents(provider, events)); err != nil {
			return err
		}
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(service.NewServiceResourceFactory(store, provider)); err != nil {
		return err
	}

	return nil
}

func registerAppDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.AppStore
	provider := options.Providers.AppProvider
	events := options.EventBus

	if store == nil {
		store = app.NewMemoryStore()
	}

	if err := registry.RegisterCommand(app.NewInstallCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(app.NewUninstallCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(app.NewStartCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(app.NewStopCommandWithEvents(store, provider, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(app.NewGetQueryWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(app.NewListQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(app.NewLogsQueryWithEvents(store, provider, events)); err != nil {
		return err
	}
	if provider != nil {
		if err := registry.RegisterQuery(app.NewTemplatesQueryWithEvents(provider, events)); err != nil {
			return err
		}
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(app.NewAppResourceFactory(store, provider)); err != nil {
		return err
	}

	return nil
}

func registerPipelineDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.PipelineStore
	events := options.EventBus

	if store == nil {
		store = pipeline.NewMemoryStore()
	}

	stepExecutor := createStepExecutor(registry)
	executor := pipeline.NewExecutor(store, stepExecutor)

	if err := registry.RegisterCommand(pipeline.NewCreateCommandWithEvents(store, executor, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(pipeline.NewDeleteCommandWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(pipeline.NewRunCommandWithEvents(store, executor, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(pipeline.NewCancelCommandWithEvents(store, executor, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(pipeline.NewGetQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(pipeline.NewListQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(pipeline.NewStatusQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(pipeline.NewValidateQueryWithEvents(events)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(pipeline.NewPipelineResourceFactory(store)); err != nil {
		return err
	}

	return nil
}

func createStepExecutor(registry *unit.Registry) pipeline.StepExecutor {
	return func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		cmd := registry.GetCommand(stepType)
		if cmd != nil {
			output, err := cmd.Execute(ctx, input)
			if err != nil {
				return nil, err
			}
			if m, ok := output.(map[string]any); ok {
				return m, nil
			}
			return map[string]any{"result": output}, nil
		}

		query := registry.GetQuery(stepType)
		if query != nil {
			output, err := query.Execute(ctx, input)
			if err != nil {
				return nil, err
			}
			if m, ok := output.(map[string]any); ok {
				return m, nil
			}
			return map[string]any{"result": output}, nil
		}

		return nil, fmt.Errorf("unit '%s' not found in registry", stepType)
	}
}

func registerAlertDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.AlertStore
	events := options.EventBus

	if store == nil {
		store = alert.NewMemoryStore()
	}

	if err := registry.RegisterCommand(alert.NewCreateRuleCommandWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewUpdateRuleCommandWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewDeleteRuleCommandWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewAcknowledgeCommandWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewResolveCommandWithEvents(store, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(alert.NewListRulesQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(alert.NewHistoryQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(alert.NewActiveQueryWithEvents(store, events)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(alert.NewAlertResourceFactory(store)); err != nil {
		return err
	}

	return nil
}

func registerRemoteDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.RemoteStore
	provider := options.Providers.RemoteProvider
	events := options.EventBus

	if store == nil {
		store = remote.NewMemoryStore()
	}

	if err := registry.RegisterCommand(remote.NewEnableCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(remote.NewDisableCommandWithEvents(store, provider, events)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(remote.NewExecCommandWithEvents(store, provider, events)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(remote.NewStatusQueryWithEvents(store, events)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(remote.NewAuditQueryWithEvents(store, events)); err != nil {
		return err
	}

	// Register ResourceFactory for dynamic resource creation
	if err := registry.RegisterResourceFactory(remote.NewRemoteResourceFactory(store)); err != nil {
		return err
	}

	return nil
}

func RegisterAllWithDefaults(registry *unit.Registry) error {
	return RegisterAll(registry)
}
