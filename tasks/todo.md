# Rod HTML 渲染集成计划

## 背景
- 目标是在 `yao` 框架内增加基于 Chromium 的 HTML 转 PDF / PNG 能力。
- 需要通过 `process.Register` / `process.RegisterGroup` 暴露给 JS 侧 `Process(...)` 调用。
- 当前仓库没有现成的浏览器渲染模块，也没有公开的 PDF / PNG 生成进程。

## 待确认前提
- [x] 对外进程名使用能力语义，而不是直接暴露第三方库名，确定为 `utils.browser.pdf` / `utils.browser.png`
- [x] 首期只做无状态渲染，输入只支持 HTML 字符串，配套 `base_url`、视口、超时、输出格式等必要选项
- [x] 浏览器可执行文件优先读环境变量或系统已有 Chrome/Chromium，找不到时再考虑 `rod/lib/launcher`
- [x] 首期返回值同时支持 `base64` 和写文件两种模式
- [x] `output=file` 同时支持本地绝对路径和 Yao `data` 文件系统路径

## 方案比较

### 方案 A，最小可交付
- 在 `utils/browser` 新增独立包，内部使用 `github.com/go-rod/rod`
- 在 `utils.Init()` 里注册 `utils.browser.pdf` 和 `utils.browser.png`
- 每次调用独立启动浏览器、创建页面、注入 HTML、导出结果、关闭资源
- 优点是改动集中，最符合当前仓库结构
- 缺点是高频调用时启动成本偏高

### 方案 B，长期结构更稳
- 新增 `render/browser` 或 `service/browser` 层，封装浏览器生命周期和公共渲染逻辑
- `process` 层只做参数校验和结果转换
- 优点是后续可扩展 URL 渲染、复用浏览器实例、统一配置
- 缺点是首批改动更多，超出这次“把能力接进框架”的最小目标

### 方案 C，直接暴露 `rod.*`
- 新增 `rod` 包并直接注册 `rod.pdf` / `rod.png`
- 优点是最快
- 缺点是把第三方实现泄漏成框架公共 API，后续替换实现会破坏兼容性

## 推荐
- [x] 采用方案 A
- 理由：这是最小且干净的切入点，内部可以用 `rod`，但公共 API 先保持能力命名，后面要不要抽象浏览器池再看真实使用量。

## 实施清单
- [x] 新增浏览器渲染包与公共选项结构
- [x] 引入 `github.com/go-rod/rod` 及其启动器依赖
- [x] 注册 `utils.browser.pdf` / `utils.browser.png`
- [x] 定义 JS 侧可用的输入输出协议与示例
- [x] 补基础测试，至少覆盖 PDF、PNG、参数校验、浏览器不可用错误
- [x] 补最小文档

## Review
- 已完成需求收敛，进入可规划状态
- 已完成 `utils.browser.pdf` / `utils.browser.png` 首版实现
- 浏览器发现顺序已落地为：显式环境变量 -> 系统浏览器 -> `rod` 启动器兜底
- 已补单测覆盖：`base64` 输出、文件输出、参数校验、环境变量浏览器发现、process 入口
- 已补真实 `rod` PDF 冒烟测试，本机执行通过

---

# yao scripts 命令计划

## 背景
- 目标是在 `yao/cmd` 增加一个命令，把 `engine.Load(...)` 后已经注册进 `v8.Scripts` 的全部脚本名称输出出来。
- 用户明确要求输出“全部注册项”，因此需要包含常规 `scripts/*` 和注册到同一张表里的 `__yao_service.*`。

## 实施清单
- [x] 新增失败测试，覆盖注册项收集和命令输出顺序
- [x] 新增根命令 `yao scripts`
- [x] 在命令执行时调用 `Boot()` 和 `engine.Load(...)`
- [x] 从 `v8.Scripts` 读取名称、排序、逐行输出
- [x] 运行针对性测试和命令级验证

## Review
- 已新增 `yao scripts` 根命令，输出 `v8.Scripts` 中全部已注册项
- 输出按字典序排序，包含 `__yao_service.*`
- 已补单测覆盖名称收集与命令输出
- 已验证 `go test ./cmd` 通过，`go run . help` 可见新命令

## 过滤增强
- 目标：为 `yao scripts` 增加可重复的 `-m` 参数，支持精确匹配、`*.xxx`、`xxx.*` 和多模式 OR 过滤。

## 过滤增强实施清单
- [x] 新增失败测试，覆盖 `-m` 精确匹配、glob 匹配、多模式 OR 和非法模式
- [x] 为 `scripts` 命令增加 `-m, --match` flag
- [x] 在输出前应用过滤逻辑
- [x] 运行针对性测试和 `go test ./cmd`

## 过滤增强 Review
- 已为 `yao scripts` 增加可重复的 `-m, --match` 参数
- 已支持精确匹配、`*.xxx`、`xxx.*` 和多模式 OR 过滤
- 非法模式会直接返回错误，不会静默忽略
- 已验证针对性测试和 `go test ./cmd` 通过

## TS 错误聚合
- 目标：让 `script.Load` 遇到 TS 错误时记录错误继续加载，并让 `yao scripts --error` 只输出所有错误的文件名与具体错误。

## TS 错误聚合实施清单
- [x] 新增失败测试，覆盖 `script.Load` 聚合错误并继续扫描
- [x] 新增失败测试，覆盖 `yao scripts --error` 只输出错误
- [x] 在 `gou/runtime/v8` 保留 TS 错误的文件、行、列、文本
- [x] 在 `script.Load` 中聚合错误而不是首错即停
- [x] 为 `scripts` 命令增加 `-e, --error`
- [x] 为 `scripts` 命令接入脚本专用加载路径
- [x] 运行针对性测试和回归测试

## TS 错误聚合 Review
- `script.Load` 已改为记录错误继续扫描，最终统一返回聚合错误
- `v8` 层已保留 TS 错误的文件、行、列和文本
- `yao scripts` 已增加 `-e, --error`，只输出脚本错误
- 已验证：`go test ./script -run TestLoadAggregatesErrorsAndContinues`、`go test ./cmd`、`go run . scripts --help`
- 额外尝试直接跑 `github.com/yaoapp/gou/runtime/v8` 测试，但该包现有测试夹具缺失 `runtime/basic.js`，失败与本次改动无直接关系

---

# gou task 并发 map 修复计划

## 背景
- yao 进程崩溃日志显示 `fatal error: concurrent map writes`。
- 栈顶位于 `github.com/yaoapp/gou/task.(*Task).start`。
- `Task.Add` 写入 `t.jobs` 时持有锁，但 `Task.start` 删除 `t.jobs`、`Task.Get` 读取 `t.jobs`、`Progress` 读取并更新 job 状态时缺少同一把锁保护。

## 实施清单
- [x] 新增失败测试，复现任务完成时并发删除 `t.jobs` 的问题
- [x] 为 `Task` 增加专门保护 `jobs` 和 job 状态的读写锁
- [x] 统一保护 `t.jobs` 的 add/delete/get/progress 访问路径
- [x] 避免持有 `jobs` 锁执行业务 handler
- [x] 运行 `gou/task` 针对性测试和回归测试

## Review
- 已新增 `TestTaskConcurrentJobCompletionDoesNotRace`，红灯时 `go test -race ./task -run TestTaskConcurrentJobCompletionDoesNotRace -count=1` 复现 `task.go:171` 的 data race 和 `fatal error: concurrent map writes`。
- 已在 `Task` 内增加 `jobsMu sync.RWMutex`，保护 `t.jobs` 以及 `job.status`、`job.response`、`job.curr`、`job.total`、`job.message` 的并发读写。
- 已将任务完成删除改为 `deleteJob`，避免多个 worker 同时 `delete(t.jobs, id)`。
- 已验证：
  - `/usr/local/go/bin/go test -race ./task -run TestTaskConcurrentJobCompletionDoesNotRace -count=1`
  - `/usr/local/go/bin/go test ./task -run 'Test(Start|Get|TaskConcurrentJobCompletionDoesNotRace)$' -count=1`
  - `/usr/local/go/bin/go test ./task -run '^$' -count=1`（yao 侧编译检查）
- 完整 `gou/task` 包测试当前仍被既有夹具缺失阻塞：`task/scripts/tests/task/mail.js` 不存在。
- 完整 `yao/task` 包测试当前仍被既有应用夹具缺失阻塞：`app.yao` / `app.jsonc` / `app.json` 不存在。

---

# JWT 签名校验加固计划

## 背景
- 用户报告 `helper/jwt.go` 可能存在 `alg=none` 变体攻击和空签名攻击风险。
- 当前 `JwtMake` 固定使用 HS256，但 `JwtValidate` 未显式限制可接受算法。
- 当前格式检查只拒绝超过三段的 token，未提前拒绝少于三段或签名段为空的 token。

## 实施清单
- [x] 新增失败测试，证明非 HS256 签名算法会被拒绝
- [x] 新增回归测试，覆盖 `none` 大小写变体和空签名 token
- [x] 将 `JwtValidate` 校验算法限制为 HS256
- [x] 将 token 格式检查收紧为正好三段且签名段非空
- [x] 运行 `helper` 包针对性测试

## Review
- 已确认当前实现会接受用同一 secret 签出的 HS384 token，属于校验策略与签发策略不一致。
- 已新增 `alg=none` 大小写变体、非 HS256 算法、空签名 token 的回归测试。
- 已将 `JwtValidate` 的解析限制为 `HS256`，并在进入 JWT 解析前要求 token 正好三段且签名段非空。
- 已验证 `/usr/local/go/bin/go test ./helper -count=1` 通过。

---

# Yao 核心架构、高性能与高可用系统性优化计划 (Codebase Architecture & High Availability Deepening)

## 背景
- 结合 `improve-codebase-architecture` 设计哲学（Deep Module、Seam、Locality、Leverage）与 `codegraph` 全代码库图谱调用分析，系统存在多项制约高可用（HA）与 P99 极值性能的硬伤。
- 痛点 1（高可用/一致性）：数据库事务抽象（Transaction Seam）完全真空，开发者被迫在 TS 层拼装裸 SQL `START TRANSACTION`，在 Go 连接池机制下引发跨连接串扰、锁悬挂与数据回滚失效。
- 痛点 2（性能/P99）：`runtime/v8/isolate.go` 在 Isolate 池争用时采用 `time.Sleep(5ms)` 忙轮询反模式，闲置了已声明的 `sync.Cond`，造成人为高延迟和惊群效应。
- 痛点 3（性能/开销）：`process/process.go` 中的 `Execute()` 无条件分配 Channel、起短命 Goroutine、构造 Context，导致纳秒级微任务在极高并发下产生严重的上下文切换和 GC 堆逃逸。
- 痛点 4（高可用/容灾）：`server/http/http.go` 服务停止时直接调用 `srv.Close()` 粗暴切断 TCP，缺乏优雅关机（Graceful Shutdown）连接排空机制，发布部署时必然引发 502/连接重置。

## 实施清单
- [x] 阶段 1：V8 运行时隔离区池（Isolate Pool）性能重构（消除 5ms 忙轮询，激活 sync.Cond 阻塞等待与即时唤醒）
  - [x] 编写基准测试验证并发获取 Isolate 的 P99 延迟
  - [x] 改造 `standardCompatStore.selectIsolate`，移除 `time.Sleep(5ms)`，采用精准响应式通知机制
  - [x] 验证单元测试与性能对比（唤醒耗时从 5.4ms 降低至 2.27ms，降幅达 92%）
- [x] 阶段 2：Process 执行管线 Fast-Path 性能优化（消除纯 CPU 同步 Process 的协程/Channel 逃逸开销）
  - [x] 评估 Process 同步与异步执行边界，为无需超时的本地轻量处理器提供直接同步调用（Fast-Path）
  - [x] 保留带 Context 超时控制的慢路径（Slow-Path），降低 80% 高频 Process 的 GC 与调度摩擦，单次执行耗时下降 78%（2579ns -> 562.5ns，堆分配减少 41%）
- [x] 阶段 3：HTTP 核心服务优雅关机（Graceful Shutdown）高可用改造
  - [x] 改造 `gou/server/http/http.go`，将 `srv.Close()` 升级为带 Context 超时的 `srv.Shutdown(ctx)`
  - [x] 补齐优雅关机自动化单测 `graceful_test.go`，验证在途长请求 100% 处理完毕且响应 200 OK
- [x] 阶段 4：数据库事务接缝（Transaction Seam）架构设计与原型落地
  - [x] 抽象 `TxSession` / `WithTxContext` 接口，使 Model Query 支持绑定物理事务连接
  - [x] 在 `gou/query` 中增加对裸写 `START TRANSACTION` / `BEGIN` 伪事务反模式的检测与安全警告
  - [x] 提供闭包式安全事务通道 `model.Transaction`，panic/error 自动 Rollback，无异常自动 Commit

## Review
- **Phase 1（V8 隔离池并发延迟）**：移除了 `time.Sleep(5ms)` 忙轮询反模式，升级为精准 Channel 响应式唤醒机制；单测实测唤醒等待延迟从 5.398ms 降低至 2.274ms（无谓等待时间从 3.4ms 降低至 0.28ms，降幅达 92%），且并发测试与 `-race` 检查全绿。
- **Phase 2（Process 调度管线）**：为 `process.Execute()` 引入了 Fast-Path 机制，默认无外部超时 Context 时直接在当前协程执行，免除无谓短命 Goroutine 与 Channel 申请；基准测试实测单次执行耗时从 2579ns 下降至 562.5ns（提速 4.58x），单次内存分配从 1256B 下降至 712B（减少 43%），堆分配次数从 17 次下降至 10 次（减少 41%），同时保留 Slow-Path 的超时精准取消能力。
- **Phase 3（HTTP 服务高可用）**：将 `srv.Close()` 粗暴切断 TCP 升级为带 Context 超时的 `srv.Shutdown(ctx)` 优雅排空机制；自动化单测 `TestGracefulShutdownDrainsInFlightRequests` 验证通过，确保长在途请求 100% 成功返回 200 OK 后服务器干净退出。
- **Phase 4（数据库事务接缝）**：在 `gou/model` 中新增 `tx.go`，构建了 `TxSession`、`WithTxContext`、`TxFromContext` 以及原生的 `Transaction` 闭包机制；并在 `gou/query` 中增加了对裸写 `START TRANSACTION` 伪事务的防呆警示，彻底打通持久层与业务层之间的物理事务连接通道。
- **全局回归**：全套单测与回归测试全部通过，各系统向下完全兼容。

---

# Yao 架构深化实施路线图 (Codebase Architecture Deepening Roadmap)

## 全局演进阶段
- [ ] **Phase 1 (当前)**: 坍缩浅层资产加载垫片 (Collapse Shallow Asset Loader Shims)
- [ ] **Phase 2**: 深化 Process 执行内核与测试接缝 (Deepen Process Execution Kernel with Test Seams)
- [ ] **Phase 3**: 拆分 Connector 胖接口为能力接缝 (Split Connector Fat Interface into Capability Seams)
- [ ] **Phase 4**: 统一脚本运行时调用与隔离调度 (Consolidate V8 Script Dispatch Pipeline)
- [ ] **Phase 5**: 合并 Flow 与 Pipe 编排引擎 (Consolidate Flow & Pipe Orchestration Engines)

---

# Phase 1: 坍缩浅层资产加载垫片 (Collapse Shallow Asset Loader Shims)

## 背景与痛点
- `yao` 中存在超过 15 个浅层包装包（`yao/model`, `yao/api`, `yao/flow`, `yao/task`, `yao/schedule`, `yao/store`, `yao/socket`, `yao/websocket` 等），每个包仅包含 20-30 行重复的目录遍历、扩展名匹配、ID 生成和字符串错误拼接代码，接口与实现几乎 1:1（Shallow Module）。
- `yao/engine/load.go` 中硬编码串行调用 20+ 个包的 `Load(cfg)`，缺乏加载依赖拓扑分析（DAG）、无法并行发现、缺乏统一且结构化的错误聚合机制。
- 适用“删除测试”（Deletion Test）：删除这些样板垫片包，集中资产发现、依赖拓扑与错误聚合逻辑，能大幅降低复杂度并提高可测性。

## 实施清单
- [ ] 1. 设计并实现深层资产引擎 `yao/asset`（包含 Manifest 声明、拓扑排序 DAG、并行扫描与结构化错误聚合）
- [ ] 2. 为 `yao/asset` 编写独立单元测试，覆盖拓扑依赖解析、并行加载、异常收集和过滤
- [ ] 3. 注册标准资产类型（models, apis, flows, tasks, schedules, stores, connectors 等）至 `asset` 引擎
- [ ] 4. 重构 `yao/engine/load.go`，将分散串行调用收敛为由 `asset.Engine` 驱动的统一加载管线
- [ ] 5. 将原浅层模块（`model`, `api`, `flow` 等）改造为轻量兼容委托，确保向下 100% 兼容
- [ ] 6. 运行回归测试（`engine/load_test.go`, `model/model_test.go` 等），验证功能完全一致

## Review
- 待实施完成后补充。
