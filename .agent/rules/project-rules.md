---
trigger: always_on
---

# Yao 项目开发规则文档

## 1. 项目概述

Yao 是一个全能应用引擎，让开发者能够创建 Web 应用、REST API、业务应用等，并以 AI 作为开发伙伴。Yao 支持使用 AI、可视化界面或手动编码来创建应用，其 DSL（领域特定语言）易于读写，与 AI 配合良好。

### 1.1 核心特性
- **AI 优先**：人机友好的 DSL 设计，支持 AI 生成和手动编写代码的无缝切换
- **一体化解决方案**：单一可执行文件包含所有必需功能
- **原生 TypeScript 支持**：内置 V8 引擎，支持 TypeScript 直接执行
- **多种编码方式**：AI 生成、可视化编辑和手动编码可在同一项目中结合使用
- **无服务器架构**：内置云函数和 API 网关
- **边缘设备支持**：支持 arm64 和 x64 芯片的边缘设备

### 1.2 技术栈
- **语言**：Go 1.23.0+
- **核心依赖**：
  - github.com/yaoapp/gou (核心框架)
  - github.com/yaoapp/kun (工具库)
  - github.com/yaoapp/xun (数据库抽象层)
  - rogchap.com/v8go (V8 JavaScript 引擎)
- **Web 框架**：Gin
- **数据库**：支持 MySQL、PostgreSQL、SQLite、MongoDB
- **前端**：React + TypeScript (Xgen UI 框架)

## 2. 项目结构规范

### 2.1 顶层目录结构

```
yao/
├── .github/           # GitHub 配置和工作流
├── .trae/            # Trae 工具配置
├── .vscode/          # VS Code 配置
├── cmd/              # 命令行接口实现
├── config/           # 配置管理
├── data/             # 静态资源和数据文件
├── docs/             # 项目文档
├── docker/           # Docker 相关文件
├── scripts/          # 构建和部署脚本
├── test/             # 测试工具和辅助函数
├── main.go           # 程序入口点
├── Makefile          # 构建配置
├── go.mod            # Go 模块定义
└── README.md         # 项目说明
```

### 2.2 功能模块目录

```
├── aigc/             # AI 生成内容模块
├── api/              # API 处理模块
├── cert/             # 证书管理
├── connector/        # 连接器模块
├── crypto/           # 加密解密功能
├── engine/           # 核心引擎
├── excel/            # Excel 处理
├── flow/             # 工作流引擎
├── fs/               # 文件系统抽象
├── helper/           # 辅助函数库
├── i18n/             # 国际化支持
├── importer/         # 数据导入模块
├── model/            # 数据模型
├── neo/              # AI 助手模块
├── openai/           # OpenAI 集成
├── pipe/             # 管道处理
├── plugin/           # 插件系统
├── query/            # 查询构建器
├── runtime/          # 运行时环境
├── schedule/         # 任务调度
├── script/           # 脚本执行
├── service/          # 服务层
├── setup/            # 初始化设置
├── share/            # 共享常量和工具
├── socket/           # WebSocket 支持
├── store/            # 存储抽象
├── studio/           # 开发工具
├── sui/              # SUI 界面框架
├── task/             # 任务管理
├── types/            # 类型定义
├── utils/            # 通用工具函数
├── volcengine/       # 火山引擎集成
├── websocket/        # WebSocket 实现
├── wework/           # 企业微信集成
├── widget/           # 组件系统
├── widgets/          # 内置组件
└── xgen/             # Xgen UI 框架
```

### 2.3 模块内部结构规范

每个功能模块应遵循以下结构：

```
module_name/
├── module_name.go      # 主要实现文件
├── module_name_test.go # 单元测试
├── process.go          # Process 接口实现
├── process_test.go     # Process 测试
├── types.go            # 类型定义
├── load.go             # 加载和初始化
├── load_test.go        # 加载测试
└── README.md           # 模块文档
```

## 3. Go 代码开发规范

### 3.1 代码风格

遵循 [Effective Go](https://go.dev/doc/effective_go) 和项目 `.cursorrules` 中定义的规范：

#### 3.1.1 命名规范
- **包名**：小写，简短，有意义的名词
- **函数名**：驼峰命名，公开函数首字母大写
- **变量名**：驼峰命名，简洁明了
- **常量名**：全大写，下划线分隔
- **接口名**：以 -er 结尾（如 Reader, Writer）

#### 3.1.2 注释规范
```go
// Package template 提供模板渲染功能
package template

// RenderTemplate 渲染模板
// code: 模板代码
// data: 模板变量数据
func RenderTemplate(code string, data map[string]interface{}) (string, error) {
    // 实现逻辑
}
```

#### 3.1.3 错误处理
```go
// 优先使用 fmt.Errorf 包装错误
if err != nil {
    return fmt.Errorf("failed to parse template '%s': %v", code, err)
}

// 使用 kun/log 进行日志记录
log.Error("RenderTemplate failed: %v", err)
log.Debug("Template '%s' compiled successfully", code)
```

### 3.2 架构设计原则

#### 3.2.1 Clean Architecture
- **Entities**：核心业务实体（types.go）
- **Use Cases**：业务逻辑（主要实现文件）
- **Interface Adapters**：接口适配（process.go）
- **Frameworks**：框架和驱动（load.go）

#### 3.2.2 依赖管理
```go
// 标准库导入
import (
    "fmt"
    "sync"
)

// 第三方库导入
import (
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

// 项目内部导入
import (
    "github.com/yaoapp/gou/process"
    "github.com/yaoapp/kun/log"
)
```

#### 3.2.3 并发处理
```go
// 使用 sync.RWMutex 保护共享资源
var (
    cache = make(map[string]*Template)
    mutex sync.RWMutex
)

// 使用 context 管理协程生命周期
func ProcessWithContext(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // 处理逻辑
    }
}
```

### 3.3 Process 接口规范

每个模块都应实现 Process 接口：

```go
// ProcessRender 渲染模板的 process 方法
func ProcessRender(process *process.Process) interface{} {
    process.ValidateArgNums(2) // 验证参数数量
    
    code := process.ArgsString(0)
    data := process.ArgsMap(1)
    
    result, err := RenderTemplate(code, data)
    if err != nil {
        exception.New(err.Error(), 500).Throw()
    }
    
    return result
}
```

## 4. 测试规范

### 4.1 测试文件组织

- 每个 `.go` 文件对应一个 `_test.go` 文件
- 测试文件与源文件在同一包中
- 使用 `github.com/stretchr/testify/assert` 进行断言

### 4.2 测试函数命名

```go
func TestFunctionName(t *testing.T) {
    // 测试实现
}

func BenchmarkFunctionName(b *testing.B) {
    // 性能测试
}

func ExampleFunctionName() {
    // 示例测试
}
```

### 4.3 Table-Driven 测试

```go
func TestRenderTemplate(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        data     map[string]interface{}
        expected string
        hasError bool
    }{
        {
            name:     "Valid template",
            code:     "hello",
            data:     map[string]interface{}{"name": "world"},
            expected: "Hello, world!",
            hasError: false,
        },
        {
            name:     "Empty code",
            code:     "",
            data:     map[string]interface{}{},
            expected: "",
            hasError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := RenderTemplate(tt.code, tt.data)
            if tt.hasError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### 4.4 测试覆盖率

- 目标覆盖率：80% 以上
- 使用 `make test` 运行测试并生成覆盖率报告
- 关键业务逻辑必须有完整的测试覆盖

### 4.5 测试环境设置

```go
func TestMain(m *testing.M) {
    // 测试前设置
    setup()
    
    // 运行测试
    code := m.Run()
    
    // 测试后清理
    cleanup()
    
    os.Exit(code)
}

func prepare(t *testing.T) {
    // 测试准备工作
    config.Conf = &config.Config{
        Mode: "test",
    }
}

func clean() {
    // 测试清理工作
}
```

## 5. 构建和部署流程

### 5.1 本地开发

```bash
# 安装依赖
go mod tidy

# 代码格式化
make fmt

# 静态检查
make vet
make lint

# 运行测试
make test

# 构建二进制文件
go build -o yao .
```

### 5.2 Makefile 目标

- `make test`：运行单元测试
- `make fmt`：格式化代码
- `make lint`：代码静态检查
- `make pack`：打包资源文件
- `make artifacts-linux`：构建 Linux 制品
- `make artifacts-macos`：构建 macOS 制品

### 5.3 Docker 构建

```dockerfile
# 多阶段构建
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o yao .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/yao .
CMD ["./yao"]
```

### 5.4 CI/CD 流程

GitHub Actions 工作流：
- **单元测试**：每次 PR 和 push 触发
- **代码质量检查**：golint、go vet、misspell
- **构建测试**：多平台构建验证
- **发布流程**：标签推送时自动发布

## 6. 版本管理和发布规范

### 6.1 版本号规范

遵循 [语义化版本](https://semver.org/lang/zh-CN/)：
- **主版本号**：不兼容的 API 修改
- **次版本号**：向下兼容的功能性新增
- **修订号**：向下兼容的问题修正

当前版本定义在 `share/const.go`：
```go
const VERSION = "0.10.5"
const PRVERSION = "f393289387f0-2025-03-24T14:30:18+0800-debug"
```

### 6.2 分支管理

- **main**：主分支，稳定版本
- **develop**：开发分支，集成新功能
- **feature/**：功能分支
- **hotfix/**：热修复分支
- **release/**：发布分支

### 6.3 提交信息规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型（type）：
- `feat`：新功能
- `fix`：修复 bug
- `docs`：文档更新
- `style`：代码格式调整
- `refactor`：重构
- `test`：测试相关
- `chore`：构建过程或辅助工具的变动

### 6.4 发布流程

1. 创建 release 分支
2. 更新版本号和 CHANGELOG
3. 运行完整测试套件
4. 合并到 main 分支
5. 创建 Git 标签
6. 自动构建和发布制品

## 7. 文档和注释规范

### 7.1 代码注释

#### 7.1.1 包注释
```go
// Package template 提供模板渲染功能，支持从 Redis 缓存加载模板内容，
// 并提供高性能的模板编译和渲染服务。
package template
```

#### 7.1.2 函数注释
```go
// RenderTemplate 渲染模板
// 
// 参数：
//   - code: 模板代码，用于从缓存中获取模板内容
//   - data: 模板变量数据，用于填充模板占位符
//
// 返回：
//   - string: 渲染后的内容
//   - error: 渲染过程中的错误
//
// 示例：
//   result, err := RenderTemplate("welcome", map[string]interface{}{
//       "name": "张三",
//       "role": "管理员",
//   })
func RenderTemplate(code string, data map[string]interface{}) (string, error) {
    // 实现逻辑
}
```

#### 7.1.3 类型注释
```go
// Template 表示一个编译后的模板实例
type Template struct {
    Code     string                 // 模板代码
    Content  string                 // 模板内容
    Data     map[string]interface{} // 模板数据
    Compiled *template.Template     // 编译后的模板
}
```

### 7.2 README 文档

每个模块都应包含 README.md：

```markdown
# 模块名称

## 概述
模块的简要描述和用途

## 功能特性
- 功能点 1
- 功能点 2

## 使用示例
```go
// 代码示例
```

## API 参考
详细的 API 文档

## 配置说明
配置参数和选项

## 注意事项
使用时的注意事项和限制
```

### 7.3 API 文档

使用 godoc 格式编写 API 文档：

```go
// ProcessRender 是模板渲染的 Process 接口实现
//
// 参数：
//   args[0] (string): 模板代码
//   args[1] (map[string]interface{}): 模板数据
//
// 返回：
//   interface{}: 渲染结果字符串
//
// 异常：
//   当模板不存在或渲染失败时抛出异常
//
// 示例：
//   process.New("utils.template.Render", "welcome", data).Run()
func ProcessRender(process *process.Process) interface{} {
    // 实现
}
```

## 8. 质量保障

### 8.1 代码质量检查

- **go fmt**：代码格式化
- **go vet**：静态分析
- **golint**：代码风格检查
- **misspell**：拼写检查
- **race detector**：数据竞争检测

### 8.2 性能监控

- 使用 `pprof` 进行性能分析
- 编写 benchmark 测试
- 监控内存使用和 GC 性能
- 关键路径的性能指标收集

### 8.3 安全规范

- 输入验证和清理
- SQL 注入防护
- XSS 攻击防护
- 敏感信息加密存储
- 访问控制和权限管理

### 8.4 日志规范

使用 `github.com/yaoapp/kun/log` 进行日志记录：

```go
// 日志级别
log.Debug("调试信息: %s", debugInfo)
log.Info("一般信息: %s", info)
log.Warn("警告信息: %s", warning)
log.Error("错误信息: %v", err)
log.Fatal("致命错误: %v", fatalErr)

// 结构化日志
log.With(log.F{
    "module": "template",
    "action": "render",
    "code":   templateCode,
}).Info("模板渲染完成")
```

## 9. 开发工具和环境

### 9.1 推荐开发环境

- **Go 版本**：1.23.0+
- **编辑器**：VS Code + Go 扩展
- **调试工具**：Delve
- **性能分析**：go tool pprof
- **依赖管理**：Go Modules

### 9.2 VS Code 配置

`.vscode/settings.json`：
```json
{
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golint",
    "go.vetOnSave": "package",
    "go.coverOnSave": true,
    "go.testFlags": ["-v", "-race"]
}
```

### 9.3 Git Hooks

使用 pre-commit hooks 确保代码质量：

```bash
#!/bin/sh
# .git/hooks/pre-commit

# 运行格式化
make fmt

# 运行静态检查
make vet

# 运行测试
make test

# 检查是否有未提交的更改
if ! git diff --cached --exit-code; then
    echo "请先提交格式化后的代码"
    exit 1
fi
```

## 10. 故障排查和调试

### 10.1 日志分析

- 使用结构化日志便于查询和分析
- 关键操作记录详细的上下文信息
- 错误日志包含完整的调用栈

### 10.2 性能调试

```go
// 使用 pprof 进行性能分析
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // 主程序逻辑
}
```

### 10.3 内存泄漏检测

- 定期运行内存分析
- 监控 goroutine 数量
- 检查资源释放情况

---

本文档将随着项目的发展持续更新，请开发团队严格遵循以上规范，确保代码质量和项目的可维护性。

最后更新：2025-03-24
版本：v1.0.0