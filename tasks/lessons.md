# Lessons Learned

## 2026-09-05: 警惕测试接缝过度工程 (No Speculative Seams in Production Flow)
- **Error Pattern**: 在重构 `gou/flow`（解决 Session 串扰与并发数据竞争）时，习惯性将 Phase 2（Process 内核）和 Phase 4（V8 调度）的测试接缝模式套用到 Flow，额外增加了 `seam.go` 并在生产执行循环（`ExecNode` / `processFlows`）中侵入 Mock 拦截逻辑。
- **Why it was wrong**:
  1. 违反了 **Simplicity First (1.1)** 和 **Surgical Changes (1.2)**：用户只要求解决 Session 串扰、并发安全与超时熔断，并未要求 Flow 层 Mock 机制；
  2. Flow 的节点本质上是 Process，上层测试若要拦截完全可以通过 Phase 2 已实现的 `process.WithTestScope` 或 `process.Register` 解决，在 Flow 层额外增加 `seam.go` 属于单用途过度设计（Over-engineering），污染了核心生产代码。
- **Defensive Rule**:
  - 严守需求边界，只解决当前问题，绝不随意添加未经用户确认的“前瞻性/投机性”抽象或辅助功能；
  - 发现过度设计时立即执行 Deletion Test 彻底清理，保持生产代码最小化与绝对纯粹。

## 2026-09-20: 杜绝 Go 互斥锁不可重入导致的系统假死 (No Reentrant Locks on sync.Mutex/RWMutex)
- **Error Pattern**: 在并发加固 `gou/runtime/v8/script.go` 时，外层公共函数 `MakeScript` 已持有写锁 `syncLock.Lock()`，而其下游内部解析流程 `TransformTS` -> `loadModule` / `buildModule` 内部又分别嵌套调用了 `syncLock.Lock()` 与 `syncLock.RLock()`。
- **Why it was wrong**:
  1. **Go 互斥锁不可重入（Non-reentrant）**：Go 标准库 `sync.Mutex` 和 `sync.RWMutex` 严禁同一个 Goroutine 重复加锁。一旦同一 Goroutine 试图获取自身已持有的互斥锁，会导致该 Goroutine 立即进入**永久自我死锁（Self-Deadlock）**；
  2. **局部盲目加锁，缺乏全链路调用审视**：在给内部函数（如 `loadModule` 读写 `ImportMap`/`Modules`）加锁时，仅关注局部变量访问，没有向上反查调用栈（Callers）是否早已处于外层互斥临界区中；
  3. **导致生产致命故障**：该死锁导致在加载包含 TypeScript `import` 依赖的项目（如 `his-adapter`）时，`script.Load` 彻底挂死，`yao start` 进程卡死没有任何输出与响应。
- **Defensive Rules**:
  - **规则 1（零重入铁律）**：在 Go 代码中严禁对同一把锁进行任何形式的嵌套获取。凡在外层已持有锁的调用链内部，绝不可再次获取该锁；
  - **规则 2（内部无锁私有化模式）**：对于已处于持锁上下文的代码，操作共享资源的辅助函数必须声明为私有/无锁版本（如明确命名为 `loadModule` 并在注释声明 `// Caller must hold syncLock`）；如果该方法需要暴露给外部独立调用，则提供无锁的内部实现与加锁的外层包装；
  - **规则 3（真实业务拓扑门禁）**：任何涉及核心运行时、锁机制或加载生命周期的变更，必须在拥有真实多脚本相互引用依赖的业务项目（如 `his-adapter`）下执行真实的 `yao start` 启动与回归测试，绝不单纯依赖无 import 的简单单测。

