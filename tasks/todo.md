# 阶段九：全面加固系统稳定性、消除并发数据竞争与下线非 MCP AI 功能

系统全栈（`yao`、`gou`、`kun`、`xun`、`v8go`）在完成 P0~P3 核心架构贯通后，针对深度审计发现的未捕获 Panic、WebSocket 假锁竞态、V8/SUI 全局 Map 并发写冲突、空指针兜底缺失以及 Flow 节点堆分配，开展全方位加固；同时按照指示彻底解耦下线非 MCP 的 Neo/AIGC 遗留代码。

---

## 一、下线非 MCP 的 Neo/AI 功能 (解耦瘦身，保留 MCP)

- [x] 1.1 `yao/service/service.go`：移除 Neo API 路由挂载 (`neo.Neo.API`) 与 `neo` 包引用，确保 MCP 协议路由 (`yaoApi.MountMCPRoutes`) 独立正常运作 <!-- id: 1.1 -->
- [x] 1.2 `yao/studio/router.go`：移除 Studio 内的 Neo API 路由挂载与 `neo` 包引用 <!-- id: 1.2 -->
- [x] 1.3 `yao/widgets/app/app.go`：解耦 `neo.Neo` 对 App Setting 的依赖 <!-- id: 1.3 -->
- [x] 1.4 `yao/engine/load.go`：清理残余的 `neo`、`aigc` 包引用与注释，消除冷启动无用加载 <!-- id: 1.4 -->
- [x] 1.5 验证编译：运行 `go build` 确保 Neo 剥离后核心服务与 CLI 正常编译 <!-- id: 1.5 -->

---

## 二、P0 严重稳定性与并发数据竞争加固

- [x] 2.1 【HTTP 网关 Panic 防御】`yao/service/middleware.go` 新增 `withRecovery` 中间件，拦截未捕获 panic 并记录结构化日志，安全返回 HTTP 500 JSON 并保障在途请求排空，补齐单测 <!-- id: 2.1 -->
- [x] 2.2 【WebSocket 并发安全与原子自增 ID】`gou/websocket/hub.go` 废除局部假锁与基于切片长度的 ID 算法，升级为 `atomic.AddUint32` 单调递增 ID，为 `clients` 挂载 `sync.RWMutex` 锁保护，补齐竞态单测 <!-- id: 2.2 -->
- [x] 2.3 【V8 模块与脚本全局注册表并发加固】`gou/runtime/v8/script.go` 消除嵌套加锁与自重入死锁；为 `Scripts` 提供 `ResetScripts()` 并由 `MakeScript` 外层互斥锁保护模块构建，读接口使用 `syncLock.RLock()` <!-- id: 2.3 -->
- [x] 2.4 【SUI 模板与脚本缓存竞态消除】`yao/sui/core/cache.go` 与 `yao/sui/core/script.go` 废除伪异步单 channel 机制，全面采用 `sync.RWMutex` 读写锁保护全局 Map，彻底消灭高并发读写崩溃 <!-- id: 2.4 -->
- [x] 2.5 【Session 与 DBClose 空指针兜底】`gou/session/session.go` 为 `Managers` 挂载读写锁并安全回退到 `"buntdb"`；`yao/share/db.go` `DBClose` 补齐 `capsule.Global` 空指针防御性校验 <!-- id: 2.5 -->

---

## 三、P1~P2 性能优化与开销消除

- [x] 3.1 【P1 响应头去反射】`gou/api/handler.go` 在 `setResponseHeaders` 中消除残留的 `Dot()` 展开与深层反射，直通单点路径寻址 <!-- id: 3.1 -->
- [x] 3.2 【P2 Flow 节点对象池化】`gou/flow/exec.go` 改造 `RunProcess` 使用 `process.AcquireProcess` 与 `defer p.Release()`，消灭高频工作流节点的堆分配与 GC 压力 <!-- id: 3.2 -->
- [x] 3.3 【P2 在途请求排空与优雅重启】实现 `share.InFlight` 并发网关过载保护与热重载排空，保障进程安全重启 <!-- id: 3.3 -->

---

## 四、全量回归与真实项目验证

- [x] 4.1 运行 `gou` 与 `yao` 跨仓库单元测试，全部通过 <!-- id: 4.1 -->
- [x] 4.2 运行静态检查 `go vet ./...`，确保 0 警告、0 错误 <!-- id: 4.2 -->
- [x] 4.3 在真实业务项目 `/Users/L/Desktop/Code/yao_projects/syd/his-adapter` 下运行 `yao start` 实测验证：彻底消除死锁，毫秒级快速启动，控制台完整打印路由、端口和运行状态 <!-- id: 4.3 -->
