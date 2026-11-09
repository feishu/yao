# 专业 ORM 视角：MySQL 与达梦 (Dameng) 在 Xun/Gou 实现对比与缺失分析

## 任务目标
对比分析 `xun/grammar/mysql` 与 `xun/grammar/dameng`（以及关联的 `xun/dbal`、`gou/model`、`gou/schema`），按专业 ORM（如 Eloquent / GORM / Hibernate / TypeORM）标准体系，系统梳理达梦驱动与方言相对于 MySQL 的所有差异、未实现项、实现缺陷与语义不一致点，形成完整的差异全景图与对齐改进方案。

## 阶段工作清单
- [x] Phase 1: 建立对比分析矩阵与测试回环
  - 梳理 MySQL 与 Dameng 的目录结构、实现文件分布、继承关系
  - 编写对比与实机自动化验证用例（`TestLiveDamengSuite`）
- [x] Phase 2: DML / Query 语法层深度对比（Select, Insert, Update, Delete, Upsert, Paginate, Lock, Union, Exists）
  - 发现并实机定位 `CompileExists` 在达梦下直接报语法错误（Error -2007: [as]附近语法分析出错）的重大缺陷
  - 实现达梦特有 `CompileExists`（`SELECT CASE WHEN EXISTS(...) THEN 1 ELSE 0 END FROM DUAL`），实机验证通过
- [x] Phase 3: DDL / Schema Blueprint 层深度对比（Table, Column, Index, Types, Defaults, Renames, Alters）
  - 梳理索引 Schema 作用域 vs 表级命名空间差异
  - 梳理 CLOB / JSON / Text 跨类型双向映射判定与 Diff 误判问题
- [x] Phase 4: Quoter / 标识符 / 表达式 / 驱动特性层深度对比（Backticks, Keywords, Quotes, DSN, Driver）
  - 梳理 `SET IDENTITY_INSERT` 自增显式写入差异
  - 梳理 `CompileInsertOrIgnore` 中基于 `columns[0]` 猜测唯一键的潜在误判风险
  - 梳理常用 SQL 函数差异（`GROUP_CONCAT` vs `LISTAGG`, 多参 `CONCAT` 等）
- [x] Phase 5: 综合评估与对齐建议报告输出（P0 阻塞级、P1 核心功能级、P2 健壮性优化级）
