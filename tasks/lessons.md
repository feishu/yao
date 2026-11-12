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


## 2026-11-09: 达梦数据库（DM8）增量迁移表结构比对失效与误报 DDL 报错 (Dameng Incremental Migrate Diff Failure)
- **Error Pattern**: 在执行 `yao migrate -n <model>` 增量比对时，表明明已经在库中存在，却持续报错：
  - `ChangeColumn id: Error -2654: 表[xxx]已含有IDENTITY列`
  - `CreatePrimary: Error -2864: 表中已存在这样的唯一关键字或主键`
  - `CreateIndex: Error -2140: 索引[xxx]已存在`
- **Why it was wrong**:
  1. **主键识别关联错误**：达梦数据库中主键约束名（`c.CONSTRAINT_NAME` 如 `CONS134221949`）与达梦为该主键自动创建的唯一索引名（`i.INDEX_NAME` 如 `INDEX33571878`）并不相等，但 `ALL_CONSTRAINTS` 中单独维护了 `INDEX_NAME` 字段。原 SQL 误用 `i.INDEX_NAME = c.CONSTRAINT_NAME` 进行 JOIN，导致 `IS_PRIMARY` 始终为 0，已有表的主键被当成了普通唯一索引，比对时系统误判“已有表没有主键”而生成 `CreatePrimary`；
  2. **自增列（IDENTITY）识别机制不符**：达梦数据库 `ALL_TAB_COLUMNS.DATA_DEFAULT` 对自增列返回 `NULL`（达梦不将自增作为普通默认值存储）。原代码判断默认值包含 `IDENTITY` 必定失效，导致已有表的自增属性丢失，类型被反向解析为 `bigInteger` 而非 `ID`，比对时触发 `ChangeColumn id`；
  3. **Interface 隐式指针类型断言盲区**：达梦方言驱动返回的默认值是 `*string` 指针，`blueprint.go` 的 `switch column.Default.(type)` 只判断了 `string` 与 `[]byte`，缺少 `*string` 的解引用，导致模型定义的 `default: 0` (int) 与已有表的指针对比始终不相等，误判列属性变化并试图重建已存在的普通索引。
- **Defensive Rules**:
  - **规则 1（方言元数据精确对齐）**：异构数据库的数据字典系统视图（如 Oracle / DM8 的 `ALL_CONSTRAINTS` / `SYSCOLUMNS`）与 MySQL `information_schema` 存在较大差异。主键关联必须通过 `(i.INDEX_NAME = c.INDEX_NAME OR i.INDEX_NAME = c.CONSTRAINT_NAME) AND c.CONSTRAINT_TYPE = 'P'`，自增列必须通过达梦底层 `SYSCOLUMNS` 的位标记 `(c.INFO2 & 1) = 1` 查询；
  - **规则 2（统一默认值基础类型）**：在 DBAL 列反向解析阶段，所有默认值必须解引用为基础类型（`string`, `int`, `float`, `bool`），严禁保留指针类型暴露给上层 `types.Compare`，避免指针与基础类型比较永远不相等导致的比对漂移；
  - **规则 3（方言 DDL 幂等防护）**：在达梦方言的 `CreateIndex`、`CreatePrimary`、`ChangeColumn` 以及 `DropTableIfExists` 中，必须针对已知表/索引已存在或不存在的错误码（如 `-2140`, `-2864`, `-2654`, `-2106`）提供幂等容错，保障异构数据库增量迁移的稳定性。

## 2026-11-11: 警惕跨网络批量匹配中的 N*M 数据库穿透放大导致接口假死 (Prevent Remote Database Query Amplification in Matching Pipelines)
- **Error Pattern**: 在导诊收敛结算（`matchDepartment`）阶段，针对全平台候选机构循环执行 `findDepartmentByKeywords`、`getDepartmentInfo`、`getDepartmentScheduleInfo` 与 `getOrganizationUuid`，在单一 HTTP 请求中产生了 150~200 次串行跨网络 SQL 查询（北京公网 MySQL），单次请求耗时高达 25.7 秒，导致前端/网关在 15 秒连接超时门限处抛出 `Initial connection: 15.01s` 挂死且无任何返回。
- **Why it was wrong**:
  1. **查询放大盲区（N*M Queries Anti-Pattern）**：在内存中已持有机构列表和多组关键词时，没有批量聚合查询，而是使用多重嵌套循环针对每个机构的每个关键词逐条执行 `LIKE '%kw%'` 远程 SQL，网络往返时延（RTT）被放大了 150 倍以上；
  2. **冗余数据反查与非幂等调用**：上一层查询已经获得了 `id, uuid, name, branch_id`，下游函数又重新向数据库查询同一条记录；且对无需变动的静态实体（如机构 UUID、院区名称）未做内存缓存，导致每个挂号链接生成都产生一次数据库往返；
  3. **非必要的全表运力穿透**：平台级多机构分诊推荐只需轻量运力提示，却对每家医院穿透查询了多张单体人员与排班大表。
- **Defensive Rules**:
  - **规则 1（批量加载+内存过滤铁律）**：在服务层或脚本引擎中，严禁在 `for` 循环内部逐个拼接 SQL 或发起跨网络查询。凡有多实体或多关键词检索需求，必须使用 `IN` 一次性拉取数据集并在内存（JavaScript / Go）中高效完成检索和匹配（0 额外网络时延）；
  - **规则 2（进程级/会话级内存缓存）**：对高频读取且生命周期稳定的元数据（如机构 UUID、院区映射），建立 Map 缓存（如 `orgUuidCache`、`campusCache`），阻断重复数据库往返；
  - **规则 3（端到端超时与延迟门限）**：任何涉及面向患者终端的交互式 API，整个调用链路必须严格控制在 3 秒以内（首选 <1s），杜绝一切接近或超过网关反向代理超时（15s/30s）的设计。

