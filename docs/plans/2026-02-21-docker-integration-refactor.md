# Docker 集成架构改进实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 系统性改进 AIMA 项目中 Docker 集成的架构设计，解决 CLI 子进程脆弱性、硬编码配置、用户进度不可见三大根本问题。

**Architecture:** 建立 `docker.Client` 接口层 → 迁移到 Docker Go SDK → 桥接 Catalog Engine YAML 到 RecipeEngine → 利用已有 RegistryProvider + EventBus 实现实时进度流。

**Tech Stack:** `github.com/docker/docker/client`（Apache 2.0），Catalog Engine YAML，`pkg/infra/eventbus`

---

## Part 1: 问题诊断（根本原因分析）

### 根本原因 A — CLI Subprocess 而非 Docker Go SDK

**核心痛点文件：** `pkg/infra/docker/simple_client.go`（161 行）

整个 Docker 操作层使用 `exec.CommandContext("docker", ...)` shell 子进程调用。这导致：
- 无类型安全（返回值全部是字符串解析）
- 无流式支持（`docker pull` 无法显示层级下载进度）
- 无 Docker Events 订阅（容器 OOM/die/health 变化无法实时感知）
- `MockClient`（`mock.go`）的方法签名与 `SimpleClient` 完全不同，**无法注入**到 `HybridEngineProvider` 中
- 依赖 `docker` CLI 在 PATH 中，跨平台不可靠

**结论：** Docker Go SDK（`github.com/docker/docker/client`，Apache 2.0）是官方程序化 API，应替换 CLI 调用。

---

### 根本原因 B — Catalog Engine YAML 已有但未被 HybridEngineProvider 使用

**关键发现：** `catalog/engines/` 目录下已有完整的引擎资产 YAML 文件：

| 文件 | 内容 |
|------|------|
| `catalog/engines/vllm/vllm-0.14.0-cu131-gb10.yaml` | 镜像名、备选镜像、启动参数、健康检查路径、API 端点 |
| `catalog/engines/asr/funasr-sensevoice-cpu.yaml` | FunASR 镜像、启动参数、资源需求 |
| `catalog/engines/tts/qwen-tts-cpu.yaml` | Qwen-TTS 镜像、启动参数、端口 |

但 `HybridEngineProvider` 的 `getDockerImages()` 和 `buildDockerCommand()` **完全忽略这些 YAML**，而是在 Go 代码中硬编码了 4 个 vLLM 镜像候选、2 个 ASR 候选、3 个 TTS 候选。

**Schema 不匹配问题：** Engine YAML 使用丰富的文档格式（`image.full_name`, `startup.default_args`, `startup.health_check`），而 `RecipeEngine` struct 使用精简格式（`Image string`, `Config map[string]any`）。需要 **EngineAssetLoader** 桥接层。

---

### 根本原因 C — RegistryProvider 已设计但未连接

ARCHITECTURE.md 已定义：
```go
type RegistryProvider interface {
    Name() string
    PullImage(ctx context.Context, image string, progress chan<- PullProgress) error
    ImageExists(ctx context.Context, image string) (bool, error)
}
```

这个接口**已经支持流式进度**（`progress chan<- PullProgress`），但 `HybridEngineProvider` 直接调用 `SimpleClient.PullImage()`，完全绕过了 `RegistryProvider`。

---

### 根本原因 D — 两个并行提供者存在重复逻辑

`docker_engine_provider.go`（476 行）和 `hybrid_engine_provider.go`（949 行）中存在重复的镜像选择、命令构建、端口默认值、健康等待循环逻辑。

`DockerEngineProvider` 在主路径中已无调用（仅在自身测试 helper 中引用），是实质性的**死代码**。

---

### 根本原因 E — 端口分配不持久化 + ServiceID 字符串分割脆弱

- `HybridServiceProvider.portCounter` 每次进程重启后都从 8000 开始，与上次会话的容器端口冲突
- Service ID 格式 `svc-{engineType}-{modelId}` 通过 `strings.Split` 在 3 处不同位置解析，若 modelId 含 `-` 则行为依赖隐式约定

---

### 用户感知痛点汇总

| 问题 | 现状 | 期望 | Phase |
|------|------|------|-------|
| 启动大模型（如 Qwen3-30B）时无任何进度 | 命令挂起 20 分钟无输出 | 实时显示 pull 进度 + 模型加载日志 | 9.3 |
| 无法查看容器日志 | 需要手动 `docker logs` | `aima service logs --follow` 流式输出 | 9.3 |
| 新增引擎类型需要改 Go 代码 | 修改 switch 语句 | 写 Engine YAML + Recipe YAML | 9.2 |
| 重启后端口冲突 | portCounter 重置为 8000 | 从 ServiceStore 恢复上次使用的端口 | 9.4 |
| 测试无法 mock Docker 操作 | MockClient 与 SimpleClient 接口不同 | 统一 docker.Client 接口 | 9.1 |

---

## Part 2: 开源项目集成建议

### ✅ 推荐集成：Docker Go SDK

- **库：** `github.com/docker/docker/client`（Apache 2.0，可商用）
- **版本：** v27.x（当前稳定版）
- **引入方式：** `go get github.com/docker/docker@v27.x`
- **二进制体积影响：** ~5MB（含 opencontainers/image-spec 等标准依赖）
- **价值：**
  - `client.ImagePull()` 返回 `io.ReadCloser` 流式 JSON（逐层进度）— 供 RegistryProvider 实现使用
  - `client.ContainerLogs(follow=true)` 实时日志流
  - `client.Events()` 提供 `<-chan events.Message`（容器 die/health_status/oom 事件）
  - `client.NewClientWithOpts(client.FromEnv)` 自动支持 `DOCKER_HOST` / Docker-over-SSH
  - 结构化 API 调用，无需解析 shell 输出

### ❌ 不推荐：docker/compose v2

`docker/compose v2` 为声明式多容器应用设计，引入完整服务图解析器。AIMA 每种引擎类型仅一个容器，Pipeline 域已在领域层处理多模型编排。引入 Compose 只会增加复杂度。

### ❌ 不推荐：Triton / BentoML

Triton 仅在需要 NVIDIA 批量推理优化时有价值（当前无此需求）；BentoML 为 GPL v3（**不可商用**）。

### ✅ 可选（未来）：Ollama Go client

`github.com/ollama/ollama/api`（MIT），当前 Ollama 支持已通过 native process 方式实现。如需更深度集成（模型管理、拉取进度）可后续添加。

---

## Part 3: 实施路线图（6 个子阶段）

### Phase 9.0: 清理死代码（前置，低风险）

**目标：** 删除与 `HybridEngineProvider` 重复的 `DockerEngineProvider`，减少后续重构的干扰。

**前置验证：**
```bash
# 确认 NewDockerServiceProvider / NewDockerEngineProvider 无主路径引用
grep -rn "NewDockerServiceProvider\|NewDockerEngineProvider" pkg/ --include="*.go" | grep -v "_test.go" | grep -v "docker_engine_provider.go"
```

**文件变更：**
- 删除 `pkg/infra/provider/docker_engine_provider.go`（476 行）
- 删除对应测试文件（如有独立测试文件）

**测试验证：** `go test ./pkg/infra/provider/... && go build ./cmd/aima/...`

---

### Phase 9.1: 建立 Docker 客户端接口层

**目标：** 将所有代码解耦于 `SimpleClient`，为 SDK 迁移和测试注入打基础。**无行为变化**。

**Step 1: 新建 `pkg/infra/docker/client.go`**

```go
package docker

import (
    "context"
    "time"
)

// ContainerEvent represents a Docker container lifecycle event.
type ContainerEvent struct {
    ContainerID string
    Type        string // "die", "health_status", "oom", "start", "stop"
    Message     string
    Timestamp   time.Time
}

// Client defines the interface for Docker container lifecycle operations.
// Image registry operations (pull, exists) are handled by RegistryProvider.
type Client interface {
    CreateAndStartContainer(ctx context.Context, name, image string, opts ContainerOptions) (string, error)
    StopContainer(ctx context.Context, containerID string, timeout int) error
    GetContainerStatus(ctx context.Context, containerID string) (string, error)
    GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error)
    StreamLogs(ctx context.Context, containerID string, since time.Time, out chan<- string) error
    ListContainers(ctx context.Context, labels map[string]string) ([]string, error)
    ContainerEvents(ctx context.Context, filters map[string]string) (<-chan ContainerEvent, error)
}
```

**Step 2: 让 `SimpleClient` 实现 `Client` 接口**
- `SimpleClient` 已有 `CreateAndStartContainer`, `StopContainer`, `GetContainerStatus`, `GetContainerLogs`, `ListContainers`
- 新增 `StreamLogs`（调用 `docker logs -f`）和 `ContainerEvents`（调用 `docker events --filter`）的 CLI 实现
- 编译验证：`var _ Client = (*SimpleClient)(nil)`

**Step 3: 新建 `pkg/infra/docker/sdk_client.go`**
- `go get github.com/docker/docker@v27.x`
- 实现 `Client` 接口，使用 `github.com/docker/docker/client`
- `NewSDKClient()` 使用 `client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())`
- `StreamLogs` 使用 `cli.ContainerLogs(ctx, id, container.LogsOptions{Follow: true})`
- `ContainerEvents` 使用 `cli.Events(ctx, events.ListOptions{Filters: ...})`

**Step 4: 更新 `MockClient` 实现接口**
- `pkg/infra/docker/mock.go` 中的 `MockClient` 实现 `Client`
- 新增 `StreamLogs` 和 `ContainerEvents` 的 mock 实现

**Step 5: `HybridEngineProvider` 注入接口**
- `dockerClient` 字段：`*docker.SimpleClient` → `docker.Client`
- `NewHybridEngineProvider` 接受 `docker.Client` 参数（默认 `NewSDKClient()`，失败 fallback `NewSimpleClient()`）

**文件变更：**
| 文件 | 操作 |
|------|------|
| `pkg/infra/docker/client.go` | 新建 |
| `pkg/infra/docker/sdk_client.go` | 新建 |
| `pkg/infra/docker/simple_client.go` | 添加 StreamLogs + ContainerEvents + 接口断言 |
| `pkg/infra/docker/mock.go` | 实现 Client 接口 |
| `pkg/infra/provider/hybrid_engine_provider.go` | dockerClient 字段改为接口 |
| `go.mod` + `go.sum` | 添加 docker SDK 依赖 |

**测试验证：**
```bash
go test ./pkg/infra/docker/... ./pkg/infra/provider/...
# 所有现有测试通过，无行为变化
```

---

### Phase 9.2: Catalog Engine YAML → RecipeEngine 桥接

**目标：** 消除 `getDockerImages()` 和 `buildDockerCommand()` 中的硬编码配置。Engine YAML 成为引擎配置的单一信息源。

**Step 1: 新建 `pkg/catalog/engine_asset_loader.go`**

```go
// EngineAsset represents parsed engine YAML file
type EngineAsset struct {
    Name             string
    Type             string   // "vllm", "asr", "tts"
    ImageFullName    string   // image.full_name
    AlternativeNames []string // image.alternative_names
    StartupArgs      []string // startup.default_args
    HealthCheckPath  string   // startup.health_check.path
    HealthTimeout    string   // startup.health_check.timeout
    DefaultPort      int      // 从 startup args 或 endpoints 推断
    Requirements     struct {
        GPURequired bool
        CPUCoresMin int
        MemoryMin   string
    }
}

// LoadEngineAssets loads all engine YAML files from catalog/engines/
func LoadEngineAssets(dir string) (map[string]EngineAsset, error)

// ToRecipeEngine converts an EngineAsset to RecipeEngine
func (a EngineAsset) ToRecipeEngine() RecipeEngine
```

**Step 2: `HybridEngineProvider` 接入 RecipeStore**
- 添加 `recipeStore catalog.RecipeStore` 字段
- `getDockerImages(engineType)` → `recipeStore.GetByEngineType(engineType)` → `recipe.Engine.Image + FallbackImages`
- `buildDockerCommand(engineType, image, ...)` → `recipe.Engine.Config["cmd"]` + 参数拼接
- `getDefaultPort(engineType)` → `recipe.Engine.Config["port"]`
- **向后兼容**：RecipeStore 无匹配时 fallback 到当前硬编码逻辑

**Step 3: 在 `RegisterAll` 中加载 Engine Assets**
- 启动时调用 `LoadEngineAssets("catalog/engines/")`
- 将结果注入 `RecipeStore`（作为内置 Recipe 的补充数据源）

**文件变更：**
| 文件 | 操作 |
|------|------|
| `pkg/catalog/engine_asset_loader.go` | 新建 |
| `pkg/catalog/engine_asset_loader_test.go` | 新建 |
| `pkg/infra/provider/hybrid_engine_provider.go` | 重构 getDockerImages/buildDockerCommand/getDefaultPort |
| `pkg/registry/register.go` | 加载 Engine Assets |

**测试验证：**
```bash
go test ./pkg/catalog/... ./pkg/infra/provider/...
# getDockerImages 返回值应与之前相同（来源从硬编码变为 YAML）
```

---

### Phase 9.3: 实时启动进度流

**目标：** 用户在 `aima service start --wait` 时看到实时的 pull 进度 + 容器启动日志。

**Step 1: 添加 `engine.start_progress` 事件**

在 `pkg/unit/engine/events.go` 中：
```go
type StartProgressEvent struct {
    ServiceID string
    Phase     string // "pulling", "starting", "loading", "ready", "failed"
    Message   string
    Progress  int    // 0-100, -1 表示不确定
    Timestamp int64
}
```

**Step 2: `startDockerWithRetry` 发布进度事件**
- Pull 阶段：使用 `RegistryProvider.PullImage(ctx, image, progress)` 替代 `SimpleClient.PullImage`
- 每收到 `PullProgress` → 发布 `StartProgressEvent{Phase: "pulling", Message: layer + status}`
- 容器启动后：发布 `StartProgressEvent{Phase: "starting"}`
- 健康等待中：使用 `SDKClient.StreamLogs()` 读取日志 → 发布 `StartProgressEvent{Phase: "loading", Message: logLine}`
- 就绪：发布 `StartProgressEvent{Phase: "ready"}` + 已有的 `engine.started` 事件

**Step 3: CLI 订阅进度事件**
- `runServiceStart --wait`：订阅 `engine.start_progress` 事件
- 渲染：pulling 阶段显示进度条，loading 阶段显示日志行，ready 显示完成

**Step 4: 新增 `aima service logs` 命令**
- 新建 `service.logs` 查询（`pkg/unit/service/query_logs.go`）
- 底层调用 `docker.Client.StreamLogs()`
- CLI 命令：`aima service logs <service-id> [--follow] [--tail N]`

**文件变更：**
| 文件 | 操作 |
|------|------|
| `pkg/unit/engine/events.go` | 添加 StartProgressEvent |
| `pkg/infra/provider/hybrid_engine_provider.go` | startDockerWithRetry 发布进度 |
| `pkg/cli/service.go` | 添加 logs 命令 + --wait 进度显示 |
| `pkg/unit/service/query_logs.go` | 新建 service.logs 查询 |

**用户体验（Phase 9.3 完成后）：**
```
$ aima service start svc-vllm-qwen3-32b --wait
Pulling image zhiwen-vllm:0128...
  [====================] layer abc123  512MB/512MB  done
  [====================] layer def456  2.1GB/2.1GB  done
Image pull complete (2.6GB in 45s)

Starting container aima-vllm-1748000000...
Container started (id: a3f2c8d1)

Loading model from /models...
  INFO  Loading weights: 23/80 layers (28%)
  INFO  Loading weights: 80/80 layers (100%)
  INFO  vLLM service ready at http://localhost:8000

Service svc-vllm-qwen3-32b started (took 4m 32s)

$ aima service logs svc-vllm-qwen3-32b --follow
2026-02-20 10:00:01  INFO  Loading model from /models
2026-02-20 10:01:15  INFO  Profiling model on GPU...
2026-02-20 10:04:30  INFO  Service ready on port 8000
```

---

### Phase 9.4: 端口分配持久化 + ServiceID 结构化

**目标：** 端口分配在进程重启后不冲突；ServiceID 解析不依赖字符串分割。

**Step 1: `ServiceID` 结构体**

```go
// pkg/unit/service/service_id.go
type ServiceID struct {
    EngineType string
    ModelID    string
}

func ParseServiceID(id string) (ServiceID, error) {
    prefix := "svc-"
    if !strings.HasPrefix(id, prefix) {
        return ServiceID{}, fmt.Errorf("invalid service ID: %s", id)
    }
    rest := id[len(prefix):]
    idx := strings.Index(rest, "-")
    if idx == -1 {
        return ServiceID{}, fmt.Errorf("invalid service ID format: %s", id)
    }
    return ServiceID{EngineType: rest[:idx], ModelID: rest[idx+1:]}, nil
}

func (s ServiceID) String() string {
    return fmt.Sprintf("svc-%s-%s", s.EngineType, s.ModelID)
}
```

**Step 2: `portCounter` 持久化恢复**
- `NewHybridServiceProvider` 注入 `serviceStore service.ServiceStore`
- 启动时查询所有现有服务，找到最大已用端口：`portCounter = maxPort + 1`
- 如果无已有服务，默认从 8000 开始

**Step 3: 新增 `[docker]` 配置节**
```toml
[docker]
host = ""           # 默认: unix:///var/run/docker.sock 或 DOCKER_HOST
timeout = "120s"    # 默认操作超时
```

**文件变更：**
| 文件 | 操作 |
|------|------|
| `pkg/unit/service/service_id.go` | 新建 |
| `pkg/unit/service/service_id_test.go` | 新建 |
| `pkg/infra/provider/hybrid_engine_provider.go` | portCounter 初始化 + ParseServiceID 替换 |
| `pkg/config/config.go` | 新增 DockerConfig |

---

## Part 4: 关键文件路径

| 文件 | Phase | 操作 | 说明 |
|------|-------|------|------|
| `pkg/infra/provider/docker_engine_provider.go` | 9.0 | **删除** | 476 行死代码 |
| `pkg/infra/docker/client.go` | 9.1 | 新建 | Client 接口 + ContainerEvent |
| `pkg/infra/docker/sdk_client.go` | 9.1 | 新建 | Docker Go SDK 实现 |
| `pkg/infra/docker/simple_client.go` | 9.1 | 更新 | 添加 StreamLogs/ContainerEvents + 接口断言 |
| `pkg/infra/docker/mock.go` | 9.1 | 更新 | 实现 Client 接口 |
| `pkg/infra/provider/hybrid_engine_provider.go` | 9.1-9.4 | 多处重构 | 核心文件 |
| `pkg/catalog/engine_asset_loader.go` | 9.2 | 新建 | YAML → RecipeEngine 桥接 |
| `pkg/unit/engine/events.go` | 9.3 | 更新 | StartProgressEvent |
| `pkg/cli/service.go` | 9.3 | 更新 | logs 命令 + 进度显示 |
| `pkg/unit/service/query_logs.go` | 9.3 | 新建 | service.logs 查询 |
| `pkg/unit/service/service_id.go` | 9.4 | 新建 | ServiceID 结构体 |
| `pkg/config/config.go` | 9.4 | 更新 | DockerConfig |
| `go.mod` + `go.sum` | 9.1 | 更新 | Docker SDK 依赖 |

---

## Part 5: 验证方法

### 每个 Phase 的验证命令

```bash
# Phase 9.0 — 删除死代码后编译 + 测试
go build ./cmd/aima/... && go test ./pkg/infra/provider/...

# Phase 9.1 — 接口层不改变行为
go test ./pkg/infra/docker/... ./pkg/infra/provider/...

# Phase 9.2 — 引擎配置从 YAML 读取
go test ./pkg/catalog/... ./pkg/infra/provider/...

# Phase 9.3 — 进度事件 + CLI 命令
go test ./pkg/unit/engine/... ./pkg/cli/...

# Phase 9.4 — ServiceID + 端口持久化
go test ./pkg/unit/service/... ./pkg/infra/provider/...

# 全量回归
go test ./... -count=1
go vet ./...
```

### E2E 验证（远程 ARM64 机器）

```bash
# 编译并部署到远程机器
GOOS=linux GOARCH=arm64 go build -o /tmp/aima ./cmd/aima
scp /tmp/aima remote:/tmp/aima

# 在远程机器上测试
ssh remote '/tmp/aima service start svc-vllm-test --wait'
# 应看到 pulling → starting → loading → ready 各阶段输出

ssh remote '/tmp/aima service logs svc-vllm-test --follow'
# 应流式输出容器日志

# 重启后端口不冲突
ssh remote 'pkill aima && /tmp/aima service start svc-vllm-test2'
# 新服务端口应在已用端口之后
```

---

## Part 6: 不做什么

1. **不引入 docker/compose v2** — Pipeline 域已处理多服务编排
2. **不用 Compose YAML 替代 Recipe YAML** — Recipe 包含硬件画像、模型下载等 Compose 不支持的信息
3. **不一次性重写 HybridEngineProvider** — 分 Phase 渐进重构，每步保持测试绿色
4. **不添加 Triton/SGLang 等新引擎** — 等有实际 Recipe 需求时再加

---

## 执行策略选择

**方案 1（推荐，子 Agent 驱动）：** 在新 session 中逐 Phase 执行，每个 Phase 完成后代码审查再继续。

**方案 2（并行 Session）：** 在新 session 中使用 `executing-plans` skill 批量执行。
