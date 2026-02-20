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

func TestGRPCServer_Execute(t *testing.T) {
	// Create test registry with mock command
	registry := unit.NewRegistry()

	// Register a test command
	testCmd := &testGRPCCommand{
		name:   "test.command",
		domain: "test",
		executeFunc: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			return map[string]any{
				"result": "success",
				"echo":   inputMap["message"],
			}, nil
		},
	}
	_ = registry.RegisterCommand(testCmd)

	// Create gateway and gRPC server
	gateway := NewGateway(registry)
	server := NewGRPCServer(gateway)

	ctx := context.Background()

	t.Run("successful command execution", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{"message": "hello"})
		req := &pb.Request{
			Type:  TypeCommand,
			Unit:  "test.command",
			Input: input,
		}

		resp, err := server.Execute(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)

		var result map[string]any
		err = json.Unmarshal(resp.Data, &result)
		require.NoError(t, err)
		assert.Equal(t, "success", result["result"])
		assert.Equal(t, "hello", result["echo"])
	})

	t.Run("command not found", func(t *testing.T) {
		req := &pb.Request{
			Type: TypeCommand,
			Unit: "nonexistent.command",
		}

		resp, err := server.Execute(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrCodeUnitNotFound, resp.Error.Code)
	})

	t.Run("invalid input JSON", func(t *testing.T) {
		req := &pb.Request{
			Type:  TypeCommand,
			Unit:  "test.command",
			Input: []byte("invalid json"),
		}

		resp, err := server.Execute(ctx, req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
	})

	t.Run("query execution", func(t *testing.T) {
		// Register a test query
		testQuery := &testGRPCQuery{
			name:   "test.query",
			domain: "test",
			executeFunc: func(ctx context.Context, input any) (any, error) {
				return map[string]any{"query_result": "data"}, nil
			},
		}
		_ = registry.RegisterQuery(testQuery)

		req := &pb.Request{
			Type: TypeQuery,
			Unit: "test.query",
		}

		resp, err := server.Execute(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		var result map[string]any
		err = json.Unmarshal(resp.Data, &result)
		require.NoError(t, err)
		assert.Equal(t, "data", result["query_result"])
	})
}

func TestGRPCServer_convertRequest(t *testing.T) {
	server := &GRPCServer{gateway: &Gateway{}}

	t.Run("valid request with options", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{"key": "value"})
		req := &pb.Request{
			Type:  TypeCommand,
			Unit:  "test.command",
			Input: input,
			Options: &pb.RequestOptions{
				TimeoutMs: 5000,
				Async:     true,
				TraceID:   "trace-123",
			},
		}

		gatewayReq, err := server.convertRequest(req)
		require.NoError(t, err)
		assert.Equal(t, TypeCommand, gatewayReq.Type)
		assert.Equal(t, "test.command", gatewayReq.Unit)
		assert.Equal(t, "value", gatewayReq.Input["key"])
		assert.Equal(t, 5*time.Second, gatewayReq.Options.Timeout)
		assert.True(t, gatewayReq.Options.Async)
		assert.Equal(t, "trace-123", gatewayReq.Options.TraceID)
	})

	t.Run("request without options", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{"key": "value"})
		req := &pb.Request{
			Type:  TypeCommand,
			Unit:  "test.command",
			Input: input,
		}

		gatewayReq, err := server.convertRequest(req)
		require.NoError(t, err)
		assert.Equal(t, TypeCommand, gatewayReq.Type)
		assert.Equal(t, 0*time.Second, gatewayReq.Options.Timeout)
	})
}

func TestGRPCServer_convertResponse(t *testing.T) {
	server := &GRPCServer{gateway: &Gateway{}}

	t.Run("successful response", func(t *testing.T) {
		resp := &Response{
			Success: true,
			Data:    map[string]any{"result": "ok"},
			Meta: &ResponseMeta{
				RequestID: "req-123",
				Duration:  100,
				TraceID:   "trace-456",
				Pagination: &Pagination{
					Page:    1,
					PerPage: 10,
					Total:   100,
				},
			},
		}

		pbResp := server.convertResponse(resp)
		assert.True(t, pbResp.Success)
		assert.NotNil(t, pbResp.Data)
		assert.NotNil(t, pbResp.Meta)
		assert.Equal(t, "req-123", pbResp.Meta.RequestID)
		assert.Equal(t, int64(100), pbResp.Meta.DurationMs)
		assert.Equal(t, "trace-456", pbResp.Meta.TraceID)
		assert.Equal(t, int32(1), pbResp.Meta.Page)
		assert.Equal(t, int32(10), pbResp.Meta.PerPage)
		assert.Equal(t, int32(100), pbResp.Meta.Total)
	})

	t.Run("error response", func(t *testing.T) {
		resp := &Response{
			Success: false,
			Error: &ErrorInfo{
				Code:    ErrCodeUnitNotFound,
				Message: "unit not found",
				Details: map[string]any{"unit": "test.unit"},
			},
		}

		pbResp := server.convertResponse(resp)
		assert.False(t, pbResp.Success)
		assert.NotNil(t, pbResp.Error)
		assert.Equal(t, ErrCodeUnitNotFound, pbResp.Error.Code)
		assert.Equal(t, "unit not found", pbResp.Error.Message)
	})
}

func TestGRPCServer_convertErrorInfo(t *testing.T) {
	server := &GRPCServer{gateway: &Gateway{}}

	t.Run("nil error", func(t *testing.T) {
		result := server.convertErrorInfo(nil)
		assert.Nil(t, result)
	})

	t.Run("error with details", func(t *testing.T) {
		errInfo := &ErrorInfo{
			Code:    "TEST_ERROR",
			Message: "test error",
			Details: map[string]any{"key": "value"},
		}

		result := server.convertErrorInfo(errInfo)
		assert.NotNil(t, result)
		assert.Equal(t, "TEST_ERROR", result.Code)
		assert.Equal(t, "test error", result.Message)
		assert.NotNil(t, result.Details)
	})
}

// Test helpers
type testGRPCCommand struct {
	name        string
	domain      string
	executeFunc func(ctx context.Context, input any) (any, error)
}

func (c *testGRPCCommand) Name() string        { return c.name }
func (c *testGRPCCommand) Domain() string      { return c.domain }
func (c *testGRPCCommand) Description() string { return "test command" }
func (c *testGRPCCommand) Examples() []unit.Example {
	return []unit.Example{{Input: map[string]any{}, Output: map[string]any{}}}
}
func (c *testGRPCCommand) InputSchema() unit.Schema  { return unit.Schema{Type: "object"} }
func (c *testGRPCCommand) OutputSchema() unit.Schema { return unit.Schema{Type: "object"} }
func (c *testGRPCCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.executeFunc != nil {
		return c.executeFunc(ctx, input)
	}
	return nil, nil
}

type testGRPCQuery struct {
	name        string
	domain      string
	executeFunc func(ctx context.Context, input any) (any, error)
}

func (q *testGRPCQuery) Name() string        { return q.name }
func (q *testGRPCQuery) Domain() string      { return q.domain }
func (q *testGRPCQuery) Description() string { return "test query" }
func (q *testGRPCQuery) Examples() []unit.Example {
	return []unit.Example{{Input: map[string]any{}, Output: map[string]any{}}}
}
func (q *testGRPCQuery) InputSchema() unit.Schema  { return unit.Schema{Type: "object"} }
func (q *testGRPCQuery) OutputSchema() unit.Schema { return unit.Schema{Type: "object"} }
func (q *testGRPCQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.executeFunc != nil {
		return q.executeFunc(ctx, input)
	}
	return nil, nil
}
