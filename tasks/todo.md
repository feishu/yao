# 系统稳定性加固与生产日志治理任务清单

## 任务背景
针对日志分析中暴露出的 6 大高发问题（JWT 401 堆栈刷屏、小程序手机号解密时序、分布式锁提前失效、MySQL 空闲连接跨网段超时断连、日志明文患者敏感信息、HIS 查询成功误判为异常），进行跨端（`yao` 引擎层与 `syd/service` 业务层）系统级协同修复与加固。

## 执行清单

### 1. JWT 401 堆栈降噪 (Guard 中间件优雅响应，杜绝 Gin Recovery Panic 堆栈)
- [x] 1.1 在 `yao/helper/jwt.go` 中新增安全校验函数 `JwtVerify(tokenString string, secret ...[]byte) (*JwtClaims, error)`，校验失败时返回明确 error 而不调用 `exception.Throw()`
- [x] 1.2 在 `yao/service/guards.go` 中重构 `guardBearerJWT`、`guardQueryJWT`、`guardCookieJWT`，使用 `JwtVerify` 替代 `JwtValidate`，Token 失效时优雅返回 HTTP 401 JSON 并 `c.Abort()`，彻底阻断 Gin Recovery Panic 堆栈打印
- [x] 1.3 在 `syd/service/scripts/guard.js` 的 `allowAnonymous` 中，当 Token 过期/无效时静默降级为匿名访问，移除无意义的 `console.error("allowAnonymous error:", error)` 刷屏
- [x] 1.4 编写针对 `JwtVerify` 与 `guards` 的单元测试，验证正常 Token 提取与过期 Token 优雅 401 响应

### 2. 小程序手机号解密时序优化与平滑容错
- [x] 2.1 在 `syd/service/scripts/service/login.ts` 中的 `miniRegisterLogin` 支持微信官方免解密直接获取手机号能力（`phoneCode` 模式调用 `gateway.getPhoneNumber`），彻底规避 `session_key` 刷新时序依赖
- [x] 2.2 针对传统 `encryptedData` + `iv` 解密失败场景，优化错误语义为 `needRelogin` 明确提示，移除敏感参数控制台打印，指导前端平滑发起会话重建

### 3. 分布式锁续期机制与超时阈值优化
- [x] 3.1 在 `syd/service/scripts/service/order/lock.ts` 中根据真实调用链路耗时调整默认 TTL（`PRESCRIPTION_FULFILLMENT` 从 60s 提升至 180s，`PAYMENT_CALLBACK` 提升至 120s）
- [x] 3.2 实现原子锁续期函数 `renewLock(lockType, resourceId, token, extensionSeconds)`（基于 Lua 脚本验证 Token 归属后延长 TTL）
- [x] 3.3 在长耗时事务与路由执行管线中集成锁保活逻辑，优化 `releaseLock` 在锁已被自然过期时的日志级别与分析信息

### 4. MySQL 空闲长连接保活与连接池管理
- [x] 4.1 在 `yao/share/db.go` 中为 `capsule` 连接池配置标准生命周期参数：`SetConnMaxIdleTime(3 * time.Minute)`、`SetConnMaxLifetime(30 * time.Minute)`、`SetMaxIdleConns(10)`、`SetMaxOpenConns(100)`
- [x] 4.2 在 `yao/share/db.go` 中建立轻量级后台保活协程（每 60 秒定时 Ping 连接池各连接），在防火墙超时前主动探活与平滑剔除失效套接字
- [x] 4.3 检查与对齐 `yao/config/types.go`，支持从环境变量（`YAO_DB_MAX_IDLE_CONNS` 等）灵活覆盖连接池参数

### 5. 生产日志脱敏（设计成不打印/静默）
- [x] 5.1 在 `syd/service/scripts/utils/his-adapter.ts` 中移除/屏蔽全量参数与响应中的身份证、手机号、真实姓名打印（设计生产模式静默）
- [x] 5.2 在 `syd/service/scripts/service/login.ts` 中清理手机号授权失败时的 `sessionKey`、`encryptedData` 明文打印
- [x] 5.3 检查腾讯健康上报 `tencent_health_report.ts`，对非关键步骤的前置空参数避免输出虚假错误
- [x] 5.4 检查微信人脸核身调用，确保身份证与姓名不再输出到日志

### 6. HIS 响应成功码修正与空数据语义分析
- [x] 6.1 深入分析 `m1303` 等查询接口在 HIS 返回 `"error": "查询成功"` 时的原始报文结构（区分“查询成功且有数据”与“查询成功但暂无报告记录”）
- [x] 6.2 在 `scripts/utils/his-adapter.ts` 中增加空结果集白名单识别：当 HIS 返回状态码虽非 0 但错误提示明确为 `"查询成功"`、`"未查询到数据"`、`"查无记录"` 时，识别为正常空结果（`isEmpty: true`），而非硬性抛出异常
- [x] 6.3 联动调整 `scripts/service/consumer/report.ts` 等消费端代码，正常接收空数据并返回友好提示，消除 `[HIS-Adapter] 请求最终失败: m1303` 误报

### 7. 编译验证与回归测试
- [x] 7.1 运行 Yao 引擎与相关模块的 Go 单元测试（`TestJwtVerify`、`TestGuardBearerJWT_*`、`TestDBConnectAndPoolSettings` 全部 PASS）
- [x] 7.2 重新构建 Yao 引擎可执行文件，验证无编译警告/报错（构建版本 `0.10.6` 成功）
- [x] 7.3 在 `syd/service` 中语法验证与脚本加载测试通过

---

## 阶段复盘与成果汇总

1. **JWT 401 降噪**: 杜绝了 Gin Recovery Panic，Token 过期/无效直接返回干净的 HTTP 401 JSON，守护日志纯净度。
2. **手机号解密时序**: 推荐采用微信免密 `phoneCode` 模式，传统解密失败时返回 `{ needRelogin: true }`，实现前端无缝静默重试。
3. **分布式锁加固**: 关键锁 TTL 延长至 120s~180s，支持原子续期；锁正常到期释放降为 Info 级别，杜绝伪报警。
4. **MySQL 空闲长连接保活**: 设定 3 分钟空闲回收、30 分钟生命周期与 60 秒主动心跳探活，彻底告别跨网段防火墙超时 `read: connection timed out` 痛点。
5. **生产日志完全脱敏**: 姓名、身份证、手机号、微信 sessionKey 在核心适配层一律静默不打印。
6. **HIS 语义精准识别**: 针对 HIS 返回 `code: 1, error: "查询成功"`（实际无数据）的特殊现象，适配器精准识别为空结果集 (`isEmpty: true`)，不再重试或抛错。
