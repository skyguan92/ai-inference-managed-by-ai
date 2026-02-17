package gateway

import (
	"context"
	"testing"

	registrypkg "github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayIntegration(t *testing.T) {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := NewGateway(registry)

	t.Run("model.list query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "model.list",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "model.list should succeed")
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Data)
		assert.NotNil(t, resp.Meta)
		assert.NotEmpty(t, resp.Meta.RequestID)
	})

	t.Run("device.detect command", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeCommand,
			Unit:  "device.detect",
			Input: map[string]any{},
		})
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Meta)
	})

	t.Run("engine.list query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "engine.list",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "engine.list should succeed")
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Data)
	})

	t.Run("inference.models query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "inference.models",
			Input: map[string]any{},
		})
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Meta)
	})

	t.Run("resource.status query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "resource.status",
			Input: map[string]any{},
		})
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Meta)
	})

	t.Run("service.list query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "service.list",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "service.list should succeed")
		assert.Nil(t, resp.Error)
	})

	t.Run("app.list query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "app.list",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "app.list should succeed")
		assert.Nil(t, resp.Error)
	})

	t.Run("pipeline.list query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "pipeline.list",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "pipeline.list should succeed")
		assert.Nil(t, resp.Error)
	})

	t.Run("alert.list_rules query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "alert.list_rules",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "alert.list_rules should succeed")
		assert.Nil(t, resp.Error)
	})

	t.Run("remote.status query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "remote.status",
			Input: map[string]any{},
		})
		assert.True(t, resp.Success, "remote.status should succeed")
		assert.Nil(t, resp.Error)
	})
}

func TestGatewayIntegrationInvalidRequests(t *testing.T) {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := NewGateway(registry)

	t.Run("nil request", func(t *testing.T) {
		resp := gw.Handle(context.Background(), nil)
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
	})

	t.Run("empty unit name", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeCommand,
			Unit:  "",
			Input: map[string]any{},
		})
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
	})

	t.Run("invalid type", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  "invalid",
			Unit:  "model.list",
			Input: map[string]any{},
		})
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
	})

	t.Run("non-existent command", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeCommand,
			Unit:  "nonexistent.command",
			Input: map[string]any{},
		})
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeUnitNotFound, resp.Error.Code)
	})

	t.Run("non-existent query", func(t *testing.T) {
		resp := gw.Handle(context.Background(), &Request{
			Type:  TypeQuery,
			Unit:  "nonexistent.query",
			Input: map[string]any{},
		})
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeUnitNotFound, resp.Error.Code)
	})
}

func TestGatewayIntegrationWithTraceID(t *testing.T) {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := NewGateway(registry)

	traceID := "test-trace-123"
	resp := gw.Handle(context.Background(), &Request{
		Type:  TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
		Options: RequestOptions{
			TraceID: traceID,
		},
	})

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, traceID, resp.Meta.TraceID)
}

func TestGatewayIntegrationMetadata(t *testing.T) {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := NewGateway(registry)

	resp := gw.Handle(context.Background(), &Request{
		Type:  TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	})

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	assert.NotEmpty(t, resp.Meta.RequestID)
	assert.NotEmpty(t, resp.Meta.TraceID)
	assert.GreaterOrEqual(t, resp.Meta.Duration, int64(0))
}

func TestGatewayIntegrationAllCommands(t *testing.T) {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := NewGateway(registry)

	commands := registry.ListCommands()
	assert.Greater(t, len(commands), 0)

	for _, cmd := range commands {
		t.Run("command/"+cmd.Name(), func(t *testing.T) {
			resp := gw.Handle(context.Background(), &Request{
				Type:  TypeCommand,
				Unit:  cmd.Name(),
				Input: map[string]any{},
			})
			assert.NotNil(t, resp)
			assert.NotNil(t, resp.Meta)
		})
	}
}

func TestGatewayIntegrationAllQueries(t *testing.T) {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := NewGateway(registry)

	queries := registry.ListQueries()
	assert.Greater(t, len(queries), 0)

	for _, q := range queries {
		t.Run("query/"+q.Name(), func(t *testing.T) {
			resp := gw.Handle(context.Background(), &Request{
				Type:  TypeQuery,
				Unit:  q.Name(),
				Input: map[string]any{},
			})
			assert.NotNil(t, resp)
			assert.NotNil(t, resp.Meta)
		})
	}
}
