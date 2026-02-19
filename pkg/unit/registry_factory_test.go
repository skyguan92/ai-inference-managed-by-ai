package unit

import (
	"errors"
	"strings"
	"testing"
)

// mockResourceFactory implements ResourceFactory for testing
type mockResourceFactory struct {
	pattern   string
	canCreate bool
	createRes Resource
	createErr error
}

func (m *mockResourceFactory) CanCreate(uri string) bool {
	if !m.canCreate {
		return false
	}
	prefix := strings.TrimSuffix(m.pattern, "*")
	return strings.HasPrefix(uri, prefix)
}
func (m *mockResourceFactory) Create(uri string) (Resource, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createRes != nil {
		return m.createRes, nil
	}
	return &regTestResource{uri: uri, domain: "test"}, nil
}
func (m *mockResourceFactory) Pattern() string { return m.pattern }

func TestRegisterResourceFactory(t *testing.T) {
	r := NewRegistry()
	factory := &mockResourceFactory{pattern: "asms://model/*", canCreate: true}

	err := r.RegisterResourceFactory(factory)
	if err != nil {
		t.Errorf("RegisterResourceFactory failed: %v", err)
	}

	err = r.RegisterResourceFactory(nil)
	if err != ErrResourceNotFound {
		t.Errorf("expected ErrResourceNotFound for nil factory, got %v", err)
	}
}

func TestGetResourceWithFactory(t *testing.T) {
	r := NewRegistry()

	// First test: directly registered resource
	res := &regTestResource{uri: "asms://model/direct", domain: "model"}
	_ = r.RegisterResource(res)

	got := r.GetResourceWithFactory("asms://model/direct")
	if got == nil {
		t.Error("GetResourceWithFactory should return directly registered resource")
	}
	if got.URI() != "asms://model/direct" {
		t.Errorf("expected URI 'asms://model/direct', got %q", got.URI())
	}

	// Second test: factory creates resource
	factory := &mockResourceFactory{pattern: "asms://model/*", canCreate: true}
	_ = r.RegisterResourceFactory(factory)

	got = r.GetResourceWithFactory("asms://model/dynamic")
	if got == nil {
		t.Error("GetResourceWithFactory should create resource via factory")
	}
	if got.URI() != "asms://model/dynamic" {
		t.Errorf("expected URI 'asms://model/dynamic', got %q", got.URI())
	}

	// Third test: not found in direct resources or factories
	got = r.GetResourceWithFactory("asms://nonexistent/abc")
	if got != nil {
		t.Error("GetResourceWithFactory should return nil for unknown URI")
	}
}

func TestGetResourceWithFactory_CannotCreate(t *testing.T) {
	r := NewRegistry()
	factory := &mockResourceFactory{pattern: "asms://model/*", canCreate: false}
	_ = r.RegisterResourceFactory(factory)

	got := r.GetResourceWithFactory("asms://model/dynamic")
	if got != nil {
		t.Error("GetResourceWithFactory should return nil when factory cannot create resource")
	}
}

func TestGetResourceWithFactory_FactoryError(t *testing.T) {
	r := NewRegistry()
	factory := &mockResourceFactory{
		pattern:   "asms://model/*",
		canCreate: true,
		createErr: errors.New("factory error"),
	}
	_ = r.RegisterResourceFactory(factory)

	got := r.GetResourceWithFactory("asms://model/dynamic")
	if got != nil {
		t.Error("GetResourceWithFactory should return nil when factory returns error")
	}
}

func TestGetResourceWithFactory_FactoryReturnsNil(t *testing.T) {
	r := NewRegistry()
	// Register a factory that returns nil resource
	_ = r.RegisterResourceFactory(&nilReturnFactory{})

	got := r.GetResourceWithFactory("asms://model/dynamic")
	if got != nil {
		t.Error("GetResourceWithFactory should return nil when factory returns nil resource")
	}
}

// nilReturnFactory always returns nil resource without error
type nilReturnFactory struct{}

func (f *nilReturnFactory) CanCreate(uri string) bool { return true }
func (f *nilReturnFactory) Create(uri string) (Resource, error) {
	return nil, nil
}
func (f *nilReturnFactory) Pattern() string { return "asms://*" }

func TestListResourceFactories(t *testing.T) {
	r := NewRegistry()

	list := r.ListResourceFactories()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	factory1 := &mockResourceFactory{pattern: "asms://model/*", canCreate: true}
	factory2 := &mockResourceFactory{pattern: "asms://engine/*", canCreate: true}

	_ = r.RegisterResourceFactory(factory1)
	_ = r.RegisterResourceFactory(factory2)

	list = r.ListResourceFactories()
	if len(list) != 2 {
		t.Errorf("expected 2 factories, got %d", len(list))
	}
}

func TestGetResourceWithFactory_MultipleFactories(t *testing.T) {
	r := NewRegistry()

	// First factory cannot handle the URI
	factory1 := &mockResourceFactory{pattern: "asms://engine/*", canCreate: false}
	// Second factory can handle the URI
	factory2 := &mockResourceFactory{
		pattern:   "asms://model/*",
		canCreate: true,
		createRes: &regTestResource{uri: "asms://model/test", domain: "model"},
	}

	_ = r.RegisterResourceFactory(factory1)
	_ = r.RegisterResourceFactory(factory2)

	got := r.GetResourceWithFactory("asms://model/test")
	if got == nil {
		t.Error("GetResourceWithFactory should return resource from second factory")
	}
}

// Ensure regTestResource from registry_test.go is accessible
var _ Resource = (*regTestResource)(nil)
