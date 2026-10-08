# SPEC-0004: 引擎生命周期收口、深层 Process 调度闭环与 DBAL 取消门禁技术规范

## Problem Statement

Yao 引擎在高并发生产环境与服务优雅退出时存在关键架构摩擦：
1. 服务运维与重启过程中，由于主入口生命周期管理分散且遗漏全局卸载，数据库连接与子进程无法确定性清理，后台协程存在潜在竞争；
2. 高频 HTTP 请求每秒无谓分配大量 Process 调度对象，CGO 跨界数据过度依赖 JSON 序列化并残留调试打印，浪费 GC 资源与 CPU 周期；
3. 大查询在客户端断开或超时后，底层行扫描仍盲目持续扫描完全量数据，浪费数据库吞吐量与连接池资源；
4. HTTP 服务端口监听异常由于判断逻辑缺陷未能可靠向调度事件广播。

## Solution

根据深模块（Deep Module）设计哲学，重构四大核心接缝：
1. 建立深层生命周期管理器 `engine.Shutdown(ctx)`，将所有排空、停止、反向卸载与资源回收完全封装在引擎内部，实现单行调用优雅关闭，并修复数据库保活竞态；
2. 修复 HTTP 服务启动异常信号广播，建立统一闭环 Process 调度器 `process.Dispatch`，自动完成 `sync.Pool` 对象复用与状态清除；
3. 在数据访问层 `xun` 行扫描主循环中注入协同上下文取消门禁；
4. 清理 V8 桥接层调试输出，为小型 Map 注入原生 CGO 属性映射快速通道。

## User Stories

1. 作为运维工程师，我希望在执行服务退出或接收系统终止信号时，引擎能够自动排空在途请求并逆序关闭所有资产、连接池和插件，从而避免产生孤儿连接和僵尸进程。
2. 作为业务开发者，我希望客户端在提前中断 HTTP 请求或超时发生时，底层正在执行的数据库查询行扫描能毫秒级停止，从而防止慢查询占用数据库连接池。
3. 作为架构师，我希望 HTTP 网关每秒处理万级请求时，Process 调度对象能够被高效复用而不是每次在堆上分配，从而显著降低垃圾回收（GC）开销与内存停顿。
4. 作为系统开发者，我希望当 HTTP 监听端口冲突或网络异常导致服务启动失败时，引擎能够立即向外部抛出明确的退出事件和错误日志，而不是静默忽略。
5. 作为脚本开发者，我希望将包含基础租户与用户上下文的小型数据传入 JavaScript 时能够直接映射，从而降低 CGO 跨界调用的微秒级延迟。

## Implementation Decisions

### 1. 引擎深层生命周期门面 (engine.Lifecycle)
- 在 `yao/engine` 模块提供 `Shutdown(ctx context.Context) error` 深接口。
- 内部顺序执行：排空在途请求（`share.DrainInFlight`）、停止 HTTP 监听、停止异步任务与调度、卸载插件、逆序卸载 DAG 资产、停止 V8 运行时、关闭外部连接器与 Query 引擎、最后关闭数据库连接池（`share.DBClose`）。
- 在 `yao/cmd/start.go` 的信号捕获与正常退出链路中使用该统一接口替换散落的各个子组件单独 `Stop()` 调用。
- 在 `yao/share/db.go` 为保活协程 `dbKeepAliveStop` 引入 `sync.Mutex` 保护，防止并发停止与重载时重复关闭 channel。

### 2. 网关异常广播与 Process 调度闭环 (gou/api & gou/process)
- 修正 `gou/server/http/http.go` 中 `srv.Serve(listener)` 的错误分支判断，确保服务启动异常能够即时推送到 `server.signal <- ERROR`。
- 在 `gou/process` 提供统一闭环执行函数：
  ```go
  type Invocation struct {
      Name   string
      Args   []interface{}
      SID    string
      Global map[string]interface{}
  }
  func Dispatch(ctx context.Context, inv Invocation) (interface{}, error)
  ```
  内部自动执行 `AcquireProcess`、上下文与变量注入、取消校验、执行与提取结果，并在 `defer` 中强制 `Release`。
- 将 `gou/api/handler.go` 中的 `executeProcess` 彻底收口为统一调度器调用。

### 3. DBAL 行扫描 Context 门禁 (xun/dbal/query)
- 在 `xun/dbal/query/support.go` 的 `mapScan` 与 `structScan` 循环内部首行增加协同取消门禁：
  ```go
  if ctx := builder.Context(); ctx != nil && ctx.Err() != nil {
      return nil, ctx.Err()
  }
  ```
  一旦上层 Context 超时或被取消，立即提前退出并关闭 `sql.Rows`。

### 4. V8 CGO 跨界数据传输净化与加速 (gou/runtime/v8/bridge)
- 彻底移除 `gou/runtime/v8/bridge/bridge.go` 中残留的 `fmt.Printf` 调试输出。
- 在 `JsValue` 对小规模 Map（长度 <= 8 且不含复杂嵌套）提供直接通过 `ObjectTemplate` 设置属性的快速通道，避免无谓的 JSON 序列化。

## Testing Decisions

- **黑盒与接口契约测试**：所有测试用例必须针对深模块的对外接口（`engine.Shutdown`、`process.Dispatch`、`builder.Select().Get()`），严禁测试模块私有内部细节。
- **并发与取消验证**：
  - 编写超时 Context 下的 DB 查询测试，验证查询在被取消后立即中断且不泄露连接；
  - 编写并发 Process 调度测试，验证在高并发调用下 `sync.Pool` 对象复用率达到预期且无脏状态复用；
  - 编写生命周期优雅关闭测试，验证在存在在途请求时 `Shutdown` 先排空请求后关闭资源的有序性。
- **参考先例（Prior Art）**：
  - 参考 `yao/share/inflight_test.go` 的在途请求并发测试；
  - 参考 `gou/runtime/v8/runner_test.go` 的并发资源释放测试。

## Out of Scope

- 不变更现有的 Yao 业务 DSL 规范（`.mod.yao`、`.http.yao` 语法保持 100% 向后兼容）；
- 不在此阶段重写所有历史 SQL 语法编译器；
- 不在此阶段引入完全重构的二进制 FlatBuffers 跨界协议。

## Further Notes

- 本技术规范严格遵循 ADR-0001 的设计决策；
- 实施阶段将通过 `/implement` 驱动 TDD（红灯测试先行，绿灯重构交付）。
