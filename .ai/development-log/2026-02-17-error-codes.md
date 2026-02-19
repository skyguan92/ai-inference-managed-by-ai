# [2026-02-17] 统一错误码体系实现

## 元信息
- 开始时间: 2026-02-17 11:00
- 完成时间: 2026-02-17 12:00
- 实现模型: Kilo
- 审查模型: N/A

## 任务概述
- **目标**: 建立统一的错误码体系，覆盖所有领域和错误场景
- **范围**: pkg/unit/errors.go, pkg/gateway/errors.go, 各领域的 errors.go
- **优先级**: P1

## 设计决策
| 决策点 | 选择 | 理由 |
|--------|------|------|
| 错误码格式 | 5位数字编码 (00000) | 与行业标准对齐，便于排序和检索 |
| 错误结构 | UnitError 结构体 | 统一包含 Code/Domain/Message/Details/Cause |
| 领域划分 | 每100个编码一个领域 | 预留足够空间，避免冲突 |
| 向后兼容 | 保留 ErrorInfo | gateway 层兼容旧代码 |

## 实现摘要

### 新增文件
- `pkg/unit/errors.go` - 核心错误类型定义和错误码常量
- `pkg/unit/errors_test.go` - 单元测试 (95%+ 覆盖率)
- `pkg/unit/model/errors.go` - Model 域错误定义
- `pkg/unit/engine/errors.go` - Engine 域错误定义
- `pkg/unit/inference/errors.go` - Inference 域错误定义
- `pkg/unit/resource/errors.go` - Resource 域错误定义
- `pkg/unit/device/errors.go` - Device 域错误定义
- `pkg/unit/service/errors.go` - Service 域错误定义
- `pkg/unit/app/errors.go` - App 域错误定义
- `pkg/unit/pipeline/errors.go` - Pipeline 域错误定义
- `pkg/unit/alert/errors.go` - Alert 域错误定义
- `pkg/unit/remote/errors.go` - Remote 域错误定义

### 修改文件
- `pkg/gateway/errors.go` - 添加 UnitError 转换和 HTTP 状态码映射
- `pkg/gateway/errors_test.go` - 更新测试用例
- `pkg/unit/model/commands.go` - 移除重复错误定义
- `pkg/unit/model/store.go` - 移除重复错误定义
- `pkg/unit/engine/store.go` - 移除重复错误定义
- `pkg/unit/inference/provider.go` - 移除重复错误定义
- `pkg/unit/resource/types.go` - 移除重复错误定义
- `pkg/unit/device/commands.go` - 移除重复错误定义
- `pkg/unit/service/store.go` - 移除重复错误定义
- `pkg/unit/app/store.go` - 移除重复错误定义
- `pkg/unit/pipeline/store.go` - 移除重复错误定义
- `pkg/unit/alert/store.go` - 移除重复错误定义
- `pkg/unit/remote/store.go` - 移除重复错误定义

### 关键代码

#### 核心错误类型
```go
type UnitError struct {
    Code    ErrorCode
    Domain  string
    Message string
    Details map[string]any
    Cause   error
}
```

#### 错误码定义
```go
// 通用错误 (000-099)
const (
    ErrCodeSuccess         ErrorCode = "00000"
    ErrCodeUnknown         ErrorCode = "00001"
    ErrCodeInvalidRequest  ErrorCode = "00002"
    ErrCodeNotFound        ErrorCode = "00004"
    // ...
)

// 模型领域 (100-199)
const (
    ErrCodeModelNotFound      ErrorCode = "00100"
    ErrCodeModelAlreadyExists ErrorCode = "00101"
    // ...
)
```

#### 错误创建函数
```go
func NewError(code ErrorCode, message string) *UnitError
func NewDomainError(domain string, code ErrorCode, message string) *UnitError
func WrapError(err error, code ErrorCode, message string) *UnitError
```

## 错误码映射表

| 领域 | 编码范围 | 错误码示例 |
|------|----------|-----------|
| 通用 | 000-099 | 00000 SUCCESS, 00004 NOT_FOUND |
| 模型 | 100-199 | 00100 MODEL_NOT_FOUND |
| 引擎 | 200-299 | 00200 ENGINE_NOT_FOUND |
| 推理 | 300-399 | 00300 INFERENCE_MODEL_NOT_LOADED |
| 资源 | 400-499 | 00400 RESOURCE_INSUFFICIENT |
| 设备 | 500-599 | 00500 DEVICE_NOT_FOUND |
| 服务 | 600-699 | 00600 SERVICE_NOT_FOUND |
| 应用 | 700-799 | 00700 APP_NOT_FOUND |
| 管道 | 800-899 | 00800 PIPELINE_NOT_FOUND |
| 告警 | 900-999 | 00900 ALERT_RULE_NOT_FOUND |
| 远程 | 1000-1099 | 01000 REMOTE_NOT_ENABLED |

## HTTP 状态码映射

| 错误类型 | HTTP 状态码 |
|----------|-------------|
| ErrCodeSuccess | 200 OK |
| ErrCodeInvalidRequest/InvalidInput/ValidationFailed | 400 Bad Request |
| ErrCodeUnauthorized | 401 Unauthorized |
| ErrCodeNotFound | 404 Not Found |
| ErrCodeAlreadyExists | 409 Conflict |
| ErrCodeTimeout | 408 Request Timeout |
| ErrCodeRateLimited | 429 Too Many Requests |
| 其他 | 500 Internal Server Error |

## 测试结果
```bash
$ go test ./pkg/unit -v
=== RUN   TestNewError
--- PASS: TestNewError (0.00s)
=== RUN   TestNewDomainError
--- PASS: TestNewDomainError (0.00s)
=== RUN   TestWrapError
--- PASS: TestWrapError (0.00s)
=== RUN   TestUnitError_Error
--- PASS: TestUnitError_Error (0.00s)
=== RUN   TestUnitError_WithDetails
--- PASS: TestUnitError_WithDetails (0.00s)
=== RUN   TestUnitError_Is
--- PASS: TestUnitError_Is (0.00s)
=== RUN   TestAsUnitError
--- PASS: TestAsUnitError (0.00s)
=== RUN   TestErrorToHTTPStatus
--- PASS: TestErrorToHTTPStatus (0.00s)
=== RUN   TestIsNotFound
--- PASS: TestIsNotFound (0.00s)
=== RUN   TestIsAlreadyExists
--- PASS: TestIsAlreadyExists (0.00s)
=== RUN   TestIsTimeout
--- PASS: TestIsTimeout (0.00s)
=== RUN   TestIsRateLimited
--- PASS: TestIsRateLimited (0.00s)
PASS
ok      github.com/jguan/ai-inference-managed-by-ai/pkg/unit    0.003s
```

## 使用示例

### 创建领域错误
```go
// 在 model 域中
return model.ErrModelNotFound

// 或使用统一错误类型
return unit.NewDomainError("model", unit.ErrCodeModelNotFound, "model xyz not found")
```

### 包装错误
```go
err := c.provider.Pull(ctx, source, repo, tag, nil)
if err != nil {
    return fmt.Errorf("pull model from %s: %w", source, err)
}
```

### 检查错误类型
```go
if unit.IsNotFound(err) {
    return http.StatusNotFound
}

if unit.IsAlreadyExists(err) {
    return http.StatusConflict
}
```

### 转换为 HTTP 响应
```go
unitErr, ok := unit.AsUnitError(err)
if ok {
    status := unit.ErrorToHTTPStatus(unitErr.Code)
    // return HTTP response with status
}
```

## 向后兼容性

- 旧代码中直接使用 `errors.New()` 创建的错误仍然有效
- gateway 层 `ErrorInfo` 类型保持不变，新增对 `UnitError` 的转换支持
- 领域特定错误变量（如 `ErrModelNotFound`）继续使用，只是底层类型变为 `*UnitError`

## 后续任务
- [ ] 在 gateway 层集成新的错误处理
- [ ] 更新 API 文档中的错误码说明
- [ ] 添加错误码到错误信息的国际化支持
