# 消息模板功能

基于 Go `text/template` 引擎的消息模板系统，支持动态变量替换和模板管理。模板内容存储在 Redis 中，提供高性能的分布式存储。

## 功能特性

- **Go 模板引擎**: 使用 `text/template` 进行模板渲染
- **Redis 存储**: 模板内容存储在 Redis 中，支持分布式部署
- **编译缓存**: 已编译的模板缓存在内存中，提高渲染性能
- **并发安全**: 使用互斥锁确保并发操作安全
- **Process 方法**: 提供 6 个 process 方法用于模板操作
- **错误处理**: 完善的错误处理和异常管理

## 使用方法

### 1. 注册模板

```go
// 通过process方法注册
yao.template.register("welcome", "Hello {{.name}}, welcome to {{.platform}}!")
```

### 2. 渲染模板

```go
// 通过process方法渲染
result := yao.template.render("welcome", {
    "name": "张三",
    "platform": "YAO"
})
// 输出: "Hello 张三, welcome to YAO!"
```

### 3. 获取模板

```go
// 获取模板内容
template := yao.template.get("welcome")
```

### 4. 列出所有模板

```go
// 列出所有已注册的模板
templates := yao.template.list()
```

### 5. 移除模板

```go
// 移除指定模板
yao.template.remove("welcome")
```

### 6. 清除缓存

```go
// 清除模板缓存
yao.template.clear()
```

## Process方法列表

| 方法 | 参数 | 返回值 | 说明 |
|------|------|--------|------|
| `render` | code(string), data(map) | string | 渲染模板 |
| `register` | code(string), content(string) | object | 注册模板 |
| `get` | code(string) | object | 获取模板内容 |
| `list` | - | object | 列出所有模板 |
| `remove` | code(string) | object | 移除模板 |
| `clear` | - | object | 清除缓存 |

## 模板语法

支持Go标准模板语法：

```go
// 变量替换
"Hello {{.name}}"

// 条件判断
"{{if .isVip}}VIP用户{{else}}普通用户{{end}}"

// 循环
"{{range .items}}{{.}}{{end}}"

// 函数调用
"{{.name | upper}}"
```

## 错误处理

所有Process方法都包含完善的错误处理：

- 模板不存在：返回404错误
- 模板解析失败：返回400错误
- 模板渲染失败：返回400错误

## 性能优化

- **模板缓存**：已编译的模板会被缓存，避免重复解析
- **并发安全**：使用读写锁确保并发访问安全
- **内存管理**：支持手动清除缓存释放内存

## 示例

### 邮件模板示例

```go
// 注册邮件模板
yao.template.register("email_welcome", `
亲爱的 {{.username}}：

欢迎加入 {{.platform}}！

您的账户信息：
- 用户名：{{.username}}
- 邮箱：{{.email}}
- 注册时间：{{.registerTime}}

{{if .isVip}}
恭喜您成为VIP用户，享受专属服务！
{{end}}

祝您使用愉快！

{{.platform}} 团队
`)

// 渲染邮件内容
content := yao.template.render("email_welcome", {
    "username": "张三",
    "platform": "YAO平台",
    "email": "zhangsan@example.com",
    "registerTime": "2024-01-15 10:30:00",
    "isVip": true
})
```

### 短信模板示例

```go
// 注册短信模板
yao.template.register("sms_verify", "【{{.platform}}】您的验证码是：{{.code}}，有效期{{.expiry}}分钟，请勿泄露。")

// 渲染短信内容
sms := yao.template.render("sms_verify", {
    "platform": "YAO",
    "code": "123456",
    "expiry": 5
})
// 输出: "【YAO】您的验证码是：123456，有效期5分钟，请勿泄露。"
```