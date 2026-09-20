# 全电脑 AI Agent Skills 资产归一化治理与分发重建任务清单

## 任务背景
本机历史上经过多套工具与手动方式引入了 271 个 Agent Skills，目前散落在 6 个不同目录中，并存在 55 个空目录失效、47 个技能脱离中央库等严重问题。
本任务旨在将 `~/.skillshub` 打造成唯一的单一真实数据源（SSOT），全量聚合散落的有效技能源码，清空目标端历史残余，并重建标准、统一、可维护的分发体系。

## 执行清单

### 1. 全量安全快照备份（零风险基石）
- [x] 1.1 创建带时间戳的统一备份目录 `~/.skills_backup_20260918_213049`
- [x] 1.2 全量归档备份 `~/.skillshub`、`~/.claude/skills`、`~/.cursor/skills`、`~/.codex/skills`、`~/.gemini/config/skills`、`~/.gemini/skills`、`~/.agents/skills`
- [x] 1.3 备份 Skills Hub 核心数据库 `skills_hub.db`
- [x] 1.4 验证备份产物完整性与文件数（共 7 个目录 + SQLite 数据库全量归档）

### 2. 数据源逆向聚合与去重补全（构建健全 SSOT）
- [x] 2.1 修复 55 个空目录：从 `~/.claude/skills` 将 55 个真实技能源码完整回填到 `~/.skillshub`
- [x] 2.2 迁移 47 个 Go 专家技能：将 `~/.gemini/config/skills` 中的 `golang-*` 实体迁移合并入 `~/.skillshub`
- [x] 2.3 验证并确保自研/特殊技能（`yao-service`、`handoff`、`cn-resume-optimizer-main`）在 `~/.skillshub` 中完整健全
- [x] 2.4 全盘自动化校验 `~/.skillshub`，确认所有 271 个技能目录均含有有效 `SKILL.md`，无任何空壳文件夹（271/271 100% 健全）
- [x] 2.5 在 `~/.skillshub` 初始化 Git 本地仓库，做首个基准版本 Commit（提交了 2368 个文件），锁定版本安全基线

### 3. 目标端安全清空（消除旧软链、重复副本与断链）
- [x] 3.1 安全清空目标工具目录中的历史旧内容：
  - `~/.claude/skills` (清空 271 项)
  - `~/.gemini/config/skills` 与 `~/.gemini/skills` (各清空 271 项)
  - `~/.cursor/skills` (清空 271 项)
  - `~/.codex/skills` (清空 276 项)
  - `~/.agents/skills` (清空 274 项)
- [x] 3.2 校验目标端目录已完全净空，无旧幽灵文件或死链残留（已全部验证 items_count=0）

### 4. 统一分发与投影重建（以 `~/.skillshub` 为唯一源头）
- [x] 4.1 重建各 AI 客户端的统一分发软链接（全量 271/271，0 错误）：
  - Claude Code（`~/.claude/skills`）-> `~/.skillshub/<skill>`
  - Gemini Antigravity（`~/.gemini/config/skills`）-> `~/.skillshub/<skill>`
  - Gemini CLI（`~/.gemini/skills`）-> `~/.skillshub/<skill>`
  - Codex（`~/.codex/skills`）-> `~/.skillshub/<skill>`
  - Agents（`~/.agents/skills`）-> `~/.skillshub/<skill>`
  - Cursor（`~/.cursor/skills`）-> `~/.skillshub/<skill>`
- [x] 4.2 校准 Skills Hub 数据库 `skills_hub.db`，同步更新 skills (271条) 与 targets (8835条) 记录为 ok
- [x] 4.3 自动化检测各目标端软链接有效性（全平台 271/271 SKILL.md 可读，0 断链）

### 5. 验收与交付
- [x] 5.1 验证 Gemini Antigravity / Claude Code / Cursor 等工具下的 Skills 可用性（全部 PASS）
- [x] 5.2 编写执行总结与复盘，提供后续 Git 同步更新使用指南

---

## 治理复盘与架构成果

1. **确立中央权威数据源 (SSOT)**：
   - 彻底摆脱 6 处重叠混乱的历史包袱，所有 271 个技能的代码物理实体唯一存放在 `~/.skillshub`。
   - 在 `~/.skillshub` 初始化了 Git 版本库并完成了 2368 个文件的全量 Baseline Commit，版本安全彻底受控。
2. **修复 55 个幽灵空目录**:
   - 修复了 Skills Hub 此前遗留的空壳目录漏洞，将真实完整的 `SKILL.md` 回填，使 `benchmark`、`browse`、`ship`、`qa` 等核心短名技能全端复活。
3. **收拢 47 个 Go 专业技能与自研技能**:
   - 将散落在 Gemini 目录的 Go 专家技能及 `yao-service` 等项目核心技能纳管回中央库。
4. **全自动零冗余分发**:
   - 6 大主流目标工具通过软链接统一投影至 `~/.skillshub`，修改一处即全端生效，极大节省磁盘空间并保持实时绝对同步。
