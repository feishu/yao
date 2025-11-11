# await 工具模块

## 概述

`await` 模块是 Yao 项目中的 JavaScript Promise 同步等待工具，支持在 Go 代码中同步等待 JavaScript Promise 解析完成。该模块经过优化，确保了高稳定性、安全性和性能。

## 主要功能

- **同步等待 Promise**：支持在 Go 中同步等待 JavaScript Promise 解析
- **超时控制**：可配置的超时机制，避免无限等待
- **V8 资源管理**：正确的 V8 对象资源释放，防止内存泄漏
- **并发安全**：支持多线程环境使用
- **错误处理**：详细的错误信息记录和异常处理
- **性能优化**：自适应时间间隔和内存优化

## 核心特性

### 1. Register 函数
```go
// 将 syncAwait 函数注入到 V8 上下文中
Register(ctx *v8.Context)
```

在 JavaScript 中使用：
```javascript
const result = await syncAwait(promise, timeoutMs);
```

### 2. ProcessAwait 函数
```go
// 处理 await 表达式的 Process 接口实现
func ProcessAwait(process *process.Process) interface{}
```

### 3. 配置管理
```go
// 设置超时配置（线程安全）
SetTimeoutConfig(defaultTimeout, maxTimeout, minTimeout time.Duration)
```

## 使用示例

### 基本用法

```go
package main

import (
    "context"
    v8 "github.com/yaoapp/gou/runtime/v8"
    "github.com/yaoapp/yao/utils/await"
)

func main() {
    // 创建 V8 上下文
    iso := v8go.NewIsolate()
    ctx := v8.NewContext(iso)
    // 使用 await 等待 Promise
    result, err := await.ProcessAwait(process)
    if err != nil {
        log.Error("Promise error: %v", err)
        return
    }
    
    log.Info("Result: %v", result)
}
```

### 配置超时

```go
// 设置超时配置
await.SetTimeoutConfig(
    10 * time.Second,  // 默认超时
    60 * time.Second,  // 最大超时
    100 * time.Millisecond,  // 最小超时
)
```

### JavaScript 中使用

```javascript
// 在 JavaScript 中使用 syncAwait
const result = await syncAwait(somePromise, 5000); // 5秒超时

// 或者不指定超时（使用默认超时）
const result = await syncAwait(somePromise);
```

## 性能特性

### 1. 内存优化
- 使用 `defer` 确保 V8 对象正确释放
- 预分配切片容量，避免动态扩容
- 自动释放临时创建的 JS 值

### 2. 时间效率
- 自适应检查间隔：根据剩余时间动态调整
- 避免 CPU 100% 占用
- 高效的微任务检查机制

### 3. 并发安全
- 使用 `sync.RWMutex` 保护全局配置
- 线程安全的配置更新
- 安全的资源管理

## 错误处理

### 错误类型
- `invalid context or promise`：无效的上下文或 Promise
- `operation cancelled`：操作被取消
- `promise timeout after xxx`：Promise 超时
- `promise rejected`：Promise 被拒绝
- `function call failed`：函数调用失败

### 日志记录
```go
// 成功解析
log.Debug("Promise resolved successfully")

// 解析被拒绝
log.Warn("Promise rejected: %v", err)

// 处理失败
log.Warn("ProcessAwait failed: %v", err)
```

## 配置选项

### 环境变量支持
后续版本将支持通过环境变量配置：
- `YAO_AWAIT_DEFAULT_TIMEOUT`：默认超时时间
- `YAO_AWAIT_MAX_TIMEOUT`：最大超时时间
- `YAO_AWAIT_MIN_TIMEOUT`：最小超时时间

## 测试

### 运行测试
```bash
go test -v ./utils/await/
```

### 性能测试
```bash
go test -bench=. ./utils/await/
```

### 测试覆盖
- ✅ 基础功能测试
- ✅ 错误处理测试
- ✅ 并发安全测试
- ✅ 性能基准测试
- ✅ 资源管理测试

## 最佳实践

### 1. 资源管理
```go
// 自动释放 V8 资源
result, err := waitPromiseWithContext(ctx, v8ctx, promise, timeout)
// V8 对象会自动通过 defer 释放
```

### 2. 错误处理
```go
result, err := ProcessAwait(process)
if err != nil {
    // 记录错误并返回
    return fmt.Errorf("await failed: %v", err)
}
```

### 3. 超时设置
```go
// 根据操作类型设置合适的超时
SetTimeoutConfig(
    5 * time.Second,  // 快速操作
    30 * time.Second, // 较慢操作
    100 * time.Millisecond,
)
```

## 注意事项

1. **V8 资源管理**：始终确保 V8 对象的正确释放
2. **超时设置**：根据实际需求设置合理的超时时间
3. **并发使用**：模块支持多线程使用，但应避免共享同一个 V8 上下文
4. **错误处理**：始终检查和处理可能的错误
5. **性能监控**：在高并发场景下监控性能指标

## 版本信息

- **当前版本**：v1.0.0
- **兼容性**：兼容 v8go 0.7+
- **Go 版本要求**：Go 1.18+

## 贡献

请遵循 Yao 项目的开发规范：
- [代码规范](.cursorrules)
- [测试要求](#测试)
- [提交信息规范](#版本管理和发布规范)

---

最后更新：2025-03-24