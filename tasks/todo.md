# 智能分诊追问引擎、红旗规则联动与知识库质量闭环修复计划

> 目标仓库：`/Users/L/Desktop/Code/yao_dev/yao` (`triagekb`)、`/Users/L/Desktop/Code/yao_projects/syd/service` (TS 状态机 / API / KB 种子与构建脚本)、`/Users/L/Desktop/Code/yao_projects/syd/portal-h5` (`app/triage/page.tsx`)
> 已确认方案决策：OQ-1 (Option A)、OQ-2 (Option A)、OQ-3 (Yes)、OQ-4 (Yes)、OQ-5 (Yes)

## 阶段工作清单

### Phase 1: 知识库规则与回归用例梳理与重构（`syd/service`）
- [x] 分析 `safety_rules.json`、`question_tree.json` (78 题)、`symptoms.json` (434 症状) 与 `tests/regression_cases.json` 的映射关系与冲突项
- [x] 修复 `safety_rules.json`：重构为 47 条有效规则，对齐 `question_id` 与 `value`，补齐共享红旗题 `primary_symptoms`（含 `RULE_CRIT_ACUTE_GLAUCOMA` 覆盖 `Q_EYE_PRESSURE_HEADACHE` 关联主诉），覆盖 59 个 `is_red_flag: true` 选项（0 悬空）
- [x] 增强 `scripts/setup/build_federated_sqlite_kb.py`：新增 `validate_seed_integrity()` 构建前强门禁，重建 `kb_triage_general.db` 与 `kb_triage_tcm.db` 且 0 门禁错误
- [x] 修复并扩充 `data/triage/tests/regression_cases.json` 至 136 例（含 `URG_01..12`、`CONV_01..04` 与全量 `EMG_01..40` 对齐）

### Phase 2: Go `triagekb` 推理引擎核心修复（`yao/triagekb`）
- [x] 更新 `types.go`：`ClinicalContext` 新增 `Answers map[string]string`；`InquiryOption` 新增 `IsUncertain`；`TriageDecision` 新增 `SafetyState`、`CanEarlyTerminate`、`AnsweredLabels`
- [x] 更新 `pipeline.go`：
  - 重构 `selectNextInquiry`：按 `question_id` 去重彻底消除同一题无限死循环；分级排序 `CRITICAL 红旗(3) > RED_FLAG(2) > DIFFERENTIAL(1)` 并按 `QuestionID` 确定性排序
  - 重构 `evaluateSafety`：基于 `ctx.Answers` 精确匹配 `(question_id, answer_value)`，移除 `IsRedFlagCandidate` 与 13 字符前缀误伤，支持 `CRITICAL` 立即 `EMERGENCY_NOW` 短路与 `URGENT` 升级收敛，并实现 `is_red_flag` 选项兜底与红旗 `UNCERTAIN` 安全兜底
  - 更新 `routeWesternServices` 与服务级早收敛（OQ-5）：融合 `option.service_hint`（鉴别/红旗选项 `+3.5`），聚合单科室疾病分并按 `ServiceCode` 确定性平手决胜，`RED_FLAG` 清零后不可翻转首选科室时直接 `CONVERGED`
- [x] 补充 `engine_test.go` 单元测试并执行 `go vet ./triagekb/... && go test -race -count=1 -v ./triagekb/...` 全部通过（8/8 含子测试）

### Phase 3: TypeScript 状态机、会话幂等与证据标签修复（`syd/service`）
- [x] 更新 `scripts/service/triage/session.ts`、`types.ts`、`engine/state_machine.ts`：
  - `evaluateViaGraphEngine` 传入 `answers: qaAnswers` 并透传 `is_uncertain`
  - `buildSafetyFromGraph` 同时映射 `EMERGENCY_NOW` (`isShortCircuit: true`) 与 `URGENT_ASSESSMENT` (`isShortCircuit: false`, 完整 `escalation`)
  - 同一 `(question_id, answer_value)` 重复提交在 `session.ts` / `state_machine.ts` 保持幂等不递增步数/事件数；同题改答不惩罚步数；`ADD_SEEDS` 做主诉并集保留
  - `URGENT_ASSESSMENT` 无剩余 `CRITICAL` 红旗题或引擎 `CONVERGED` 时直接收敛为 `COMPLETED`
- [x] 更新 `scripts/service/triage/api.ts`、`engine/department_matcher.ts`：
  - 从 Go `decision.answered_labels` 自动回填真实中文选项文案至 `accumulated_evidence`，消除 `"已确认特征"`
  - 按引擎 `can_early_terminate` 与 `SAFETY_STATE` 准确控制提前收敛入口；在 `COMPLETED` + `URGENT_ASSESSMENT` 返回完整 `safety_state`、`escalation` 与 `emergency_action`
- [x] 更新并验证 `triage_engine_test.ts`、`triage_graph_test.ts`、`triage_test.ts`、`platform_triage_test.ts` 全部通过

### Phase 4: H5 前端紧急告警状态同步修复（`syd/portal-h5`）
- [x] 修复 `app/triage/page.tsx` 与 `components/triage/types.ts`：抽取 `applyStepOutcome`，在 `COMPLETED` 分支同步 `setEscalation` / `setEmergencyAction`，确保 `URGENT_ASSESSMENT` 下 `InformedConsentModal` 展示完整规则标题与警告正文
- [x] 更新 `components/triage/result/triage-result-card.tsx`：在 `result.safety_state === 'URGENT_ASSESSMENT'` 时于结果卡顶部持久渲染琥珀色紧急就医警示横幅

### Phase 5: 编译新二进制并执行跨仓端到端全量验证
- [x] 构建本地生产二进制 `/Users/L/Desktop/Code/yao_dev/yao/dist/yao`
- [x] 验收通过全部 Go 测试、Yao 业务/回归套件及用户原红眼病/青光眼 API Trace

### Phase 6: 导诊步骤接口响应慢/超时排查与性能手术优化（`department_matcher.ts`）
- [x] 故障根因复现与定位：`pushStep` 在问答触发 `CLINICAL_RECOMMENDATION` 时调用 `matchDepartment`，平台模式下针对 10 个机构循环执行 `findDepartmentByKeywords`、`getDepartmentInfo`、`getDepartmentScheduleInfo` 和 `getOrganizationUuid`，产生 150~200 次跨网串行远程 MySQL 查询（北京 39.101.71.171），耗时高达 25.7 秒，远超 Next.js / 浏览器 15 秒连接超时门限
- [x] 方案实施：
  - [x] 机构范围收敛与机构 UUID 内存预热：`activeOrgs` 从 10 限制为 5 家；建立 `orgUuidCache`，直接由 `activeOrgs` 预热，消除所有重复 `models.sys.organization.Find`
  - [x] 科室批量加载与内存关键词秒级检索：将单关键词多次 `like '%kw%'` 远程 SQL 改为单次批量查询 `models.sys.his_departments`（`org_id in [orgIds]`），在内存中执行关键字优先匹配
  - [x] 院区信息与科室详情去重缓存：直接复用科室记录已有字段，避免重复查询 `sys_his_departments`，对 `sys_campus` 建立内存缓存
  - [x] 排班与运力查询轻量化：避免对每个机构穿透多张单体表，平台模式下按需批量聚合或直接复用默认安全运力
- [x] 验证闭环：
  - [x] 使用 `time yao run scripts.service.triage.api.pushStep ...` 验证端到端耗时从 25.7 秒降至 2.7 秒以内（冷启动包含 DAG 模型加载共 8.7s）
  - [x] 使用 `curl` 验证 HTTP 接口 `POST /api/v1/triage/session/step` 在运行的 HTTP 服务上 3 秒内返回 200 OK
  - [x] 确保 `scripts.test.platform_triage_test.run` (17/17)、`triage_engine_test.run` (34/34)、`triage_test.run` (136/136)、`triage_graph_test.run` (7/7) 全量测试持续 100% 通过
  - [x] 模拟前端请求 `http://localhost:3000/api/backend/v1/triage/session/step` 验证经 Next.js 代理正常 200 返回，0 超时，前端无任何挂起



## 验收结果摘要
- `python3 scripts/setup/build_federated_sqlite_kb.py`: 通过（主诉=434, 服务=30, 规则=47, 题库=78, 已覆盖红旗选项=59）
- `go vet ./triagekb/... && go test -race -count=1 -v ./triagekb/...`: 8 个测试套件全部 `PASS`
- `dist/yao run scripts.test.triage_graph_test.run`: `7/7` 通过，`criticalRecall = "100%"`
- `dist/yao run scripts.test.triage_engine_test.run`: `34/34` 通过（含红眼病/青光眼 YES-NO-UNCERTAIN、中文标签、幂等、种子并集）
- `dist/yao run scripts.test.triage_test.run`: `136/136` 通过（急救 40/40 = `100.0%`，紧急升级 12/12 = `100.0%`，常规 73/73，边界 11/11）
- `dist/yao run scripts.test.platform_triage_test.run`: `17/17` 通过
- `npm run type-check` (`portal-h5`): `app/triage/` 与 `components/triage/` 下 **0 个 TypeScript 编译错误**（其余无关历史测试/旧目录存在原有 TS 报错）
