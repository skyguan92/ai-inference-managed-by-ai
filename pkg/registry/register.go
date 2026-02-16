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

	return nil
}

func registerInferenceDomain(registry *unit.Registry, options *Options) error {
	provider := options.Providers.InferenceProvider

	if err := registry.RegisterCommand(inference.NewChatCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewCompleteCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewEmbedCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewTranscribeCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewSynthesizeCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewGenerateImageCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewGenerateVideoCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewRerankCommand(provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(inference.NewDetectCommand(provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(inference.NewModelsQuery(provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(inference.NewVoicesQuery(provider)); err != nil {
		return err
	}

	return nil
}

func registerResourceDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.ResourceStore
	provider := options.Providers.ResourceProvider

	if store == nil {
		store = resource.NewMemoryStore()
	}

	if err := registry.RegisterCommand(resource.NewAllocateCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(resource.NewReleaseCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(resource.NewUpdateSlotCommand(store)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(resource.NewStatusQuery(provider, store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(resource.NewBudgetQuery(provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(resource.NewAllocationsQuery(store)); err != nil {
		return err
	}

	if provider != nil {
		if err := registry.RegisterQuery(resource.NewCanAllocateQuery(provider)); err != nil {
			return err
		}
	}

	return nil
}

func registerServiceDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.ServiceStore
	provider := options.Providers.ServiceProvider

	if store == nil {
		store = service.NewMemoryStore()
	}

	if err := registry.RegisterCommand(service.NewCreateCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewDeleteCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewScaleCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewStartCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(service.NewStopCommand(store, provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(service.NewGetQuery(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(service.NewListQuery(store)); err != nil {
		return err
	}
	if provider != nil {
		if err := registry.RegisterQuery(service.NewRecommendQuery(provider)); err != nil {
			return err
		}
	}

	return nil
}

func registerAppDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.AppStore
	provider := options.Providers.AppProvider

	if store == nil {
		store = app.NewMemoryStore()
	}

	if err := registry.RegisterCommand(app.NewInstallCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(app.NewUninstallCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(app.NewStartCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(app.NewStopCommand(store, provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(app.NewGetQuery(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(app.NewListQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(app.NewLogsQuery(store, provider)); err != nil {
		return err
	}
	if provider != nil {
		if err := registry.RegisterQuery(app.NewTemplatesQuery(provider)); err != nil {
			return err
		}
	}

	return nil
}

func registerPipelineDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.PipelineStore

	if store == nil {
		store = pipeline.NewMemoryStore()
	}

	stepExecutor := createStepExecutor(registry)
	executor := pipeline.NewExecutor(store, stepExecutor)

	if err := registry.RegisterCommand(pipeline.NewCreateCommand(store, executor)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(pipeline.NewDeleteCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(pipeline.NewRunCommand(store, executor)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(pipeline.NewCancelCommand(store, executor)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(pipeline.NewGetQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(pipeline.NewListQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(pipeline.NewStatusQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(pipeline.NewValidateQuery()); err != nil {
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

	if store == nil {
		store = alert.NewMemoryStore()
	}

	if err := registry.RegisterCommand(alert.NewCreateRuleCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewUpdateRuleCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewDeleteRuleCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewAcknowledgeCommand(store)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(alert.NewResolveCommand(store)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(alert.NewListRulesQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(alert.NewHistoryQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(alert.NewActiveQuery(store)); err != nil {
		return err
	}

	return nil
}

func registerRemoteDomain(registry *unit.Registry, options *Options) error {
	store := options.Stores.RemoteStore
	provider := options.Providers.RemoteProvider

	if store == nil {
		store = remote.NewMemoryStore()
	}

	if err := registry.RegisterCommand(remote.NewEnableCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(remote.NewDisableCommand(store, provider)); err != nil {
		return err
	}
	if err := registry.RegisterCommand(remote.NewExecCommand(store, provider)); err != nil {
		return err
	}

	if err := registry.RegisterQuery(remote.NewStatusQuery(store)); err != nil {
		return err
	}
	if err := registry.RegisterQuery(remote.NewAuditQuery(store)); err != nil {
		return err
	}

	return nil
}

func RegisterAllWithDefaults(registry *unit.Registry) error {
	return RegisterAll(registry)
}
