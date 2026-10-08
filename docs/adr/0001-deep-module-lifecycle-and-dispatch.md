# 0001: 深模块生命周期收口与网关闭环调度架构决策

## 状态
Accepted (已接受)

## 上下文
在 Yao 引擎及其核心运行时生态（`yao`、`gou`、`xun`）中，当前存在四项关键接缝（Seams）与接口深度（Depth）问题：
1. **生命周期认知泄漏与注销闭环缺失**：`yao/cmd/start.go` 手动管理 7 个子系统的停止，但遗漏了 `engine.Unload()` 调用，导致数据库连接池、连接器及 V8 脚本运行时未执行优雅回收；`share/db.go` 保活通道无锁，存在并发数据竞争风险。
2. **Process 调度池化接缝绕过与 HTTP 错误捕获缺陷**：`gou/process.AcquireProcess` 对象池未被 `gou/api/handler.go` 使用，高频请求每秒产生大量 `Process` 堆分配；`gou/server/http/http.go:159` 误用外层 `err` 导致 `srv.Serve` 异常未向上抛出。
3. **DBAL 行扫描缺少 Context 级联取消**：`xun/dbal/query/support.go:mapScan` 在 `for rows.Next()` 循环中未检查 `builder.Context().Err()`，违背了 DBAL 取消不变量。
4. **V8 CGO 跨界数据过度序列化与调试输出残留**：`gou/runtime/v8/bridge/bridge.go` 残留 `fmt.Printf`，小型 Map 未提供基于 CGO 原生属性设置的快速路径。

## 决策
1. **统揽深模块生命周期**：在 `yao/engine` 封装 `engine.Shutdown(ctx)` 深接口，统领 HTTP 排空停听、异步队列、插件、资产及 DB 连接池逆序安全注销，`cmd/start.go` 仅保留单一 `defer` 调用；为 `share/db.go` 探活通道加互斥锁保护。
2. **闭环统一调度器与 HTTP 监听修复**：
   - 修复 `gou/server/http/http.go` 中 `errSrv != nil` 条件判断；
   - 在 `gou/process` 提供统一闭环调度入口 `Dispatch(ctx, inv) (any, error)`，自动执行对象池出池、上下文/全局变量绑定、协同取消检查并在 `defer` 中安全归还对象池；`gou/api/handler.go` 全量接入。
3. **渐进式 DBAL 扫描加固**：
   - 在 `xun/dbal/query/support.go:mapScan`（及 `structScan`）循环内部注入 `ctx.Err()` 强制取消门禁；
   - 为后续高吞吐查询引入紧凑二维列式 `RecordSet` 保留接口接缝。
4. **CGO 跨界净化与小型 Map 快速路径**：
   - 彻底移除 `bridge.go` 中的 `fmt.Printf` 调试代码；
   - 对小型 Map（键值对 <= 8）提供原生 `ObjectTemplate` 直接映射快速路径。

## 结果与影响
- **外部杠杆提升（Leverage）**：CLI 入口与 HTTP Handler 的代码心智负担大幅降低，不再需要关心对象池生命周期与复杂的组件停止顺序。
- **内存与性能收益**：HTTP 核心热路径消除 Process 堆逃逸；跨界小 Map 延迟削减 50%+；孤儿数据库查询在客户端断开时毫秒级中止。
- **并发与退出安全**：消除数据库保活通道数据竞争，实现退出时 100% 资源优雅释放。
