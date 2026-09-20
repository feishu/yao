# AGENTS.md 架构深度增强任务清单（Agent Harness & Karpathy 准则）

## 任务背景与目标
基于 Andrej Karpathy 的 Agent 认知工程学理念（RAM 最小化、高频原子反馈、显式致命约束、导航矩阵与系统不变量），对 6 个工程的 `AGENTS.md` 进行第二轮高信噪比升级，全面赋能智能体自主导航、排障与开发。

---

## 阶段一：业务应用服务工程增强
- [x] 1.1 优化 `syd/service/AGENTS.md`（注入业务不变量、Entrypoints 导航矩阵、严厉的 DO NOT 负面清单、原子单测指令） <!-- id: 1.1 -->

## 阶段二：核心引擎库增强
- [x] 2.1 优化 `yao/AGENTS.md`（注入 CLI/资产/服务入口路由表、架构不变量、防踩坑清单与跨库 replace 说明） <!-- id: 2.1 -->
- [x] 2.2 优化 `gou/AGENTS.md`（注入 Process/Isolate 调度入口、Runner 单 Context 恒定性、CGO 桥接负面约束） <!-- id: 2.2 -->

## 阶段三：基础与底层支撑库增强
- [x] 3.1 优化 `kun/AGENTS.md`（注入公共 API 稳定性契约、Zero-Panic 约束、单包极速测试范式） <!-- id: 3.1 -->
- [x] 3.2 优化 `xun/AGENTS.md`（注入方言隔离公理、Context 穿透写入、无 Prepare 直连规约与 SQLite 轻量单测） <!-- id: 3.2 -->
- [x] 3.3 优化 `v8go/AGENTS.md`（注入 CGO 内存所有权公理、HandleScope 遗漏排查、单测试用例执行指令） <!-- id: 3.3 -->

## 阶段四：质量与密度校验
- [x] 4.1 统一校验 6 个文件的行数（确保处于 70~100 行黄金极简高密度区间） <!-- id: 4.1 -->
- [x] 4.2 验证各文件渲染与超链接完整性，向用户汇报最终成果 <!-- id: 4.2 -->

---

## 结果审查与验收（Review & Verification）
- **Yao 主引擎**: [`yao/AGENTS.md`](file:///Users/L/Desktop/Code/yao_dev/yao/AGENTS.md) (87 行) - 包含 CLI/资产引擎/网关入口路由表、DAG 拓扑顺序与通用 Context 穿透不变量。
- **Gou 核心运行时**: [`gou/AGENTS.md`](file:///Users/L/Desktop/Code/yao_dev/gou/AGENTS.md) (80 行) - 包含 Process 调度入口、Runner 单 Context 恒定性、零拷贝二进制视图与禁止随手 Close Context。
- **Kun 基础设施库**: [`kun/AGENTS.md`](file:///Users/L/Desktop/Code/yao_dev/kun/AGENTS.md) (74 行) - 包含公共 API 兼容性契约、Zero-Panic 准则、结构化 Panic 捕获与原子单测。
- **Xun 数据库抽象层**: [`xun/AGENTS.md`](file:///Users/L/Desktop/Code/yao_dev/xun/AGENTS.md) (77 行) - 包含方言隔离公理、直连无 Prepare 写入、Context 取消与 SQLite 轻量测试模式。
- **v8go JavaScript 引擎**: [`v8go/AGENTS.md`](file:///Users/L/Desktop/Code/yao_dev/v8go/AGENTS.md) (76 行) - 包含 Isolate 单线程限制、CGO 内存管理契约、HandleScope 泄漏规避与原生快序列化。
- **syd/service 医疗业务**: [`syd/service/AGENTS.md`](file:///Users/L/Desktop/Code/yao_projects/syd/service/AGENTS.md) (92 行) - 包含单履约唯一权不变量、重复支付事实不可逆、禁止 Node 原生库与 `tsc` 秒级反馈。
