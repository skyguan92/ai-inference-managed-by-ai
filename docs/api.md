# API 文档

AIMA 提供统一的 API 接口，支持多种协议：HTTP RESTful API、MCP (Model Context Protocol)、gRPC 和 CLI。

## 目录

- [通用约定](#通用约定)
- [HTTP API](#http-api)
- [MCP 协议集成](#mcp-协议集成)
- [gRPC 接口](#grpc-接口)
- [认证方式](#认证方式)
- [错误处理](#错误处理)
- [分页](#分页)
- [流式响应](#流式响应)

---

## 通用约定

### 请求格式

```go
type Request struct {
    Type    string         `json:"type"`    // "command" | "query" | "resource" | "workflow"
    Unit    string         `json:"unit"`    // "model.pull" | "inference.chat"
    Input   map[string]any `json:"input"`
    Options RequestOptions `json:"options"` // timeout, async, trace_id
}

type RequestOptions struct {
    Timeout time.Duration `json:"timeout,omitempty"`
    Async   bool          `json:"async,omitempty"`
    TraceID string        `json:"trace_id,omitempty"`
}
```

### 响应格式

```go
type Response struct {
    Success bool       `json:"success"`
    Data    any        `json:"data,omitempty"`
    Error   *ErrorInfo `json:"error,omitempty"`
    Meta    *ResponseMeta `json:"meta,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`     // "MODEL_NOT_FOUND", "INSUFFICIENT_RESOURCES"
    Message string `json:"message"`
    Details any    `json:"details,omitempty"`
}

type ResponseMeta struct {
    RequestID  string        `json:"request_id"`
    Duration   int64         `json:"duration_ms"`
    TraceID    string        `json:"trace_id,omitempty"`
    Pagination *Pagination   `json:"pagination,omitempty"`
}
```

---

## HTTP API

### 基础信息

| 属性 | 值 |
|------|-----|
| 基础 URL | `http://localhost:9090` |
| 内容类型 | `application/json` |
| 字符编码 | UTF-8 |

### 端点列表

#### 执行接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v2/execute` | 通用执行接口 |
| POST | `/api/v2/stream` | 流式执行接口 |
| POST | `/api/v2/command/{unit}` | 执行特定命令 |
| POST | `/api/v2/query/{unit}` | 执行特定查询 |
| GET  | `/api/v2/resource/{uri}` | 获取资源 |
| GET  | `/api/v2/resource/{uri}/watch` | 订阅资源变更 (SSE) |

#### 工作流接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/api/v2/workflows` | 列出工作流 |
| POST | `/api/v2/workflow/{name}/run` | 运行工作流 |
| GET  | `/api/v2/workflow/{name}/runs/{run_id}` | 获取运行状态 |
| POST | `/api/v2/workflow/{name}/runs/{run_id}/cancel` | 取消运行 |

#### 系统接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/api/v2/health` | 健康检查 |
| GET  | `/api/v2/metrics` | 指标数据 (Prometheus) |
| GET  | `/api/v2/units` | 列出所有原子单元 |
| GET  | `/api/v2/schema/{unit}` | 获取单元 Schema |

### 示例请求

#### 执行命令

```bash
curl -X POST http://localhost:9090/api/v2/execute \
  -H "Content-Type: application/json" \
  -d '{
    "type": "command",
    "unit": "model.pull",
    "input": {
      "source": "ollama",
      "repo": "llama3.2"
    },
    "options": {
      "timeout": "10m",
      "trace_id": "trace_123"
    }
  }'
```

响应：

```json
{
  "success": true,
  "data": {
    "model_id": "llama3.2:latest",
    "status": "ready",
    "size_bytes": 3825393664
  },
  "meta": {
    "request_id": "req_abc123",
    "duration_ms": 5234,
    "trace_id": "trace_123"
  }
}
```

#### 执行查询

```bash
curl -X POST http://localhost:9090/api/v2/execute \
  -H "Content-Type: application/json" \
  -d '{
    "type": "query",
    "unit": "model.list",
    "input": {
      "type": "llm",
      "limit": 10,
      "offset": 0
    }
  }'
```

响应：

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "llama3.2:latest",
        "name": "llama3.2",
        "type": "llm",
        "format": "gguf",
        "status": "ready",
        "size_bytes": 3825393664
      }
    ],
    "total": 5
  },
  "meta": {
    "request_id": "req_def456",
    "duration_ms": 123,
    "pagination": {
      "page": 1,
      "per_page": 10,
      "total": 5
    }
  }
}
```

#### 获取资源

```bash
# 获取设备信息
curl http://localhost:9090/api/v2/resource/asms://device/0/info

# 获取模型详情
curl http://localhost:9090/api/v2/resource/asms://model/llama3.2:latest
```

#### 订阅资源变更 (SSE)

```bash
curl http://localhost:9090/api/v2/resource/asms://device/0/metrics/watch \
  -H "Accept: text/event-stream"
```

响应（Server-Sent Events）：

```
event: update
data: {"uri":"asms://device/0/metrics","timestamp":"2026-02-17T10:30:00Z","operation":"update","data":{"utilization":0.45,"temperature":65,"power":120}}

event: update
data: {"uri":"asms://device/0/metrics","timestamp":"2026-02-17T10:30:01Z","operation":"update","data":{"utilization":0.50,"temperature":67,"power":125}}
```

#### 流式推理

```bash
curl -X POST http://localhost:9090/api/v2/stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "type": "command",
    "unit": "inference.chat",
    "input": {
      "model": "llama3.2",
      "messages": [{"role": "user", "content": "你好"}],
      "stream": true
    }
  }'
```

响应：

```
event: chunk
data: {"data":"你","metadata":null}

event: chunk
data: {"data":"好","metadata":null}

event: chunk
data: {"data":"！","metadata":null}

event: chunk
data: {"data":null,"metadata":{"usage":{"prompt_tokens":10,"completion_tokens":3}},"done":true}
```

---

## MCP 协议集成

AIMA 完全支持 [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)，允许 AI Agent 通过标准化接口访问 AIMA 的所有功能。

### MCP Server 启动

```bash
aima mcp serve

# 或指定传输方式
aima mcp serve --transport stdio
aima mcp serve --transport sse --port 9091
```

### 提供的 Tools

所有 Command 和 Query 都会自动映射为 MCP Tools。

命名规则：
- `model.pull` → `model_pull`
- `inference.chat` → `inference_chat`

### Tool 示例

```json
{
  "name": "model_pull",
  "description": "Pull a model from source registry",
  "inputSchema": {
    "type": "object",
    "properties": {
      "source": {
        "type": "string",
        "enum": ["ollama", "huggingface", "modelscope"],
        "description": "Source registry"
      },
      "repo": {
        "type": "string",
        "description": "Model repository name"
      },
      "tag": {
        "type": "string",
        "description": "Model tag (optional)"
      }
    },
    "required": ["source", "repo"]
  }
}
```

### MCP Client 配置示例

```json
{
  "mcpServers": {
    "aima": {
      "command": "aima",
      "args": ["mcp", "serve", "--transport", "stdio"],
      "env": {
        "AIMA_API_KEY": "your-api-key"
      }
    }
  }
}
```

### SSE 传输配置

```json
{
  "mcpServers": {
    "aima": {
      "url": "http://localhost:9091/mcp",
      "headers": {
        "Authorization": "Bearer your-api-key"
      }
    }
  }
}
```

---

## gRPC 接口

AIMA 提供 gRPC 接口用于高性能场景。

### Proto 定义

```protobuf
syntax = "proto3";
package aima.v2;

service AIMAService {
  rpc Execute(Request) returns (Response);
  rpc ExecuteStream(Request) returns (stream StreamChunk);
  rpc WatchResource(WatchRequest) returns (stream ResourceUpdate);
  rpc HealthCheck(HealthRequest) returns (HealthResponse);
}

message Request {
  string type = 1;
  string unit = 2;
  bytes input = 3;
  RequestOptions options = 4;
}

message Response {
  bool success = 1;
  bytes data = 2;
  ErrorInfo error = 3;
  ResponseMeta meta = 4;
}

message StreamChunk {
  bytes data = 1;
  bytes metadata = 2;
  bool done = 3;
  ErrorInfo error = 4;
}
```

### gRPC 端口

默认 gRPC 端口：`50051`

```bash
aima start --grpc-port 50051
```

---

## 认证方式

### API Key 认证

在请求头中携带 API Key：

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:9090/api/v2/execute
```

### 配置文件

```toml
[security]
api_key = "your-secure-api-key"
rate_limit_per_min = 120
```

### 生成 API Key

```bash
aima admin generate-api-key
```

---

## 错误处理

### 错误码列表

| 错误码 | 说明 | HTTP 状态码 |
|--------|------|-------------|
| `INVALID_REQUEST` | 请求格式错误 | 400 |
| `UNIT_NOT_FOUND` | 原子单元不存在 | 404 |
| `RESOURCE_NOT_FOUND` | 资源不存在 | 404 |
| `EXECUTION_FAILED` | 执行失败 | 500 |
| `TIMEOUT` | 执行超时 | 504 |
| `UNAUTHORIZED` | 未授权 | 401 |
| `FORBIDDEN` | 禁止访问 | 403 |
| `RATE_LIMITED` | 请求过于频繁 | 429 |
| `INSUFFICIENT_RESOURCES` | 资源不足 | 503 |
| `MODEL_NOT_FOUND` | 模型不存在 | 404 |
| `ENGINE_NOT_RUNNING` | 引擎未运行 | 503 |
| `VALIDATION_ERROR` | 参数验证失败 | 400 |

### 错误响应示例

```json
{
  "success": false,
  "error": {
    "code": "MODEL_NOT_FOUND",
    "message": "Model 'llama3.2' not found",
    "details": {
      "model_id": "llama3.2",
      "suggestions": ["llama3.2:latest", "llama3.1:latest"]
    }
  },
  "meta": {
    "request_id": "req_err789",
    "duration_ms": 45
  }
}
```

---

## 分页

列表查询支持分页参数：

```json
{
  "type": "query",
  "unit": "model.list",
  "input": {
    "limit": 20,
    "offset": 40
  }
}
```

响应包含分页元数据：

```json
{
  "success": true,
  "data": {
    "items": [...],
    "total": 150
  },
  "meta": {
    "pagination": {
      "page": 3,
      "per_page": 20,
      "total": 150
    }
  }
}
```

---

## 流式响应

流式响应使用 Server-Sent Events (SSE) 协议。

### SSE 格式

```
event: chunk
data: {"data":"..."}

event: chunk
data: {"data":"...","metadata":{...}}

event: chunk
data: {"done":true}
```

### 客户端示例 (JavaScript)

```javascript
const eventSource = new EventSource('/api/v2/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    type: 'command',
    unit: 'inference.chat',
    input: { model: 'llama3.2', messages: [...], stream: true }
  })
});

eventSource.onmessage = (event) => {
  const chunk = JSON.parse(event.data);
  if (chunk.done) {
    eventSource.close();
  } else {
    console.log(chunk.data);
  }
};
```

---

## 性能指标

| 指标 | 值 |
|------|-----|
| HTTP 延迟 (p99) | < 50ms |
| gRPC 延迟 (p99) | < 20ms |
| 并发请求 | 1000+ |
| 流式吞吐 | 10 MB/s |

## 更多示例

查看 [examples/](../examples/) 目录获取更多使用示例。
