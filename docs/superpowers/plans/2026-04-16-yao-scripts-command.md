# yao scripts 命令 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `yao scripts` 根命令，在应用加载完成后输出排序后的全部已注册脚本名称，并支持通过可重复的 `-m` 模式过滤以及通过 `-e` 输出聚合到的 TS 错误。

**Architecture:** 命令层负责触发脚本专用加载路径。`script.Load` 改为在遍历 `scripts` 和 `services` 时收集错误并继续扫描，最后统一返回聚合错误；`v8` 层保留 TS 错误的结构化位置信息。`scripts --error` 直接消费这些聚合错误，只输出错误项，不输出正常脚本名。

**Tech Stack:** Go、cobra、`github.com/yaoapp/gou/runtime/v8`

---

### Task 1: 补失败测试

**Files:**
- Create: `cmd/scripts_test.go`
- Test: `cmd/scripts_test.go`

- [ ] **Step 1: 写一个失败测试，验证注册项名称会按字典序返回**

```go
func TestRegisteredScriptNamesSorted(t *testing.T) {
    // 预填充无序脚本名称，断言返回值已排序
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd -run 'TestRegisteredScriptNamesSorted|TestScriptsCmdOutputsAllRegisteredScripts'`
Expected: FAIL，提示缺少待实现函数或命令输出不符合预期

- [ ] **Step 3: 再写一个失败测试，验证命令输出全部注册项**

```go
func TestScriptsCmdOutputsAllRegisteredScripts(t *testing.T) {
    // 断言包含普通脚本和 __yao_service.*，并按顺序输出
}
```

- [ ] **Step 4: 再次运行测试确认失败原因正确**

Run: `go test ./cmd -run 'TestRegisteredScriptNamesSorted|TestScriptsCmdOutputsAllRegisteredScripts'`
Expected: FAIL，失败点指向尚未实现的命令逻辑

### Task 2: 实现命令

**Files:**
- Create: `cmd/scripts.go`
- Modify: `cmd/root.go`
- Test: `cmd/scripts_test.go`

- [ ] **Step 1: 实现名称收集函数**

```go
func registeredScriptNames() []string {
    // 从 v8.Scripts 取 key，排序后返回
}
```

- [ ] **Step 2: 实现 `scripts` 命令**

```go
var scriptsCmd = &cobra.Command{
    Use:   "scripts",
    Short: L("Show registered scripts"),
    Long:  L("Show registered scripts"),
}
```

- [ ] **Step 3: 在命令执行中加载应用并输出名称**

```go
Boot()
err := engine.Load(config.Conf, engine.LoadOption{Action: "scripts"})
```

- [ ] **Step 4: 把命令注册到 `rootCmd`**

Run: 修改 `cmd/root.go` 的 `rootCmd.AddCommand(...)`
Expected: `yao scripts` 成为根命令可用子命令

### Task 3: 验证

**Files:**
- Modify: `cmd/scripts.go`
- Test: `cmd/scripts_test.go`

- [ ] **Step 1: 运行命令测试**

Run: `go test ./cmd -run 'TestRegisteredScriptNamesSorted|TestScriptsCmdOutputsAllRegisteredScripts'`
Expected: PASS

- [ ] **Step 2: 运行 `cmd` 包全量测试**

Run: `go test ./cmd`
Expected: PASS

- [ ] **Step 3: 更新本计划和 `tasks/todo.md` 的 Review**

Run: 补充结果说明
Expected: 文档与实际实现一致

### Task 4: 增加 `-m` 过滤能力

**Files:**
- Modify: `cmd/scripts.go`
- Modify: `cmd/scripts_test.go`
- Test: `cmd/scripts_test.go`

- [ ] **Step 1: 先写失败测试，覆盖 `-m` 精确匹配**

```go
func TestFilterScriptNamesExactMatch(t *testing.T) {
    // 断言 a.b 只命中 a.b
}
```

- [ ] **Step 2: 写失败测试，覆盖 glob 匹配和多模式 OR**

```go
func TestFilterScriptNamesWildcardMatch(t *testing.T) {}
func TestFilterScriptNamesMultiplePatterns(t *testing.T) {}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./cmd -run 'TestFilterScriptNames'`
Expected: FAIL，失败点指向尚未实现的过滤逻辑

- [ ] **Step 4: 实现 `-m, --match` flag 和过滤函数**

```go
func filterScriptNames(names []string, patterns []string) ([]string, error) {
    // 无模式返回全部；有模式时命中任一模式即保留
}
```

- [ ] **Step 5: 在命令输出前应用过滤**

Run: 修改 `printRegisteredScripts(...)`
Expected: `yao scripts -m "*.xxx" -m "a.b"` 只输出匹配项

- [ ] **Step 6: 运行针对性测试和 `go test ./cmd`**

Run: `go test ./cmd -run 'TestFilterScriptNames|TestScriptsCmd' && go test ./cmd`
Expected: PASS

### Task 5: 增加聚合错误与 `--error`

**Files:**
- Modify: `script/script.go`
- Modify: `script/script_test.go`
- Modify: `cmd/scripts.go`
- Modify: `cmd/scripts_test.go`
- Modify: `engine/load.go`
- Modify: `../gou/runtime/v8/script.go`

- [ ] **Step 1: 先写失败测试，覆盖 `script.Load` 遇错继续扫描**

```go
func TestLoadAggregatesErrorsAndContinues(t *testing.T) {
    // 坏 ts 不阻断后续 good ts / service 加载
}
```

- [ ] **Step 2: 再写失败测试，覆盖 `scripts --error` 只输出错误**

```go
func TestScriptsCmdOutputsOnlyScriptErrors(t *testing.T) {
    // 断言只输出 file:line:column text
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./script ./cmd -run 'TestLoadAggregatesErrorsAndContinues|TestScriptsCmdOutputsOnlyScriptErrors'`
Expected: FAIL，失败点指向尚未实现的聚合错误逻辑

- [ ] **Step 4: 在 `v8` 层补结构化 TS 错误**

```go
type TransformError struct {
    Items []TransformErrorItem
}
```

- [ ] **Step 5: 在 `script.Load` 中聚合错误并继续扫描**

```go
func Load(cfg config.Config) error {
    // 单文件失败时记录错误并返回 nil，让 Walk 继续
}
```

- [ ] **Step 6: 给 `scripts` 命令增加 `-e, --error`**

Run: 修改 `cmd/scripts.go`
Expected: `yao scripts --error` 只打印脚本错误，不打印正常脚本名

- [ ] **Step 7: 给脚本命令增加轻量加载路径或脚本错误读取入口**

Run: 修改 `engine/load.go`
Expected: `scripts` 相关 action 不被其他加载器错误干扰

- [ ] **Step 8: 运行针对性测试和回归**

Run: `go test ./script ./cmd && go test ./engine`
Expected: PASS
