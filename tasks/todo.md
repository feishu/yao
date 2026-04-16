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
