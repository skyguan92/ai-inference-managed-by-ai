# 性能基准报告

**日期**: 2026-02-17  
**执行环境**: ARM64 Linux  
**Go 版本**: (系统默认)  
**测试包**: `pkg/benchmark`

---

## 执行摘要

本次基准测试评估了 AIMA 核心组件的性能特征，包括 Registry 查找、Gateway 执行、Schema 验证和事件发布。基于测试结果，标准 Registry 实现已经具有出色的性能，优化方案需要针对特定场景定制。

---

## 测试结果汇总

### 1. Registry 性能

| 测试项 | ops/sec | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|---------|
| Registry.GetCommand | 30,697,942 | 39.16 ns | 0 B | 0 |
| Registry.GetQuery | 30,715,028 | 39.20 ns | 0 B | 0 |
| Registry.GetResource | 31,454,520 | 38.39 ns | 0 B | 0 |
| Registry.Get (通用) | 30,740,370 | 39.15 ns | 0 B | 0 |
| Registry.ConcurrentReads | 10,288,890 | 116.6 ns | 0 B | 0 |

**分析**:
- 标准 Registry 性能非常优秀，单次查找仅需 ~40ns
- 已经使用 `sync.RWMutex`，无内存分配
- 并发读取性能良好，约 116ns/op

### 2. Gateway 执行性能

| 测试项 | ops/sec | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|---------|
| Gateway.ExecuteQuery | 531,846 | 3.138 μs | 1,361 B | 28 |
| Gateway.ExecuteQuery(Parallel) | 1,550,434 | 863.8 ns | 1,363 B | 28 |
| Gateway.ExecuteCommand | 374,246 | 3.126 μs | 1,549 B | 29 |
| Gateway.ValidateRequest | 324,819 | 3.089 μs | 1,361 B | 28 |

**分析**:
- 并行执行显著提升吞吐量（3.6x 提升）
- 每次请求产生 ~28 次内存分配
- 主要开销来自上下文创建和错误处理

### 3. Schema 验证性能

| 测试项 | ops/sec | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|---------|
| Schema.Validate(String) | 31,872,530 | 34.66 ns | 16 B | 1 |
| Schema.Validate(Number) | 57,252,442 | 17.81 ns | 8 B | 1 |
| Schema.Validate(Object) | 3,158,122 | 380.8 ns | 480 B | 2 |
| Schema.Validate(Array) | 11,578,826 | 98.48 ns | 24 B | 1 |

**分析**:
- 基本类型验证非常快（<35ns）
- 对象验证开销较大（380ns/op，480B 分配）
- 使用反射是主要瓶颈

### 4. 事件系统性能

| 测试项 | ops/sec | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|---------|
| EventBus.Publish | 15,291,399 | 78.76 ns | 84 B | 0 |
| EventBus.Publish(Parallel) | 3,488,612 | 300.1 ns | 96 B | 0 |

**分析**:
- 事件发布性能优秀
- 并行场景有锁竞争，延迟增加

### 5. 内存分配测试

| 测试项 | ops/sec | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|---------|
| ObjectCreation | 196,485,055 | 6.104 ns | 0 B | 0 |
| MapCreation | 37,281,333 | 32.36 ns | 0 B | 0 |
| Context.WithValues | 4,184,770 | 268.5 ns | 175 B | 7 |
| Context.GetValues | 64,029,286 | 18.10 ns | 0 B | 0 |

**分析**:
- 上下文读取非常快（18ns）
- 上下文写入有开销（268ns，175B）

### 6. Command 执行性能

| 测试项 | ops/sec | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|---------|
| Command.Execute(MemoryStore) | 1,141,958 | 1.081 μs | 732 B | 7 |
| Command.Execute(WithEvents) | 499,632 | 2.047 μs | 1,421 B | 18 |

**分析**:
- 带事件发布的执行慢 2x
- 事件系统增加了 9 次分配

---

## 优化方案评估

### 方案 1: 优化 Gateway（Response 对象池）

| 测试项 | 延迟/op | 内存/op | 分配/op | 改善 |
|--------|---------|---------|---------|------|
| Gateway.Standard | 3.2μs | 1,361 B | 28 | - |
| Gateway.Optimized | 3.0μs | 1,266 B | 26 | 7% ↓ |

**结论**: ⚠️ **场景依赖**
- 内存分配减少 7% (从 28 到 26 allocs/op)
- 内存使用减少 7% (从 1361B 到 1266B)
- 适合高并发、GC 敏感场景使用
- 实现文件: `pkg/gateway/optimized_gateway.go`

### 方案 2: 优化 Registry（带缓存层 - 已废弃）

| 测试项 | 延迟/op | 与标准版对比 |
|--------|---------|--------------|
| OptimizedRegistry.GetCommand (热缓存) | 132.2 ns | 3.4x 慢 |
| OptimizedRegistry.GetCommand (冷缓存) | 50.64 ns | 1.3x 慢 |

**结论**: ❌ **不采用**  
标准 Registry 已经使用了 `sync.RWMutex`，性能已经很好。额外的缓存层增加了不必要的开销。

### 方案 2: 对象池

| 测试项 | 延迟/op | 内存/op | 分配/op |
|--------|---------|---------|---------|
| RequestPool | 142.6 ns | 336 B | 2 |
| RequestPool (无池) | 0.86 ns | 0 B | 0 |
| MapPool | 2.45 ns | 0 B | 0 |
| MapPool (无池) | 1.40 ns | 0 B | 0 |
| EventPool | 0.56 ns | 0 B | 0 |
| EventPool (无池) | 0.10 ns | 0 B | 0 |

**结论**: ⚠️ **场景依赖**  
在简单基准测试中，对象池由于获取/放回开销反而更慢。但在高并发、高 GC 压力的场景下，对象池能减少 GC 暂停时间。

### 方案 3: Schema 缓存

| 测试项 | 延迟/op | 与无缓存对比 |
|--------|---------|--------------|
| SchemaCache | 118.5 ns | 3.6x 慢 |
| NoCache | 32.44 ns | - |

**结论**: ❌ **不采用**  
由于 Schema 验证本身很快，缓存查找反而增加了开销。

---

## 关键发现

1. **Registry 无需优化**: 标准实现已使用 `sync.RWMutex`，性能已达纳秒级

2. **Gateway 是主要瓶颈**: 每次请求 28 次内存分配，可通过以下方式优化：
   - 复用 Request 对象
   - 减少上下文创建开销
   - 延迟分配错误对象

3. **Schema 验证可优化**: 对象验证使用反射是主要开销，考虑：
   - 预编译验证规则
   - 使用代码生成代替反射

4. **事件系统性能良好**: 无需额外优化

---

## 推荐优化措施

### 高优先级

1. **Gateway 内存分配优化**
   ```go
   // 使用 sync.Pool 复用 Request 和 Response 对象
   // 减少每次请求的 28 次分配到 <10 次
   ```

2. **Schema 验证代码生成**
   ```go
   // 为常用结构体生成验证代码，替代反射
   // 预期提升：380ns -> ~50ns (7x 提升)
   ```

### 中优先级

3. **上下文优化**
   ```go
   // 批量设置上下文值，减少多次 map 操作
   // 预期提升：268ns -> ~150ns
   ```

4. **事件发布异步化**
   ```go
   // 使用带缓冲的 channel 批量处理事件
   // 预期提升：减少 50% 的事件处理延迟
   ```

### 低优先级

5. **对象池** - 仅在 GC 压力高的场景启用

---

## 性能目标

| 组件 | 当前 | 目标 | 优化路径 |
|------|------|------|----------|
| Registry.Get | 40ns | 40ns | 已达标，无需优化 |
| Gateway.Handle (标准) | 3.1μs | 1.5μs | 内存分配优化 |
| Gateway.Handle (优化) | 3.0μs | 1.5μs | Response 对象池 |
| Schema.Validate(Object) | 380ns | 80ns | 代码生成 |
| Event.Publish | 78ns | 78ns | 已达标 |

---

## 附录：原始测试数据

```bash
$ go test -bench=. -benchmem ./pkg/benchmark/...

BenchmarkRegistry_GetCommand-20                    30697942        39.16 ns/op       0 B/op       0 allocs/op
BenchmarkGateway_ExecuteQuery-20                     531846       3.138 μs/op    1361 B/op      28 allocs/op
BenchmarkSchema_Validate_Object-20                  3158122       380.8 ns/op     480 B/op       2 allocs/op
BenchmarkEventBus_Publish-20                       15291399       78.76 ns/op      84 B/op       0 allocs/op
...
```

---

*报告生成时间: 2026-02-17*  
*测试环境: Linux ARM64*  
*基准测试代码: pkg/benchmark/*
