# Lessons Learned

## 2026-09-05: 警惕测试接缝过度工程 (No Speculative Seams in Production Flow)
- **Error Pattern**: 在重构 `gou/flow`（解决 Session 串扰与并发数据竞争）时，习惯性将 Phase 2（Process 内核）和 Phase 4（V8 调度）的测试接缝模式套用到 Flow，额外增加了 `seam.go` 并在生产执行循环（`ExecNode` / `processFlows`）中侵入 Mock 拦截逻辑。
- **Why it was wrong**:
  1. 违反了 **Simplicity First (1.1)** 和 **Surgical Changes (1.2)**：用户只要求解决 Session 串扰、并发安全与超时熔断，并未要求 Flow 层 Mock 机制；
  2. Flow 的节点本质上是 Process，上层测试若要拦截完全可以通过 Phase 2 已实现的 `process.WithTestScope` 或 `process.Register` 解决，在 Flow 层额外增加 `seam.go` 属于单用途过度设计（Over-engineering），污染了核心生产代码。
- **Defensive Rule**:
  - 严守需求边界，只解决当前问题，绝不随意添加未经用户确认的“前瞻性/投机性”抽象或辅助功能；
  - 发现过度设计时立即执行 Deletion Test 彻底清理，保持生产代码最小化与绝对纯粹。
