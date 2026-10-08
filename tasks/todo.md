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

- [x] **Phase 2: 流式接口与 V8 运行时内存泄漏及死锁隐患系统性修复 (SPEC-0003)**
  - [x] 2.1 修复 `gou/api/handler.go`：通道关闭检查 `, ok`，超时等待退出，清理 Runner 全局挂载函数
  - [x] 2.2 修复 `gou/runtime/v8/runner.go`：`runner.reset()` 增加 `ssEvent` / `cancel` 全局变量防御性清除
  - [x] 2.3 修复 `yao/moapi/process.go`：消除裸写入死锁与通道已关闭 Panic，增加 `WaitGroup` 与 Context 监听
  - [x] 2.4 优化 `yao/helper/process.go`：`ProcessSleep` 增加 `process.Context` 取消感知
  - [x] 2.5 编写与运行单元测试：验证流式接口正常关闭、异常断开、并发压测与零泄漏（`TestStreamHandlerChannelCloseAndClientCancel`、`TestRunnerResetCleansGlobalSSEventAndCancel`）全部 PASS
  - [x] 2.6 生成现代化专业 HTML 架构与修复分析报告（`logs/stream_v8_memory_leak_fix_report.html`）

---

## 阶段验收评审 (Review)
1. **多核多协程 V8 调度完全解绑全局锁 (Phase 1)**：`v8go` 在释放和重置上下文句柄时，由全局竞争降级为 Context 实例私有锁，高并发压测下再无互斥串行化停顿。
2. **微架构零逃逸短路收益达成 (Phase 1)**：日志系统在级别关闭时实现 **0 B/op** 零堆分配与 **< 2.5ns/op** 极致短路，彻底根除了全生态各调用点无意义的 `fmt.Sprintf` 逃逸；异常提取实现零正则纯字节状态转移。
3. **全链路流式连接泄漏与死锁彻底根除 (Phase 2)**：
   - 修复了 `c.Stream` 对已关闭通道未检查 `ok` 导致的伪随机空帧广播死循环；
   - 增加了 `streamHandler` 退出时的显式 `cancel()` 广播与 15s 超时安全等待，根除了后台阻塞导致的 Handler 永久挂起；
   - 修复了 `moapi` 的裸发送可能引发的 `panic: send on closed channel` 以及协程孤立问题；
   - 实现了 Runner 在复位时对 `ssEvent`、`cancel` 等动态注入全局函数的深度清除，阻断了闭包与通道在 `iso.cbs` 中的长期累积慢泄漏；
   - `ProcessSleep` 增加了 `process.Context` 感知能力，支持毫秒级中断响应。
4. **全生态 100% 编译与单元测试回归稳健**：涵盖 `v8go`、`kun`、`gou`、`yao` 全部模块，`make vet` 零警告，`go build -o dist/yao .` 编译无误。HTML 交互式报告已输出至 `logs/stream_v8_memory_leak_fix_report.html`。

---

## 阶段三：深模块生命周期收口、深层 Process 调度闭环与 DBAL 门禁 (SPEC-0004)

- [x] **Phase 3.1: 引擎生命周期深度收口与优雅注销 (yao)**
  - [x] 3.1.1 在 `yao/engine/load.go` 中封装统一深接口 `Shutdown(ctx context.Context, stoppers ...func() error) error`（整合排空、停听、取消异步任务、插件卸载、资产卸载、V8停机、DB连接池安全关闭）
  - [x] 3.1.2 改造 `yao/cmd/start.go`，将分散的子系统 defer 统一收口为单一 `defer engine.Shutdown(...)`
  - [x] 3.1.3 为 `yao/share/db.go` 的保活协程通道控制增加互斥锁保护，消除并发重载与停止的数据竞争
  - [x] 3.1.4 编写针对性单元测试验证优雅注销与生命周期有序性（`TestShutdown`、`TestShutdownWithStoppers` 100% PASS）

- [x] **Phase 3.2: HTTP 服务异常信号修复与 Process 闭环调度器 (gou)**
  - [x] 3.2.1 修复 `gou/server/http/http.go:159` 中 `err != nil` 检查笔误，改为 `errSrv != nil` 并校验信号捕获
  - [x] 3.2.2 在 `gou/process/process.go` 中实现统一闭环执行入口 `Dispatch(ctx context.Context, inv Invocation) (interface{}, error)`（自动管理出池、注入、协同取消与 defer 放回，修复 `process.Reset()` 全局 Map 清除问题）
  - [x] 3.2.3 改造 `gou/api/handler.go` 中的 `executeProcess`，全量收口至 `process.Dispatch`
  - [x] 3.2.4 编写并发测试验证在高并发调度下 Process 对象池复用且无状态污染（`TestDispatch`、`TestDispatchConcurrent` 100% PASS）

- [x] **Phase 3.3: Xun DBAL 行扫描 Context 协同取消门禁 (xun)**
  - [x] 3.3.1 在 `xun/dbal/query/support.go` 的 `mapScan`、`recordSetScan`、`structScan` 的 `for rows.Next()` 循环头部加入 `ctx.Err()` 强制取消门禁
  - [x] 3.3.2 编写针对性单测验证在 context 提前取消或超时场景下，行扫描能够立即中断并返回 context 错误（`TestScanContextCancellationDuringIteration` 100% PASS）

- [x] **Phase 3.4: V8 CGO 跨界数据传输净化与小对象加速 (gou/runtime/v8)**
  - [x] 3.4.1 彻底清除 `gou/runtime/v8/bridge/bridge.go:425` 中的 `fmt.Printf` 调试残留
  - [x] 3.4.2 在 `gou/runtime/v8/bridge/bridge.go:JsValue` 为小规模 Map（长度 <= 8 且标量值）引入基于 ObjectTemplate 的属性直接注入快速路径
  - [x] 3.4.3 运行单元与回归测试验证跨界性能与行为完全兼容（`TestJsValueSmallMapFastPath` 100% PASS）

- [x] **Phase 3.5: 全生态编译与回归验证**
  - [x] 3.5.1 在 `xun`、`gou`、`yao` 分别运行针对性单元测试并保证 100% PASS
  - [x] 3.5.2 在 `yao` 与 `gou`、`xun` 中运行 `make vet` 静态分析 100% 干净通过，并重新编译 `dist/yao` 二进制验证通过（版本正常输出 `0.10.7`）

---

## 阶段三验收评审 (Review)
1. **引擎生命周期深度收口 (Phase 3.1)**：
   - 彻底打破原来在 CLI 启动入口与主服务之间分散执行子系统停止的脆弱接缝，封装统一深接口 `engine.Shutdown(ctx, stoppers...)`；
   - 实现了对“服务终止回调 -> 在途请求排空 (DrainInFlight) -> 插件终止 -> DAG 资产逆序卸载 -> V8 停机 -> 连接器/Query 注销 -> Session 停止 -> 数据库连接池安全关闭”的确定性链条式管理；
   - 修复了 `yao/share/db.go` 中数据库保活协程 `dbKeepAliveStop` 的通道并发竞争漏洞。
2. **调度闭环与异常信号广播 (Phase 3.2)**：
   - 修正了 HTTP 服务启动时底层错误捕获的逻辑笔误，杜绝端口冲突静默失败；
   - 封装了统一且不可绕过的 Process 闭环调度器 `process.Dispatch`，自动完成 `sync.Pool` 对象出池、上下文隔离注入、Context 协同取消检查与 `defer p.Release()` 安全回收；彻底修复了 `process.Reset()` 中误删外部全局字典键的隐蔽 Bug。
3. **数据访问层扫描取消门禁 (Phase 3.3)**：
   - 在 `xun` 的 `mapScan`、`recordSetScan` 与 `structScan` 中全面引入 `builder.Context()` 取消门禁；当客户端连接断开或 Context 超时时，立即中止后续行迭代并释放 `sql.Rows`，阻断大查询占用数据库资源。
4. **CGO 桥接传输净化与小对象加速 (Phase 3.4)**：
   - 清除了 `gou/runtime/v8/bridge/bridge.go` 中残留的 `fmt.Printf` 调试输出；
   - 为 <= 8 项标量 Map 注入了基于 `v8go.ObjectTemplate` 的直接映射快速路径，跳过无谓的 JSON 序列化与反序列化。
5. **全生态零警告与产物稳健交付 (Phase 3.5)**：
   - `xun`、`gou`、`yao` 三大核心仓库单测 100% PASS；
   - 全生态 `make vet` 静态分析 0 警告 0 错误；
   - `dist/yao` 重新构建成功并验证正常运行。


