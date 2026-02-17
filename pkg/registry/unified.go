package registry

import (
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type UnifiedRegistry struct {
	registry *unit.Registry
	options  *Options
}

func NewUnifiedRegistry() *UnifiedRegistry {
	return &UnifiedRegistry{
		registry: unit.NewRegistry(),
		options:  &Options{},
	}
}

func (r *UnifiedRegistry) RegisterAll() error {
	if err := r.RegisterModelDomain(); err != nil {
		return err
	}
	if err := r.RegisterDeviceDomain(); err != nil {
		return err
	}
	if err := r.RegisterEngineDomain(); err != nil {
		return err
	}
	if err := r.RegisterInferenceDomain(); err != nil {
		return err
	}
	if err := r.RegisterResourceDomain(); err != nil {
		return err
	}
	if err := r.RegisterServiceDomain(); err != nil {
		return err
	}
	if err := r.RegisterAppDomain(); err != nil {
		return err
	}
	if err := r.RegisterPipelineDomain(); err != nil {
		return err
	}
	if err := r.RegisterAlertDomain(); err != nil {
		return err
	}
	if err := r.RegisterRemoteDomain(); err != nil {
		return err
	}
	return nil
}

func (r *UnifiedRegistry) RegisterModelDomain() error {
	return registerModelDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterDeviceDomain() error {
	return registerDeviceDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterEngineDomain() error {
	return registerEngineDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterInferenceDomain() error {
	return registerInferenceDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterResourceDomain() error {
	return registerResourceDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterServiceDomain() error {
	return registerServiceDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterAppDomain() error {
	return registerAppDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterPipelineDomain() error {
	return registerPipelineDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterAlertDomain() error {
	return registerAlertDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) RegisterRemoteDomain() error {
	return registerRemoteDomain(r.registry, r.options)
}

func (r *UnifiedRegistry) Registry() *unit.Registry {
	return r.registry
}

func (r *UnifiedRegistry) WithStores(stores Stores) *UnifiedRegistry {
	r.options.Stores = stores
	return r
}

func (r *UnifiedRegistry) WithProviders(providers Providers) *UnifiedRegistry {
	r.options.Providers = providers
	return r
}
