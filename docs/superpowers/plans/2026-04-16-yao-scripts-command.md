# yao scripts 命令 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `yao scripts` 根命令，在应用加载完成后输出排序后的全部已注册脚本名称。

**Architecture:** 命令层负责触发 `Boot()` 和 `engine.Load(...)`，名称收集逻辑单独做成小函数，直接从 `v8.Scripts` 读取并排序，保证测试可以在不启动完整引擎的情况下验证输出行为。

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
