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
