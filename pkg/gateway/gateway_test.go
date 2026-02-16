package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type mockCommand struct {
	name    string
	domain  string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockCommand) Name() string              { return m.name }
func (m *mockCommand) Domain() string            { return m.domain }
func (m *mockCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockCommand) Execute(ctx context.Context, input any) (any, error) {
	if m.execute != nil {
		return m.execute(ctx, input)
	}
	return map[string]any{"success": true}, nil
}
func (m *mockCommand) Description() string      { return "mock command" }
func (m *mockCommand) Examples() []unit.Example { return nil }

type mockQuery struct {
	name    string
	domain  string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockQuery) Name() string              { return m.name }
func (m *mockQuery) Domain() string            { return m.domain }
func (m *mockQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockQuery) Execute(ctx context.Context, input any) (any, error) {
	if m.execute != nil {
		return m.execute(ctx, input)
	}
	return map[string]any{"result": "ok"}, nil
}
func (m *mockQuery) Description() string      { return "mock query" }
func (m *mockQuery) Examples() []unit.Example { return nil }

type mockResource struct {
	uri   string
	get   func(ctx context.Context) (any, error)
	watch func(ctx context.Context) (<-chan unit.ResourceUpdate, error)
}

func (m *mockResource) URI() string         { return m.uri }
func (m *mockResource) Domain() string      { return "test" }
func (m *mockResource) Schema() unit.Schema { return unit.Schema{} }
func (m *mockResource) Get(ctx context.Context) (any, error) {
	if m.get != nil {
		return m.get(ctx)
	}
	return map[string]any{"data": "resource"}, nil
}
func (m *mockResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	if m.watch != nil {
		return m.watch(ctx)
	}
	ch := make(chan unit.ResourceUpdate)
	close(ch)
	return ch, nil
}

func TestNewGateway(t *testing.T) {
	t.Run("with nil registry", func(t *testing.T) {
		g := NewGateway(nil)
		if g == nil {
			t.Fatal("expected gateway, got nil")
		}
		if g.registry == nil {
			t.Error("expected registry to be initialized")
		}
		if g.requestTimeout != DefaultTimeout {
			t.Errorf("expected timeout %v, got %v", DefaultTimeout, g.requestTimeout)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		customTimeout := 10 * time.Second
		g := NewGateway(nil, WithTimeout(customTimeout))
		if g.requestTimeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, g.requestTimeout)
		}
	})

	t.Run("with provided registry", func(t *testing.T) {
		reg := unit.NewRegistry()
		g := NewGateway(reg)
		if g.registry != reg {
			t.Error("expected same registry instance")
		}
	})
}

func TestHandle_Validation(t *testing.T) {
	g := NewGateway(nil)

	t.Run("nil request", func(t *testing.T) {
		resp := g.Handle(context.Background(), nil)
		if resp.Success {
			t.Error("expected failure for nil request")
		}
		if resp.Error == nil {
			t.Fatal("expected error")
		}
		if resp.Error.Code != ErrCodeInvalidRequest {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidRequest, resp.Error.Code)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: "",
			Unit: "test.unit",
		})
		if resp.Success {
			t.Error("expected failure for empty type")
		}
		if resp.Error.Code != ErrCodeInvalidRequest {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidRequest, resp.Error.Code)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: "invalid",
			Unit: "test.unit",
		})
		if resp.Success {
			t.Error("expected failure for invalid type")
		}
		if resp.Error.Code != ErrCodeInvalidRequest {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidRequest, resp.Error.Code)
		}
	})

	t.Run("empty unit", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: TypeCommand,
			Unit: "",
		})
		if resp.Success {
			t.Error("expected failure for empty unit")
		}
		if resp.Error.Code != ErrCodeInvalidRequest {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidRequest, resp.Error.Code)
		}
	})
}

func TestHandle_Command(t *testing.T) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.ping", domain: "test"}
	if err := reg.RegisterCommand(cmd); err != nil {
		t.Fatalf("failed to register command: %v", err)
	}
	g := NewGateway(reg)

	t.Run("command found", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type:  TypeCommand,
			Unit:  "test.ping",
			Input: map[string]any{"msg": "hello"},
		})
		if !resp.Success {
			t.Errorf("expected success, got error: %v", resp.Error)
		}
		if resp.Data == nil {
			t.Error("expected data")
		}
		if resp.Meta == nil {
			t.Fatal("expected meta")
		}
		if resp.Meta.RequestID == "" {
			t.Error("expected request_id")
		}
		if resp.Meta.TraceID == "" {
			t.Error("expected trace_id")
		}
		if resp.Meta.Duration < 0 {
			t.Error("expected non-negative duration")
		}
	})

	t.Run("command not found", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: TypeCommand,
			Unit: "unknown.command",
		})
		if resp.Success {
			t.Error("expected failure for unknown command")
		}
		if resp.Error.Code != ErrCodeUnitNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeUnitNotFound, resp.Error.Code)
		}
	})

	t.Run("command execution error", func(t *testing.T) {
		errorCmd := &mockCommand{
			name:   "test.error",
			domain: "test",
			execute: func(ctx context.Context, input any) (any, error) {
				return nil, errors.New("execution failed")
			},
		}
		reg.RegisterCommand(errorCmd)

		resp := g.Handle(context.Background(), &Request{
			Type: TypeCommand,
			Unit: "test.error",
		})
		if resp.Success {
			t.Error("expected failure")
		}
		if resp.Error.Code != ErrCodeExecutionFailed {
			t.Errorf("expected code %s, got %s", ErrCodeExecutionFailed, resp.Error.Code)
		}
	})
}

func TestHandle_Query(t *testing.T) {
	reg := unit.NewRegistry()
	q := &mockQuery{name: "test.get", domain: "test"}
	if err := reg.RegisterQuery(q); err != nil {
		t.Fatalf("failed to register query: %v", err)
	}
	g := NewGateway(reg)

	t.Run("query found", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "test.get",
			Input: map[string]any{"id": "123"},
		})
		if !resp.Success {
			t.Errorf("expected success, got error: %v", resp.Error)
		}
		if resp.Data == nil {
			t.Error("expected data")
		}
	})

	t.Run("query not found", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: TypeQuery,
			Unit: "unknown.query",
		})
		if resp.Success {
			t.Error("expected failure for unknown query")
		}
		if resp.Error.Code != ErrCodeUnitNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeUnitNotFound, resp.Error.Code)
		}
	})

	t.Run("query execution error", func(t *testing.T) {
		errorQuery := &mockQuery{
			name:   "test.query_error",
			domain: "test",
			execute: func(ctx context.Context, input any) (any, error) {
				return nil, errors.New("query failed")
			},
		}
		reg.RegisterQuery(errorQuery)

		resp := g.Handle(context.Background(), &Request{
			Type: TypeQuery,
			Unit: "test.query_error",
		})
		if resp.Success {
			t.Error("expected failure")
		}
		if resp.Error.Code != ErrCodeExecutionFailed {
			t.Errorf("expected code %s, got %s", ErrCodeExecutionFailed, resp.Error.Code)
		}
	})
}

func TestHandle_Resource(t *testing.T) {
	reg := unit.NewRegistry()
	res := &mockResource{uri: "asms://test/resource"}
	if err := reg.RegisterResource(res); err != nil {
		t.Fatalf("failed to register resource: %v", err)
	}
	g := NewGateway(reg)

	t.Run("resource found", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: TypeResource,
			Unit: "asms://test/resource",
		})
		if !resp.Success {
			t.Errorf("expected success, got error: %v", resp.Error)
		}
		if resp.Data == nil {
			t.Error("expected data")
		}
	})

	t.Run("resource not found", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: TypeResource,
			Unit: "asms://unknown/resource",
		})
		if resp.Success {
			t.Error("expected failure for unknown resource")
		}
		if resp.Error.Code != ErrCodeResourceNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeResourceNotFound, resp.Error.Code)
		}
	})

	t.Run("resource get error", func(t *testing.T) {
		errorRes := &mockResource{
			uri: "asms://test/error",
			get: func(ctx context.Context) (any, error) {
				return nil, errors.New("resource error")
			},
		}
		reg.RegisterResource(errorRes)

		resp := g.Handle(context.Background(), &Request{
			Type: TypeResource,
			Unit: "asms://test/error",
		})
		if resp.Success {
			t.Error("expected failure")
		}
		if resp.Error.Code != ErrCodeExecutionFailed {
			t.Errorf("expected code %s, got %s", ErrCodeExecutionFailed, resp.Error.Code)
		}
	})
}

func TestHandle_Timeout(t *testing.T) {
	reg := unit.NewRegistry()
	slowCmd := &mockCommand{
		name:   "test.slow",
		domain: "test",
		execute: func(ctx context.Context, input any) (any, error) {
			select {
			case <-time.After(5 * time.Second):
				return "done", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	reg.RegisterCommand(slowCmd)

	g := NewGateway(reg, WithTimeout(100*time.Millisecond))

	resp := g.Handle(context.Background(), &Request{
		Type: TypeCommand,
		Unit: "test.slow",
		Options: RequestOptions{
			Timeout: 50 * time.Millisecond,
		},
	})

	if resp.Success {
		t.Error("expected failure due to timeout")
	}
}

func TestHandle_CustomTimeout(t *testing.T) {
	reg := unit.NewRegistry()
	fastCmd := &mockCommand{
		name:   "test.fast",
		domain: "test",
		execute: func(ctx context.Context, input any) (any, error) {
			return "fast", nil
		},
	}
	reg.RegisterCommand(fastCmd)

	g := NewGateway(reg)

	resp := g.Handle(context.Background(), &Request{
		Type: TypeCommand,
		Unit: "test.fast",
		Options: RequestOptions{
			Timeout: 1 * time.Second,
		},
	})

	if !resp.Success {
		t.Errorf("expected success, got error: %v", resp.Error)
	}
}

func TestHandle_TraceID(t *testing.T) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.trace", domain: "test"}
	reg.RegisterCommand(cmd)
	g := NewGateway(reg)

	t.Run("custom trace_id", func(t *testing.T) {
		customTraceID := "custom_trace_123"
		resp := g.Handle(context.Background(), &Request{
			Type: TypeCommand,
			Unit: "test.trace",
			Options: RequestOptions{
				TraceID: customTraceID,
			},
		})
		if !resp.Success {
			t.Errorf("expected success, got error: %v", resp.Error)
		}
		if resp.Meta.TraceID != customTraceID {
			t.Errorf("expected trace_id %s, got %s", customTraceID, resp.Meta.TraceID)
		}
	})

	t.Run("auto generated trace_id", func(t *testing.T) {
		resp := g.Handle(context.Background(), &Request{
			Type: TypeCommand,
			Unit: "test.trace",
		})
		if !resp.Success {
			t.Errorf("expected success, got error: %v", resp.Error)
		}
		if resp.Meta.TraceID == "" {
			t.Error("expected auto-generated trace_id")
		}
	})
}

func TestHandle_Workflow(t *testing.T) {
	g := NewGateway(nil)

	resp := g.Handle(context.Background(), &Request{
		Type: TypeWorkflow,
		Unit: "test.workflow",
	})

	if resp.Success {
		t.Error("expected failure for workflow (not implemented)")
	}
	if resp.Error.Code != ErrCodeInternalError {
		t.Errorf("expected code %s, got %s", ErrCodeInternalError, resp.Error.Code)
	}
}

func TestGateway_Accessors(t *testing.T) {
	reg := unit.NewRegistry()
	g := NewGateway(reg, WithTimeout(5*time.Second))

	t.Run("Registry", func(t *testing.T) {
		if g.Registry() != reg {
			t.Error("expected same registry")
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		if g.Timeout() != 5*time.Second {
			t.Errorf("expected 5s timeout, got %v", g.Timeout())
		}
	})
}

func TestRequest_Options(t *testing.T) {
	req := &Request{
		Type:  TypeCommand,
		Unit:  "test.unit",
		Input: map[string]any{"key": "value"},
		Options: RequestOptions{
			Timeout: 10 * time.Second,
			Async:   true,
			TraceID: "trace123",
		},
	}

	if req.Options.Timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", req.Options.Timeout)
	}
	if !req.Options.Async {
		t.Error("expected async to be true")
	}
	if req.Options.TraceID != "trace123" {
		t.Errorf("expected trace123, got %s", req.Options.TraceID)
	}
}

func TestResponse_Pagination(t *testing.T) {
	resp := &Response{
		Success: true,
		Data:    []string{"a", "b", "c"},
		Meta: &ResponseMeta{
			RequestID: "req123",
			Duration:  100,
			Pagination: &Pagination{
				Page:    1,
				PerPage: 10,
				Total:   100,
			},
		},
	}

	if resp.Meta.Pagination.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Meta.Pagination.Page)
	}
	if resp.Meta.Pagination.Total != 100 {
		t.Errorf("expected total 100, got %d", resp.Meta.Pagination.Total)
	}
}
