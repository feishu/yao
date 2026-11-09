# 达梦数据库下 `yao migrate -n syd_ih.mo_drug_store` 报错排查与修复计划

## 问题现象
执行 `yao migrate -n syd_ih.mo_drug_store` 时报错：
```
更新表结构 model: syd_ih.mo_drug_store (syd_ih_mo_inst_drug_store)	失败: strconv.ParseFloat: parsing "'0.00'": invalid syntax
```

## 根因分析
1. **达梦系统表默认值格式带单引号**：
   - 在达梦数据库中，`ALL_TAB_COLUMNS.DATA_DEFAULT` 存储的是建表时的原始 SQL 默认值表达式。对于设置了字符串或带引号默认值（例如模型中 `"default": "0.00"`）的列，系统表返回的默认值包含单引号 `'0.00'`，且可能包含换行或空格。
   - `xun/grammar/dameng/schema.go` 的 `GetColumnListing` 中使用 `fmt.Sprintf("%v", dataDefault)` 直接将 `"'0.00'"` 赋给 `column.Default`。
2. **`gou/schema/xun/blueprint.go` 解析浮点/整数默认值未去除外层引号**：
   - 在 `TableToBlueprint` -> `parseColumnType` 中，对于 `float/decimal/double` 类型，直接执行 `any.Of(v).CFloat()`。
   - 内部通过 `strconv.ParseFloat(v, 64)` 解析字符串，遇到带单引号的 `"'0.00'"` 抛出语法错误 panic。
   - 同样对于 `integer` 和 `boolean` 类型列，如果默认值为带单引号的字符串（如 `"'0'"`、`"'1'"`），也会导致转换失败或类型错判。
3. **`gou/schema/types/diff.go` 中默认值比对类型不一致问题**：
   - 模型 Blueprint 中默认值若是字符串 `"0.00"` 或由 JSON 反序列化的 `float64(0)`，与已有表转换后的数值默认值直接使用 `origin.Default != column.Default` 比对会导致类型不匹配，误判默认值发生变更。

## 任务执行清单
- [x] 1. 在 `xun/grammar/dameng/schema.go` 中规范化处理列默认值，去除外层包裹的单引号及空白，并正确识别 `NULL` 默认值
- [x] 2. 在 `gou/schema/xun/blueprint.go` 中优化 `ColumnToBlueprint` 与 `parseColumnType`，对默认值安全去除单引号及空白，并使用安全的数值转换避免 panic
- [x] 3. 在 `gou/schema/types/diff.go` 中完善默认值相等性比对逻辑（`isDefaultEqual`），支持数值与数字字符串等价比较
- [x] 4. 在 `kun/any/any.go` 与 `kun/num/number.go`、`xun/xun.go` 中优化 `CFloat64` / `Float64`，支持去除两端多余引号的数字字符串解析
- [x] 5. 重新编译 `yao` 并部署到测试环境进行验证（执行 `yao migrate -n syd_ih.mo_drug_store` 输出 SUCCESS）
- [x] 6. 验证全量/其它模型 migrate 不受回归影响，并总结复盘

## 修复总结与验证结果
1. **修复成果**：
   - `xun/grammar/dameng/schema.go`：在反查达梦系统表列结构时，自动识别并清理 `DATA_DEFAULT` 外层包裹的单引号及换行符，将 `NULL` 正确解析为 nil。
   - `gou/schema/xun/blueprint.go`：新增 `cleanDefaultString` 函数，对所有从表反向映射到 Blueprint 的列默认值统一清理单引号；对于 `decimal/float/integer/boolean` 等类型增加防御性类型安全转换，杜绝因语法异常导致 panic。
   - `gou/schema/types/diff.go`：引入 `isDefaultEqual`，使 `"0.00"` 与 `0.0`、`"0"` 与 `0`、`"false"` 与 `0` 等跨类型等价默认值能够正确判定为一致，消除误判的列结构变更操作。
   - `kun/any`、`kun/num`、`xun`：全局兜底数值解析函数，对带引号字符串进行剥离保护。
2. **验证结果**：
   - `yao migrate -n syd_ih.mo_drug_store` -> `SUCCESS`
   - `yao migrate -n sys.menu` -> `SUCCESS`
   - `yao migrate -n syd_ih.payment_route_runtime` -> `SUCCESS`
