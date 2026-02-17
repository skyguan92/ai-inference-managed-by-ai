// Package unit defines the core atomic unit interfaces and registry for AIMA.
package unit

import (
	"errors"
	"sync"
)

var (
	ErrCommandAlreadyRegistered  = errors.New("command already registered")
	ErrQueryAlreadyRegistered    = errors.New("query already registered")
	ErrResourceAlreadyRegistered = errors.New("resource already registered")
	ErrFactoryAlreadyRegistered  = errors.New("resource factory already registered")
	ErrCommandNotFound           = errors.New("command not found")
	ErrQueryNotFound             = errors.New("query not found")
	ErrResourceNotFound          = errors.New("resource not found")
)

// Registry is the central registry for all atomic units (Commands, Queries, Resources).
// It provides thread-safe registration and lookup operations.
type Registry struct {
	commands          map[string]Command
	queries           map[string]Query
	resources         map[string]Resource
	resourceFactories []ResourceFactory
	mu                sync.RWMutex
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		commands:          make(map[string]Command),
		queries:           make(map[string]Query),
		resources:         make(map[string]Resource),
		resourceFactories: make([]ResourceFactory, 0),
	}
}

// RegisterCommand registers a Command with the registry.
// Returns ErrCommandAlreadyRegistered if a command with the same name exists.
// Returns ErrCommandNotFound if cmd is nil.
func (r *Registry) RegisterCommand(cmd Command) error {
	if cmd == nil {
		return ErrCommandNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := cmd.Name()
	if _, exists := r.commands[name]; exists {
		return ErrCommandAlreadyRegistered
	}

	r.commands[name] = cmd
	return nil
}

// RegisterQuery registers a Query with the registry.
// Returns ErrQueryAlreadyRegistered if a query with the same name exists.
// Returns ErrQueryNotFound if q is nil.
func (r *Registry) RegisterQuery(q Query) error {
	if q == nil {
		return ErrQueryNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := q.Name()
	if _, exists := r.queries[name]; exists {
		return ErrQueryAlreadyRegistered
	}

	r.queries[name] = q
	return nil
}

// RegisterResource registers a Resource with the registry.
// Returns ErrResourceAlreadyRegistered if a resource with the same URI exists.
// Returns ErrResourceNotFound if res is nil.
func (r *Registry) RegisterResource(res Resource) error {
	if res == nil {
		return ErrResourceNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	uri := res.URI()
	if _, exists := r.resources[uri]; exists {
		return ErrResourceAlreadyRegistered
	}

	r.resources[uri] = res
	return nil
}

// RegisterResourceFactory registers a ResourceFactory with the registry.
// It is used to handle dynamic URI patterns like asms://model/{id}.
func (r *Registry) RegisterResourceFactory(factory ResourceFactory) error {
	if factory == nil {
		return ErrResourceNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.resourceFactories = append(r.resourceFactories, factory)
	return nil
}

// GetCommand retrieves a Command by name. Returns nil if not found.
func (r *Registry) GetCommand(name string) Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.commands[name]
}

// GetQuery retrieves a Query by name. Returns nil if not found.
func (r *Registry) GetQuery(name string) Query {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.queries[name]
}

// GetResource retrieves a Resource by URI. If not found directly, it tries
// to create a Resource using registered ResourceFactory instances.
// Returns nil if not found in any collection or factory.
func (r *Registry) GetResource(uri string) Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First, try to find directly registered resource
	if res, ok := r.resources[uri]; ok {
		return res
	}

	// Return nil - caller should use GetResourceWithFactory for factory creation
	return nil
}

// GetResourceWithFactory tries to create a Resource using registered factories.
// It first checks directly registered resources, then tries each factory.
func (r *Registry) GetResourceWithFactory(uri string) Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First, try to find directly registered resource
	if res, ok := r.resources[uri]; ok {
		return res
	}

	// Try each factory
	for _, factory := range r.resourceFactories {
		if factory.CanCreate(uri) {
			res, err := factory.Create(uri)
			if err == nil && res != nil {
				return res
			}
		}
	}

	return nil
}

// ListCommands returns all registered Commands.
func (r *Registry) ListCommands() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
	}
	return result
}

// ListQueries returns all registered Queries.
func (r *Registry) ListQueries() []Query {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Query, 0, len(r.queries))
	for _, q := range r.queries {
		result = append(result, q)
	}
	return result
}

// ListResources returns all registered Resources.
func (r *Registry) ListResources() []Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Resource, 0, len(r.resources))
	for _, res := range r.resources {
		result = append(result, res)
	}
	return result
}

// ListResourceFactories returns all registered ResourceFactories.
func (r *Registry) ListResourceFactories() []ResourceFactory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ResourceFactory, 0, len(r.resourceFactories))
	result = append(result, r.resourceFactories...)
	return result
}

// Get retrieves an atomic unit by name/URI. It searches Commands, Queries, and Resources
// in that order. Returns nil if not found in any collection.
func (r *Registry) Get(name string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cmd, ok := r.commands[name]; ok {
		return cmd
	}

	if q, ok := r.queries[name]; ok {
		return q
	}

	if res, ok := r.resources[name]; ok {
		return res
	}

	return nil
}

// UnregisterCommand removes a Command by name.
// Returns true if the command was found and removed, false otherwise.
func (r *Registry) UnregisterCommand(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[name]; exists {
		delete(r.commands, name)
		return true
	}
	return false
}

// UnregisterQuery removes a Query by name.
// Returns true if the query was found and removed, false otherwise.
func (r *Registry) UnregisterQuery(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.queries[name]; exists {
		delete(r.queries, name)
		return true
	}
	return false
}

// UnregisterResource removes a Resource by URI.
// Returns true if the resource was found and removed, false otherwise.
func (r *Registry) UnregisterResource(uri string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[uri]; exists {
		delete(r.resources, uri)
		return true
	}
	return false
}

// CommandCount returns the number of registered Commands.
func (r *Registry) CommandCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.commands)
}

// QueryCount returns the number of registered Queries.
func (r *Registry) QueryCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.queries)
}

// ResourceCount returns the number of registered Resources.
func (r *Registry) ResourceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.resources)
}
