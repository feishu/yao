---
date: 2026-04-14
topic: yao-browser-rendering
---

# Yao 浏览器渲染能力 Requirements

## Problem Frame
Yao 当前缺少框架内建的 HTML 渲染能力。业务如果要把 HTML 生成 PDF 或 PNG，通常只能额外接 Node 服务、外部 Chrome 微服务，或者引入命令行工具，链路重、部署重、排障也重。

这次要补的是一个框架级能力，让 JS 流程可以直接调用统一进程，把 HTML 字符串渲染成 PDF 或 PNG，同时保持接口语义稳定，不把底层第三方库直接暴露成公共 API。

## Requirements

**公共接口**
- R1. 框架必须提供 `utils.browser.pdf` 进程，用于将 HTML 字符串渲染为 PDF。
- R2. 框架必须提供 `utils.browser.png` 进程，用于将 HTML 字符串渲染为 PNG。
- R3. 公共进程名必须使用能力语义，不能直接暴露为 `rod.pdf`、`rod.png` 等第三方库命名。

**输入与输出**
- R4. 首版输入只支持 HTML 字符串，不支持 URL 输入，也不支持本地 HTML 文件路径输入。
- R5. 两个进程都必须支持 `base64` 返回模式，供 JS 侧直接消费或转存。
- R6. 两个进程都必须支持写文件模式，调用方通过选项决定输出到文件。
- R7. 写文件模式必须同时支持本地绝对路径和 Yao `data` 文件系统路径。
- R8. 首版必须提供最小但够用的渲染选项，至少覆盖 `base_url`、超时、视口尺寸，以及 PDF/PNG 各自必要的导出参数。

**运行时行为**
- R9. 底层实现使用 `github.com/go-rod/rod`，但该实现细节不得泄漏为公共 DSL 契约。
- R10. 浏览器发现顺序必须优先使用显式环境变量或系统已有 Chrome/Chromium，找不到时再尝试 `rod` 的启动/下载机制。
- R11. 参数错误、浏览器不可用、渲染失败时，返回给调用方的错误必须是框架可读的业务异常，而不是底层浏览器堆栈直出。
- R12. 首版必须是无状态渲染能力，不包含登录态页面访问、复杂交互脚本、浏览器池复用等更重的浏览器编排能力。

**验证与可用性**
- R13. 该能力必须能在 CLI `yao run` 与 JS `Process(...)` 调用场景下保持一致行为。
- R14. 至少需要有基础测试覆盖 PDF 成功路径、PNG 成功路径、参数校验失败、浏览器不可用失败。
- R15. 必须提供最小调用示例，说明 `base64` 模式和文件输出模式如何使用。

## Success Criteria
- 可以通过 `Process("utils.browser.pdf", html, options)` 生成可用 PDF。
- 可以通过 `Process("utils.browser.png", html, options)` 生成可用 PNG。
- JS 侧不需要额外起独立渲染服务，就能完成基础报表和截图生成。
- 首版接口边界清楚，后续即使替换底层实现，也不需要修改现有 JS 调用名。

## Scope Boundaries
- 首版不支持 URL 渲染。
- 首版不支持本地 HTML 文件路径输入。
- 首版不做浏览器实例池或长生命周期会话管理。
- 首版不覆盖需要登录态、点击交互、复杂等待条件的自动化浏览器场景。
- 首版不承诺公开 `rod` 级别 API。

## Key Decisions
- 公共接口使用 `utils.browser.*`：框架暴露的是能力，不是第三方实现。
- 首版只接受 HTML 字符串：先把用户当前真实需求打穿，避免把 URL、文件、网络等待策略一起引入。
- 输出同时支持 `base64` 和文件：两者都是低成本高价值能力，缺任一边都会很快补票。
- 文件输出同时支持绝对路径和 `data` 路径：既满足临时落盘，也贴合 Yao 现有文件系统使用习惯。
- 浏览器发现采用“显式配置/系统浏览器优先，`rod` 兜底”：兼顾部署可控性和可用性。

## Dependencies / Assumptions
- 假设目标运行环境允许启动 Chrome/Chromium 类浏览器。
- 假设仓库后续会把 `github.com/go-rod/rod` 纳入依赖并处理对应运行时要求。
- 已验证当前仓库没有现成的浏览器渲染模块，只有附件层对 PDF/PNG MIME 和存储的支持。

## Outstanding Questions

### Resolve Before Planning
- 无

### Deferred to Planning
- [Affects R8][Technical] `options` 的最终字段命名和默认值如何设计，才能和现有 `process.ArgsMap(...)` 风格保持一致。
- [Affects R7][Technical] `data` 文件系统路径与本地绝对路径的判定规则怎么定义，才能减少歧义。
- [Affects R10][Needs research] 浏览器发现逻辑中，环境变量键名、系统路径扫描顺序、以及 `rod/lib/launcher` 的兜底策略如何排序更稳。
- [Affects R14][Needs research] 在当前仓库测试环境下，浏览器相关测试如何保证可重复且不过度脆弱。

## Alternatives Considered
- 直接暴露 `rod.pdf` / `rod.png`：实现快，但把第三方库名写进公共契约，后续替换实现的代价太高。
- 先建完整浏览器服务层：长期结构更稳，但对这次需求来说过重，容易提前抽象。
- 当前选择是最小可交付路径：先做框架内建能力，保持接口中性，后续再视真实使用量决定是否抽象浏览器池。

## Next Steps
-> /ce:plan for structured implementation planning
