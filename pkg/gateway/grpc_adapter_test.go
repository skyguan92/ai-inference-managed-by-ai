package gateway

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway/proto/pb"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGRPCAdapter_NewGRPCAdapter(t *testing.T) {
	registry := unit.NewRegistry()
	gateway := NewGateway(registry)

	t.Run("default options", func(t *testing.T) {
		adapter := NewGRPCAdapter(gateway)
		assert.NotNil(t, adapter)
		assert.NotNil(t, adapter.Server())
		assert.Equal(t, gateway, adapter.Gateway())
		assert.Equal(t, ":9091", adapter.options.Address)
		assert.Equal(t, 100*1024*1024, adapter.options.MaxMessageSize)
	})

	t.Run("with custom address", func(t *testing.T) {
		adapter := NewGRPCAdapter(gateway, WithAddress(":9092"))
		assert.Equal(t, ":9092", adapter.options.Address)
	})

	t.Run("with max message size", func(t *testing.T) {
		adapter := NewGRPCAdapter(gateway, WithMaxMessageSize(50*1024*1024))
		assert.Equal(t, 50*1024*1024, adapter.options.MaxMessageSize)
	})

	t.Run("with TLS", func(t *testing.T) {
		adapter := NewGRPCAdapter(gateway, WithTLS("cert.pem", "key.pem"))
		assert.True(t, adapter.options.EnableTLS)
		assert.Equal(t, "cert.pem", adapter.options.TLSCertFile)
		assert.Equal(t, "key.pem", adapter.options.TLSKeyFile)
	})
}

func TestGRPCAdapter_Execute(t *testing.T) {
	// Create test registry with mock command
	registry := unit.NewRegistry()

	testCmd := &testAdapterCommand{
		name:   "test.command",
		domain: "test",
		executeFunc: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			return map[string]any{
				"result": "success",
				"value":  inputMap["value"],
			}, nil
		},
	}
	_ = registry.RegisterCommand(testCmd)

	gateway := NewGateway(registry)
	adapter := NewGRPCAdapter(gateway)

	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		input := map[string]any{"value": 42}
		resp, err := adapter.Execute(ctx, TypeCommand, "test.command", input)

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)

		var result map[string]any
		err = json.Unmarshal(resp.Data, &result)
		require.NoError(t, err)
		assert.Equal(t, "success", result["result"])
		assert.Equal(t, float64(42), result["value"])
	})

	t.Run("command not found", func(t *testing.T) {
		resp, err := adapter.Execute(ctx, TypeCommand, "nonexistent.command", nil)

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeUnitNotFound, resp.Error.Code)
	})
}

func TestGRPCAdapter_ExecuteStream(t *testing.T) {
	registry := unit.NewRegistry()

	// Register a streaming command
	streamingCmd := &testAdapterStreamingCommand{
		testAdapterCommand: testAdapterCommand{
			name:   "test.stream",
			domain: "test",
		},
	}
	_ = registry.RegisterCommand(streamingCmd)

	gateway := NewGateway(registry)
	adapter := NewGRPCAdapter(gateway)

	ctx := context.Background()

	t.Run("streaming command", func(t *testing.T) {
		input := map[string]any{"message": "hello"}
		chunkChan, err := adapter.ExecuteStream(ctx, TypeCommand, "test.stream", input)

		require.NoError(t, err)

		// Collect chunks
		var chunks []*pb.Chunk
		done := make(chan struct{})
		go func() {
			for chunk := range chunkChan {
				chunks = append(chunks, chunk)
				if chunk.Done {
					break
				}
			}
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for stream")
		}

		// Should have received at least one chunk
		assert.GreaterOrEqual(t, len(chunks), 1)
		assert.True(t, chunks[len(chunks)-1].Done)
	})

	t.Run("non-streaming command", func(t *testing.T) {
		// Register a non-streaming command
		nonStreamingCmd := &testAdapterCommand{
			name:   "test.nonstream",
			domain: "test",
		}
		_ = registry.RegisterCommand(nonStreamingCmd)

		input := map[string]any{}
		chunkChan, err := adapter.ExecuteStream(ctx, TypeCommand, "test.nonstream", input)

		// Should fail because command doesn't support streaming
		require.NoError(t, err)

		// Should receive an error chunk
		var errorChunk *pb.Chunk
		for chunk := range chunkChan {
			if chunk.Error != nil {
				errorChunk = chunk
				break
			}
		}

		assert.NotNil(t, errorChunk)
		assert.Contains(t, errorChunk.Error.Message, "does not support streaming")
	})
}

func TestGRPCAdapter_Options(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := DefaultGRPCAdapterOptions()
		assert.Equal(t, ":9091", opts.Address)
		assert.Equal(t, 100*1024*1024, opts.MaxMessageSize)
		assert.Equal(t, 30*time.Second, opts.ReadTimeout)
		assert.Equal(t, 30*time.Second, opts.WriteTimeout)
		assert.False(t, opts.EnableTLS)
	})
}

// Test helpers
type testAdapterCommand struct {
	name        string
	domain      string
	executeFunc func(ctx context.Context, input any) (any, error)
}

func (c *testAdapterCommand) Name() string        { return c.name }
func (c *testAdapterCommand) Domain() string      { return c.domain }
func (c *testAdapterCommand) Description() string { return "test command" }
func (c *testAdapterCommand) Examples() []unit.Example {
	return []unit.Example{{Input: map[string]any{}, Output: map[string]any{}}}
}
func (c *testAdapterCommand) InputSchema() unit.Schema  { return unit.Schema{Type: "object"} }
func (c *testAdapterCommand) OutputSchema() unit.Schema { return unit.Schema{Type: "object"} }
func (c *testAdapterCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.executeFunc != nil {
		return c.executeFunc(ctx, input)
	}
	return nil, nil
}

type testAdapterStreamingCommand struct {
	testAdapterCommand
}

func (c *testAdapterStreamingCommand) SupportsStreaming() bool {
	return true
}

func (c *testAdapterStreamingCommand) ExecuteStream(ctx context.Context, input any, output chan<- unit.StreamChunk) error {
	// Send some test chunks
	for i := 0; i < 3; i++ {
		select {
		case output <- unit.StreamChunk{
			Type: "content",
			Data: map[string]any{"chunk": i},
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Send done signal
	select {
	case output <- unit.StreamChunk{Type: "done"}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
