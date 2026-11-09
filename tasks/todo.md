# 达梦数据库下 `yao migrate -n sys.menu` 报错排查与修复计划

## 问题现象
执行 `yao migrate -n sys.menu`（不带 `--reset` 的增量比对迁移）时报错：
```
更新表结构 model: sys.menu (sys_menus)	FAILURE
ChangeColumn id: Error -2654: 第1 行附近出现错误:
表[sys_menus]已含有IDENTITY列
CreatePrimary: Error -2864: 第1 行附近出现错误:
表中已存在这样的唯一关键字或主键
```

## 任务执行清单
- [x] 1. 分析根因：排查为什么在已建好表的情况下，增量迁移会生成 `ChangeColumn id` 和 `CreatePrimary`，分析 `GetTable` 探测结果与模型 Blueprint 的 Diff 差异
- [x] 2. 检查达梦方言中的表结构反向工程（`GetColumnListing`、`GetIndexListing`、`GetPrimary` 或主键/自增属性识别）
- [x] 3. 检查 Xun 的 `TableDiff` / `TableSave` 逻辑中针对达梦方言的 `ChangeColumn` 与 `CreatePrimary` 生成规则
- [x] 4. 实施针对性修复（在 `xun/grammar/dameng` 与 `gou/schema/xun` 中补齐主键与自增列解析、默认值指针解引用与 DDL 幂等保护）
- [x] 5. 重新编译并执行 `cd /Users/L/Desktop/Code/yao_projects/syd/service && ./yao migrate -n sys.menu` 进行严格验证（输出 SUCCESS）
- [x] 6. 验证全量/其它模型的增量 migrate 及 reset migrate 不受回归影响，并总结复盘

## 根因与修复总结
1. **主键识别失效导致 `CreatePrimary: -2864`**：
   - 原 SQL 使用 `i.INDEX_NAME = c.CONSTRAINT_NAME` 关联，达梦中主键约束名与索引名并不相等，但 `ALL_CONSTRAINTS` 中单独维护了 `INDEX_NAME` 字段。改用 `(i.INDEX_NAME = c.INDEX_NAME OR i.INDEX_NAME = c.CONSTRAINT_NAME) AND c.CONSTRAINT_TYPE = 'P'`，并增加 `ALL_CONSTRAINTS` 的直接查询兜底，使达梦主键 100% 正确被识别。
2. **自增列（IDENTITY）识别失效导致 `ChangeColumn id: -2654`**：
   - 达梦系统表 `ALL_TAB_COLUMNS.DATA_DEFAULT` 对自增列返回 `NULL`，原代码通过判断默认值包含 `IDENTITY` 必定失效。通过达梦系统视图 `SYSCOLUMNS`（`INFO2 & 1 = 1`）精准提取自增列名并标记 `column.Extra = AutoIncrement`，使反向解析的列类型与模型一致为 `ID`，消除了误判的 `ChangeColumn id`。
3. **默认值指针未解引用导致列属性误判触发重建索引**：
   - 达梦驱动返回的默认值是 `*string` 指针，`blueprint.go` 缺少 `*string` 的解引用，导致模型定义的 `default: 0` (int) 与已有表的指针对比始终不相等，误判列属性变化并试图重建已存在的普通索引。现已统一家解引用并对达梦 `CreateIndex: -2140` 增加幂等容错。
4. **编译与回归验证**：
   - 重新构建 `yao` 并部署到 `syd/service/yao`，执行 `yao migrate -n sys.menu` 和 `yao migrate -n sys.menu --reset` 均 100% 输出 `SUCCESS`。
