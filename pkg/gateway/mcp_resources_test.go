package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestMCPAdapter_ListResources(t *testing.T) {
	t.Run("with resources", func(t *testing.T) {
		reg := unit.NewRegistry()
		res := &mockResource{
			uri: "asms://test/resource",
			get: func(ctx context.Context) (any, error) {
				return map[string]any{"data": "test"}, nil
			},
		}
		_ = reg.RegisterResource(res)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		resources := adapter.ListResources()

		if len(resources) != 1 {
			t.Errorf("expected 1 resource, got %d", len(resources))
		}

		if resources[0].URI != "asms://test/resource" {
			t.Errorf("expected URI asms://test/resource, got %s", resources[0].URI)
		}

		if resources[0].MimeType != "application/json" {
			t.Errorf("expected mimeType application/json, got %s", resources[0].MimeType)
		}
	})

	t.Run("with nil gateway", func(t *testing.T) {
		adapter := NewMCPAdapter(nil)
		resources := adapter.ListResources()

		if len(resources) != 0 {
			t.Errorf("expected 0 resources, got %d", len(resources))
		}
	})

	t.Run("with empty registry", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)
		resources := adapter.ListResources()

		if len(resources) != 0 {
			t.Errorf("expected 0 resources, got %d", len(resources))
		}
	})
}

func TestMCPAdapter_handleResourcesList(t *testing.T) {
	reg := unit.NewRegistry()
	res := &mockResource{
		uri: "asms://test/resource",
		get: func(ctx context.Context) (any, error) {
			return map[string]any{"data": "test"}, nil
		},
	}
	_ = reg.RegisterResource(res)

	g := NewGateway(reg)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		ID:      1,
		Method:  "resources/list",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(*MCPResourcesListResult)
	if !ok {
		t.Fatalf("expected MCPResourcesListResult, got %T", resp.Result)
	}

	if len(result.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(result.Resources))
	}
}

func TestMCPAdapter_ReadResource(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		reg := unit.NewRegistry()
		res := &mockResource{
			uri: "asms://test/resource",
			get: func(ctx context.Context) (any, error) {
				return map[string]any{"data": "test", "value": 123}, nil
			},
		}
		_ = reg.RegisterResource(res)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		result, err := adapter.ReadResource(context.Background(), "asms://test/resource")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Contents) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Contents))
		}

		if result.Contents[0].URI != "asms://test/resource" {
			t.Errorf("expected URI asms://test/resource, got %s", result.Contents[0].URI)
		}

		if result.Contents[0].Text == "" {
			t.Error("expected text content")
		}
	})

	t.Run("resource not found", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		_, err := adapter.ReadResource(context.Background(), "asms://unknown/resource")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("with nil gateway", func(t *testing.T) {
		adapter := NewMCPAdapter(nil)

		_, err := adapter.ReadResource(context.Background(), "asms://test/resource")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("resource get error", func(t *testing.T) {
		reg := unit.NewRegistry()
		res := &mockResource{
			uri: "asms://test/error",
			get: func(ctx context.Context) (any, error) {
				return nil, errors.New("resource error")
			},
		}
		_ = reg.RegisterResource(res)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		_, err := adapter.ReadResource(context.Background(), "asms://test/error")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("with bytes data", func(t *testing.T) {
		reg := unit.NewRegistry()
		res := &mockResource{
			uri: "asms://test/bytes",
			get: func(ctx context.Context) (any, error) {
				return []byte("raw bytes data"), nil
			},
		}
		_ = reg.RegisterResource(res)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		result, err := adapter.ReadResource(context.Background(), "asms://test/bytes")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Contents[0].Text != "raw bytes data" {
			t.Errorf("expected text 'raw bytes data', got %s", result.Contents[0].Text)
		}
	})
}

func TestMCPAdapter_handleResourcesRead(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		reg := unit.NewRegistry()
		res := &mockResource{
			uri: "asms://test/resource",
			get: func(ctx context.Context) (any, error) {
				return map[string]any{"data": "test"}, nil
			},
		}
		_ = reg.RegisterResource(res)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "resources/read",
			Params:  json.RawMessage(`{"uri":"asms://test/resource"}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(*MCPResourceReadResult)
		if !ok {
			t.Fatalf("expected MCPResourceReadResult, got %T", resp.Result)
		}

		if len(result.Contents) != 1 {
			t.Errorf("expected 1 content, got %d", len(result.Contents))
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "resources/read",
			Params:  json.RawMessage(`invalid json`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}

		if resp.Error.Code != MCPErrorCodeInvalidParams {
			t.Errorf("expected error code %d, got %d", MCPErrorCodeInvalidParams, resp.Error.Code)
		}
	})

	t.Run("empty URI", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "resources/read",
			Params:  json.RawMessage(`{"uri":""}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("resource not found", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "resources/read",
			Params:  json.RawMessage(`{"uri":"asms://unknown/resource"}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}

		if resp.Error.Code != MCPErrorCodeResourceNotFound {
			t.Errorf("expected error code %d, got %d", MCPErrorCodeResourceNotFound, resp.Error.Code)
		}
	})
}

func TestMCPAdapter_resourceToMCPResource(t *testing.T) {
	reg := unit.NewRegistry()
	g := NewGateway(reg)
	adapter := NewMCPAdapter(g)

	res := &mockResourceWithSchema{
		uri:    "asms://test/resource",
		schema: unit.Schema{Description: "test resource description"},
	}

	mcpRes := adapter.resourceToMCPResource(res)

	if mcpRes.URI != "asms://test/resource" {
		t.Errorf("expected URI asms://test/resource, got %s", mcpRes.URI)
	}

	if mcpRes.Name != "asms://test/resource" {
		t.Errorf("expected name asms://test/resource, got %s", mcpRes.Name)
	}

	if mcpRes.Description != "test resource description" {
		t.Errorf("expected description 'test resource description', got %s", mcpRes.Description)
	}

	if mcpRes.MimeType != "application/json" {
		t.Errorf("expected mimeType application/json, got %s", mcpRes.MimeType)
	}
}

type mockResourceWithSchema struct {
	uri    string
	schema unit.Schema
	get    func(ctx context.Context) (any, error)
}

func (m *mockResourceWithSchema) URI() string         { return m.uri }
func (m *mockResourceWithSchema) Domain() string      { return "test" }
func (m *mockResourceWithSchema) Schema() unit.Schema { return m.schema }
func (m *mockResourceWithSchema) Get(ctx context.Context) (any, error) {
	if m.get != nil {
		return m.get(ctx)
	}
	return nil, nil
}
func (m *mockResourceWithSchema) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate)
	close(ch)
	return ch, nil
}
