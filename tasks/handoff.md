# Yao Dev (高性能后端平台) MCP 声明式架构研发交接与接续实施指引

本文档作为 `yao_dev` 项目在实现**声明式 API-to-MCP (Model Context Protocol)** 核心能力后的全局交接基准。后续会话无论由任何 Agent 或工程师接手，均可通过本指南快速恢复完整上下文，并无缝进入下一阶段的业务试点与生产化推进。

---

## 1. 项目核心背景与不可变原则

### 1.1 战略定位与架构边界
1. **纯粹的高性能企业级后端平台 + MCP 服务提供者**:
   - **坚决不内置 Agent**: 官方代码（`github/yao` 1.0.0 分支）走向了庞大的 Agent 内置沙箱与容器虚拟化（包含 666+ 个文件与重型容器依赖）。而 `yao_dev` 的战略定位是纯粹的**高性能企业级业务后端基座**。业务 Agent 留给外部生态（如 Claude Desktop、Cursor、Dify、Coze、自研 Agent 编排框架），Yao 仅负责输出高可靠、带事务、带鉴权的 MCP 业务工具能力。
2. **无需 Model 转 MCP**:
   - 严禁直接将底层数据库 Model/Table 批量暴露为 MCP Tools，防止 LLM 直接穿透底层表 CRUD，杜绝业务状态机绕过与数据完整性破坏。
3. **精准声明式 API-to-MCP**:
   - 仅将具备明确业务动作、适合 AI 调用的 API 暴露为 MCP 工具；
   - **就地声明原则**: 业务开发者直接在现有的 `apis/*.http.yao` 路由定义中添加 `"mcp": true` 或结构化对象，严禁额外引入冗余的 `*.mcp.yao` 或孤立的 Schema 映射文件。

### 1.2 仓库与双仓协作结构
- **核心框架库 (`gou`)**: `/Users/L/Desktop/Code/yao_dev/gou`
  - 负责底层 DSL 解析（`gou/api`）、V8 运行时调度、轻量 MCP 核心服务器与协议分发器（`gou/mcp/server`）。
- **引擎应用主仓 (`yao`)**: `/Users/L/Desktop/Code/yao_dev/yao`
  - 负责业务引擎装载、API 自动投影为 MCP Tool、Gin 路由生命周期挂载（`yao/service`）、进程上下文（Session/Authorized）透传调度。

---

## 2. 已完成里程碑与工程资产 (MCP 核心已 100% 达成)

截至当前版本，`yao_dev` 声明式 API-to-MCP 架构已全部实现并通过完整单测与端到端集成测试，核心资产沉淀如下：

### 2.1 核心模块资产明细

| 模块 / 路径 | 核心职责与关键实现 | 关联测试用例 |
| :--- | :--- | :--- |
| **`gou/api/types.go`** | 扩展路由 DSL：新增 `MCP *MCPConfig` 与 `RawMCP interface{}`，定义 `MCPConfig` 与 `MCPParam`，实现 `NormalizeMCP()` 自动兼容布尔简写与高级对象配置。 | `gou/api/mcp_test.go` (`TestPathMCPDeclaration`) |
| **`gou/api/api.go`** | 路由解析闭环：在 `LoadSource` 解析阶段自动调用 `NormalizeMCP`，完成就地规范化。 | `gou/api/mcp_test.go` |
| **`gou/mcp/server/broker.go`** | **安全会话总线 (`SessionBroker`)**：封装带有互斥与原子状态机的 `Session`，提供幂等安全 `Close()` 与防并发关闭 panic 的 `Send()`，支持背压与缓冲满丢包防范，挂载请求元信息缓存。 | `gou/mcp/server/broker_test.go` (`TestSessionBrokerLifecycle`, `TestSessionIdempotentClose`, `TestSessionBufferFull`, `TestSessionConcurrentSendAndClose`) |
| **`gou/mcp/server/server.go`** | 纯净版 MCP Server 引擎：完整实现 JSON-RPC 2.0 协议（`initialize`, `tools/list`, `tools/call`, `ping`），接入 `SessionBroker`，并将 HTTP 传输层上下文（Headers / Query / Remote）通过 `RequestInfo` 注入 context。 | `gou/mcp/server/server_test.go` (`TestMCPServerProtocols`) |
| **`gou/mcp/server/registry.go`** | 分组注册表：提供 `GetServer(group)`，按业务域/租户隔离 MCP 服务实例。 | `gou/mcp/server/server_test.go` |
| **`gou/mcp/mcp.go`** | 核心门面接缝：暴露顶层 `GetServer(group)` 访问入口。 | `gou/mcp/server/server_test.go` |
| **`yao/api/mcp.go`** | **自动投影适配器 + 安全鉴权接缝**：<br>1. 扫描系统加载的 API 路由，将声明 `MCP` 的端点投影为 `types.Tool`；<br>2. **智能 Schema 合成器 (`synthesizeSchemaFromIn`)**：自动解析 `:payload`、`$param.*`、`$query.*` 生成严格 JSON Schema，杜绝 AI 幻觉；<br>3. **Auth Adapter 鉴权接缝 (`executeGuards`)**：在工具调用前自动执行 API 声明的 Guard 链（如 `bearer-jwt` 等），拦截未授权请求，放行授权请求并将解析得到的 `__sid` 和 `__authorized` 上下文安全注入 `process.Process`；<br>4. 调度桥接器：将 LLM 的调用入参精准还原为 API `in` 参数切片。 | `yao/api/mcp_test.go` (`TestMCPProjectionAndExecution`, `TestMCPSSEFullBidirectionalWorkflow`, `TestSmartSchemaSynthesizer`, `TestMCPGuardAuthentication`) |
| **`yao/service/service.go`** | 路由生命周期挂载：在 `Start` 与 `Restart` 时自动挂载 `/v1/mcps/:group/sse`、`/v1/mcps/:group/messages` 与 `/v1/mcps/:group`。 | `yao/api/mcp_test.go` |

---

## 3. 验证基线与质量门禁

在接续任何新开发或修改前，必须运行以下门禁命令确保 100% 绿灯：

```bash
# Go 环境：推荐使用系统已验证的 Go 1.23.4
export PATH=/usr/local/go/bin:$PATH

# 1. 验证 gou 运行时 DSL 规范化、SessionBroker 并发安全与 MCP Server 协议测试（开启 -race 检测）
cd /Users/L/Desktop/Code/yao_dev/gou
go test -v -race -count=1 ./mcp/server
go test -v -count=1 ./api -run TestPathMCPDeclaration

# 2. 验证 yao 顶层 API 投影、双向 SSE 握手、Guard 认证拦截放行与端到端 Process 执行测试（开启 -race 检测）
cd /Users/L/Desktop/Code/yao_dev/yao
go test -v -race -count=1 ./api -run "TestMCP|TestSmartSchemaSynthesizer"

# 3. 静态检查
go vet ./mcp/server
cd /Users/L/Desktop/Code/yao_dev/yao && go vet ./api/...
```

### 当前基线测试运行结果：
- `gou/mcp/server`: `TestSessionBrokerLifecycle` -> **PASS (0.00s)**
- `gou/mcp/server`: `TestSessionIdempotentClose` -> **PASS (0.00s)**
- `gou/mcp/server`: `TestSessionBufferFull` -> **PASS (0.00s)**
- `gou/mcp/server`: `TestSessionConcurrentSendAndClose` (50 并发) -> **PASS (0.01s, 0 race)**
- `gou/mcp/server`: `TestMCPServerProtocols` -> **PASS (0.00s)**
- `gou/api`: `TestPathMCPDeclaration` -> **PASS (0.00s)**
- `yao/api`: `TestMCPProjectionAndExecution` -> **PASS (0.00s)**
- `yao/api`: `TestMCPSSEFullBidirectionalWorkflow` -> **PASS (0.02s, 0 race)**
- `yao/api`: `TestSmartSchemaSynthesizer` -> **PASS (0.00s)**
- `yao/api`: `TestMCPGuardAuthentication` (拦截/放行/上下文/SSE继承) -> **PASS (0.02s, 0 race)**

---

## 4. 核心架构认知与避坑准则

接手的工程师或 Agent 务必注意以下经实践验证的技术要点：

1. **SSE 双向消息通道规范（必须符合 Anthropic/MCP 标准）**:
   - 客户端建立 `GET /v1/mcps/:group/sse` 长连接后，服务端下发 `event: endpoint` 告知客户端推送消息的 URL（如 `/v1/mcps/:group/messages?session_id=xxx`）。
   - 客户端后续向 `/messages?session_id=xxx` 发送 POST JSON-RPC 请求时，**POST 请求自身必须立即返回 `202 Accepted`**；而 RPC 的响应体（Response）**必须异步通过最初建立的 SSE 流（`event: message`）推回给客户端**。
   - 若直接在 POST 中响应结果，标准客户端（如 Claude Desktop）会挂起直至超时。当前代码已完整支持该标准机制，并兼顾无 session 直连降级模式。
2. **零手写 Schema 与智能参数推导**:
   - 开发者只需标注 `"mcp": true`，`synthesizeSchemaFromIn()` 会自动解析 `in: ["$param.id", "$query.status", ":payload"]`，提取变量名并推导字段类型，保证生成的 `inputSchema` 具备完整的字段声明。
   - 若开发者需要更精细控制，可使用结构化声明：
     ```json
     "mcp": {
       "name": "cancel_appointment",
       "description": "取消患者挂号预约",
       "params": {
         "id": { "type": "string", "description": "预约记录ID", "required": true },
         "reason": { "type": "string", "description": "取消原因" }
       }
     }
     ```
3. **Process 上下文契约**:
   - `process.Process` 中用户会话与安全身份态是通过 `p.WithGlobal(map[string]interface{}{"__authorized": auth})` 与 `p.WithSID(sid)` 进行透传的，保证被 MCP 调用的内部 Process 与常规 HTTP API 拥有完全一致的权限切面。

---

## 5. 下一阶段接续实施路线 (Next Steps)

### 阶段目标：从引擎能力迈向真实业务试点与生产联调

1. **真实业务接口试点声明 (Pilot in Real Business APIs)**:
   - 选择 1~2 个现有典型的业务 API（例如医院系统的 `apis/v1/patient.http.yao` 或 `apis/appointment.http.yao`）；
   - 在关键接口（如查询排班、挂号、取消预约）上添加 `"mcp": true` 或精细化 `mcp` 声明；
   - 启动本地 Yao 引擎（`./yao start`），检查日志中 MCP 工具注册输出。
2. **外部主流 Agent 客户端真实联调**:
   - **Claude Desktop 配置**:
     修改 `~/Library/Application Support/Claude/claude_desktop_config.json`：
     ```json
     {
       "mcpServers": {
         "hospital_service": {
           "url": "http://127.0.0.1:5099/v1/mcps/default/sse"
         }
       }
     }
     ```
   - **Cursor / 自研 Agent 编排平台接入**:
     配置 MCP SSE URL，触发 Agent 自然语言推理，验证其自动识别工具、传参调用、获取返回值及完成业务闭环。
3. **企业级权限与安全切面加固**:
   - 验证 API 声明的 `guard`（如 `bearer-jwt`）在 MCP 场景下的鉴权拦截；
   - 支持客户端在请求头中携带 `Authorization: Bearer <token>` 并在挂载的 SSE 与 Messages 通道中自动解析并绑定到 Process。
4. **代码提交与发布**:
   - 对 `gou` 仓库（`types.go`, `api.go`, `mcp/server/`）与 `yao` 仓库（`api/mcp.go`, `service/service.go`）进行分支提交。

---

## 6. 新会话进入与执行检查清单 (Session Checklist)

新 Agent/开发者开始工作时，请按以下步骤执行：
- [ ] 1. 检查工作区 Git 状态：`git status`（确认在 `yao_dev/yao` 和 `yao_dev/gou` 分支）。
- [ ] 2. 运行第 3 节的质量门禁命令，确保测试 100% 绿灯。
- [ ] 3. 查阅 `tasks/todo.md` 与本 `tasks/handoff.md` 确认当前任务排期。
- [ ] 4. 若进行业务接口试点，创建并遵循最小侵入变更原则，实施后使用端到端测试或 curl/MCP Inspector 验证。
