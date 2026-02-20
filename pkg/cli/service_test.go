package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func makeServiceRoot(t *testing.T) *RootCommand {
	t.Helper()
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	return &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}
}

func TestNewServiceCommand(t *testing.T) {
	root := makeServiceRoot(t)
	cmd := NewServiceCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "service", cmd.Use)

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 5)

	names := make([]string, len(subCommands))
	for i, c := range subCommands {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "create")
	assert.Contains(t, names, "start")
	assert.Contains(t, names, "stop")
	assert.Contains(t, names, "status")
	assert.Contains(t, names, "list")
}

func TestNewServiceCreateCommand_Flags(t *testing.T) {
	root := makeServiceRoot(t)
	cmd := NewServiceCreateCommand(root)
	assert.NotNil(t, cmd)

	modelFlag := cmd.Flags().Lookup("model")
	assert.NotNil(t, modelFlag)
	assert.Equal(t, "m", modelFlag.Shorthand)

	deviceFlag := cmd.Flags().Lookup("device")
	assert.NotNil(t, deviceFlag)
	assert.Equal(t, "gpu", deviceFlag.DefValue)

	portFlag := cmd.Flags().Lookup("port")
	assert.NotNil(t, portFlag)

	gpuLayersFlag := cmd.Flags().Lookup("gpu-layers")
	assert.NotNil(t, gpuLayersFlag)
}

func TestNewServiceStartCommand_Flags(t *testing.T) {
	root := makeServiceRoot(t)
	cmd := NewServiceStartCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "start <service-id>", cmd.Use)

	waitFlag := cmd.Flags().Lookup("wait")
	assert.NotNil(t, waitFlag)
	assert.Equal(t, "w", waitFlag.Shorthand)

	asyncFlag := cmd.Flags().Lookup("async")
	assert.NotNil(t, asyncFlag)
	assert.Equal(t, "a", asyncFlag.Shorthand)

	timeoutFlag := cmd.Flags().Lookup("timeout")
	assert.NotNil(t, timeoutFlag)
}

func TestNewServiceStopCommand_Flags(t *testing.T) {
	root := makeServiceRoot(t)
	cmd := NewServiceStopCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "stop <service-id>", cmd.Use)

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
}

func TestNewServiceStatusCommand(t *testing.T) {
	root := makeServiceRoot(t)
	cmd := NewServiceStatusCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "status <service-id>", cmd.Use)
}

func TestNewServiceListCommand_Flags(t *testing.T) {
	root := makeServiceRoot(t)
	cmd := NewServiceListCommand(root)
	assert.NotNil(t, cmd)

	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "s", statusFlag.Shorthand)

	modelFlag := cmd.Flags().Lookup("model")
	assert.NotNil(t, modelFlag)
	assert.Equal(t, "m", modelFlag.Shorthand)
}

func TestRunServiceCreate_ModelNotFound(t *testing.T) {
	root := makeServiceRoot(t)
	// No model.get query registered, so model lookup fails
	err := runServiceCreate(context.Background(), root, "my-svc", "model-123", "gpu", 0, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}

func TestRunServiceCreate_ServiceCreated(t *testing.T) {
	registry := unit.NewRegistry()
	// Register model.get to succeed
	_ = registry.RegisterQuery(&testServiceQuery{
		name: "model.get",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"id": "model-123", "name": "llama3"}, nil
		},
	})
	// Register service.create to succeed
	_ = registry.RegisterCommand(&testServiceCommand{
		name: "service.create",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"service_id": "svc-001", "status": "created"}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceCreate(context.Background(), root, "my-svc", "model-123", "gpu", 8080, 20)
	require.NoError(t, err)
}

func TestRunServiceCreate_CPUDevice(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testServiceQuery{
		name: "model.get",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"id": "model-123"}, nil
		},
	})
	_ = registry.RegisterCommand(&testServiceCommand{
		name: "service.create",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			// CPU device should result in medium resource class
			assert.EqualValues(t, "medium", inputMap["resource_class"])
			return map[string]any{"service_id": "svc-001"}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceCreate(context.Background(), root, "", "model-123", "cpu", 0, -1)
	require.NoError(t, err)
}

func TestRunServiceCreate_ServiceFails(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testServiceQuery{
		name: "model.get",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"id": "model-123"}, nil
		},
	})
	// No service.create registered -> will fail

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceCreate(context.Background(), root, "", "model-123", "gpu", 0, -1)
	require.Error(t, err)
}

func TestRunServiceStart_Success(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testServiceCommand{
		name: "service.start",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"status": "running"}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceStart(context.Background(), root, "svc-001", true, 60, false)
	require.NoError(t, err)
}

func TestRunServiceStart_Async(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testServiceCommand{
		name: "service.start",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"status": "starting"}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceStart(context.Background(), root, "svc-001", false, 60, true)
	require.NoError(t, err)
}

func TestRunServiceStart_Fails(t *testing.T) {
	root := makeServiceRoot(t)
	// No service.start registered
	err := runServiceStart(context.Background(), root, "svc-001", true, 60, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start service failed")
}

func TestRunServiceStop_Success(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testServiceCommand{
		name: "service.stop",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceStop(context.Background(), root, "svc-001", false)
	require.NoError(t, err)
}

func TestRunServiceStop_Force(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&testServiceCommand{
		name: "service.stop",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			assert.Equal(t, true, inputMap["force"])
			return map[string]any{"success": true}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceStop(context.Background(), root, "svc-001", true)
	require.NoError(t, err)
}

func TestRunServiceStop_Fails(t *testing.T) {
	root := makeServiceRoot(t)
	err := runServiceStop(context.Background(), root, "svc-001", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stop service failed")
}

func TestRunServiceStatus_Success(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testServiceQuery{
		name: "service.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"status": "running", "health": "healthy"}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceStatus(context.Background(), root, "svc-001")
	require.NoError(t, err)
}

func TestRunServiceStatus_Fails(t *testing.T) {
	root := makeServiceRoot(t)
	err := runServiceStatus(context.Background(), root, "svc-001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check status failed")
}

func TestRunServiceList_Success(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testServiceQuery{
		name: "service.list",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"services": []map[string]any{
					{"id": "svc-001", "status": "running"},
				},
				"total": 1,
			}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceList(context.Background(), root, "", "")
	require.NoError(t, err)
}

func TestRunServiceList_WithFilters(t *testing.T) {
	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&testServiceQuery{
		name: "service.list",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			assert.Equal(t, "running", inputMap["status"])
			assert.Equal(t, "model-123", inputMap["model_id"])
			return map[string]any{"services": []map[string]any{}, "total": 0}, nil
		},
	})

	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runServiceList(context.Background(), root, "running", "model-123")
	require.NoError(t, err)
}

func TestRunServiceList_Fails(t *testing.T) {
	root := makeServiceRoot(t)
	err := runServiceList(context.Background(), root, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list services failed")
}

// Test helper types for service CLI tests

type testServiceCommand struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *testServiceCommand) Name() string              { return m.name }
func (m *testServiceCommand) Domain() string            { return "service" }
func (m *testServiceCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *testServiceCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *testServiceCommand) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}
func (m *testServiceCommand) Description() string      { return "" }
func (m *testServiceCommand) Examples() []unit.Example { return nil }

type testServiceQuery struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *testServiceQuery) Name() string              { return m.name }
func (m *testServiceQuery) Domain() string            { return "service" }
func (m *testServiceQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *testServiceQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *testServiceQuery) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}
func (m *testServiceQuery) Description() string      { return "" }
func (m *testServiceQuery) Examples() []unit.Example { return nil }
