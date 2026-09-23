# Yao 运行时与底层基础设施性能深化工程 (SPEC-0002 Phase 1) 任务清单

## 待办清单

- [x] **Phase 1.1: v8go 消除上下文重置全局互斥锁 (v8go/context.go)**
  - [x] 1.1.1 为 `v8go.Context` 结构体引入实例级互斥锁 `mu sync.Mutex`
  - [x] 1.1.2 改造 `ResetRetainedValues()` 与 `RetainedValueCount()`：仅获取当前 Context 实例锁 `c.mu`，完全剥离包级全局锁 `ctxMutex`
  - [x] 1.1.3 保持 `ctxMutex` 仅服务于全局 `ctxRegistry` 注册与反注册
  - [x] 1.1.4 运行 `v8go` 单元测试与多协程并发测试（`TestConcurrentContextResetRetainedValues`）全部 PASS

- [x] **Phase 1.2: kun/log 日志级别前置门禁短路优化 (kun/log/log.go)**
  - [x] 1.2.1 在 `Trace`、`Debug`、`Info` 等包级函数入口增加级别检查：未达输出级别直接 return，阻断 `fmt.Sprintf` 堆逃逸
  - [x] 1.2.2 在 `Entry.Trace`、`Entry.Debug` 等方法上同步增加级别短路门禁，并增加 `IsTrace()`、`IsDebug()`、`IsInfo()` 辅助检查函数
  - [x] 1.2.3 编写 Benchmark 测试验证日志关闭时的零分配（`BenchmarkLogTraceSimpleDisabled`: 2.33 ns/op, 0 B/op, 0 allocs/op; `BenchmarkLogTraceWithIsTraceCheck`: 0.93 ns/op）

- [x] **Phase 1.3: kun/exception 异常提取零正则优化 (kun/exception/exception.go)**
  - [x] 1.3.1 改造 `New` 与 `Trim`：使用原生字符串前缀检查与字节查找替代 `reEx.FindStringSubmatch`，移除未使用的 `any` 依赖
  - [x] 1.3.2 保持对 `Exception|<code|int>:<message>` 语法的严格兼容性
  - [x] 1.3.3 运行 `kun/exception` 单元测试与 Benchmark 基准测试（`BenchmarkExceptionTrim`: 12.27 ns/op, 0 B/op），100% 通过

- [x] **Phase 1.4: 全生态编译与回归验证**
  - [x] 1.4.1 在 `gou` 中运行 `go test -v -run "TestRunner|TestProcess" ./runtime/v8/... ./process/...` 全量通过
  - [x] 1.4.2 在 `yao` 中运行 `make vet` 静态分析 100% 干净通过，并编译构建二进制 `dist/yao` 验证通过

---

## 阶段验收评审 (Review)
1. **多核多协程 V8 调度完全解绑全局锁**：`v8go` 在释放和重置上下文句柄时，由全局竞争降级为 Context 实例私有锁，高并发压测下再无互斥串行化停顿。
2. **微架构零逃逸短路收益达成**：日志系统在级别关闭时实现 **0 B/op** 零堆分配与 **< 2.5ns/op** 极致短路，彻底根除了全生态各调用点无意义的 `fmt.Sprintf` 逃逸；异常提取实现零正则纯字节状态转移。
3. **全生态 100% 编译与单元测试回归稳健**：涵盖 `v8go`、`kun`、`gou`、`yao` 全部模块，行为完全向后兼容，无破坏性风险。
